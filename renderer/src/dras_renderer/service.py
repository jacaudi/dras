"""Orchestrate cache → s3 → decode → render for one render request."""

from __future__ import annotations

import threading
from dataclasses import dataclass, replace
from datetime import UTC, datetime

from cachetools import LRUCache

from dras_renderer.cache import RenderCache
from dras_renderer.decode import DecodedScan, decode_level2_archive
from dras_renderer.metrics import S3_ERRORS_TOTAL
from dras_renderer.render import RenderOptions, render_base_reflectivity
from dras_renderer.s3 import S3Client, S3Error
from dras_renderer.version import VERSION


@dataclass(frozen=True)
class RenderRequest:
    station: str
    product: str = "base_reflectivity"
    range_km: float = 230.0
    width: int = 800
    height: int = 800
    # Optional view-center override. None means "center on the radar".
    center_lat: float | None = None
    center_lon: float | None = None


@dataclass(frozen=True)
class RenderMetadata:
    station: str
    product: str
    scan_time: datetime
    elevation_deg: float
    vcp: int
    renderer_version: str
    # Seconds elapsed between the newest chunk's S3 LastModified and the moment
    # we (re)entered the rendering step. Recomputed on cache hits so the value
    # always reflects "right now", not when the PNG was rendered.
    data_age_at_render: float


@dataclass(frozen=True)
class RenderResponse:
    png: bytes
    metadata: RenderMetadata


def _fmt_coord(v: float | None) -> str:
    """Format a coord for the cache key. ``None`` → ``"-"``."""
    return "-" if v is None else f"{v:.3f}"


class ServiceError(Exception):
    """Domain-level error with a stable code for HTTP mapping."""

    def __init__(self, code: str, detail: str) -> None:
        super().__init__(f"{code}: {detail}")
        self.code = code
        self.detail = detail


class RenderService:
    """Single-product orchestration. v1 supports base_reflectivity only."""

    def __init__(self, *, s3: S3Client, cache: RenderCache) -> None:
        self._s3 = s3
        self._cache = cache
        # Parallel LRUCache keyed identically to the PNG cache, holding metadata
        # so cache hits can return the same envelope as fresh renders.
        # maxsize=256 matches typical NEXRAD station count (~160) with headroom.
        # cachetools caches are not thread-safe; _render_lock serializes access.
        self._meta: LRUCache[tuple[str, str], RenderMetadata] = LRUCache(maxsize=256)
        self._render_lock = threading.Lock()

    def render(self, req: RenderRequest) -> RenderResponse:
        if req.product != "base_reflectivity":
            raise ServiceError(
                "unsupported_product",
                f"product {req.product!r} not supported in v1",
            )

        # Serialize renders: cachetools caches are not thread-safe, and renders
        # are dispatched from the asyncio threadpool (asyncio.to_thread). Lock
        # contention cost is dominated by the render itself (seconds), so the
        # serialization penalty is negligible.
        with self._render_lock:
            try:
                volume = self._s3.latest_volume(req.station)
            except S3Error as exc:
                S3_ERRORS_TOTAL.inc()
                raise ServiceError("internal", f"S3 list failed: {exc}") from exc
            if volume is None:
                raise ServiceError(
                    "no_recent_scan",
                    f"no Level II volume found for {req.station}",
                )

            # The volume's latest_chunk_time uniquely identifies a snapshot
            # (volume slot numbers are reused). Cache key folds in render
            # parameters that change pixel output — different center/range/
            # size combos must not share a cache slot.
            cache_key = (
                f"{volume.latest_chunk_time.isoformat()}"
                f"|{req.width}x{req.height}"
                f"|r{req.range_km:.1f}"
                f"|c{_fmt_coord(req.center_lat)},{_fmt_coord(req.center_lon)}"
            )
            cached_png = self._cache.get(req.station, cache_key)
            cached_meta = self._meta.get((req.station, cache_key))
            if cached_png is not None and cached_meta is not None:
                # Recompute data_age_at_render so the cached envelope reports
                # current age, not the age at the time of the original render.
                fresh_age = (
                    datetime.now(UTC) - volume.latest_chunk_uploaded_at
                ).total_seconds()
                return RenderResponse(
                    png=cached_png,
                    metadata=replace(cached_meta, data_age_at_render=fresh_age),
                )

            try:
                volume_bytes = self._s3.download_volume(volume)
            except S3Error as exc:
                S3_ERRORS_TOTAL.inc()
                raise ServiceError("internal", f"S3 download failed: {exc}") from exc

            try:
                decoded: DecodedScan = decode_level2_archive(volume_bytes)
            except ValueError as exc:
                raise ServiceError("decode_failed", str(exc)) from exc

            opts = RenderOptions(
                width=req.width,
                height=req.height,
                range_km=req.range_km,
                center_lat=req.center_lat,
                center_lon=req.center_lon,
            )
            # Capture render_time as close to the render call as possible so
            # data_age_at_render reflects the freshness perceived by callers.
            render_time = datetime.now(UTC)
            data_age = (render_time - volume.latest_chunk_uploaded_at).total_seconds()
            png = render_base_reflectivity(decoded, opts)

            meta = RenderMetadata(
                station=req.station,
                product="base_reflectivity",
                scan_time=decoded.scan_time,
                elevation_deg=decoded.elevation_deg,
                vcp=decoded.vcp,
                renderer_version=VERSION,
                data_age_at_render=data_age,
            )
            self._cache.set(req.station, cache_key, png)
            self._meta[(req.station, cache_key)] = meta

            return RenderResponse(png=png, metadata=meta)

"""Orchestrate cache → s3 → decode → render for one render request."""

from __future__ import annotations

import threading
from dataclasses import dataclass
from datetime import datetime

from cachetools import LRUCache

from dras_renderer.cache import RenderCache
from dras_renderer.decode import DecodedScan, decode_level2_archive
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


@dataclass(frozen=True)
class RenderMetadata:
    station: str
    product: str
    scan_time: datetime
    elevation_deg: float
    vcp: int
    renderer_version: str


@dataclass(frozen=True)
class RenderResponse:
    png: bytes
    metadata: RenderMetadata


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
            volume = self._s3.latest_volume(req.station)
            if volume is None:
                raise ServiceError(
                    "no_recent_scan",
                    f"no Level II volume found for {req.station}",
                )

            # The volume's latest_chunk_time uniquely identifies a snapshot
            # (volume slot numbers are reused). Use ISO timestamp as the cache key.
            cache_key = volume.latest_chunk_time.isoformat()
            cached_png = self._cache.get(req.station, cache_key)
            cached_meta = self._meta.get((req.station, cache_key))
            if cached_png is not None and cached_meta is not None:
                return RenderResponse(png=cached_png, metadata=cached_meta)

            try:
                volume_bytes = self._s3.download_volume(volume)
            except S3Error as exc:
                raise ServiceError("internal", f"S3 download failed: {exc}") from exc

            try:
                decoded: DecodedScan = decode_level2_archive(volume_bytes)
            except ValueError as exc:
                raise ServiceError("decode_failed", str(exc)) from exc

            opts = RenderOptions(width=req.width, height=req.height, range_km=req.range_km)
            png = render_base_reflectivity(decoded, opts)

            meta = RenderMetadata(
                station=req.station,
                product="base_reflectivity",
                scan_time=decoded.scan_time,
                elevation_deg=decoded.elevation_deg,
                vcp=decoded.vcp,
                renderer_version=VERSION,
            )
            self._cache.set(req.station, cache_key, png)
            self._meta[(req.station, cache_key)] = meta

            return RenderResponse(png=png, metadata=meta)

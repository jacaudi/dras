"""Chunks-bucket client for ``s3://unidata-nexrad-level2-chunks/``.

Volumes cycle 0-999 in dedicated slot directories. The freshest volume
for a station is the one whose newest chunk filename has the largest
``YYYYMMDD-HHMMSS`` prefix. Lex-order matches chronological within and
across volumes because the chunk filename starts with the volume's start
timestamp. We discover this by fanning out one LIST per slot, picking
the slot with the latest chunk, and (when caching is enabled) memoizing
the result per-station for a short TTL. In-progress volumes are returned
as-is — the renderer will operate on whatever chunks are present.
"""

from __future__ import annotations

import bz2
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass
from datetime import UTC, datetime
from typing import Any, cast

import boto3  # type: ignore[import-untyped]
from botocore import UNSIGNED  # type: ignore[import-untyped]
from botocore.config import Config as BotocoreConfig  # type: ignore[import-untyped]
from botocore.exceptions import ClientError  # type: ignore[import-untyped]
from cachetools import TTLCache

VOLUME_SLOTS: range = range(1000)
"""Volume IDs cycle in [0, 999]."""


class S3Error(Exception):
    """Generic failure talking to S3."""


class VolumeNotFoundError(S3Error):
    """No chunks present for the station in any volume slot."""


@dataclass(frozen=True)
class LatestVolume:
    """Pointer to the freshest volume in the chunks bucket for a station."""

    station_id: str
    volume_number: int
    chunk_keys: tuple[str, ...]  # sorted by chunk-num (== lex order due to zero-padding)
    latest_chunk_time: datetime


def _make_config(anonymous: bool) -> BotocoreConfig:
    kwargs: dict[str, Any] = {
        "connect_timeout": 5,
        "read_timeout": 15,
        "retries": {"max_attempts": 3, "mode": "standard"},
        "max_pool_connections": 64,
    }
    if anonymous:
        kwargs["signature_version"] = UNSIGNED
    return BotocoreConfig(**kwargs)


class S3Client:
    """Anonymous S3 client for the NOAA chunks bucket."""

    def __init__(
        self,
        *,
        bucket: str,
        region: str,
        anonymous: bool = True,
        list_workers: int = 64,
        latest_volume_ttl: float = 30.0,
    ) -> None:
        self.bucket = bucket
        self.region = region
        self.list_workers = list_workers
        self.latest_volume_ttl = latest_volume_ttl
        self._client: Any = boto3.client(
            "s3", region_name=region, config=_make_config(anonymous)
        )
        # Negative results (None for unknown stations) are cached too — that's
        # deliberate, to suppress 1000-LIST fan-out on typo'd station IDs.
        self._latest_cache: TTLCache[str, LatestVolume | None] = TTLCache(
            maxsize=256, ttl=latest_volume_ttl,
        )

    def latest_volume(self, station_id: str) -> LatestVolume | None:
        """Return the freshest volume for ``station_id``, or ``None`` if no chunks exist.

        In-progress volumes (no ``E`` chunk yet) are returned as the freshest
        if they win on chunk-timestamp. Result is cached per-station for
        ``self.latest_volume_ttl`` seconds; the cached value is returned by
        identity, so repeated callers within the TTL all see the same instance.
        """
        try:
            return self._latest_cache[station_id]
        except KeyError:
            pass
        result = self._compute_latest_volume(station_id)
        self._latest_cache[station_id] = result
        return result

    def _compute_latest_volume(self, station_id: str) -> LatestVolume | None:
        def chunks_for(vol_num: int) -> tuple[int, list[str]]:
            return vol_num, self._list_keys(f"{station_id}/{vol_num}/")

        with ThreadPoolExecutor(max_workers=self.list_workers) as executor:
            results = list(executor.map(chunks_for, VOLUME_SLOTS))

        non_empty = [(v, sorted(keys)) for v, keys in results if keys]
        if not non_empty:
            return None

        # The volume with the largest "newest chunk filename" wins.
        # Compare on the filename only (after last '/') so that slot numbers
        # like 5 vs 17 don't distort the lex comparison — only the
        # YYYYMMDD-HHMMSS prefix in the filename determines recency.
        best_vol, best_chunks = max(non_empty, key=lambda item: item[1][-1].rsplit("/", 1)[-1])
        latest_filename = best_chunks[-1].rsplit("/", 1)[-1]
        ts_prefix = latest_filename[:15]  # "YYYYMMDD-HHMMSS"
        # Slots are reused (volume IDs cycle 0-999). A slot mid-overwrite can hold
        # chunks from two distinct volumes; keep only the winning volume's chunks.
        volume_chunks = tuple(
            k for k in best_chunks if k.rsplit("/", 1)[-1].startswith(ts_prefix)
        )
        latest_time = datetime.strptime(ts_prefix, "%Y%m%d-%H%M%S").replace(tzinfo=UTC)

        return LatestVolume(
            station_id=station_id,
            volume_number=best_vol,
            chunk_keys=volume_chunks,
            latest_chunk_time=latest_time,
        )

    def download_volume(self, volume: LatestVolume) -> bytes:
        """Fetch all chunks for ``volume``, concatenate in chunk-num order, bunzip."""
        compressed_parts: list[bytes] = []
        for key in volume.chunk_keys:
            try:
                resp = self._client.get_object(Bucket=self.bucket, Key=key)
            except ClientError as e:
                code = e.response.get("Error", {}).get("Code", "")
                raise S3Error(f"S3 get_object failed on {key}: {code}") from e
            compressed_parts.append(cast(bytes, resp["Body"].read()))

        try:
            return bz2.decompress(b"".join(compressed_parts))
        except OSError as e:
            raise S3Error(f"bzip2 decompress failed: {e}") from e

    def _list_keys(self, prefix: str) -> list[str]:
        keys: list[str] = []
        paginator = self._client.get_paginator("list_objects_v2")
        for page in paginator.paginate(Bucket=self.bucket, Prefix=prefix):
            for entry in page.get("Contents", []):
                keys.append(entry["Key"])
        return keys

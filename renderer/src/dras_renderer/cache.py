"""In-memory LRU cache for rendered PNG bytes."""

from __future__ import annotations

from cachetools import LRUCache


class RenderCache:
    """Thread-naive LRU cache keyed by (station_id, scan_filename).

    The renderer is single-replica and FastAPI handles requests sequentially
    per worker, so locking is not required for correctness here. If we ever
    add async concurrency for renders, revisit.
    """

    def __init__(self, max_size: int) -> None:
        self._inner: LRUCache[tuple[str, str], bytes] = LRUCache(maxsize=max_size)

    def get(self, station_id: str, scan_filename: str) -> bytes | None:
        return self._inner.get((station_id, scan_filename))

    def set(self, station_id: str, scan_filename: str, png: bytes) -> None:
        self._inner[(station_id, scan_filename)] = png

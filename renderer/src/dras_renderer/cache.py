"""In-memory LRU cache for rendered PNG bytes."""

from __future__ import annotations

from cachetools import LRUCache


class RenderCache:
    """LRU cache keyed by (station_id, scan_filename).

    Holds rendered PNG bytes in memory, bounded by ``max_size``. Single
    operations on the underlying ``cachetools.LRUCache`` are safe under
    CPython's GIL, and this class performs no read-modify-write sequence,
    so no locking is required as written. Note that ``get`` promotes the
    entry to most-recently-used (standard LRU semantics). If we ever add
    a single-flight render path that does miss-then-set across awaits,
    revisit and add cross-call coordination.
    """

    def __init__(self, max_size: int) -> None:
        self._inner: LRUCache[tuple[str, str], bytes] = LRUCache(maxsize=max_size)

    def get(self, station_id: str, scan_filename: str) -> bytes | None:
        return self._inner.get((station_id, scan_filename))

    def set(self, station_id: str, scan_filename: str, png: bytes) -> None:
        self._inner[(station_id, scan_filename)] = png

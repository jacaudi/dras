"""LRU cache for rendered PNG bytes keyed by (station, scan_filename)."""

from dras_renderer.cache import RenderCache


def test_get_returns_none_when_missing() -> None:
    cache = RenderCache(max_size=4)
    assert cache.get("KATX", "x.ar2v") is None


def test_set_then_get_returns_bytes() -> None:
    cache = RenderCache(max_size=4)
    cache.set("KATX", "x.ar2v", b"\x89PNG fake")
    assert cache.get("KATX", "x.ar2v") == b"\x89PNG fake"


def test_lru_evicts_oldest() -> None:
    cache = RenderCache(max_size=2)
    cache.set("KATX", "a", b"1")
    cache.set("KATX", "b", b"2")
    cache.set("KATX", "c", b"3")
    assert cache.get("KATX", "a") is None
    assert cache.get("KATX", "b") == b"2"
    assert cache.get("KATX", "c") == b"3"


def test_keys_distinguish_station_and_filename() -> None:
    cache = RenderCache(max_size=4)
    cache.set("KATX", "x", b"katx")
    cache.set("KRAX", "x", b"krax")
    assert cache.get("KATX", "x") == b"katx"
    assert cache.get("KRAX", "x") == b"krax"

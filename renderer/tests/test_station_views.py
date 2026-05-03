"""Per-station view preset resolution."""

from __future__ import annotations

from dras_renderer.station_views import resolve


def test_resolve_known_view() -> None:
    v = resolve("KATX", "metro")
    assert v is not None
    assert v.center_lat == 47.61
    assert v.center_lon == -122.33
    assert v.range_km == 70.0


def test_resolve_is_case_insensitive() -> None:
    a = resolve("katx", "Metro")
    b = resolve("KATX", "metro")
    assert a == b


def test_resolve_unknown_station_returns_none() -> None:
    """Unknown stations don't error — caller falls back to radar default."""
    assert resolve("KZZZ", "metro") is None


def test_resolve_unknown_view_returns_none() -> None:
    assert resolve("KATX", "supermetro") is None


def test_resolve_no_view_returns_none() -> None:
    assert resolve("KATX", None) is None
    assert resolve("KATX", "") is None

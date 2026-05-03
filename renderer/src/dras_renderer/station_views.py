"""Per-station named view presets.

A "view" is an opinionated framing — recenter + zoom — for a particular
station. Lets callers say ``?view=metro`` instead of having to know the
center coordinates and range for every radar.

Adding a station: append an entry to ``_VIEWS``. Adding a view name:
add a key under the station's dict. Unknown station/view combos resolve
to ``None`` and the caller falls back to the radar-centered default.
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class StationView:
    """Override values applied on top of the request defaults."""

    center_lat: float
    center_lon: float
    range_km: float


# Known views, keyed by (station_id, view_name). All station_ids are
# upper-case ICAO codes; view names are lower-case.
_VIEWS: dict[tuple[str, str], StationView] = {
    # KATX (Camano Island) → Seattle metro. Center on downtown Seattle,
    # 70 km range covers Seattle / Bellevue / Redmond / Tacoma / Everett /
    # Bremerton — the Puget Sound conurbation.
    ("KATX", "metro"): StationView(
        center_lat=47.61,
        center_lon=-122.33,
        range_km=70.0,
    ),
}


def resolve(station: str, view: str | None) -> StationView | None:
    """Return overrides for ``(station, view)``, or None if unknown.

    Unknown station/view combos return None rather than raising — the
    request still succeeds with the radar-centered default. This keeps
    the view system additive: stations without bespoke presets aren't
    broken by a caller that asks for ``?view=metro``.
    """
    if not view:
        return None
    return _VIEWS.get((station.upper(), view.lower()))

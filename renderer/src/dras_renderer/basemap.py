"""Basemap layer helpers for the radar PPI render.

Each function takes a Cartopy GeoAxes and adds one layer. Functions are
pure with respect to the axes (they mutate it) and have no return value
unless documented otherwise. Loaders that read shapefiles wrap them in
@lru_cache(maxsize=1) so repeated renders don't re-read disk.
"""

from __future__ import annotations

import logging
from functools import lru_cache
from typing import Any

import cartopy.crs as ccrs  # type: ignore[import-untyped]
import cartopy.feature as cfeature  # type: ignore[import-untyped]
import cartopy.io.shapereader as shapereader  # type: ignore[import-untyped]
from adjustText import adjust_text  # type: ignore[import-untyped]
from cartopy.feature import ShapelyFeature
from matplotlib.patches import Rectangle
from matplotlib.patheffects import withStroke
from shapely.geometry import box as _shapely_box  # type: ignore[import-untyped]
from shapely.geometry.base import BaseGeometry  # type: ignore[import-untyped]

# Color palette — single source of truth for the renderer's look.
LAND_COLOR = "#f5f0e6"      # warm cream
WATER_COLOR = "#cfe6f3"     # light blue
LAKE_OUTLINE = "#4a6da7"
COUNTY_COLOR = "#d0d0d0"
COASTLINE_COLOR = "#3a3a3a"
INTERSTATE_COLOR = "#b04a3a"
SECONDARY_ROAD_COLOR = "#888888"


def add_land_water_fill(
    ax: Any, extent: tuple[float, float, float, float]
) -> None:
    """Paint the entire extent with WATER_COLOR, then overlay land polygons.

    The ocean fill is a Rectangle in PlateCarree coords sized to ``extent``
    and drawn at zorder=0 — Cartopy's OCEAN feature is unreliable for
    inland waterways like Puget Sound (the Sound is technically tidal but
    NE classifies it inconsistently across resolutions).

    The land overlay then masks the ocean wherever there's land. This is
    the same trick the NWS RIDGE basemap uses.
    """
    west, east, south, north = extent
    ax.add_patch(
        Rectangle(
            (west, south),
            east - west,
            north - south,
            facecolor=WATER_COLOR,
            edgecolor="none",
            transform=ccrs.PlateCarree(),
            zorder=0,
        )
    )
    ax.add_feature(
        cfeature.LAND.with_scale("10m"),
        facecolor=LAND_COLOR,
        edgecolor="none",
        zorder=1,
    )


@lru_cache(maxsize=1)
def _county_records() -> tuple[BaseGeometry, ...]:
    """Load Natural Earth admin_2_counties_lakes (10m), cached.

    Returns a tuple of shapely geometries. We strip attributes — we don't
    need names, populations, or any other metadata to draw outlines.
    """
    path = shapereader.natural_earth(
        category="cultural", name="admin_2_counties_lakes", resolution="10m"
    )
    return tuple(record.geometry for record in shapereader.Reader(path).records())


def add_counties(ax: Any) -> None:
    """Draw US county boundaries as thin gray lines under states.

    Counties shape comes from Natural Earth admin_2_counties_lakes (10m),
    which excludes lake polygons (so we don't get spurious "county"
    boundaries cutting through the Great Lakes / Lake Champlain).
    """
    feat = ShapelyFeature(
        _county_records(),
        ccrs.PlateCarree(),
        facecolor="none",
        edgecolor=COUNTY_COLOR,
        linewidth=0.3,
    )
    ax.add_feature(feat, zorder=2)


# Road classes we keep (filters out local/residential/unclassified). The
# Natural Earth roads dataset's "type" attribute uses these category names.
_KEEP_ROAD_CLASSES = frozenset(
    {"Major Highway", "Secondary Highway", "Beltway", "Bypass"}
)
_INTERSTATE_CLASSES = frozenset({"Major Highway", "Beltway"})


@lru_cache(maxsize=1)
def _road_records() -> tuple[tuple[BaseGeometry, str], ...]:
    """Load Natural Earth roads (10m), cached. Returns ((geometry, class), ...)."""
    path = shapereader.natural_earth(
        category="cultural", name="roads", resolution="10m"
    )
    out: list[tuple[BaseGeometry, str]] = []
    for record in shapereader.Reader(path).records():
        attrs = record.attributes
        road_class = attrs.get("type") or attrs.get("TYPE") or ""
        if road_class not in _KEEP_ROAD_CLASSES:
            continue
        out.append((record.geometry, road_class))
    return tuple(out)


def add_roads(
    ax: Any, extent: tuple[float, float, float, float]
) -> None:
    """Draw highways inside ``extent``: interstates dull-red, others gray.

    Geometries are pre-filtered against ``extent`` with a shapely box so
    we don't ask matplotlib to draw segments that fall entirely outside
    the visible area.
    """
    west, east, south, north = extent
    bbox = _shapely_box(west, south, east, north)

    interstates: list[BaseGeometry] = []
    secondaries: list[BaseGeometry] = []
    for geom, road_class in _road_records():
        if not geom.intersects(bbox):
            continue
        if road_class in _INTERSTATE_CLASSES:
            interstates.append(geom)
        else:
            secondaries.append(geom)

    if secondaries:
        ax.add_feature(
            ShapelyFeature(
                secondaries,
                ccrs.PlateCarree(),
                facecolor="none",
                edgecolor=SECONDARY_ROAD_COLOR,
                linewidth=0.5,
            ),
            zorder=4,
        )
    if interstates:
        # Drawn after secondaries so interstates win at intersections.
        ax.add_feature(
            ShapelyFeature(
                interstates,
                ccrs.PlateCarree(),
                facecolor="none",
                edgecolor=INTERSTATE_COLOR,
                linewidth=0.9,
            ),
            zorder=4,
        )


@lru_cache(maxsize=1)
def _populated_places_records() -> tuple[tuple[float, float, str, int], ...]:
    """Load Natural Earth populated_places once.

    Returns a tuple of (lon, lat, name, scalerank) tuples. Caching avoids
    re-reading the shapefile on every render — the file is small but the
    repeated I/O + shapely geometry construction is wasteful.
    """
    path = shapereader.natural_earth(
        category="cultural", name="populated_places", resolution="10m"
    )
    out: list[tuple[float, float, str, int]] = []
    for record in shapereader.Reader(path).records():
        attrs = record.attributes
        name = attrs.get("NAME") or attrs.get("name")
        if not name:
            continue
        # SCALERANK is the Natural Earth "global importance" rank; lower
        # = more prominent. 10 is the missing/sentinel value.
        try:
            scalerank = int(attrs.get("SCALERANK", 99))
        except (TypeError, ValueError):
            scalerank = 99
        geom = record.geometry
        out.append((float(geom.x), float(geom.y), str(name), scalerank))
    return tuple(out)


def add_cities(
    ax: Any,
    extent: tuple[float, float, float, float],
    max_scalerank: int,
    *,
    deconflict: bool = True,
) -> None:
    """Plot Natural Earth populated places inside ``extent``.

    All labels get a white halo via ``path_effects.withStroke`` so they
    stay legible when they fall over reflectivity colors.

    When ``deconflict`` is True, ``adjustText.adjust_text`` repositions
    overlapping labels iteratively. We catch and log any adjustText
    failure (e.g., degenerate layouts) and fall back to halo-only
    placement — the labels are still readable, just possibly overlapping.
    """
    west, east, south, north = extent
    texts = []
    for lon0, lat0, name, scalerank in _populated_places_records():
        if scalerank > max_scalerank:
            continue
        if not (west <= lon0 <= east and south <= lat0 <= north):
            continue
        ax.plot(
            lon0, lat0, "o",
            markersize=2.5, color="black",
            transform=ccrs.PlateCarree(), zorder=5,
        )
        t = ax.text(
            lon0 + 0.04, lat0 + 0.02, name,
            fontsize=7, color="black",
            transform=ccrs.PlateCarree(), zorder=6,
            clip_on=True,
            path_effects=[withStroke(linewidth=2, foreground="white")],
        )
        texts.append(t)

    if deconflict and texts:
        try:
            adjust_text(
                texts,
                ax=ax,
                expand_points=(1.2, 1.4),
                arrowprops=None,  # no leader lines in v1
                only_move={"text": "xy"},
            )
        except Exception:  # pragma: no cover - defensive
            logging.getLogger(__name__).warning(
                "adjustText failed; falling back to halo-only placement",
                exc_info=True,
            )

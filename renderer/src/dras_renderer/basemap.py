"""Basemap layer helpers for the radar PPI render.

Each function takes a Cartopy GeoAxes and adds one layer. Functions are
pure with respect to the axes (they mutate it) and have no return value
unless documented otherwise. Loaders that read shapefiles wrap them in
@lru_cache(maxsize=1) so repeated renders don't re-read disk.
"""

from __future__ import annotations

from functools import lru_cache

import cartopy.crs as ccrs
import cartopy.feature as cfeature
import cartopy.io.shapereader as shapereader
from cartopy.feature import ShapelyFeature
from matplotlib.axes import Axes
from matplotlib.patches import Rectangle

# Color palette — single source of truth for the renderer's look.
LAND_COLOR = "#f5f0e6"      # warm cream
WATER_COLOR = "#cfe6f3"     # light blue
LAKE_OUTLINE = "#4a6da7"
COUNTY_COLOR = "#d0d0d0"
COASTLINE_COLOR = "#3a3a3a"
INTERSTATE_COLOR = "#b04a3a"
SECONDARY_ROAD_COLOR = "#888888"


def add_land_water_fill(
    ax: Axes, extent: tuple[float, float, float, float]
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
def _county_records() -> tuple:
    """Load Natural Earth admin_2_counties_lakes (10m), cached.

    Returns a tuple of shapely geometries. We strip attributes — we don't
    need names, populations, or any other metadata to draw outlines.
    """
    path = shapereader.natural_earth(
        category="cultural", name="admin_2_counties_lakes", resolution="10m"
    )
    return tuple(record.geometry for record in shapereader.Reader(path).records())


def add_counties(ax: Axes) -> None:
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

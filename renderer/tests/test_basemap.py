"""Basemap layer tests."""

from __future__ import annotations

import gzip
from pathlib import Path

import cartopy.crs as ccrs
import matplotlib.pyplot as plt
import pytest

from dras_renderer.basemap import add_land_water_fill
from dras_renderer.decode import DecodedScan, decode_level2_archive

FIXTURE = Path(__file__).parent / "fixtures" / "KATX_test.ar2v.gz"


@pytest.fixture(scope="session")
def decoded() -> DecodedScan:
    return decode_level2_archive(gzip.decompress(FIXTURE.read_bytes()))


def _make_axes() -> tuple[plt.Figure, plt.Axes]:
    fig = plt.figure(figsize=(4, 4))
    ax = fig.add_subplot(1, 1, 1, projection=ccrs.PlateCarree())
    ax.set_extent((-123.0, -121.5, 47.0, 48.5), crs=ccrs.PlateCarree())
    return fig, ax


def test_add_land_water_fill_adds_two_features() -> None:
    """add_land_water_fill paints ocean then masks with land — two features."""
    fig, ax = _make_axes()
    try:
        baseline_features = len(ax._children)
        add_land_water_fill(ax, extent=(-123.0, -121.5, 47.0, 48.5))
        added = len(ax._children) - baseline_features
        assert added >= 2  # one for ocean fill, one for land polygon
    finally:
        plt.close(fig)


def test_add_counties_loads_records_once() -> None:
    """County records are cached — repeated calls hit the lru_cache."""
    from dras_renderer.basemap import _county_records, add_counties

    _county_records.cache_clear()
    add_counties(_make_axes()[1])
    assert _county_records.cache_info().hits == 0
    add_counties(_make_axes()[1])
    assert _county_records.cache_info().hits >= 1


def test_add_counties_adds_artist() -> None:
    from dras_renderer.basemap import add_counties

    fig, ax = _make_axes()
    try:
        baseline = len(ax._children)
        add_counties(ax)
        assert len(ax._children) > baseline
    finally:
        plt.close(fig)

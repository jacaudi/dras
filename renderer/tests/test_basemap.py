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


def test_add_roads_filters_by_class() -> None:
    """Only Major and Secondary Highway classes are rendered (no local
    or unclassified roads)."""
    from dras_renderer.basemap import _road_records

    _road_records.cache_clear()
    records = _road_records()
    classes = {cls for _, cls in records}
    # NE roads dataset uses these top-level types we keep:
    assert classes <= {"Major Highway", "Secondary Highway", "Beltway", "Bypass"}


def test_add_roads_adds_artist_when_extent_contains_roads() -> None:
    """Calling add_roads over a US metro extent adds at least one feature."""
    from dras_renderer.basemap import add_roads

    fig, ax = _make_axes()
    try:
        baseline = len(ax._children)
        add_roads(ax, extent=(-123.0, -121.5, 47.0, 48.5))  # KATX metro
        assert len(ax._children) > baseline
    finally:
        plt.close(fig)


def test_add_cities_deconflict_runs(decoded) -> None:
    """With deconflict=True, no two visible city label bboxes overlap.

    Render at the KATX metro extent (Bremerton/Seattle/Tacoma cluster)
    where the un-deconflicted labels visibly stack. Compare bbox pairs
    in display coords.
    """
    import matplotlib.pyplot as plt
    from dras_renderer.basemap import add_cities

    fig, ax = _make_axes()
    ax.set_extent((-122.7, -121.9, 47.2, 47.9), crs=ccrs.PlateCarree())
    try:
        add_cities(ax, extent=(-122.7, -121.9, 47.2, 47.9), max_scalerank=8,
                   deconflict=True)
        fig.canvas.draw()  # adjustText needs a canvas to compute bboxes

        labels = [c for c in ax.texts if c.get_text()]
        assert len(labels) >= 2
        renderer = fig.canvas.get_renderer()
        bboxes = [t.get_window_extent(renderer=renderer) for t in labels]
        for i in range(len(bboxes)):
            for j in range(i + 1, len(bboxes)):
                # overlaps returns True if the bboxes touch or overlap.
                assert not bboxes[i].overlaps(bboxes[j]), (
                    f"labels {labels[i].get_text()!r} and "
                    f"{labels[j].get_text()!r} overlap after deconfliction"
                )
    finally:
        plt.close(fig)


def test_add_cities_label_has_white_halo() -> None:
    """Every city label has a white path_effects stroke for legibility."""
    from matplotlib.patheffects import withStroke
    from dras_renderer.basemap import add_cities

    fig, ax = _make_axes()
    try:
        add_cities(ax, extent=(-123.0, -121.5, 47.0, 48.5), max_scalerank=8,
                   deconflict=False)
        for t in ax.texts:
            effects = t.get_path_effects()
            assert effects, f"label {t.get_text()!r} has no path effects"
            kinds = {type(e).__name__ for e in effects}
            assert "withStroke" in kinds or any(isinstance(e, withStroke)
                                                for e in effects)
    finally:
        plt.close(fig)

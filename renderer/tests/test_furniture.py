"""Tests for cartographic furniture (colorbar, scale bar, N arrow, footer)."""

from __future__ import annotations

import gzip
from pathlib import Path

import cartopy.crs as ccrs
import matplotlib.pyplot as plt
import pytest

from dras_renderer.decode import DecodedScan, decode_level2_archive
from dras_renderer.furniture import add_colorbar
from dras_renderer.render import RenderOptions

FIXTURE = Path(__file__).parent / "fixtures" / "KATX_test.ar2v.gz"


@pytest.fixture(scope="session")
def decoded() -> DecodedScan:
    return decode_level2_archive(gzip.decompress(FIXTURE.read_bytes()))


def _make_axes() -> tuple[plt.Figure, plt.Axes]:
    fig = plt.figure(figsize=(4, 4))
    ax = fig.add_subplot(1, 1, 1, projection=ccrs.PlateCarree())
    ax.set_extent((-123.0, -121.5, 47.0, 48.5), crs=ccrs.PlateCarree())
    return fig, ax


def test_add_colorbar_creates_inset_axes() -> None:
    """add_colorbar creates a new inset axes inside the parent figure."""
    fig, ax = _make_axes()
    try:
        before = len(fig.axes)
        add_colorbar(ax, RenderOptions())
        assert len(fig.axes) == before + 1
    finally:
        plt.close(fig)

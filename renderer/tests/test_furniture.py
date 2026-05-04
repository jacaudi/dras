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


def test_add_scale_bar_adds_text_and_line() -> None:
    """Scale bar adds at least one Line2D and one Text artist with '20 km'."""
    from dras_renderer.furniture import add_scale_bar

    fig, ax = _make_axes()
    try:
        add_scale_bar(ax, length_km=20)
        labels = [t.get_text() for t in ax.texts]
        assert any("20 km" in lab for lab in labels)
    finally:
        plt.close(fig)


def test_add_north_arrow_adds_n_label() -> None:
    """N arrow adds an 'N' text label."""
    from dras_renderer.furniture import add_north_arrow

    fig, ax = _make_axes()
    try:
        add_north_arrow(ax)
        labels = [t.get_text() for t in ax.texts]
        assert "N" in labels
    finally:
        plt.close(fig)


def test_add_footer_text_includes_station_and_age(decoded) -> None:
    """Footer text mentions station, scan time, and data-age annotation."""
    from dras_renderer.furniture import add_footer

    fig = plt.figure(figsize=(8, 8.6))
    radar_ax = fig.add_axes((0, 0.08, 1, 0.92))
    footer_ax = fig.add_axes((0, 0, 1, 0.08))

    try:
        add_footer(footer_ax, decoded, data_age_seconds=12.0,
                   renderer_version="9.9.9")
        labels = " ".join(t.get_text() for t in footer_ax.texts)
        assert decoded.station_id in labels
        assert "9.9.9" in labels
        assert "12s" in labels
    finally:
        plt.close(fig)

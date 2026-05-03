"""Render Level II to PNG bytes."""

from __future__ import annotations

import gzip
import io
from pathlib import Path

import matplotlib.pyplot as plt
import pytest
from PIL import Image

from dras_renderer.decode import DecodedScan, decode_level2_archive
from dras_renderer.render import (
    RenderOptions,
    _render_figure,
    render_base_reflectivity,
)

FIXTURE = Path(__file__).parent / "fixtures" / "KATX_test.ar2v.gz"


@pytest.fixture(scope="session")
def decoded() -> DecodedScan:
    return decode_level2_archive(gzip.decompress(FIXTURE.read_bytes()))


def test_render_returns_png_bytes(decoded: DecodedScan) -> None:
    png = render_base_reflectivity(decoded, RenderOptions())
    assert png.startswith(b"\x89PNG")


def test_render_dimensions_match_options(decoded: DecodedScan) -> None:
    opts = RenderOptions(width=400, height=400)
    png = render_base_reflectivity(decoded, opts)
    img = Image.open(io.BytesIO(png))
    assert img.size == (400, 400)


def test_render_respects_range_km(decoded: DecodedScan) -> None:
    """Larger range_km produces an image of the same dimensions but
    different content (different geographic extent)."""
    a = render_base_reflectivity(decoded, RenderOptions(range_km=100))
    b = render_base_reflectivity(decoded, RenderOptions(range_km=300))
    assert a != b
    img_a = Image.open(io.BytesIO(a))
    img_b = Image.open(io.BytesIO(b))
    assert img_a.size == img_b.size


def test_render_center_override_changes_output(decoded: DecodedScan) -> None:
    """Recentering on a different point produces a visually different image
    even at the same range — the basemap and PPI clip differently."""
    radar_centered = render_base_reflectivity(
        decoded, RenderOptions(range_km=70.0)
    )
    seattle_centered = render_base_reflectivity(
        decoded,
        RenderOptions(range_km=70.0, center_lat=47.61, center_lon=-122.33),
    )
    assert radar_centered != seattle_centered


def test_render_clutter_filter_changes_output(decoded: DecodedScan) -> None:
    """Disabling the clutter filter must produce a different image —
    otherwise we'd know the filter is silently a no-op."""
    filtered = render_base_reflectivity(decoded, RenderOptions(clutter_filter=True))
    raw = render_base_reflectivity(decoded, RenderOptions(clutter_filter=False))
    assert filtered != raw


def test_render_title_includes_scan_time_and_data_age(decoded: DecodedScan) -> None:
    """When data_age_seconds is provided, the axes title is overridden to
    include the volume start (scan_time iso), the station id, and the
    +Δ data-age annotation.

    Uses 30.0 to dodge banker's-rounding ambiguity on .5 values.
    """
    fig, ax = _render_figure(decoded, RenderOptions(), data_age_seconds=30.0)
    try:
        title = ax.get_title()
    finally:
        plt.close(fig)

    assert decoded.scan_time.isoformat() in title
    # Pin the joined token so a future format drift (e.g. dropping the Δ or
    # the trailing "s") is caught, not just the substrings independently.
    assert "+Δ 30s" in title
    assert decoded.station_id in title


def test_render_default_title_unchanged_when_no_age(decoded: DecodedScan) -> None:
    """Without data_age_seconds, leave Py-ART's default title intact —
    don't pin the exact text (locks us to a Py-ART version), but assert
    it's non-empty and lacks the "+Δ" annotation.
    """
    fig, ax = _render_figure(decoded, RenderOptions(), data_age_seconds=None)
    try:
        title = ax.get_title()
    finally:
        plt.close(fig)

    assert title  # non-empty
    assert "+Δ" not in title


def test_matplotlib_uses_agg_backend() -> None:
    """matplotlib.use('Agg') in render.py must select the Agg backend at import time.

    This regression-locks the headless-backend selection across the M5 change
    that drops the redundant ``os.environ["MPLBACKEND"] = "Agg"`` line — the
    explicit ``matplotlib.use("Agg")`` call is authoritative.
    """
    import matplotlib

    # Importing dras_renderer.render forces matplotlib.use('Agg') to execute.
    import dras_renderer.render  # noqa: F401

    assert matplotlib.get_backend().lower() == "agg"

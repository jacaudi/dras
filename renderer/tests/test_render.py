"""Render Level II to PNG bytes."""

from __future__ import annotations

import gzip
import io
from pathlib import Path

import pytest
from PIL import Image

from dras_renderer.decode import DecodedScan, decode_level2_archive
from dras_renderer.render import RenderOptions, render_base_reflectivity

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

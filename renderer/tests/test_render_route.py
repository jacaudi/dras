"""End-to-end route test for /render/{station}."""

from __future__ import annotations

import base64
import gzip
from datetime import UTC, datetime
from pathlib import Path
from unittest.mock import patch

import pytest
from fastapi.testclient import TestClient

from dras_renderer.app import build_app
from dras_renderer.s3 import LatestVolume, S3Error

FIXTURE = Path(__file__).parent / "fixtures" / "KATX_test.ar2v.gz"


@pytest.fixture(scope="session")
def fixture_bytes() -> bytes:
    return gzip.decompress(FIXTURE.read_bytes())


def _vol() -> LatestVolume:
    return LatestVolume(
        station_id="KATX",
        volume_number=492,
        chunk_keys=("KATX/492/20260501-180941-001-S",),
        latest_chunk_time=datetime(2026, 5, 1, 18, 9, 41, tzinfo=UTC),
    )


def test_render_route_returns_envelope(fixture_bytes: bytes) -> None:
    with patch("dras_renderer.s3.S3Client.latest_volume", return_value=_vol()), \
         patch("dras_renderer.s3.S3Client.download_volume", return_value=fixture_bytes):
        client = TestClient(build_app())
        resp = client.get("/render/KATX")

    assert resp.status_code == 200
    body = resp.json()
    assert body["metadata"]["station"] == "KATX"
    assert body["metadata"]["product"] == "base_reflectivity"
    assert body["metadata"]["renderer_version"]
    png_bytes = base64.b64decode(body["image"])
    assert png_bytes.startswith(b"\x89PNG")


def test_render_route_no_recent_volume() -> None:
    with patch("dras_renderer.s3.S3Client.latest_volume", return_value=None):
        client = TestClient(build_app())
        resp = client.get("/render/KATX")

    assert resp.status_code == 404
    body = resp.json()
    assert body["error"] == "no_recent_scan"


def test_render_route_decode_failed() -> None:
    with patch("dras_renderer.s3.S3Client.latest_volume", return_value=_vol()), \
         patch("dras_renderer.s3.S3Client.download_volume", return_value=b"junk"):
        client = TestClient(build_app())
        resp = client.get("/render/KATX")

    assert resp.status_code == 502
    body = resp.json()
    assert body["error"] == "decode_failed"


def test_render_route_s3_failure() -> None:
    with patch("dras_renderer.s3.S3Client.latest_volume", return_value=_vol()), \
         patch("dras_renderer.s3.S3Client.download_volume", side_effect=S3Error("boom")):
        client = TestClient(build_app())
        resp = client.get("/render/KATX")

    assert resp.status_code == 500
    body = resp.json()
    assert body["error"] == "internal"


def test_render_route_lowercases_station(fixture_bytes: bytes) -> None:
    """Station IDs in the URL are normalized to uppercase before going to S3."""
    captured: dict[str, str] = {}

    def capture(self: object, station: str) -> LatestVolume:
        captured["station"] = station
        return _vol()

    with patch("dras_renderer.s3.S3Client.latest_volume", autospec=True, side_effect=capture), \
         patch("dras_renderer.s3.S3Client.download_volume", return_value=fixture_bytes):
        client = TestClient(build_app())
        resp = client.get("/render/katx")

    assert resp.status_code == 200
    assert captured["station"] == "KATX"

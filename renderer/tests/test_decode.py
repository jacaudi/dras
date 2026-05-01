"""Decode the bundled Level II fixture and assert metadata."""

from __future__ import annotations

import gzip
from pathlib import Path

import pytest

from dras_renderer.decode import DecodedScan, decode_level2_archive

FIXTURE = Path(__file__).parent / "fixtures" / "KATX_test.ar2v.gz"

# Update this constant in lockstep with the fixture (see fixtures/README.md).
FIXTURE_DATE_PREFIX = "2026-05-01"


@pytest.fixture
def fixture_bytes() -> bytes:
    return gzip.decompress(FIXTURE.read_bytes())


def test_decode_returns_metadata(fixture_bytes: bytes) -> None:
    scan = decode_level2_archive(fixture_bytes)
    assert isinstance(scan, DecodedScan)
    assert scan.station_id == "KATX"
    # Lowest tilt elevation is conventionally 0.5°; allow ±0.2° drift.
    assert 0.3 <= scan.elevation_deg <= 0.7
    # VCP for the bundled scan; small int (e.g. 215, 35, 12).
    assert scan.vcp > 0
    assert scan.scan_time.isoformat().startswith(FIXTURE_DATE_PREFIX)


def test_decode_raises_on_truncated_input() -> None:
    with pytest.raises(ValueError):
        decode_level2_archive(b"not-a-real-volume-scan")


def test_radar_object_has_reflectivity_field(fixture_bytes: bytes) -> None:
    scan = decode_level2_archive(fixture_bytes)
    assert "reflectivity" in scan.radar.fields

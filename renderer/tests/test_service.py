"""Orchestration: cache → s3 → decode → render."""

from __future__ import annotations

import gzip
from datetime import UTC, datetime
from pathlib import Path
from unittest.mock import MagicMock

import pytest

from dras_renderer.cache import RenderCache
from dras_renderer.s3 import LatestVolume, S3Error
from dras_renderer.service import RenderRequest, RenderService, ServiceError

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


def test_renders_and_caches(fixture_bytes: bytes) -> None:
    s3 = MagicMock()
    s3.latest_volume.return_value = _vol()
    s3.download_volume.return_value = fixture_bytes
    cache = RenderCache(max_size=8)

    svc = RenderService(s3=s3, cache=cache)
    resp = svc.render(RenderRequest(station="KATX"))

    assert resp.png.startswith(b"\x89PNG")
    assert resp.metadata.station == "KATX"
    assert resp.metadata.product == "base_reflectivity"
    assert resp.metadata.vcp > 0
    assert s3.download_volume.call_count == 1

    # Second call hits the cache.
    svc.render(RenderRequest(station="KATX"))
    assert s3.download_volume.call_count == 1


def test_no_recent_volume_raises() -> None:
    s3 = MagicMock()
    s3.latest_volume.return_value = None
    svc = RenderService(s3=s3, cache=RenderCache(max_size=8))

    with pytest.raises(ServiceError) as excinfo:
        svc.render(RenderRequest(station="KATX"))
    assert excinfo.value.code == "no_recent_scan"


def test_decode_failure_raises() -> None:
    s3 = MagicMock()
    s3.latest_volume.return_value = _vol()
    s3.download_volume.return_value = b"not-a-volume"
    svc = RenderService(s3=s3, cache=RenderCache(max_size=8))

    with pytest.raises(ServiceError) as excinfo:
        svc.render(RenderRequest(station="KATX"))
    assert excinfo.value.code == "decode_failed"


def test_s3_download_failure_raises_internal() -> None:
    s3 = MagicMock()
    s3.latest_volume.return_value = _vol()
    s3.download_volume.side_effect = S3Error("network blew up")
    svc = RenderService(s3=s3, cache=RenderCache(max_size=8))

    with pytest.raises(ServiceError) as excinfo:
        svc.render(RenderRequest(station="KATX"))
    assert excinfo.value.code == "internal"


def test_unsupported_product_raises() -> None:
    s3 = MagicMock()
    svc = RenderService(s3=s3, cache=RenderCache(max_size=8))

    with pytest.raises(ServiceError) as excinfo:
        svc.render(RenderRequest(station="KATX", product="velocity"))
    assert excinfo.value.code == "unsupported_product"
    s3.latest_volume.assert_not_called()


def test_render_increments_s3_errors_on_list_failure() -> None:
    """An S3Error from latest_volume must increment renderer_s3_errors_total."""
    from dras_renderer.metrics import S3_ERRORS_TOTAL

    s3 = MagicMock()
    s3.latest_volume.side_effect = S3Error("simulated list failure")
    svc = RenderService(s3=s3, cache=RenderCache(max_size=8))

    before = S3_ERRORS_TOTAL._value.get()  # type: ignore[attr-defined]
    with pytest.raises(ServiceError) as excinfo:
        svc.render(RenderRequest(station="KATX"))
    assert excinfo.value.code == "internal"
    after = S3_ERRORS_TOTAL._value.get()  # type: ignore[attr-defined]
    assert after == before + 1
    s3.download_volume.assert_not_called()


def test_render_increments_s3_errors_on_download_failure() -> None:
    """An S3Error from download_volume must also increment renderer_s3_errors_total."""
    from dras_renderer.metrics import S3_ERRORS_TOTAL

    s3 = MagicMock()
    s3.latest_volume.return_value = _vol()
    s3.download_volume.side_effect = S3Error("simulated download failure")
    svc = RenderService(s3=s3, cache=RenderCache(max_size=8))

    before = S3_ERRORS_TOTAL._value.get()  # type: ignore[attr-defined]
    with pytest.raises(ServiceError) as excinfo:
        svc.render(RenderRequest(station="KATX"))
    assert excinfo.value.code == "internal"
    after = S3_ERRORS_TOTAL._value.get()  # type: ignore[attr-defined]
    assert after == before + 1

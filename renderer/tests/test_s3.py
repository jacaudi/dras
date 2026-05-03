"""Chunks-bucket S3 client tests (moto-backed)."""

from __future__ import annotations

import inspect
from collections.abc import Iterator
from datetime import UTC, datetime

import boto3
import pytest
from moto import mock_aws

from dras_renderer.s3 import (
    LatestVolume,
    S3Client,
    S3Error,
    VolumeNotFoundError,
)


def test_latest_volume_ttl_default_is_5_seconds() -> None:
    """Lock the S3Client.latest_volume_ttl default at 5s.

    The TTL exists to amortize the 1000-LIST fan-out across back-to-back
    renders. Anything longer (the original 30s) can hide minute-old data
    from the freshness checks. 5s still amortizes hot loops while keeping
    user-visible freshness perception correct.
    """
    assert inspect.signature(S3Client).parameters["latest_volume_ttl"].default == 5.0

BUCKET = "unidata-nexrad-level2-chunks"


def _put_chunk(s3: object, key: str, payload: bytes) -> None:
    s3.put_object(Bucket=BUCKET, Key=key, Body=payload)


@pytest.fixture
def mock_bucket() -> Iterator[str]:
    """Two KATX volumes. Vol 5 is older and complete. Vol 17 is newer and in-progress (no E)."""
    with mock_aws():
        s3 = boto3.client("s3", region_name="us-east-1")
        s3.create_bucket(Bucket=BUCKET)

        # Older complete volume in slot 5 (ends with E).
        _put_chunk(s3, "KATX/5/20260429-120000-001-S", b"v5-S")
        _put_chunk(s3, "KATX/5/20260429-120000-002-I", b"v5-I")
        _put_chunk(s3, "KATX/5/20260429-120000-003-E", b"v5-E")

        # Newer in-progress volume in slot 17 (no E chunk yet).
        _put_chunk(s3, "KATX/17/20260429-120500-001-S", b"v17-S")
        _put_chunk(s3, "KATX/17/20260429-120500-002-I", b"v17-I")

        yield BUCKET


def _make_client(latest_volume_ttl: float = 0.0) -> S3Client:
    return S3Client(
        bucket=BUCKET,
        region="us-east-1",
        anonymous=False,
        list_workers=4,
        latest_volume_ttl=latest_volume_ttl,
    )


def test_latest_volume_picks_max_chunk_timestamp(mock_bucket: str) -> None:
    """In-progress vol 17 has a newer timestamp than complete vol 5; vol 17 wins."""
    with mock_aws():
        client = _make_client()
        v = client.latest_volume("KATX")
        assert isinstance(v, LatestVolume)
        assert v.volume_number == 17
        assert v.chunk_keys == (
            "KATX/17/20260429-120500-001-S",
            "KATX/17/20260429-120500-002-I",
        )
        assert v.latest_chunk_time == datetime(2026, 4, 29, 12, 5, 0, tzinfo=UTC)
        # moto sets LastModified to "now" at put time.
        assert v.latest_chunk_uploaded_at.tzinfo is not None
        assert v.latest_chunk_uploaded_at <= datetime.now(UTC)


def test_latest_volume_uploaded_at_is_max_winning_chunk_lastmodified() -> None:
    """``latest_chunk_uploaded_at`` must equal the max LastModified across the
    winning slot's filtered chunks."""
    import time

    with mock_aws():
        s3 = boto3.client("s3", region_name="us-east-1")
        s3.create_bucket(Bucket=BUCKET)
        # Two chunks in one slot, with a small delay so LastModified differs.
        _put_chunk(s3, "KATX/4/20260429-120000-001-S", b"a")
        time.sleep(0.05)
        _put_chunk(s3, "KATX/4/20260429-120000-002-I", b"b")

        # Read the actual LastModified of each chunk via list_objects_v2.
        listing = s3.list_objects_v2(Bucket=BUCKET, Prefix="KATX/4/")
        last_mods = {entry["Key"]: entry["LastModified"] for entry in listing["Contents"]}
        expected_max = max(last_mods.values())

        client = _make_client()
        v = client.latest_volume("KATX")
        assert v is not None
        assert v.latest_chunk_uploaded_at == expected_max


def test_latest_volume_returns_none_for_unknown_station(mock_bucket: str) -> None:
    with mock_aws():
        client = _make_client()
        assert client.latest_volume("KZZZ") is None


def test_latest_volume_caches_within_ttl(mock_bucket: str) -> None:
    """Two calls within the TTL must return the exact same instance (cache hit)."""
    with mock_aws():
        client = _make_client(latest_volume_ttl=60.0)
        v1 = client.latest_volume("KATX")
        v2 = client.latest_volume("KATX")
        assert v1 is v2


def test_download_volume_concatenates_chunks(mock_bucket: str) -> None:
    """Vol 17's two chunk bodies are concatenated as-is — Py-ART handles internal bzip2."""
    with mock_aws():
        client = _make_client()
        v = client.latest_volume("KATX")
        assert v is not None
        body = client.download_volume(v)
        assert body == b"v17-S" + b"v17-I"


def test_download_volume_raises_s3error_on_missing_chunk(mock_bucket: str) -> None:
    """If a chunk listed in the volume disappears between list and download, S3Error."""
    with mock_aws():
        client = _make_client()
        v = client.latest_volume("KATX")
        assert v is not None
        s3 = boto3.client("s3", region_name="us-east-1")
        s3.delete_object(Bucket=BUCKET, Key=v.chunk_keys[0])
        with pytest.raises(S3Error):
            client.download_volume(v)


def test_anonymous_mode_uses_unsigned_requests() -> None:
    """When anonymous=True, S3 requests must NOT carry an Authorization header.

    This is a behavior-level check — it does not reach into private botocore
    state. A regression that flipped the client to signed mode would be
    caught here even if botocore's internal config layout changed.
    """
    captured: dict[str, dict[str, str]] = {}

    def capture(**kwargs: object) -> None:
        # botocore's `before-sign.s3` hook fires on every prepared request,
        # including under moto. The request object exposes its headers as a
        # case-insensitive mapping; coerce to plain dict for the assertion.
        request = kwargs.get("request")
        if request is not None:
            captured["headers"] = dict(request.headers)  # type: ignore[attr-defined]

    with mock_aws():
        s3 = boto3.client("s3", region_name="us-east-1")
        s3.create_bucket(Bucket=BUCKET)

        client = S3Client(bucket=BUCKET, region="us-east-1", anonymous=True)
        client._client.meta.events.register("before-sign.s3", capture)

        # latest_volume issues paginated LIST calls; we don't care about the
        # result (the bucket is empty), only that a real signed/unsigned LIST
        # is dispatched so the before-sign hook captures its headers.
        client.latest_volume("KATX")

    headers = captured.get("headers")
    assert headers is not None, "no request was captured by the before-sign.s3 hook"
    # Header keys are case-insensitive in HTTP; normalize for the membership check.
    lowered = {k.lower() for k in headers}
    assert "authorization" not in lowered, (
        f"anonymous client must not sign requests; got headers: {headers}"
    )


def test_volume_not_found_error_subclasses_s3_error() -> None:
    """VolumeNotFoundError must subclass S3Error so callers can catch the base."""
    assert issubclass(VolumeNotFoundError, S3Error)


def test_latest_volume_filters_chunks_to_winning_timestamp() -> None:
    """A slot mid-overwrite holds chunks from two distinct volumes.
    Only the winning volume's chunks should be returned."""
    with mock_aws():
        s3 = boto3.client("s3", region_name="us-east-1")
        s3.create_bucket(Bucket=BUCKET)
        # Slot 9 holds two chunks of an older volume (12:00:00) and three
        # chunks of a newer volume (12:05:00) that's mid-overwrite.
        _put_chunk(s3, "KATX/9/20260429-120000-001-S", b"old-S")
        _put_chunk(s3, "KATX/9/20260429-120000-002-I", b"old-I")
        _put_chunk(s3, "KATX/9/20260429-120500-001-S", b"new-S")
        _put_chunk(s3, "KATX/9/20260429-120500-002-I", b"new-I")
        _put_chunk(s3, "KATX/9/20260429-120500-003-E", b"new-E")

        client = _make_client()
        v = client.latest_volume("KATX")
        assert v is not None
        assert v.volume_number == 9
        assert v.chunk_keys == (
            "KATX/9/20260429-120500-001-S",
            "KATX/9/20260429-120500-002-I",
            "KATX/9/20260429-120500-003-E",
        )
        # download_volume must produce only the new volume's payloads, concatenated as-is.
        body = client.download_volume(v)
        assert body == b"new-S" + b"new-I" + b"new-E"


def test_latest_volume_with_tied_prefixes_returns_a_valid_slot() -> None:
    """Tie case: when two slots share the YYYYMMDD-HHMMSS prefix of their
    newest chunk, the picker must return one of them deterministically and
    return only that slot's chunks.

    For well-formed NEXRAD filenames the prefix-only and full-filename
    comparisons agree whenever timestamps differ (the prefix is the
    leftmost varying field, so it dominates lex order). The two algorithms
    can only disagree when prefixes tie — and even then, either slot is
    a valid answer. This test pins the post-condition: the returned
    chunk_keys belong to a single, real slot.
    """
    with mock_aws():
        s3 = boto3.client("s3", region_name="us-east-1")
        s3.create_bucket(Bucket=BUCKET)
        # Same timestamp prefix; suffix differs across slots.
        _put_chunk(s3, "KATX/5/20260429-120000-001-S", b"a")
        _put_chunk(s3, "KATX/5/20260429-120000-002-I", b"b")
        _put_chunk(s3, "KATX/5/20260429-120000-003-E", b"c")
        _put_chunk(s3, "KATX/17/20260429-120000-001-S", b"x")
        _put_chunk(s3, "KATX/17/20260429-120000-002-I", b"y")

        client = _make_client()
        v = client.latest_volume("KATX")
        assert v is not None
        assert v.volume_number in {5, 17}
        assert all(k.startswith(f"KATX/{v.volume_number}/20260429-120000-") for k in v.chunk_keys)
        assert v.latest_chunk_time == datetime(2026, 4, 29, 12, 0, 0, tzinfo=UTC)


def test_download_volume_concatenates_in_chunk_keys_order() -> None:
    """Output bytes must be in chunk_keys order regardless of fetch completion order.

    This is the correctness invariant of the parallel download path: a refactor
    to ``as_completed`` (which yields by completion order) would break it.
    """
    with mock_aws():
        s3 = boto3.client("s3", region_name="us-east-1")
        s3.create_bucket(Bucket=BUCKET)
        # 8 chunks with bodies that encode their chunk-num so order is verifiable.
        for n in range(1, 9):
            _put_chunk(s3, f"KATX/3/20260429-130000-00{n}-I", f"chunk-{n:02d}".encode())
        client = S3Client(
            bucket=BUCKET,
            region="us-east-1",
            anonymous=False,
            list_workers=4,
            download_workers=8,
            latest_volume_ttl=0.0,
        )
        v = client.latest_volume("KATX")
        assert v is not None
        body = client.download_volume(v)
        assert body == b"".join(f"chunk-{n:02d}".encode() for n in range(1, 9))

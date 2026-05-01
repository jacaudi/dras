"""Decode NEXRAD Level II Archive bytes via Py-ART.

The byte stream may come from either an archive `_V06` object or a chunks-bucket
volume assembled by ``S3Client.download_volume`` — they are byte-identical at
this layer. Py-ART's ``read_nexrad_archive`` accepts a file-like via
``prepare_for_read``, which short-circuits to the file-like when ``hasattr(filename, 'read')``.
"""

from __future__ import annotations

import io
from dataclasses import dataclass
from datetime import UTC, datetime
from typing import Any, cast

import pyart  # type: ignore[import-untyped]


@dataclass
class DecodedScan:
    """Decoded Level II volume + extracted metadata for the API response."""

    radar: Any  # pyart.core.Radar — typed Any to avoid the heavy import in callers.
    station_id: str
    scan_time: datetime
    elevation_deg: float
    vcp: int


def decode_level2_archive(volume_bytes: bytes) -> DecodedScan:
    """Decode a Level II Archive volume.

    Args:
        volume_bytes: raw Level II Archive bytes (byte-identical whether sourced
            from an archive _V06 object or assembled from chunks).

    Raises:
        ValueError: if Py-ART cannot parse the input.
    """
    try:
        radar = pyart.io.read_nexrad_archive(io.BytesIO(volume_bytes))
        return DecodedScan(
            radar=radar,
            station_id=_extract_station(radar),
            scan_time=_extract_scan_time(radar),
            elevation_deg=float(radar.fixed_angle["data"][0]),
            vcp=_extract_vcp(radar),
        )
    except ValueError:
        raise  # Already the right type (e.g. unexpected time-units format).
    except Exception as exc:  # Py-ART raises a varied set of errors; catch all.
        raise ValueError(f"failed to decode Level II archive: {exc}") from exc


def _extract_station(radar: Any) -> str:
    # Py-ART sets metadata['instrument_name'] from the volume header's ICAO bytes.
    name = cast(str, radar.metadata.get("instrument_name", ""))
    return name.upper()


def _extract_scan_time(radar: Any) -> datetime:
    # make_time_unit_str produces "seconds since YYYY-MM-DDTHH:MM:SSZ".
    units = cast(str, radar.time["units"])
    prefix = "seconds since "
    if not units.startswith(prefix):
        raise ValueError(f"unexpected Py-ART time units format: {units!r}")
    iso = units[len(prefix) :].removesuffix("Z")
    return datetime.fromisoformat(iso).replace(tzinfo=UTC)


def _extract_vcp(radar: Any) -> int:
    """Read VCP number from the volume coverage pattern, if present.

    Uses ``is not None`` rather than truthy fallback so a present-but-zero
    value isn't silently treated as missing.
    """
    raw = radar.metadata.get("vcp_pattern")
    if raw is None:
        raw = radar.metadata.get("vcp", 0)
    try:
        return int(raw)
    except (TypeError, ValueError):
        return 0

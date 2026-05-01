"""Render a decoded Level II volume to a PPI PNG."""

from __future__ import annotations

import io
import math
import os
from dataclasses import dataclass

# Headless backend MUST be selected before importing pyplot.  Use
# matplotlib.use() (not just setdefault) so it wins even when the
# macOS backend has already been detected by the env.
os.environ["MPLBACKEND"] = "Agg"
import matplotlib

matplotlib.use("Agg")

import cartopy.crs as ccrs  # type: ignore[import-untyped]
import cartopy.feature as cfeature  # type: ignore[import-untyped]
import matplotlib.pyplot as plt
import pyart  # type: ignore[import-untyped]

from dras_renderer.decode import DecodedScan


@dataclass(frozen=True)
class RenderOptions:
    """Options controlling render output."""

    width: int = 800
    height: int = 800
    range_km: float = 230.0
    dpi: int = 100

    # Reflectivity color scale bounds, dBZ.
    vmin: float = -20.0
    vmax: float = 75.0


def render_base_reflectivity(scan: DecodedScan, opts: RenderOptions) -> bytes:
    """Render the lowest-tilt base reflectivity as a PPI on a Cartopy basemap.

    Returns PNG bytes sized exactly to ``(opts.width, opts.height)``.
    """
    fig = plt.figure(
        figsize=(opts.width / opts.dpi, opts.height / opts.dpi),
        dpi=opts.dpi,
    )
    try:
        radar = scan.radar
        lat = float(radar.latitude["data"][0])
        lon = float(radar.longitude["data"][0])

        projection = ccrs.LambertConformal(central_latitude=lat, central_longitude=lon)
        ax = fig.add_subplot(1, 1, 1, projection=projection)

        # 1° lat ≈ 111 km everywhere; 1° lon ≈ 111 cos(lat) km. Without the
        # cos(lat) correction the east-west extent stretches by 33% at KATX
        # (lat ~48°) and ~50% at high-latitude AK stations.
        delta_lat = opts.range_km / 111.0
        delta_lon = delta_lat / max(math.cos(math.radians(lat)), 1e-6)
        ax.set_extent(
            [lon - delta_lon, lon + delta_lon, lat - delta_lat, lat + delta_lat],
            crs=ccrs.PlateCarree(),
        )

        ax.add_feature(cfeature.STATES.with_scale("50m"), edgecolor="gray", linewidth=0.5)
        ax.add_feature(cfeature.COASTLINE.with_scale("50m"), edgecolor="black", linewidth=0.5)

        display = pyart.graph.RadarMapDisplay(radar)
        display.plot_ppi_map(
            "reflectivity",
            sweep=0,
            ax=ax,
            colorbar_flag=False,
            title_flag=True,
            vmin=opts.vmin,
            vmax=opts.vmax,
            cmap="pyart_NWSRef",
            embellish=False,  # We add our own basemap features above.
        )

        buf = io.BytesIO()
        # IMPORTANT: do NOT pass bbox_inches="tight" — it crops to content
        # and produces non-deterministic output dimensions, which would
        # break test_render_dimensions_match_options. Caller asks for an
        # exact (width, height); honor it.
        fig.savefig(buf, format="png", dpi=opts.dpi)
        return buf.getvalue()
    finally:
        plt.close(fig)

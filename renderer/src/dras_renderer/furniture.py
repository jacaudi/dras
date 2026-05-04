"""Cartographic furniture: colorbar, scale bar, N arrow, footer.

Furniture functions add visual elements that are part of the *map*, not
the data — anything a cartographer would call "marginalia." Each function
takes the radar axes (and any other context it needs) and mutates it.
"""

from __future__ import annotations

import math

import cartopy.crs as ccrs
import matplotlib.pyplot as plt
from matplotlib.axes import Axes
from matplotlib.cm import ScalarMappable
from matplotlib.colors import Normalize
from mpl_toolkits.axes_grid1.inset_locator import inset_axes


def add_colorbar(ax: Axes, opts) -> None:
    """Inset horizontal reflectivity colorbar in the lower-left corner.

    Uses the same ``pyart_NWSRef`` cmap and (vmin, vmax) the radar plot
    is rendered with so the inset is exactly the active scale.
    """
    cax = inset_axes(
        ax,
        width="28%", height="3%",
        loc="lower left",
        bbox_to_anchor=(0.02, 0.02, 1, 1),
        bbox_transform=ax.transAxes,
        borderpad=0,
    )
    cax.set_facecolor((1, 1, 1, 0.85))
    sm = ScalarMappable(norm=Normalize(vmin=opts.vmin, vmax=opts.vmax),
                        cmap="pyart_NWSRef")
    cb = plt.colorbar(sm, cax=cax, orientation="horizontal")
    cb.set_ticks([-20, 0, 20, 40, 60, 75])
    cb.ax.tick_params(labelsize=6, length=2, pad=1)
    cb.set_label("dBZ", fontsize=6, labelpad=1)


def add_scale_bar(ax: Axes, length_km: float = 20.0) -> None:
    """Draw a fixed-length scale bar in the lower-right of the radar axes.

    Length is converted from km to projected coords using the latitude
    of the axes' center — accurate to within ~1% for any reasonable
    radar zoom (the projection distortion over 20 km at mid-latitudes
    is negligible).
    """
    west, east, south, north = ax.get_extent(crs=ccrs.PlateCarree())
    center_lat = (south + north) / 2.0
    # 1° lon ≈ 111 km · cos(lat); invert for "what fraction of a degree
    # is `length_km`?".
    deg_per_km = 1.0 / (111.0 * max(math.cos(math.radians(center_lat)), 1e-6))
    bar_deg = length_km * deg_per_km

    # Anchor at 5% in from the right edge, 5% up from the bottom.
    x_end = east - 0.05 * (east - west)
    x_start = x_end - bar_deg
    y = south + 0.05 * (north - south)

    ax.plot(
        [x_start, x_end], [y, y],
        color="black", linewidth=2,
        transform=ccrs.PlateCarree(), zorder=10,
    )
    ax.text(
        (x_start + x_end) / 2, y + 0.01 * (north - south),
        f"{int(length_km)} km",
        fontsize=7, color="black", ha="center", va="bottom",
        transform=ccrs.PlateCarree(), zorder=10,
    )


def add_north_arrow(ax: Axes) -> None:
    """Place a small N + upward arrow glyph in the upper-right of the axes.

    Uses axes-fraction coords so position is invariant to data extent.
    """
    ax.annotate(
        "N",
        xy=(0.95, 0.93),
        xytext=(0.95, 0.97),
        xycoords="axes fraction",
        textcoords="axes fraction",
        ha="center", va="bottom",
        fontsize=10, fontweight="bold", color="black",
        arrowprops=dict(facecolor="black", edgecolor="black",
                        width=2, headwidth=8, headlength=8),
        zorder=10,
    )


def add_footer(
    footer_ax: Axes,
    scan,  # DecodedScan
    data_age_seconds: float | None,
    renderer_version: str,
) -> None:
    """Render the metadata strip below the radar plot.

    Single line, center-aligned. Format:
      ``KATX • 0.5° base reflectivity • 2026-05-03 19:12 UTC • age 12s • dras-renderer v9.9.9``
    """
    footer_ax.set_facecolor("white")
    footer_ax.set_xticks([])
    footer_ax.set_yticks([])
    for spine in footer_ax.spines.values():
        spine.set_visible(False)

    age_part = f"age {int(round(data_age_seconds))}s" if data_age_seconds is not None else "age unknown"
    when = scan.scan_time.strftime("%Y-%m-%d %H:%M UTC")
    text = (
        f"{scan.station_id}  •  {scan.elevation_deg:.1f}° base reflectivity  "
        f"•  {when}  •  {age_part}  •  dras-renderer v{renderer_version}"
    )
    footer_ax.text(
        0.5, 0.5, text,
        ha="center", va="center", fontsize=8, color="#333",
        transform=footer_ax.transAxes,
    )

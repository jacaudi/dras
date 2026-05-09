"""Cartographic furniture: colorbar, scale bar, N arrow, footer.

Furniture functions add visual elements that are part of the *map*, not
the data — anything a cartographer would call "marginalia." Each function
takes the radar axes (and any other context it needs) and mutates it.
"""

from __future__ import annotations

import math
from typing import TYPE_CHECKING, Any

import cartopy.crs as ccrs  # type: ignore[import-untyped]
import matplotlib.pyplot as plt
from matplotlib.cm import ScalarMappable
from matplotlib.colors import Normalize
from mpl_toolkits.axes_grid1.inset_locator import inset_axes  # type: ignore[import-untyped]

if TYPE_CHECKING:
    from dras_renderer.decode import DecodedScan
    from dras_renderer.render import RenderOptions


def add_colorbar(ax: Any, opts: RenderOptions) -> None:
    """Inset horizontal reflectivity colorbar in the lower-left corner.

    Uses the same ``pyart_NWSRef`` cmap and (vmin, vmax) the radar plot
    is rendered with so the inset is exactly the active scale.

    Sized for legibility: thicker bar, integer ticks every 10 dBZ, a
    thin black border, and an explicit "dBZ" unit label.
    """
    cax = inset_axes(
        ax,
        width="34%", height="4.5%",
        loc="lower left",
        bbox_to_anchor=(0.02, 0.025, 1, 1),
        bbox_transform=ax.transAxes,
        borderpad=0,
    )
    cax.set_facecolor((1, 1, 1, 0.9))
    sm = ScalarMappable(norm=Normalize(vmin=opts.vmin, vmax=opts.vmax),
                        cmap="pyart_NWSRef")
    cb = plt.colorbar(sm, cax=cax, orientation="horizontal")
    # Ticks every 20 dBZ — 11 ticks at 10-dBZ spacing crowd into the
    # 340 px bar and the appended upper bound (e.g. 75) glues onto the
    # last decade tick (70). Even spacing reads cleanly at this size.
    lo = int(math.ceil(opts.vmin / 20.0) * 20)
    hi = int(math.floor(opts.vmax / 20.0) * 20)
    ticks = list(range(lo, hi + 1, 20))
    cb.set_ticks(ticks)
    cb.ax.tick_params(labelsize=9, length=3, pad=2)
    # cb.outline is typed `Spine | None`; the matplotlib stubs flag
    # method calls as "Spine not callable [operator]". At runtime it's
    # always a Spine on a freshly constructed colorbar — silence mypy.
    cb.outline.set_edgecolor("black")  # type: ignore[operator]
    cb.outline.set_linewidth(0.8)  # type: ignore[operator]
    # Inline "dBZ" label to the right of the bar (instead of a centered
    # label below) so the bar + ticks + unit fit in a single horizontal
    # band without flowing past the radar axes bottom.
    bar_box = cax.get_position()
    ax.text(
        bar_box.x1 + 0.005, (bar_box.y0 + bar_box.y1) / 2,
        "dBZ",
        transform=ax.figure.transFigure,
        ha="left", va="center",
        fontsize=9, fontweight="bold", color="black",
        zorder=11,
    )


def add_scale_bar(ax: Any, length_km: float = 20.0) -> None:
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


def add_radar_marker(
    ax: Any, lat: float, lon: float, station_id: str
) -> None:
    """Mark the radar site with a scope-style glyph + station label.

    Drawn as a small dot inside an open ring — a "scope center" cue that
    reads as radar at a glance and gives the eye a fixed origin for the
    sweep. The station ID sits to the right with a white halo so it
    survives over reflectivity returns.
    """
    import matplotlib.patheffects as path_effects

    pc = ccrs.PlateCarree()
    # Outer ring: open circle, 10pt diameter — the "scope" silhouette.
    ax.plot(
        lon, lat, marker="o", markersize=10,
        markerfacecolor="none", markeredgecolor="black", markeredgewidth=1.2,
        transform=pc, zorder=11,
    )
    # Inner dot: solid 3pt — the antenna location.
    ax.plot(
        lon, lat, marker="o", markersize=3,
        markerfacecolor="black", markeredgecolor="black",
        transform=pc, zorder=12,
    )
    label = ax.text(
        lon, lat, f"  {station_id}",
        transform=pc,
        fontsize=8, fontweight="bold", color="black",
        ha="left", va="center", zorder=12,
    )
    label.set_path_effects([
        path_effects.Stroke(linewidth=2.5, foreground="white"),
        path_effects.Normal(),
    ])


def add_north_arrow(ax: Any) -> None:
    """Place a small N + upward arrow glyph in the upper-right of the axes.

    Uses axes-fraction coords so position is invariant to data extent.
    The arrow is drawn separately from the "N" label so the head points
    *up* (toward the label) — ``annotate`` always points its arrow toward
    ``xy``, so the previous single-call form had the head at the bottom.
    """
    # Upward-pointing arrow: tail at 0.91, head at 0.96.
    ax.annotate(
        "",
        xy=(0.95, 0.96),
        xytext=(0.95, 0.91),
        xycoords="axes fraction",
        textcoords="axes fraction",
        arrowprops=dict(facecolor="black", edgecolor="black",
                        width=2, headwidth=8, headlength=8),
        zorder=10,
    )
    ax.text(
        0.95, 0.97, "N",
        transform=ax.transAxes,
        ha="center", va="bottom",
        fontsize=12, fontweight="bold", color="black",
        zorder=10,
    )


def add_footer(
    footer_ax: Any,
    scan: DecodedScan,
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
        ha="center", va="center", fontsize=12, color="#333",
        transform=footer_ax.transAxes,
    )

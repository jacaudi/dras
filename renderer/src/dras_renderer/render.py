"""Render a decoded Level II volume to a PPI PNG."""

from __future__ import annotations

import io
import math
from dataclasses import dataclass
from typing import Any

# Headless backend MUST be selected before importing pyplot. ``matplotlib.use``
# is authoritative — it overrides the env-var-based detection that runs at
# import time. No need to also set MPLBACKEND in the process env: ``use`` wins.
import matplotlib

matplotlib.use("Agg")

import cartopy.crs as ccrs  # type: ignore[import-untyped]
import cartopy.feature as cfeature  # type: ignore[import-untyped]
import matplotlib.pyplot as plt
import pyart  # type: ignore[import-untyped]
from matplotlib.axes import Axes
from matplotlib.figure import Figure

from dras_renderer import basemap, furniture
from dras_renderer.decode import DecodedScan
from dras_renderer.version import VERSION


DEFAULT_RANGE_KM: float = 150.0
"""Canonical default radar view range, in km half-extent.

Single source of truth — referenced by ``RenderOptions.range_km``,
``service.RenderRequest.range_km``, and the HTTP Query default in
``app.py``. Change this constant and all three layers move together.

150 km is wide enough to keep regional context (Olympic Peninsula,
Cascades, BC border for KATX) while still leaving named cities
legible; tighter values (50–75 km) lose the synoptic feel, and the
radar's nominal 230 km range pushes named places to pinpricks.
"""

DEFAULT_RANGE_MIN_KM: float = 10.0
DEFAULT_RANGE_MAX_KM: float = 460.0

DEFAULT_WIDTH: int = 1000
DEFAULT_HEIGHT: int = 1100
"""Canonical default radar plot dimensions, in pixels.

Single source of truth — referenced by ``RenderOptions``,
``service.RenderRequest``, and the HTTP Query defaults in ``app.py``.

Width is fixed by ``range_km`` (x-half-extent); height is taller than
width by design so the projected view extends further N-S, keeping
edge-of-coverage features (e.g. Mt Rainier ~150 km south of KATX) off
the south margin instead of glued to it.
"""


@dataclass(frozen=True)
class RenderOptions:
    """Options controlling render output."""

    width: int = DEFAULT_WIDTH
    height: int = DEFAULT_HEIGHT
    range_km: float = DEFAULT_RANGE_KM
    dpi: int = 100

    # White margin (in pixels) between the radar axes and the figure edge.
    # Frames the basemap so it doesn't bleed to the PNG border.
    border_px: int = 15

    # Reflectivity color scale bounds, dBZ.
    vmin: float = -20.0
    vmax: float = 75.0

    # View center. Default (None, None) centers on the radar location.
    # Override to re-center on a metro area or arbitrary point — the PPI
    # is plotted in full and ``set_extent`` clips the visible region.
    center_lat: float | None = None
    center_lon: float | None = None

    # Clutter filter. Set ``clutter_filter=False`` to disable (renders the
    # raw Py-ART output, useful for QA / debugging).
    clutter_filter: bool = True
    # Reflectivity floor in dBZ. Anything weaker is suppressed — kills
    # noise floor + most ground/sea clutter and biota. 15 dBZ is a typical
    # NWS display threshold: keeps moderate-light rain (~drizzle and up)
    # while rejecting weak biological scatter and sub-precipitation echo.
    clutter_min_dbz: float = 15.0
    # Cross-correlation coefficient (RhoHV) floor. Real precip is ~>0.95;
    # non-meteorological returns (clutter, biology, AP) drop below ~0.85.
    # Only applied if the field is present (NEXRAD has been dual-pol since
    # the 2013 upgrade — every operational radar carries it).
    clutter_min_rhohv: float = 0.85
    # Despeckle: drop isolated gates smaller than this many connected
    # cells. 10 is conservative; raise to be more aggressive.
    despeckle_size: int = 10

    # Map enrichment toggles (cities are the biggest contributor to
    # render time among these, mostly because populated_places is the
    # largest pre-warmed shapefile).
    show_lakes: bool = True
    show_borders: bool = True
    show_cities: bool = True
    # SCALERANK is a Natural Earth importance score; lower = bigger city.
    # ≤4 ≈ "regional/global cities only" (Seattle, Tacoma). ≤6 includes
    # mid-size suburbs (Bellevue, Redmond). 8 includes most named towns
    # (Renton, Bremerton, Everett, Bellingham, Olympia, Aberdeen). 10 is
    # the Natural Earth maximum — every named populated place.
    cities_max_scalerank: int = 8

    # New basemap layers (Task 4–7).
    show_counties: bool = True
    show_roads: bool = True
    show_peaks: bool = True

    # Cartographic furniture (Task 9–12).
    show_colorbar: bool = True
    show_scale_bar: bool = True
    show_north_arrow: bool = True
    show_footer: bool = True
    # Mark the radar antenna location with a scope-style glyph + station
    # label. Useful as a fixed reference point for the sweep origin.
    show_radar_marker: bool = True

    # Label deconfliction (Task 8). When False, falls back to the existing
    # fixed-offset placement with a white text halo.
    deconflict_labels: bool = True


def render_base_reflectivity(
    scan: DecodedScan,
    opts: RenderOptions,
    *,
    data_age_seconds: float | None = None,
) -> bytes:
    """Render the lowest-tilt base reflectivity as a PPI on a Cartopy basemap.

    Returns PNG bytes sized exactly to ``(opts.width, opts.height)``.

    When ``data_age_seconds`` is provided, the axes title is overridden to
    show both the volume start time (``scan.scan_time``) and the freshest
    chunk's age at render time, so callers can see "+Δ Ns" at a glance.
    When ``None``, Py-ART's default title is left intact.
    """
    fig, _ax = _render_figure(scan, opts, data_age_seconds)
    try:
        buf = io.BytesIO()
        # IMPORTANT: do NOT pass bbox_inches="tight" — it crops to content
        # and produces non-deterministic output dimensions, which would
        # break test_render_dimensions_match_options. Caller asks for an
        # exact (width, height); honor it.
        fig.savefig(buf, format="png", dpi=opts.dpi)
        return buf.getvalue()
    finally:
        plt.close(fig)


def _render_figure(
    scan: DecodedScan,
    opts: RenderOptions,
    data_age_seconds: float | None,
) -> tuple[Figure, Axes]:
    """Build the matplotlib figure + axes for a render.

    Split out from ``render_base_reflectivity`` so tests can introspect the
    axes (title, artists) before the figure is closed. The caller owns the
    returned ``Figure`` and is responsible for calling ``plt.close(fig)``.
    """
    radar = scan.radar
    radar_lat = float(radar.latitude["data"][0])
    radar_lon = float(radar.longitude["data"][0])

    center_lat = opts.center_lat if opts.center_lat is not None else radar_lat
    center_lon = opts.center_lon if opts.center_lon is not None else radar_lon

    projection = ccrs.LambertConformal(
        central_latitude=center_lat,
        central_longitude=center_lon,
    )

    footer_height_frac = 0.08 if opts.show_footer else 0.0
    footer_px = round(opts.height * footer_height_frac)
    total_height_px = opts.height + footer_px

    fig = plt.figure(
        figsize=(opts.width / opts.dpi, total_height_px / opts.dpi),
        dpi=opts.dpi,
    )

    # Carve the figure into [radar plot on top, footer strip below].
    # ax is a Cartopy GeoAxes (since projection= is passed) — typed as Any
    # because mpl stubs return plain Axes from add_axes, which lacks the
    # GeoAxes-specific set_extent/add_feature methods we need below.
    # ``border_px`` insets the radar axes on all four sides so the basemap
    # is framed by white margin instead of bleeding to the figure edge.
    radar_top_frac = opts.height / total_height_px
    inset_x = opts.border_px / opts.width
    inset_y = opts.border_px / total_height_px
    ax: Any = fig.add_axes(
        (
            inset_x,
            (1 - radar_top_frac) + inset_y,
            1 - 2 * inset_x,
            radar_top_frac - 2 * inset_y,
        ),
        projection=projection,
    )
    footer_ax = (
        fig.add_axes((0, 0, 1, 1 - radar_top_frac)) if opts.show_footer else None
    )

    # 1° lat ≈ 111 km everywhere; 1° lon ≈ 111 cos(lat) km. Without the
    # cos(lat) correction the east-west extent stretches by 33% at KATX
    # (lat ~48°) and ~50% at high-latitude AK stations.
    #
    # ``range_km`` defines the E-W half-extent. The N-S half-extent
    # scales with the radar axes' pixel aspect — when the figure is
    # taller than wide, the visible view extends further north/south so
    # edge-of-coverage features (e.g. Mt Rainier south of KATX) don't
    # crowd against the bottom margin.
    ax_w_px = (1 - 2 * inset_x) * opts.width
    ax_h_px = (radar_top_frac - 2 * inset_y) * total_height_px
    y_aspect = ax_h_px / ax_w_px

    delta_lon = opts.range_km / (
        111.0 * max(math.cos(math.radians(center_lat)), 1e-6)
    )
    delta_lat = (opts.range_km / 111.0) * y_aspect
    extent = (
        center_lon - delta_lon,
        center_lon + delta_lon,
        center_lat - delta_lat,
        center_lat + delta_lat,
    )
    ax.set_extent(extent, crs=ccrs.PlateCarree())
    # Lock the visible projected box to a rectangle sized to the axes
    # aspect ratio, in meters. The projection is centered on
    # (center_lat, center_lon), so (0, 0) in projected coords is the
    # view center; a symmetric ±range_m_x by ±range_m_y box fills the
    # axes box exactly.
    #
    # Without this, Cartopy's set_extent projects the lat/lon corners
    # into a slightly-curved quadrilateral whose bounding rectangle has
    # a ~2.5% non-unit aspect at ~48° lat — and Cartopy's default
    # ``adjustable="box"`` then shrinks the axes box to match, leaving
    # uneven white margins inside the figure. Locking the projected
    # limits sidesteps the auto-aspect entirely so all four margins
    # equal the figure-level border_px inset.
    range_m_x = opts.range_km * 1000.0
    range_m_y = range_m_x * y_aspect
    ax.set_xlim(-range_m_x, range_m_x)
    ax.set_ylim(-range_m_y, range_m_y)

    basemap.add_land_water_fill(ax, extent)

    if opts.show_counties:
        basemap.add_counties(ax)

    # Remaining basemap layers, in the design z-order:
    # states → roads → coastline → lakes (outlined) → borders.
    # Lakes come after coastline so their outline isn't cut by the
    # coastline stroke where they touch the shore.
    ax.add_feature(cfeature.STATES.with_scale("50m"), edgecolor="gray", linewidth=0.5)
    if opts.show_roads:
        basemap.add_roads(ax, extent)
    ax.add_feature(cfeature.COASTLINE.with_scale("50m"), edgecolor="black", linewidth=0.5)
    if opts.show_lakes:
        # Two layers — the global ``lakes`` shapefile carries the Great
        # Lakes / large international lakes, and the NA-specific
        # ``lakes_north_america`` carries metro-scale lakes (Lake
        # Washington, Lake Tahoe, Lake Pontchartrain) that the global
        # layer drops at every resolution. Both fill with WATER_COLOR
        # since the 10m LAND polygon covers the bottom water rectangle
        # wherever it touches a lake bed — a hollow outline alone leaves
        # cream-on-cream and the lake disappears.
        ax.add_feature(
            cfeature.LAKES.with_scale("10m"),
            facecolor=basemap.WATER_COLOR,
            edgecolor=basemap.LAKE_OUTLINE,
            linewidth=0.4,
            zorder=3,
        )
        basemap.add_inland_lakes(ax)
    if opts.show_borders:
        ax.add_feature(
            cfeature.BORDERS.with_scale("50m"),
            edgecolor="gray",
            linewidth=0.5,
        )

    gatefilter = _build_clutter_filter(radar, opts) if opts.clutter_filter else None

    display = pyart.graph.RadarMapDisplay(radar)
    # sweep=0 == lowest tilt: Py-ART sorts sweeps by ascending elevation.
    display.plot_ppi_map(
        "reflectivity",
        sweep=0,
        ax=ax,
        gatefilter=gatefilter,
        colorbar_flag=False,
        title_flag=True,
        vmin=opts.vmin,
        vmax=opts.vmax,
        cmap="pyart_NWSRef",
        embellish=False,  # We add our own basemap features above.
    )

    # Override Py-ART's default title to surface both the volume start time
    # and the freshest-chunk age — answers "is this image stale?" at a glance.
    # MUST come after plot_ppi_map, which sets its own title via title_flag=True.
    if data_age_seconds is not None and not opts.show_footer:
        ax.set_title(
            f"{scan.station_id} {scan.elevation_deg:.1f} Deg. "
            f"{scan.scan_time.isoformat()}  +Δ {data_age_seconds:.0f}s"
        )
    elif opts.show_footer:
        ax.set_title("")  # suppress Py-ART's auto title

    if opts.show_cities:
        basemap.add_cities(
            ax,
            extent,
            opts.cities_max_scalerank,
            deconflict=opts.deconflict_labels,
        )

    if opts.show_peaks:
        basemap.add_peaks(ax, extent, scan.station_id)

    if opts.show_radar_marker:
        furniture.add_radar_marker(ax, radar_lat, radar_lon, scan.station_id)

    if opts.show_colorbar:
        furniture.add_colorbar(ax, opts)

    if opts.show_scale_bar:
        furniture.add_scale_bar(ax)

    if opts.show_north_arrow:
        furniture.add_north_arrow(ax)

    if opts.show_footer and footer_ax is not None:
        furniture.add_footer(
            footer_ax, scan, data_age_seconds, renderer_version=VERSION,
        )

    return fig, ax


def _build_clutter_filter(radar: Any, opts: RenderOptions) -> Any:
    """Construct a Py-ART GateFilter for non-meteorological echo removal.

    Stack:
      1. exclude_invalid: drops missing/masked gates outright.
      2. exclude_below(reflectivity, ~5 dBZ): kills noise floor and most
         clutter, biology, and weak returns.
      3. exclude_below(cross_correlation_ratio, ~0.85): the dual-pol
         "is this meteorological?" test. Real precip >~0.95; non-met <~0.85.
      4. despeckle_field: drops isolated single-gate noise pixels left
         over after the threshold passes.
    """
    gf = pyart.filters.GateFilter(radar)
    gf.exclude_invalid("reflectivity")
    gf.exclude_below("reflectivity", opts.clutter_min_dbz)
    if "cross_correlation_ratio" in radar.fields:
        gf.exclude_below("cross_correlation_ratio", opts.clutter_min_rhohv)
    pyart.correct.despeckle_field(
        radar, "reflectivity", gatefilter=gf, size=opts.despeckle_size
    )
    return gf

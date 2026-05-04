"""Cartographic furniture: colorbar, scale bar, N arrow, footer.

Furniture functions add visual elements that are part of the *map*, not
the data — anything a cartographer would call "marginalia." Each function
takes the radar axes (and any other context it needs) and mutates it.
"""

from __future__ import annotations

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

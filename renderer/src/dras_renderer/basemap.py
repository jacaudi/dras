"""Basemap layer helpers for the radar PPI render.

Each function takes a Cartopy GeoAxes and adds one layer. Functions are
pure with respect to the axes (they mutate it) and have no return value
unless documented otherwise. Loaders that read shapefiles wrap them in
@lru_cache(maxsize=1) so repeated renders don't re-read disk.
"""

from __future__ import annotations

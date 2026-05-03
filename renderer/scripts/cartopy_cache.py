#!/usr/bin/env python3
"""Warm and verify the renderer's Cartopy Natural Earth cache.

Used twice in the Dockerfile:
  1. In the builder stage (as root) to populate /root/.local/share/cartopy.
  2. In the runtime stage, run twice — once as the baked-in `renderer` user
     and once as an arbitrary policy-pinned uid (e.g. 65532) — to assert
     the cache COPY landed correctly and is reachable under any uid.

Loading a Natural Earth feature for the first time downloads it; subsequent
loads read from the on-disk cache. We assert each layer yields a non-empty
geometry collection so a missing or unreadable cache fails the build instead
of first render in production.

Usage:
    python cartopy_cache.py [<label>]

The optional ``label`` argument is echoed in the success message — handy for
distinguishing the three Dockerfile invocations in build logs (``builder``,
``uid renderer``, ``uid 65532``).
"""
from __future__ import annotations

import sys

import cartopy.feature as cf  # type: ignore[import-untyped]
import cartopy.io.shapereader as shp  # type: ignore[import-untyped]

# Layers used by the renderer's PPI plot. Keep this list in sync with
# render.py — adding a layer there without warming it here means first
# render in production downloads it, which on read-only/uid-mismatch
# pods fails outright.
_PHYSICAL_50M = ("STATES", "COASTLINE", "LAKES", "BORDERS")
_POPULATED_PLACES_RES = "10m"


def main() -> int:
    counts: dict[str, int] = {}
    for name in _PHYSICAL_50M:
        feature = getattr(cf, name).with_scale("50m")
        counts[name.lower()] = len(list(feature.geometries()))

    cities_path = shp.natural_earth(
        category="cultural",
        name="populated_places",
        resolution=_POPULATED_PLACES_RES,
    )
    counts["cities"] = len(list(shp.Reader(cities_path).records()))

    if not all(counts.values()):
        empty = [name for name, n in counts.items() if not n]
        print(
            f"cartopy cache verification failed: empty layers {empty}",
            file=sys.stderr,
        )
        return 1

    label = sys.argv[1] if len(sys.argv) > 1 else "default"
    summary = ", ".join(f"{n} {name}" for name, n in counts.items())
    print(f"cartopy cache verified ({label}): {summary}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

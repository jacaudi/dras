# syntax=docker/dockerfile:1.7

ARG PYTHON_VERSION=3.12
ARG VERSION="development"

FROM python:${PYTHON_VERSION}-slim-bookworm AS builder

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1 \
    UV_LINK_MODE=copy \
    UV_PYTHON_DOWNLOADS=never

# Build deps for any wheels that need C compilation.
RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential \
        libgeos-dev \
        libproj-dev \
        proj-data \
        proj-bin \
        libgdal-dev \
        ca-certificates \
        curl \
    && rm -rf /var/lib/apt/lists/*

# Install uv.
COPY --from=ghcr.io/astral-sh/uv:0.11 /uv /usr/local/bin/uv

WORKDIR /app
# Two-stage uv sync, by design — DO NOT consolidate:
#   1. dependency-only sync (skips the local project) cached on
#      pyproject.toml + uv.lock; reused as long as deps don't change.
#   2. full sync (after src/ + README copies) installs the local package
#      itself. README is required because hatchling reads it for the
#      editable build's metadata.
# Collapsing into a single COPY+sync invalidates the heavy dep layer on
# every src change.
COPY pyproject.toml uv.lock /app/
RUN uv sync --frozen --no-dev --no-install-project

COPY src /app/src
COPY README.md /app/README.md
RUN uv sync --frozen --no-dev

# Pre-warm the Cartopy Natural Earth cache so first request is fast. Cartopy
# stores under ~/.local/share/cartopy by default; we run as root in this stage
# so it lands in /root/.local/share/cartopy. The script is the single source
# of truth for the layer list — keep it in sync with render.py.
COPY scripts/cartopy_cache.py /app/scripts/cartopy_cache.py
RUN /app/.venv/bin/python /app/scripts/cartopy_cache.py builder

###############################################################################
FROM python:${PYTHON_VERSION}-slim-bookworm AS runtime

ARG VERSION
# Runtime defaults match the chart (runAsUser: 65532). No /etc/passwd
# entry is created for the runtime uid — the image is uid-agnostic post
# #108: HOME is pinned via ENV so $HOME resolves regardless of which uid
# the pod runs as, and the cache directory is world-readable+traversable
# (chmod below). Operators can still override runAsUser to satisfy
# cluster policy without breaking renders.
#
# MPLCONFIGDIR points at /tmp so Matplotlib never needs to write to $HOME
# on import. Mirrors the chart-side setting added for #101 — bake it into
# the image so the image is self-sufficient even without the chart's env
# block.
ENV DRAS_RENDERER_VERSION=${VERSION} \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    MPLBACKEND=Agg \
    HOME=/home/renderer \
    MPLCONFIGDIR=/tmp/matplotlib \
    PATH="/app/.venv/bin:$PATH"

# Runtime libs only — no -dev packages. bookworm-slim ships GDAL 3.6 binaries
# under the libgdal32 SONAME package (verified from libgdal-dev's runtime deps
# in this build context). The number suffix tracks the SO-version, not the
# upstream release: GDAL 3.6.x → libgdal32.
RUN apt-get update && apt-get install -y --no-install-recommends \
        libgeos-c1v5 \
        libproj25 \
        proj-data \
        libgdal32 \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /home/renderer

WORKDIR /app
COPY --from=builder /app/.venv /app/.venv
COPY --from=builder /app/scripts /app/scripts
# Copy the warmed Cartopy cache and make every path under HOME
# world-readable+traversable so the image works under any non-root uid
# (operator-overridden runAsUser values, cluster policy, etc.). Issue #107.
COPY --from=builder /root/.local/share/cartopy /home/renderer/.local/share/cartopy
RUN chmod -R a+rX /home/renderer
# Verify the warmed cache is reachable under the chart-default uid. If a
# future change re-introduces uid-coupled perms, the build fails here
# instead of at first request in production.
USER 65532
RUN /app/.venv/bin/python /app/scripts/cartopy_cache.py "uid 65532"
USER root
COPY src /app/src
RUN chmod -R a+rX /app/src

USER 65532
EXPOSE 8080

CMD ["dras-renderer"]

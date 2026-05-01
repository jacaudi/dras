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
COPY --from=ghcr.io/astral-sh/uv:0.5 /uv /usr/local/bin/uv

WORKDIR /app
COPY pyproject.toml uv.lock /app/
RUN uv sync --frozen --no-dev

# Copy source after dependency install for better layer caching.
COPY src /app/src
RUN uv sync --frozen --no-dev

# Pre-warm the Cartopy Natural Earth cache so first request is fast. Cartopy
# stores under ~/.local/share/cartopy by default; we run as root in this stage
# so it lands in /root/.local/share/cartopy.
RUN /app/.venv/bin/python -c "\
import cartopy.feature as cf;\
list(cf.STATES.with_scale('50m').geometries());\
list(cf.COASTLINE.with_scale('50m').geometries());\
print('cartopy cache warmed')\
"

###############################################################################
FROM python:${PYTHON_VERSION}-slim-bookworm AS runtime

ARG VERSION
ENV DRAS_RENDERER_VERSION=${VERSION} \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    MPLBACKEND=Agg \
    PATH="/app/.venv/bin:$PATH"

# Runtime libs only — no -dev packages. Use libgdal36 (bookworm slim ships GDAL
# 3.6 series; libgdal32 is older Debian).
RUN apt-get update && apt-get install -y --no-install-recommends \
        libgeos-c1v5 \
        libproj25 \
        proj-data \
        libgdal36 \
        ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --create-home --uid 10001 renderer

WORKDIR /app
COPY --from=builder /app/.venv /app/.venv
# Copy the warmed Cartopy cache and chown it for the renderer user.
COPY --from=builder --chown=renderer:renderer /root/.local/share/cartopy /home/renderer/.local/share/cartopy
COPY --chown=renderer:renderer src /app/src

USER renderer
EXPOSE 8080

CMD ["dras-renderer"]

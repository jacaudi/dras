"""FastAPI application factory."""

from __future__ import annotations

from fastapi import FastAPI

from dras_renderer.version import VERSION


def build_app() -> FastAPI:
    """Build and return the FastAPI application."""
    app = FastAPI(title="dras-renderer", version=VERSION)

    @app.get("/healthz")
    async def healthz() -> dict[str, str]:
        return {"status": "ok", "renderer_version": VERSION}

    return app

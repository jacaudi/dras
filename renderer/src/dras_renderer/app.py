"""FastAPI application factory."""

from __future__ import annotations

import asyncio
import base64
from typing import cast

from fastapi import FastAPI, HTTPException, Query, Request
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field

from dras_renderer.cache import RenderCache
from dras_renderer.config import Config
from dras_renderer.s3 import S3Client
from dras_renderer.service import (
    RenderRequest,
    RenderResponse,
    RenderService,
    ServiceError,
)
from dras_renderer.version import VERSION

# HTTP status mapping per ServiceError.code.
_STATUS_FOR_CODE: dict[str, int] = {
    "station_unknown": 404,
    "no_recent_scan": 503,
    "decode_failed": 502,
    "unsupported_product": 400,
    "internal": 500,
}


class MetadataModel(BaseModel):
    station: str
    product: str
    scan_time: str  # ISO-8601 UTC
    elevation_deg: float
    vcp: int
    renderer_version: str


class RenderEnvelope(BaseModel):
    image: str = Field(..., description="base64-encoded PNG bytes")
    metadata: MetadataModel


def build_app(config: Config | None = None) -> FastAPI:
    cfg = config or Config.from_env()
    app = FastAPI(title="dras-renderer", version=VERSION)
    app.state.config = cfg
    app.state.cache = RenderCache(max_size=cfg.cache_size)
    app.state.s3 = S3Client(bucket=cfg.s3_bucket, region=cfg.aws_region, anonymous=True)
    app.state.service = RenderService(s3=app.state.s3, cache=app.state.cache)

    @app.get("/healthz")
    async def healthz() -> dict[str, str]:
        return {"status": "ok", "renderer_version": VERSION}

    @app.get("/render/{station}", response_model=RenderEnvelope)
    async def render(
        request: Request,
        station: str,
        product: str = Query("base_reflectivity"),
        range_km: float = Query(230.0, ge=10.0, le=460.0),
        width: int = Query(800, ge=200, le=4000),
        height: int = Query(800, ge=200, le=4000),
    ) -> RenderEnvelope:
        svc = cast(RenderService, request.app.state.service)
        req = RenderRequest(
            station=station.upper(),
            product=product,
            range_km=range_km,
            width=width,
            height=height,
        )
        try:
            resp: RenderResponse = await asyncio.to_thread(svc.render, req)
        except ServiceError as exc:
            raise HTTPException(
                status_code=_STATUS_FOR_CODE.get(exc.code, 500),
                detail={"error": exc.code, "detail": exc.detail},
            ) from exc

        return RenderEnvelope(
            image=base64.b64encode(resp.png).decode("ascii"),
            metadata=MetadataModel(
                station=resp.metadata.station,
                product=resp.metadata.product,
                scan_time=resp.metadata.scan_time.isoformat(),
                elevation_deg=resp.metadata.elevation_deg,
                vcp=resp.metadata.vcp,
                renderer_version=resp.metadata.renderer_version,
            ),
        )

    @app.exception_handler(HTTPException)
    async def _http_exc_handler(_request: Request, exc: HTTPException) -> JSONResponse:
        # Surface our service-error envelope verbatim.
        if isinstance(exc.detail, dict) and "error" in exc.detail:
            return JSONResponse(status_code=exc.status_code, content=exc.detail)
        return JSONResponse(
            status_code=exc.status_code,
            content={"error": "internal", "detail": str(exc.detail)},
        )

    return app

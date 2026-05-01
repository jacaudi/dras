"""Runtime configuration loaded from environment variables."""

from __future__ import annotations

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Config:
    """Renderer process configuration.

    Attributes:
        port: TCP port for HTTP server.
        log_level: stdlib logging level name (DEBUG, INFO, WARNING, ERROR).
        cache_size: maximum LRU cache entries (per-scan rendered PNGs).
        s3_bucket: NOAA real-time chunks bucket. Override only for testing.
        aws_region: bucket region.
    """

    port: int = 8080
    log_level: str = "INFO"
    cache_size: int = 100
    s3_bucket: str = "unidata-nexrad-level2-chunks"
    aws_region: str = "us-east-1"

    @classmethod
    def from_env(cls) -> Config:
        return cls(
            port=int(os.environ.get("PORT", "8080")),
            log_level=os.environ.get("LOG_LEVEL", "INFO").upper(),
            cache_size=int(os.environ.get("CACHE_SIZE", "100")),
            s3_bucket=os.environ.get("S3_BUCKET", "unidata-nexrad-level2-chunks"),
            aws_region=os.environ.get("AWS_REGION", "us-east-1"),
        )

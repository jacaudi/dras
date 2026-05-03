"""Process entrypoint for ``dras-renderer``."""

from __future__ import annotations

import logging

import uvicorn

from dras_renderer.app import build_app
from dras_renderer.config import Config

_UVICORN_LEVELS = {"critical", "error", "warning", "info", "debug", "trace"}


def _uvicorn_log_level(level: str) -> str:
    """Normalize stdlib log-level aliases (e.g. WARN) to uvicorn's vocabulary."""
    normalized = level.strip().lower()
    if normalized == "warn":
        normalized = "warning"
    if normalized == "fatal":
        normalized = "critical"
    return normalized if normalized in _UVICORN_LEVELS else "info"


class _HealthzAccessFilter(logging.Filter):
    """Drop /healthz access-log lines unless the root logger is at DEBUG.

    Probes hit /healthz every few seconds; surfacing every one at INFO
    drowns the real request log. uvicorn.access records carry the
    request path at args[2].
    """

    def filter(self, record: logging.LogRecord) -> bool:
        args = record.args
        if isinstance(args, tuple) and len(args) >= 3 and args[2] == "/healthz":
            return logging.getLogger().getEffectiveLevel() <= logging.DEBUG
        return True


def run() -> None:
    cfg = Config.from_env()
    logging.basicConfig(
        level=cfg.log_level,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )
    logging.getLogger("uvicorn.access").addFilter(_HealthzAccessFilter())
    uvicorn.run(
        build_app(),
        host="0.0.0.0",  # service is namespace-internal; not a public bind
        port=cfg.port,
        log_level=_uvicorn_log_level(cfg.log_level),
    )


if __name__ == "__main__":
    run()

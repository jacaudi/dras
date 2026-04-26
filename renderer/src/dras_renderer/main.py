"""Process entrypoint for ``dras-renderer``."""

from __future__ import annotations

import logging

import uvicorn

from dras_renderer.app import build_app
from dras_renderer.config import Config


def run() -> None:
    cfg = Config.from_env()
    logging.basicConfig(
        level=cfg.log_level,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )
    uvicorn.run(
        build_app(),
        host="0.0.0.0",  # noqa: S104 — service is namespace-internal
        port=cfg.port,
        log_level=cfg.log_level.lower(),
    )


if __name__ == "__main__":
    run()

"""Prometheus metrics shared across the app.

Custom registry so we expose only renderer-scoped metrics, not the default
process/platform collectors.
"""

from __future__ import annotations

from prometheus_client import CollectorRegistry, Counter, Histogram

REGISTRY = CollectorRegistry(auto_describe=True)

REQUESTS_TOTAL = Counter(
    "renderer_requests_total",
    "Total /render/{station} requests, labeled by outcome.",
    labelnames=("outcome",),  # ok | error_<code>
    registry=REGISTRY,
)

RENDER_DURATION = Histogram(
    "renderer_render_duration_seconds",
    "End-to-end render time for /render/{station}.",
    buckets=(0.1, 0.25, 0.5, 1, 2, 5, 10, 20, 30, 60),
    registry=REGISTRY,
)

S3_ERRORS_TOTAL = Counter(
    "renderer_s3_errors_total",
    "S3 list/download failures (mapped from S3Error to 'internal' service errors).",
    registry=REGISTRY,
)

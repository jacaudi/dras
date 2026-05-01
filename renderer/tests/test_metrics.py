"""/metrics exposition format."""

from __future__ import annotations

from fastapi.testclient import TestClient

from dras_renderer.app import build_app


def test_metrics_endpoint_exposes_default_metrics() -> None:
    client = TestClient(build_app())
    resp = client.get("/metrics")
    assert resp.status_code == 200
    body = resp.text
    assert "renderer_requests_total" in body
    assert "renderer_render_duration_seconds" in body
    assert "renderer_s3_errors_total" in body

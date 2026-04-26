"""Health endpoint returns 200 with renderer version."""

from fastapi.testclient import TestClient

from dras_renderer.app import build_app


def test_healthz_returns_ok() -> None:
    client = TestClient(build_app())
    resp = client.get("/healthz")
    assert resp.status_code == 200
    body = resp.json()
    assert body["status"] == "ok"
    assert "renderer_version" in body

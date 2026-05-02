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


def test_render_response_scan_time_is_iso8601() -> None:
    """MetadataModel.scan_time is now `datetime`; JSON output must remain ISO-8601 with TZ."""
    from datetime import UTC, datetime

    from fastapi.testclient import TestClient

    from dras_renderer.service import RenderMetadata, RenderResponse

    fake_meta = RenderMetadata(
        station="KATX",
        product="base_reflectivity",
        scan_time=datetime(2026, 4, 29, 12, 5, 0, tzinfo=UTC),
        elevation_deg=0.5,
        vcp=212,
        renderer_version="test",
    )
    fake_resp = RenderResponse(png=b"\x89PNG\r\n\x1a\n", metadata=fake_meta)

    class StubService:
        def render(self, req):
            return fake_resp

    app = build_app()
    app.state.service = StubService()

    with TestClient(app) as tc:
        r = tc.get("/render/KATX")
    assert r.status_code == 200
    body = r.json()
    st = body["metadata"]["scan_time"]
    # Pin the canonical wire format: Pydantic v2 emits UTC datetimes as
    # "...Z" (RFC 3339). The previous code path used datetime.isoformat(),
    # which emits "...+00:00"; both encode the same instant, but the wire
    # format DID change with M12 — pin it so a future Pydantic regression
    # back to "+00:00" is caught here, not by a downstream consumer.
    assert st == "2026-04-29T12:05:00Z"

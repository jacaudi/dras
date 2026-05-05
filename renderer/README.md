# dras-renderer

NEXRAD Level II rendering service used by [dras](../dras) in advanced
deployments. Discovers the freshest volume for a station in the NOAA
real-time chunks bucket (`s3://unidata-nexrad-level2-chunks/`), assembles
the chunks, decodes with [Py-ART](https://arm-doe.github.io/pyart/), and
renders the lowest-tilt base reflectivity PPI as a PNG with a Cartopy
basemap.

## API

`GET /render/{station}` — returns a JSON envelope:

```json
{
  "image": "<base64-encoded PNG>",
  "metadata": {
    "station": "KATX",
    "product": "base_reflectivity",
    "scan_time": "2026-04-26T15:32:00Z",
    "elevation_deg": 0.5,
    "vcp": 215,
    "renderer_version": "v3.0.0"
  }
}
```

Query parameters (all optional):

| Param | Default | Range | Notes |
|---|---|---|---|
| `product` | `base_reflectivity` | — | Only value supported in v1. |
| `range_km` | `230` | `[10, 460]` | Render extent radius. 460 is the Level II max unambiguous range. |
| `width` | `1000` | `[200, 4000]` | Output PNG width. |
| `height` | `1000` | `[200, 4000]` | Radar plot height; total PNG height adds an ~8% footer strip. |

Error responses use a stable envelope:

```json
{ "error": "no_recent_scan", "detail": "no Level II volume found for KATX" }
```

| HTTP | error code | meaning |
|---|---|---|
| 400 | `unsupported_product` | Product is not `base_reflectivity`. |
| 404 | `station_unknown` | Reserved; not currently emitted. |
| 404 | `no_recent_scan` | No volume present in any slot for the station. |
| 502 | `decode_failed` | Py-ART couldn't parse the assembled volume. |
| 500 | `internal` | S3 listing/download failed (mapped from `S3Error`). |

`GET /healthz` — liveness/readiness probe; returns `{"status": "ok", "renderer_version": "..."}`.

`GET /metrics` — Prometheus exposition. Metrics:
- `renderer_requests_total{outcome=...}` — request counter labeled `ok` or `error_<code>`.
- `renderer_render_duration_seconds` — end-to-end render histogram (100ms–60s buckets).
- `renderer_s3_errors_total` — S3 list/download failure counter.

## View presets

The `?view=<name>` query parameter applies an opinionated framing for a
station (recenter + zoom). Presets are opt-in — without `?view=`, the
renderer centers on the radar with the default `range_km`.

Currently defined:

| Station | View    | Center               | Range  |
|---------|---------|----------------------|--------|
| KATX    | `metro` | 47.61°N, -122.33°E   | 70 km  |

Example:

```
GET /render/KATX?view=metro
```

### Adding a view preset

View presets live in `src/dras_renderer/station_views.py`. To add a new
framing for a station:

1. Append an entry to `_VIEWS` keyed on `(station_id, view_name)`.
2. Set `center_lat`, `center_lon`, and `range_km` to your desired framing.
3. Add a row to the table above.

Example — adding a Bay Area view for KMUX:

```python
("KMUX", "metro"): StationView(
    center_lat=37.77,
    center_lon=-122.42,
    range_km=70.0,
),
```

## Local development

Requires Python 3.12 and [uv](https://docs.astral.sh/uv/).

```bash
cd renderer
uv sync
uv run pytest                       # tests (~25 s warm)
uv run dras-renderer                # run on :8080
curl http://127.0.0.1:8080/healthz
```

The first `uv sync` takes a few minutes — Py-ART, Cartopy, and matplotlib are heavy.

## Testing

`pytest`. Tests use:
- `moto` for S3 mocking.
- A real ~8 MB KATX volume scan checked into `tests/fixtures/` (assembled from chunks, see `tests/fixtures/README.md` for provenance).
- `fastapi.testclient.TestClient` for end-to-end route tests.

Run a single suite:

```bash
uv run pytest tests/test_render.py -v
```

The first run downloads Cartopy's Natural Earth shapefiles into the user cache (~5 MB, one-time). The Dockerfile pre-warms this cache so containerized cold starts don't pay the download cost.

## Building the container

```bash
docker build --build-arg VERSION=v0.0.0-local -t dras-renderer:local .
docker run --rm -p 8080:8080 dras-renderer:local
curl http://127.0.0.1:8080/healthz
```

Build takes ~5–10 min (Py-ART + Cartopy + GDAL). Final image ~600–800 MB.

## Configuration

| env | default | meaning |
|---|---|---|
| `PORT` | `8080` | TCP listen port. |
| `LOG_LEVEL` | `INFO` | stdlib logging level. |
| `CACHE_SIZE` | `100` | LRU entries (per-snapshot rendered PNGs). |
| `S3_BUCKET` | `unidata-nexrad-level2-chunks` | NOAA real-time chunks bucket. Override only for testing. |
| `AWS_REGION` | `us-east-1` | Bucket region. |
| `DRAS_RENDERER_VERSION` | `development` | Reported by `/healthz` and the metadata envelope. Set by CI from the git tag. |

## Architecture

Single FastAPI worker per replica. The render pipeline is serialized inside `RenderService` with a `threading.Lock` (cachetools caches aren't thread-safe, and renders dispatched via `asyncio.to_thread` could otherwise race). One render at a time per replica is acceptable since renders take seconds and traffic is low (driven by dras's polling).

Two in-memory caches:
- **Volume-pointer cache** (per-station, ~30 s TTL) inside `S3Client.latest_volume`. Suppresses the 1000-LIST fan-out on a burst of requests.
- **Rendered-PNG cache** (LRU, `CACHE_SIZE`) keyed by `(station, latest_chunk_time.isoformat())`. A second request hitting while the same volume is freshest does not re-decode/re-render.

Both reset on restart; no persistence.

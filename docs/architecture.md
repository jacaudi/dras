# Architecture

DRAS is two services in one repository, deployed independently.

## Component split

```
┌─────────────────────────┐         ┌────────────────────────────┐
│ dras (Go)               │  HTTP   │ dras-renderer (Python)     │
│  — orchestrator         │ ──────► │  — Py-ART decoder          │
│  — NWS API polling      │  JSON   │  — matplotlib/Cartopy      │
│  — VCP change detection │ ◄────── │      renderer              │
│  — Pushover notifier    │  PNG    │  — chunks-bucket fetch     │
│                         │ (base64)│                            │
└─────────────────────────┘         └─────────────┬──────────────┘
                                                  │
                                                  ▼
                                     s3://unidata-nexrad-level2-chunks/
```

## Modes

DRAS picks an `image.Source` at startup based on `RENDERER_URL`:

- **Basic** (default) — `internal/image.Service` downloads `radar.weather.gov/ridge/standard/{station}_0.gif` directly.
- **Advanced** — `internal/renderer.Client` calls the renderer's `/render/{station}` endpoint and decodes the JSON envelope.
- **Disabled** — neither, no image attached to notifications.

The two paths are mutually exclusive. Selection happens once at startup; no runtime switching, no per-request fallback.

## Renderer pipeline

For each `/render/{station}` request:

1. **Latest-volume discovery** — `S3Client.latest_volume(station)` fans out one `list_objects_v2` call per volume slot (0–999), picks the slot whose newest chunk filename has the largest `YYYYMMDD-HHMMSS` prefix, and caches the result per station for ~30 s. Volume IDs cycle, so the slot number alone doesn't identify recency — the chunk timestamp does.
2. **Slot-reuse safety** — a slot mid-overwrite can hold chunks from two distinct volumes. `latest_volume` filters chunks to those sharing the winning timestamp prefix, so `download_volume` cannot concatenate cross-volume chunks into a Frankenstein blob.
3. **Volume assembly** — `download_volume` fetches all chunks in chunk-num order and concatenates them as-is. Real chunks are AR2V-framed: chunk 1 begins with the 24-byte `AR2V0006...` header and every chunk carries LDM-record-framed bzip2 sections. The concatenation IS a `_V06` Level II Archive file. Py-ART handles internal bzip2 itself; we never call `bz2.decompress` at this layer.
4. **Decode** — `pyart.io.read_nexrad_archive(BytesIO(...))` parses the blob into a Radar object.
5. **Render** — lowest-tilt sweep on a Cartopy LambertConformal basemap with the NWSRef colormap. Output sized exactly to `width × height` (no `bbox_inches="tight"` so dimensions are deterministic). The longitude span uses `cos(lat)` so high-latitude stations don't get a stretched view.

## Caches

Two layers, both in-memory, both bounded, both lost on restart:

| Cache | Where | Key | Bound | TTL |
|---|---|---|---|---|
| Volume-pointer | `S3Client._latest_cache` | `station_id` | 256 entries | 30 s (config) |
| Rendered PNG | `RenderCache` (Task 4) + `RenderService._meta` | `(station_id, latest_chunk_time.isoformat())` | `CACHE_SIZE` (config, default 100) | LRU eviction |

The PNG cache key uses the volume's `latest_chunk_time` because slot numbers are reused — a slot's contents change over time.

## Concurrency

- The route is `async def`. The blocking sync `RenderService.render` runs via `await asyncio.to_thread(...)` so the asyncio event loop stays responsive (e.g. `/healthz` is not stalled by a render in progress).
- `cachetools` caches are not thread-safe. `RenderService` holds a `threading.Lock` and serializes all renders. Render contention is rare (driven by dras's polling cadence) and dwarfed by per-render latency, so the serialization cost is negligible.
- The `S3Client` boto3 client is documented thread-safe; the 64-way LIST fan-out uses a `ThreadPoolExecutor`.

## Failure handling

Hard-fail in advanced mode: any error from the renderer surfaces to dras, which logs a warning and sends the Pushover notification **with no attachment**. dras does not fall back to the ridge GIF.

Renderer error envelope: `{"error": "<code>", "detail": "<message>"}`.

| HTTP | code | meaning |
|---|---|---|
| 400 | `unsupported_product` | Product is not `base_reflectivity`. |
| 404 | `station_unknown` | Reserved; not currently emitted. |
| 502 | `decode_failed` | Py-ART couldn't parse the assembled volume. |
| 503 | `no_recent_scan` | No volume present in any slot for the station. |
| 500 | `internal` | S3 listing/download failed (mapped from `S3Error`). |

The dras-side renderer client surfaces the code and detail verbatim in its error string.

## Observability

`/metrics` (Prometheus exposition):

- `renderer_requests_total{outcome=...}` — counter labeled `ok` or `error_<code>`.
- `renderer_render_duration_seconds` — histogram (100 ms – 60 s buckets).
- `renderer_s3_errors_total` — counter for S3 list/download failures.

`/healthz`: `{"status":"ok","renderer_version":"..."}`.

## S3 source notes

The renderer reads from `s3://unidata-nexrad-level2-chunks/`. NOAA deprecated the older `s3://noaa-nexrad-level2/` archive bucket on **2025-09-01**. The chunks bucket is real-time (chunks land within seconds of being broadcast); volumes complete in ~5 minutes. Anonymous access; no IAM, no credentials. Egress to `*.s3.amazonaws.com:443` required.

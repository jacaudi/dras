[![On Merge](https://github.com/jacaudi/dras/actions/workflows/on-merge.yml/badge.svg)](https://github.com/jacaudi/dras/actions/workflows/on-merge.yml) [![Versioned Release](https://github.com/jacaudi/dras/actions/workflows/on-release.yml/badge.svg)](https://github.com/jacaudi/dras/actions/workflows/on-release.yml)

# DRAS — Doppler Radar Alerting Service

DRAS monitors one or more WSR-88D sites via the NWS API and sends a
Pushover notification with a radar image whenever the radar's status,
mode, or VCP changes.

## Two deployment modes

DRAS picks an image source at startup based on `RENDERER_URL`:

| Mode | Image source | Footprint | When to choose |
|---|---|---|---|
| **Basic** (default) | NWS pre-rendered ridge GIF (600×550) | Single small Go binary (~15 MB) | Quick setup, low resource budget, casual use. |
| **Advanced** | `dras-renderer` decoding NEXRAD Level II from S3 | Adds a ~700 MB Python container | Higher-quality rendering, custom range/dimensions, future products (velocity, composite). |

Modes are mutually exclusive. Set `RENDERER_URL` to opt into advanced
mode; leave it unset (and keep `RADAR_IMAGE_ENABLED=true`) for the
existing basic behavior.

## What is monitored

Per station:

- Volume Coverage Pattern (VCP) — Clear Air or Precipitation Mode
- Operational Status
- Power Source
- Generator State

When the VCP changes, the freshly fetched image (ridge GIF in basic mode,
rendered Level II PPI in advanced mode) is attached to the Pushover
notification.

---

## Quick start (Basic deployment)

### Standalone container

```bash
docker run -d \
  -e STATION_IDS=KRAX \
  -e PUSHOVER_USER_KEY=<KEY> \
  -e PUSHOVER_API_TOKEN=<TOKEN> \
  ghcr.io/jacaudi/dras:latest
```

### Kubernetes

See [`examples/kubernetes.yaml`](examples/kubernetes.yaml) for a
deployment + configmap + secret skeleton.

### Binary

```bash
go install github.com/jacaudi/dras@latest
STATION_IDS=KRAX PUSHOVER_USER_KEY=... PUSHOVER_API_TOKEN=... dras
```

---

## Advanced deployment

Run `dras` alongside the [`dras-renderer`](./renderer/) container and
point `RENDERER_URL` at it.

### Docker Compose

```yaml
services:
  dras-renderer:
    image: ghcr.io/jacaudi/dras-renderer:latest
    restart: unless-stopped

  dras:
    image: ghcr.io/jacaudi/dras:latest
    environment:
      RENDERER_URL: http://dras-renderer:8080
      PUSHOVER_API_TOKEN: ...
      PUSHOVER_USER_KEY: ...
      STATION_IDS: KATX,KRAX
    depends_on:
      - dras-renderer
    restart: unless-stopped
```

### What you get

Rendered base reflectivity from the most recent Level II volume scan,
NWSRef color scale, Cartopy basemap with state and coastline outlines.
The renderer fetches volumes from `s3://unidata-nexrad-level2-chunks/`
(NOAA real-time public bucket; anonymous access).

### Trade-offs

- Extra container, ~700 MB image, ~512 MB RAM minimum per replica.
- Egress to `*.s3.amazonaws.com` required for the renderer.
- If the renderer is unreachable, the Pushover notification still goes
  out — text-only, no image. DRAS does not fall back to the ridge GIF in
  advanced mode.

---

## Configuration

| env | default | meaning |
|---|---|---|
| `STATION_IDS` | required | Space/comma/semicolon-separated 4-letter NEXRAD station IDs (e.g. `KATX,KRAX`). |
| `PUSHOVER_API_TOKEN` | required (unless `DRYRUN=true`) | Pushover API token. |
| `PUSHOVER_USER_KEY` | required (unless `DRYRUN=true`) | Pushover user key. |
| `INTERVAL` | `10` | Poll cadence in **minutes** (integer). |
| `DRYRUN` | `false` | Disable Pushover; use test stations KATX/KRAX. |
| `RENDERER_URL` | unset | **Advanced mode:** HTTP endpoint of `dras-renderer`. Empty → basic mode. |
| `RENDERER_TIMEOUT` | `30s` | HTTP timeout for renderer calls. |
| `RADAR_IMAGE_ENABLED` | `true` | **Basic mode:** enable/disable ridge image attach. Ignored in advanced mode. |
| `RADAR_IMAGE_URL_TEMPLATE` | NWS Ridge GIF | Basic mode: override the per-station image URL. Use `{station}` as the placeholder. |
| `RADAR_IMAGE_RETENTION` | `1h` | Basic mode: sliding window of polled images kept per station (Go duration). |
| `ALERT_VCP` | `true` | Notify on VCP changes. |
| `ALERT_STATUS` | `false` | Notify on operational status changes. |
| `ALERT_OPERABILITY` | `false` | Notify on operability status changes. |
| `ALERT_POWER_SOURCE` | `false` | Notify on power source changes. |
| `ALERT_GEN_STATE` | `false` | Notify on generator state changes. |

---

## Architecture

In advanced mode:

```
┌────────────┐  HTTP/JSON   ┌────────────────────┐
│   dras     │ ───────────▶ │   dras-renderer    │
│   (Go)     │              │ Py-ART, matplotlib │
│            │ ◀─────────── │ Cartopy, FastAPI   │
└────────────┘   PNG (b64)  └─────────┬──────────┘
                                      │
                                      ▼
              s3://unidata-nexrad-level2-chunks/
              (real-time NEXRAD Level II chunks)
```

In basic mode `dras` downloads `radar.weather.gov/ridge/standard/{station}_0.gif`
directly.

---

## Repository layout

- [`dras/`](./dras/) — the Go service (orchestrator + Pushover notifier).
- [`renderer/`](./renderer/) — the optional Python rendering service.
- [`examples/`](./examples/) — Kubernetes deployment example.

---

## Development

- Working on `dras` (Go): see [`dras/README.md`](./dras/README.md).
- Working on `dras-renderer` (Python): see [`renderer/README.md`](./renderer/README.md).

## Versioning

Semantic versioning, automated releases via [uplift](https://uplift.dev/).
Commits follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` — minor version bump (new features)
- `fix:` — patch version bump (bug fixes)
- `feat!:` or `BREAKING CHANGE:` — major version bump

Both container images (`dras` and `dras-renderer`) ship under one
unified version tag.

## Contributing

Feature improvements and bug reports welcome via PRs. Thanks!

## License

[MIT](LICENSE)

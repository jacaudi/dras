# Configuration

All configuration is via environment variables. DRAS reads them once at startup; restart to change values.

## Required

| env | meaning |
|---|---|
| `STATION_IDS` | Space/comma/semicolon-separated 4-letter NEXRAD station IDs (e.g. `KATX,KRAX`). |
| `PUSHOVER_API_TOKEN` | Pushover API token. Skipped when `DRYRUN=true`. |
| `PUSHOVER_USER_KEY` | Pushover user key. Skipped when `DRYRUN=true`. |

## Mode selection

| env | default | meaning |
|---|---|---|
| `RENDERER_URL` | unset | **Advanced mode** when set. HTTP endpoint of `dras-renderer` (e.g. `http://dras-renderer:8080`). Empty → basic mode. |
| `RENDERER_TIMEOUT` | `30s` | HTTP timeout for renderer calls (Go duration: `15s`, `1m`, etc.). |

If `RENDERER_URL` is set, basic-mode `RADAR_IMAGE_*` settings are ignored — DRAS logs a warning at startup if both are present.

## Polling and runtime

| env | default | meaning |
|---|---|---|
| `INTERVAL` | `10` | Poll cadence in **minutes** (integer ≥ 1). |
| `DRYRUN` | `false` | Disable Pushover; use test stations `KATX`/`KRAX`. |

## Logging

| env | default | meaning |
|---|---|---|
| `LOG_LEVEL` | `INFO` | Case-insensitive: `DEBUG`, `INFO`, `WARN` (or `WARNING`), `ERROR`, `FATAL` (mapped to `ERROR`). Unknown values fall back to `INFO`. |
| `LOG_FORMAT` | `text` | `text` for stdlib `slog.NewTextHandler` (`time=... level=... msg=... k=v`), `json` for `slog.NewJSONHandler` (one JSON object per line). |

## Alert toggles

Each governs whether a change in that field triggers a notification.

| env | default | event |
|---|---|---|
| `ALERT_VCP` | `true` | Volume Coverage Pattern change (clear-air ↔ precipitation, etc.). |
| `ALERT_STATUS` | `false` | Operational status change. |
| `ALERT_OPERABILITY` | `false` | Operability status change. |
| `ALERT_POWER_SOURCE` | `false` | Power-source change (utility ↔ generator). |
| `ALERT_GEN_STATE` | `false` | Generator-state change. |

## Basic mode (legacy ridge GIF)

Ignored in advanced mode.

| env | default | meaning |
|---|---|---|
| `RADAR_IMAGE_ENABLED` | `true` | Enable/disable image attach in basic mode. |
| `RADAR_IMAGE_URL_TEMPLATE` | NWS Ridge GIF | Override the per-station image URL. Use `{station}` as the placeholder. |
| `RADAR_IMAGE_RETENTION` | `1h` | Sliding window of polled images kept per station (Go duration). |

## Renderer-only settings

These apply to the `dras-renderer` container, not `dras`. Override only for testing.

| env | default | meaning |
|---|---|---|
| `PORT` | `8080` | TCP listen port. |
| `LOG_LEVEL` | `INFO` | stdlib logging level. `WARN`/`FATAL` are normalized to `WARNING`/`CRITICAL`. |
| `CACHE_SIZE` | `100` | LRU entries (per-snapshot rendered PNGs). |
| `S3_BUCKET` | `unidata-nexrad-level2-chunks` | NOAA real-time chunks bucket. |
| `AWS_REGION` | `us-east-1` | Bucket region. |
| `DRAS_RENDERER_VERSION` | `development` | Reported by `/healthz` and the metadata envelope. Set by CI from the git tag. |

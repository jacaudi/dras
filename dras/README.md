# dras (Go service)

The orchestrator: NWS API polling, VCP/status change detection, Pushover
notification dispatch. The image source is pluggable via
`internal/image.Source`; the legacy ridge fetcher lives in `internal/image`,
the HTTP renderer client in `internal/renderer`.

For top-level project info see the [repo README](../README.md).

## Local development

```bash
cd dras
go test -count=1 ./...
PUSHOVER_API_TOKEN=x PUSHOVER_USER_KEY=x STATION_IDS=KATX DRY_RUN=true \
  go run .
```

## Run against the renderer

Two terminals:

```bash
# Terminal 1 — renderer
cd renderer
uv run dras-renderer
```

```bash
# Terminal 2 — dras pointed at it
cd dras
PUSHOVER_API_TOKEN=x PUSHOVER_USER_KEY=x STATION_IDS=KATX DRY_RUN=true \
  RENDERER_URL=http://127.0.0.1:8080 \
  go run .
```

Look for `Radar image source enabled [mode=advanced, ...]` in the logs.

## Building the container

```bash
cd dras
docker build --build-arg VERSION=v0.0.0-local -t dras:local .
docker run --rm -e DRY_RUN=true -e STATION_IDS=KATX dras:local
```

## Code layout

- `main.go` — entrypoint, mode selection (basic vs advanced).
- `internal/config` — env-var loading and validation.
- `internal/image` — ridge GIF fetcher (basic mode); also defines the `Source` interface and the `Image` struct.
- `internal/renderer` — renderer HTTP client (advanced mode); implements `image.Source`.
- `internal/monitor` — polling loop, change detection, notification dispatch.
- `internal/notify` — Pushover client (with attachment support).
- `internal/radar` — `radar.Data` model, comparison, station-ID utilities.
- `internal/logger` — structured-ish logger.
- `internal/version` — build-time version metadata.

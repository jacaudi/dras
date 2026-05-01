[![On Merge](https://github.com/jacaudi/dras/actions/workflows/on-merge.yml/badge.svg)](https://github.com/jacaudi/dras/actions/workflows/on-merge.yml) [![Versioned Release](https://github.com/jacaudi/dras/actions/workflows/on-release.yml/badge.svg)](https://github.com/jacaudi/dras/actions/workflows/on-release.yml)

# DRAS — Doppler Radar Alerting Service

DRAS watches WSR-88D radar sites via the NWS API and sends a Pushover notification — with a radar image attached — when the site's mode, VCP, or operational status changes.

## Two modes

| Mode | Image source | When to choose |
|---|---|---|
| **Basic** (default) | NWS pre-rendered ridge GIF | Quick setup, single small Go binary, casual use. |
| **Advanced** | `dras-renderer` decoding NEXRAD Level II | Higher-quality rendering, custom range/dimensions, future products. Adds a ~700 MB Python sidecar. |

Set `RENDERER_URL` to opt into advanced mode; leave it unset for basic.

---

## Quickstart — Basic

```bash
docker run -d \
  -e STATION_IDS=KRAX \
  -e PUSHOVER_USER_KEY=<KEY> \
  -e PUSHOVER_API_TOKEN=<TOKEN> \
  ghcr.io/jacaudi/dras:latest
```

That's it. Default `INTERVAL` is `10` minutes per station. See [docs/configuration.md](./docs/configuration.md) for alert toggles and the full env-var matrix.

## Quickstart — Advanced

```yaml
# docker-compose.yml
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

The renderer is namespace-internal; needs egress to `*.s3.amazonaws.com`. Cold start ~10–30 s while Cartopy hydrates shapefiles (the container pre-warms this).

---

## Documentation

In-depth docs live under [`docs/`](./docs/):

- [Configuration](./docs/configuration.md) — every env var, every default.
- [Architecture](./docs/architecture.md) — service split, S3 source, caching, error contract.
- [Deployment](./docs/deployment.md) — Compose / Kubernetes recipes, sizing, networking.
- [Development](./docs/development.md) — running locally, testing, CI.

Component-specific READMEs:

- [`dras/`](./dras/) — Go orchestrator.
- [`renderer/`](./renderer/) — Python rendering service.

## License

[MIT](LICENSE)

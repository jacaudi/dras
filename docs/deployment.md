# Deployment

Two modes share `dras`; advanced mode adds the `dras-renderer` sidecar.

See [Configuration](./configuration.md) for the full env-var matrix.

## Basic mode

### Standalone container

```bash
docker run -d \
  -e STATION_IDS=KRAX \
  -e PUSHOVER_USER_KEY=<KEY> \
  -e PUSHOVER_API_TOKEN=<TOKEN> \
  ghcr.io/jacaudi/dras:latest
```

### Docker Compose

```yaml
services:
  dras:
    image: ghcr.io/jacaudi/dras:latest
    environment:
      STATION_IDS: KATX,KRAX
      PUSHOVER_USER_KEY: ...
      PUSHOVER_API_TOKEN: ...
    restart: unless-stopped
```

### Kubernetes

See [`examples/kubernetes.yaml`](../examples/kubernetes.yaml) for a deployment + configmap + secret skeleton.

## Advanced mode

Run `dras-renderer` alongside `dras` and point `RENDERER_URL` at it. The renderer is **not** intended to be exposed publicly — keep it namespace-internal.

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
      RENDERER_TIMEOUT: 30s
      PUSHOVER_API_TOKEN: ...
      PUSHOVER_USER_KEY: ...
      STATION_IDS: KATX,KRAX
    depends_on:
      - dras-renderer
    restart: unless-stopped
```

### Kubernetes

Two Deployments + one Service for the renderer. Helm via [bjw-s `app-template`](https://github.com/bjw-s/helm-charts/tree/main/charts/other/app-template) is the conventional pattern in this org; a project-owned chart is tracked in the issue tracker.

Networking notes:

- Renderer needs egress to `*.s3.amazonaws.com:443` (NetworkPolicy if applicable).
- dras → renderer on port `8080`.
- Renderer is not behind any Gateway / Ingress — internal-only.

## Resource sizing

| Component | Image | RAM (steady) | CPU (idle) | Notes |
|---|---|---|---|---|
| `dras` | ~15 MB | ~30 MB | negligible | Pure Go, single binary. |
| `dras-renderer` | ~700 MB | ~500 MB | bursty during render | Py-ART + matplotlib + Cartopy stack. ~5–15 s per render (cold), ~1–3 s (warm cache). First request after cold start may take 10–30 s while Cartopy hydrates Natural Earth shapefiles — the Dockerfile pre-warms this cache so production cold starts skip it. |

A single renderer replica is the intended deployment. Renders are serialized with a `threading.Lock`; multiple replicas don't add throughput unless you front them with a load balancer that distributes per-station requests, and the per-station volume-pointer cache is per-process so cache benefits don't compound across replicas.

## Versioning

Both images publish at the same release tag (e.g. `v3.0.0`). Bump them together when upgrading; they share a single git tag and CI release flow.

```yaml
# advanced compose pinned to a specific release
services:
  dras-renderer:
    image: ghcr.io/jacaudi/dras-renderer:v3.0.0
  dras:
    image: ghcr.io/jacaudi/dras:v3.0.0
```

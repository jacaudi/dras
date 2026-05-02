# dras Helm chart

Helm chart for [dras](https://github.com/jacaudi/dras) — NEXRAD radar monitoring with optional renderer service.

Published as an OCI artifact at `oci://ghcr.io/jacaudi/charts/dras`. Built on top of [`bjw-s/app-template`](https://github.com/bjw-s-labs/helm-charts/tree/main/charts/other/app-template) v4.6.2.

## Modes

The chart deploys dras in one of two modes via the top-level `mode` value:

| `mode` | Resources | When to use |
|---|---|---|
| `standard` | `Deployment/dras` | dras only — uses the legacy NWS radar image fetcher. |
| `advanced` *(default)* | `Deployment/dras`, `Deployment/dras-renderer`, `Service/dras-renderer` | dras + renderer for live Level II rendering. `RENDERER_URL` is wired automatically. |

## Prerequisites

- Helm 3.16 or newer (OCI dependency support).
- A Pushover account with API token + user key.
- A Kubernetes Secret containing `PUSHOVER_API_TOKEN` and `PUSHOVER_USER_KEY`. **The chart does not create this Secret for you.**

## Installing

### 1. Create the Pushover Secret

```bash
kubectl create secret generic dras-pushover \
  --namespace monitoring \
  --from-literal=PUSHOVER_API_TOKEN='your-token' \
  --from-literal=PUSHOVER_USER_KEY='your-user-key'
```

For production, source these values from External Secrets Operator, SOPS, or your secret manager of choice.

### 2. Install the chart

Advanced mode (default):

```bash
helm install dras oci://ghcr.io/jacaudi/charts/dras --version vX.Y.Z \
  --namespace monitoring --create-namespace \
  --set dras.stationIds=KATX,KRAX \
  --set dras.pushover.existingSecret=dras-pushover
```

Standard mode (dras only):

```bash
helm install dras oci://ghcr.io/jacaudi/charts/dras --version vX.Y.Z \
  --namespace monitoring --create-namespace \
  --set mode=standard \
  --set dras.stationIds=KATX,KRAX \
  --set dras.pushover.existingSecret=dras-pushover
```

## Required values

| Key | Description |
|---|---|
| `mode` | `standard` or `advanced`. Defaults to `advanced`. |
| `dras.stationIds` | Comma-separated NEXRAD station IDs (e.g. `KATX,KRAX`). 4 uppercase letters per station. |
| `dras.pushover.existingSecret` | Name of an existing Secret containing `PUSHOVER_API_TOKEN` and `PUSHOVER_USER_KEY`. |
| `image.dras.repository` | dras image repository. Default `ghcr.io/jacaudi/dras`. |
| `image.renderer.repository` | renderer image repository. Default `ghcr.io/jacaudi/dras-renderer`. Required only in advanced mode. |

Image `tag` defaults to `.Chart.AppVersion` when empty (the chart's app version is lockstepped to the unified release tag).

## Common overrides

```yaml
mode: advanced

image:
  dras:
    tag: v2.7.0    # pin instead of moving with chart upgrades

dras:
  stationIds: KATX,KRAX,KMUX
  interval: "10"
  pushover:
    existingSecret: dras-pushover

renderer:
  s3:
    bucket: unidata-nexrad-level2-chunks
    region: us-east-1

# Pass-through to bjw-s/app-template — full power for tuning.
controllers:
  dras:
    replicas: 1
    containers:
      app:
        resources:
          requests: { cpu: 10m, memory: 32Mi }
          limits:   { memory: 64Mi }
  renderer:
    replicas: 1                # >1 not currently useful (per-process LRU + lock)
    containers:
      app:
        resources:
          requests: { cpu: 100m, memory: 384Mi }
          limits:   { memory: 768Mi }
```

## Notes

- The renderer is **internal-only** — no Ingress, no Gateway route. dras reaches it via the cluster-internal `RENDERER_URL`.
- The renderer reads from the public `unidata-nexrad-level2-chunks` S3 bucket using anonymous (UNSIGNED) requests. The renderer's per-process volume-pointer cache means `replicas: 1` is the recommended setting; multiple replicas don't compound cache benefits.
- Renderer egress: `*.s3.amazonaws.com:443`. If you enable a `NetworkPolicy` (via the `networkpolicies:` pass-through to app-template), allow this explicitly.
- The dras container has no HTTP health endpoint as of dras v2.6.0; probes are disabled by default. This may change in a future dras release.

## Versioning

The chart's `version` and `appVersion` are kept in lockstep with the unified dras release tag. A chart version `2.7.0` deploys `ghcr.io/jacaudi/dras:v2.7.0` and `ghcr.io/jacaudi/dras-renderer:v2.7.0` by default.

## Source

- Repo: <https://github.com/jacaudi/dras>
- Issues: <https://github.com/jacaudi/dras/issues>

[![On Merge](https://github.com/jacaudi/dras/actions/workflows/on-merge.yml/badge.svg)](https://github.com/jacaudi/dras/actions/workflows/on-merge.yml) [![Versioned Release](https://github.com/jacaudi/dras/actions/workflows/on-release.yml/badge.svg)](https://github.com/jacaudi/dras/actions/workflows/on-release.yml)

# DRAS — Doppler Radar Notification Service

 This programs monitors either a single, or multiple, WSR-88D sites and sends alerts via Pushover based on change in status.

## What Is Monitored

 The following attributes are monitored per each site

- Volume Coverage Pattern (VCP) — Clear Air or Precipitation Mode
- Operational Status
- Power Source
- Generator State

Each poll downloads the latest radar image and keeps a sliding window of the last
hour of images per station (configurable via `RADAR_IMAGE_RETENTION`). When a VCP
change is detected, the freshly downloaded image is attached to the Pushover
notification. All radar image requests use the same User-Agent as the NWS API
calls.

## How To Use

### Requirements

- Pushover Account
- A device with the pushover mobile application installed

### Binary Method

 1. Assuming you have Go installed on your system, head over to the Github [Releases](https://github.com/jacaudi/dras/releases) and grab the URL to the latest release.
 2. Run go install URL (e.g. `go install github.com/jacaudi/dras@v1.0.0`)
 3. Be sure to set the following Environmental Variables
    - `STATION_IDS` — WSR-88D (Radar) Sites (e.g. KRAX - Raleigh/Durham)
    - `PUSHOVER_USER_KEY` — Your Pushover User Key
    - `PUSHOVER_API_TOKEN` — Your Pushover API Token
    - `ALERT_VCP` — Enable Alerts on changes to Volume Coverage Pattern (default: `true`)
    - `ALERT_STATUS` — Enable Alerts on changes to radar operational status (default: `false`)
    - `ALERT_OPERABILITY` — Enable Alerts on changes to radar operability status (default: `false`)
    - `ALERT_POWER_SOURCE` — Enable Alerts on changes to radar power source (default: `false`)
    - `ALERT_GEN_STATE` — Enable Alerts on changes to generator state (default: `false`)
    - `RADAR_IMAGE_ENABLED` — Poll the radar image every check and attach it to VCP-change notifications (default: `true`)
    - `RADAR_IMAGE_URL_TEMPLATE` — Override the radar image URL. Use `{station}` as the station-ID placeholder (default: NWS Ridge GIF — the highest-resolution per-station single image NWS publishes via static URL)
    - `RADAR_IMAGE_RETENTION` — Sliding window of polled images kept per station, parsed as a Go duration like `1h` or `30m` (default: `1h`)
 4. Enjoy!

### Standalone Container Method

```bash
docker pull ghcr.io/jacaudi/dras:v1.0.0

docker run -d \
  -e STATION_IDS=KRAX \
  -e PUSHOVER_USER_KEY=<KEY> \
  -e PUSHOVER_API_TOKEN=<TOKEN> \
  -e ALERT_VCP=false \
  -e ALERT_STATUS=true \
  -e ALERT_OPERABILITY=true \
  ghcr.io/jacaudi/dras:v1.0.0
```

### Kubernetes Method

 See the [kubernetes](examples/kubernetes.yaml) file in [examples](examples) folder — It contains an example deployment, configmap, and secret.

## Versioning

This project uses [Semantic Versioning](https://semver.org/) with automated releases via [uplift](https://uplift.dev/).

Commits should follow [Conventional Commits](https://www.conventionalcommits.org/):
- `feat:` - Minor version bump (new features)
- `fix:` - Patch version bump (bug fixes)
- `feat!:` or `BREAKING CHANGE:` - Major version bump

Releases are automatically created when changes are merged to main.

## How To Contribute

This project welcomes any feature improvements or bugs found via PRs. Thank you!

## License

[MIT](LICENSE)

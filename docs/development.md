# Development

Each component has its own dev guide:

- [`dras/README.md`](../dras/README.md) — Go service: `go test`, `go run`, container build.
- [`renderer/README.md`](../renderer/README.md) — Python service: `uv sync`, `pytest`, ruff/mypy, container build.

## Running both together

Two terminals — renderer first, then dras pointed at it:

```bash
# Terminal 1
cd renderer
uv run dras-renderer
```

```bash
# Terminal 2
cd dras
PUSHOVER_API_TOKEN=x PUSHOVER_USER_KEY=x STATION_IDS=KATX DRYRUN=true \
  RENDERER_URL=http://127.0.0.1:8080 \
  go run .
```

Watch dras's logs for `Radar image source enabled [mode=advanced, ...]`. With `DRYRUN=true`, no Pushover messages are sent; dras runs against the test stations `KATX`/`KRAX`.

## Testing the full pipeline

Renderer tests use a real ~8 MB KATX volume fixture (`renderer/tests/fixtures/KATX_test.ar2v.gz`) checked in for deterministic decoder/renderer behavior. The fixture is regenerated only when the assertions need bumping.

dras tests include `TestRendererModeDeliversAttachment` (`dras/internal/monitor/monitor_test.go`) which exercises the full renderer-mode path with a stubbed renderer HTTP server.

## CI

- `.github/workflows/pr.yml` runs Go lint+test, renderer ruff+mypy+pytest, and Docker build validation for both images on every PR.
- `.github/workflows/ci-cd.yml` releases on merge to `main` and builds+pushes both container images at the release tag.

## Conventional commits

Required. The release tooling depends on commit prefixes for semver bumps:

- `feat:` — minor bump
- `fix:` — patch bump
- `feat!:` or `BREAKING CHANGE:` — major bump

Mixing `feat:` for bug fixes will incorrectly bump minor; mixing `fix:` for features will skip a deserved minor bump.

## Plans

Design and implementation plans live in `docs/plans/`. That directory is gitignored — plans are local-only for the working engineer; share via PR description rather than checking in.

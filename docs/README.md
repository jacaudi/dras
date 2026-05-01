# DRAS Documentation

In-depth documentation for the **D**oppler **R**adar **A**lerting **S**ervice. The top-level [README](../README.md) is a quickstart; everything below covers the system in detail.

## Index

| Doc | Covers |
|---|---|
| [Configuration](./configuration.md) | Every environment variable, defaults, and which mode it applies to. |
| [Architecture](./architecture.md) | Service split (Go orchestrator + Python renderer), data flow, S3 source, caching, error contract. |
| [Deployment](./deployment.md) | Docker Compose and Kubernetes recipes for both basic and advanced modes; networking and resource sizing notes. |
| [Development](./development.md) | Working on either component locally; links to component READMEs. |
| [`plans/`](./plans/) | (Local-only, gitignored) Design and implementation plans for major features. |

## Component documentation

- [`dras/README.md`](../dras/README.md) — Go service: code layout, local-dev recipes, container build.
- [`renderer/README.md`](../renderer/README.md) — Python renderer: API reference, metrics, tests, container build.

## Conventions

- Conventional Commits enforced (semver bumps depend on it). `fix:` for bug fixes, `feat:` for new features. See [Versioning](#versioning) below.
- Both container images ship under one unified version tag.
- Modes (basic / advanced) are mutually exclusive and selected at startup via `RENDERER_URL`.

## Versioning

Semantic versioning, automated releases via [uplift](https://uplift.dev/). Commit-prefix → semver bump:

| Prefix | Bump |
|---|---|
| `feat:` | minor |
| `fix:` | patch |
| `feat!:` or `BREAKING CHANGE:` | major |

Releases are cut on merge to `main`. Both `ghcr.io/jacaudi/dras` and `ghcr.io/jacaudi/dras-renderer` images publish at the same tag.

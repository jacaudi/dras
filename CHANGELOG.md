# Changelog

## [2.8.2](https://github.com/jacaudi/dras/compare/v2.8.1...v2.8.2) (2026-05-03)

### Bug Fixes

* **renderer:** raise clutter floor to 15 dBZ; tighter default zoom; more towns ([36a53e7](https://github.com/jacaudi/dras/commit/36a53e79f3e57e388f977faa48fe0f55c5ec76bb))

## [2.8.1](https://github.com/jacaudi/dras/compare/v2.8.0...v2.8.1) (2026-05-03)

### Bug Fixes

* **chart:** startupProbe gates dras on renderer /healthz; rename back to 'renderer' ([b2680cd](https://github.com/jacaudi/dras/commit/b2680cd73ddbb470d62de94ba79aa75b233fbbea))

## [2.8.0](https://github.com/jacaudi/dras/compare/v2.7.4...v2.8.0) (2026-05-03)

### Features

* **renderer,chart:** clutter+cities+zoom + single-pod sidecar layout ([#109](https://github.com/jacaudi/dras/issues/109)) ([e68b698](https://github.com/jacaudi/dras/commit/e68b6989b40a5448419739543b02dbbfa4b0ef39))

## [2.7.4](https://github.com/jacaudi/dras/compare/v2.7.3...v2.7.4) (2026-05-03)

### Bug Fixes

* **renderer:** suppress /healthz access log unless at DEBUG ([591e5e9](https://github.com/jacaudi/dras/commit/591e5e9d1916fac0c302de6d5703c301799cdbf9))

## [2.7.3](https://github.com/jacaudi/dras/compare/v2.7.2...v2.7.3) (2026-05-03)

### Bug Fixes

* **renderer:** make image uid-agnostic — closes [#107](https://github.com/jacaudi/dras/issues/107) ([#108](https://github.com/jacaudi/dras/issues/108)) ([4a9e359](https://github.com/jacaudi/dras/commit/4a9e359b160b048627e31fc94c6f34387b3fe011)), closes [#101](https://github.com/jacaudi/dras/issues/101)

## [2.7.2](https://github.com/jacaudi/dras/compare/v2.7.1...v2.7.2) (2026-05-03)

### Bug Fixes

* **chart,dras:** renderer crash loop and upstream-fetch resilience ([3b256f0](https://github.com/jacaudi/dras/commit/3b256f0f47b403751f549daf308a084a8543fc20)), closes [#103](https://github.com/jacaudi/dras/issues/103) [#101](https://github.com/jacaudi/dras/issues/101) [#103](https://github.com/jacaudi/dras/issues/103) [#101](https://github.com/jacaudi/dras/issues/101) [#103](https://github.com/jacaudi/dras/issues/103) [#101](https://github.com/jacaudi/dras/issues/101)
* **renderer:** map no_recent_scan to 404 instead of 503 ([#106](https://github.com/jacaudi/dras/issues/106)) ([f0cdf2f](https://github.com/jacaudi/dras/commit/f0cdf2f19315169a8ad827dc3493949811c81c86)), closes [#105](https://github.com/jacaudi/dras/issues/105) [#104](https://github.com/jacaudi/dras/issues/104)

## [2.7.1](https://github.com/jacaudi/dras/compare/v2.7.0...v2.7.1) (2026-05-02)

### Bug Fixes

* **docs:** update README badges to current CI workflows ([7e72b1a](https://github.com/jacaudi/dras/commit/7e72b1a7ccc5292932e7c2c3f91d61853c507d85))

## [2.7.0](https://github.com/jacaudi/dras/compare/v2.6.0...v2.7.0) (2026-05-02)

### Bug Fixes

* **chart:** restore stationIds pattern; pass un-prefixed version to helm-publish ([3dd5e21](https://github.com/jacaudi/dras/commit/3dd5e21dc18c242534acd1a38be846f56fca7be8))
* **ci:** pin helm-unittest plugin to v1.0.3 (0.6.5 never existed) ([af10f13](https://github.com/jacaudi/dras/commit/af10f13c99de8d71817c3ac4fef35e767bacce61))
* **dras:** plumb context.Context through image.Source.Fetch ([d558c2f](https://github.com/jacaudi/dras/commit/d558c2fd2beea064cc9ce8428fc519271bc2a933)), closes [#85](https://github.com/jacaudi/dras/issues/85)
* **renderer:** assert anonymous S3 requests are unsigned via header inspection ([efbb10d](https://github.com/jacaudi/dras/commit/efbb10d437963da92023701cb57f5a1fb104e519)), closes [#85](https://github.com/jacaudi/dras/issues/85)
* **renderer:** attribute S3 errors at service layer; type scan_time as datetime ([be37b90](https://github.com/jacaudi/dras/commit/be37b9083c2aee0911d41ecc2551efaef221d6e8)), closes [#85](https://github.com/jacaudi/dras/issues/85)
* **renderer:** compare slot freshness on YYYYMMDD-HHMMSS prefix only ([fef13b2](https://github.com/jacaudi/dras/commit/fef13b20650cc287f730314fc0536db5d89305c6)), closes [#85](https://github.com/jacaudi/dras/issues/85)
* **renderer:** document deviation from chunks/ fixture design ([ce5c943](https://github.com/jacaudi/dras/commit/ce5c943ca455212f97b70714773db8f2948432ba)), closes [#85](https://github.com/jacaudi/dras/issues/85)
* **renderer:** document Dockerfile uv sync layering and verify cartopy cache at build ([f6dd64f](https://github.com/jacaudi/dras/commit/f6dd64fb7808113c7058191389e19bb8484cc7e2)), closes [#85](https://github.com/jacaudi/dras/issues/85)
* **renderer:** drop redundant MPLBACKEND env var and document lowest-tilt sweep ([7958e8c](https://github.com/jacaudi/dras/commit/7958e8cb1cd2f043d7e146f7c3d6b6cb5206a000)), closes [#85](https://github.com/jacaudi/dras/issues/85)
* **renderer:** parallelize chunk downloads in download_volume ([78e9a12](https://github.com/jacaudi/dras/commit/78e9a129474f7a1ac9be0ebfe1ae72a60575a200)), closes [#85](https://github.com/jacaudi/dras/issues/85)


### Features

* **chart:** add values.schema.json and template fail clauses ([4a0750e](https://github.com/jacaudi/dras/commit/4a0750ec93741d14aca083c6fcc3d37c2f087696))
* **chart:** scaffold helm chart with standard-mode dras-only render ([28e6393](https://github.com/jacaudi/dras/commit/28e639323943ff6b291bba925cea378b6ac5ffce))
* **chart:** wire advanced mode — renderer service, RENDERER_URL, and S3 envs ([4f8d47f](https://github.com/jacaudi/dras/commit/4f8d47f89badeb65e9e0fd866f38960761354392))
* **chart:** wire top-level image block as operator override surface ([78044de](https://github.com/jacaudi/dras/commit/78044de1d79dae24a06941e881c54a2e479e6b9b))
* **release:** sync chart version with release tag via semantic-release ([86d2bdb](https://github.com/jacaudi/dras/commit/86d2bdb46d63b51ce79420ef6c2f633e1b511951))

## [2.6.0](https://github.com/jacaudi/dras/compare/v2.5.0...v2.6.0) (2026-05-01)

### Bug Fixes

* address comprehensive review findings ([151568c](https://github.com/jacaudi/dras/commit/151568cc05d7dfa96dfdeba992318b5ce5f5fc9e))
* **ci:** set Go working-directory to dras/, use libgdal32 in renderer image ([381af7f](https://github.com/jacaudi/dras/commit/381af7fb5a42f0e90cb4089a92994509a6bd5329))
* **docker:** move .dockerignore into dras/ build context ([888f7e2](https://github.com/jacaudi/dras/commit/888f7e2fc4317c6e705a0582df7f0b9e4eac9db1))
* **renderer:** bound metadata cache, lock renders, dispatch off event loop ([0f3d72f](https://github.com/jacaudi/dras/commit/0f3d72fcee7011a4fe23b2d77a361ef974b53510))
* **renderer:** correct longitude stretch and drop redundant projection arg ([9bc3716](https://github.com/jacaudi/dras/commit/9bc3716c9b9cb87258691d6c5526d4ee03a63fa8))
* **renderer:** filter stale chunks on slot reuse, bound latest-volume cache ([1c8a557](https://github.com/jacaudi/dras/commit/1c8a557db9b2541d746bdc7f76683209a1bfb8d9))
* **renderer:** split dependency sync from project install in Dockerfile ([4bcbe2f](https://github.com/jacaudi/dras/commit/4bcbe2f278740b0e62bc5c1b08d4213f8a82c55e))
* **renderer:** stop bunzipping in download_volume; chunks are AR2V-framed ([1fa52b5](https://github.com/jacaudi/dras/commit/1fa52b51d5b8e4bb9b3cd40382039ad920f0b04e))
* **renderer:** tighten decode error wrapping and metadata extraction ([32cf5bd](https://github.com/jacaudi/dras/commit/32cf5bd623bc4616964be94632e11655b048ecd8))


### Features

* **dras:** add internal/renderer HTTP client ([c9fd7d8](https://github.com/jacaudi/dras/commit/c9fd7d811bb3159f7aadd405409e5f09eca4cbc9))
* **dras:** wire renderer mode selection via RENDERER_URL ([0a67a39](https://github.com/jacaudi/dras/commit/0a67a391fe9bb65216616a6a8fe3eb36299cc9b7))
* **renderer:** add chunks-bucket S3 client with fan-out latest-volume ([6d22b6e](https://github.com/jacaudi/dras/commit/6d22b6e33da9290d862f892eb1ac47744aa38ec5))
* **renderer:** add FastAPI skeleton with /healthz endpoint ([32b2f80](https://github.com/jacaudi/dras/commit/32b2f802b18d730c61c0d0922a941fd28d1589b0))
* **renderer:** add LRU cache for rendered PNG bytes ([ce71472](https://github.com/jacaudi/dras/commit/ce7147222baa7f3f71490964aade5de12b293b13))
* **renderer:** add multi-stage Dockerfile ([36551c2](https://github.com/jacaudi/dras/commit/36551c263314d43d3e66b37ecfe62e3cb60b4ccb))
* **renderer:** add Py-ART decoder + assembled Level II fixture ([3c78e1e](https://github.com/jacaudi/dras/commit/3c78e1e9b8821c312dea356f4addaf6f03bafbf9))
* **renderer:** expose Prometheus metrics on /metrics ([d87444d](https://github.com/jacaudi/dras/commit/d87444d870589700da91d937f949475ad6fa5448))
* **renderer:** render base reflectivity PPI to PNG with Cartopy basemap ([423f958](https://github.com/jacaudi/dras/commit/423f9580da320aac08ec95a891077f08e4802d2f))
* **renderer:** wire /render/{station} with cache + S3 + decode + render ([2db239b](https://github.com/jacaudi/dras/commit/2db239b03641d90960f9c4b8fa9e6f25de1294b4))

## [2.5.0](https://github.com/jacaudi/dras/compare/v2.4.0...v2.5.0) (2026-04-26)

### Features

* **notify:** attach radar image to startup notification ([7293646](https://github.com/jacaudi/dras/commit/7293646c0e9d4a7abf8ae8b6dcb8794a4a7eef6f))

## [2.4.0](https://github.com/jacaudi/dras/compare/v2.3.3...v2.4.0) (2026-04-26)

### Features

* **log:** show configured stations and rendered URLs in polling-enabled log ([845c693](https://github.com/jacaudi/dras/commit/845c6932fc9dc5c67304a73926bbdb01a41009ca))

## [2.3.3](https://github.com/jacaudi/dras/compare/v2.3.2...v2.3.3) (2026-04-26)

### Bug Fixes

* **deps:** bump github.com/jacaudi/nws to v0.1.0 ([95611b9](https://github.com/jacaudi/dras/commit/95611b95442b0483eb8613a3948acbc318ae3614))

## [2.3.2](https://github.com/jacaudi/dras/compare/v2.3.1...v2.3.2) (2026-04-26)

### Bug Fixes

* **ci:** publish v<major>, v<major>.<minor>, latest container tags ([a1ba477](https://github.com/jacaudi/dras/commit/a1ba477953bbf387c7c93dcfb280bdb1dce8f2d1))

## [2.3.1](https://github.com/jacaudi/dras/compare/v2.3.0...v2.3.1) (2026-04-26)

### Bug Fixes

* **ci:** chain container build after release to bypass skip-ci block ([21c7c79](https://github.com/jacaudi/dras/commit/21c7c79cbdb066cfc0a66411e204c7e225492daa))

## [2.3.0](https://github.com/jacaudi/dras/compare/v2.2.4...v2.3.0) (2026-04-25)

### Features

* **image:** retain hourly history, send User-Agent on image fetches ([b818292](https://github.com/jacaudi/dras/commit/b818292eda28cbe67871131079dcb5d92b2d911a))
* poll radar image and attach to VCP-change notifications ([b805392](https://github.com/jacaudi/dras/commit/b805392d32651a575cb82c62da8740eb443cd676))

## [2.2.4](https://github.com/jacaudi/dras/compare/v2.2.3...v2.2.4) (2026-04-07)

### Bug Fixes

* **ci:** remove invalid template variable from github assets config ([687d770](https://github.com/jacaudi/dras/commit/687d7701182e3932093125beb3cc5cf8f597399f))

## [2.2.3](https://github.com/jacaudi/dras/compare/v2.2.2...v2.2.3) (2026-04-07)

### Bug Fixes

* **ci:** prevent duplicate GitHub Release from goreleaser ([4b2f7ac](https://github.com/jacaudi/dras/commit/4b2f7acf0deec8ec61c1966d9878211315e97a6c))

## [2.2.2](https://github.com/jacaudi/dras/compare/v2.2.1...v2.2.2) (2026-04-07)

### Bug Fixes

* **ci:** skip goreleaser dirty check during semantic-release ([83d6c7a](https://github.com/jacaudi/dras/commit/83d6c7a5f1f4272659455b5b15d140607794924f))

## [2.2.1](https://github.com/jacaudi/dras/compare/v2.2.0...v2.2.1) (2026-04-06)

### Bug Fixes

* **ci:** preserve v prefix in container image tags ([6a47d28](https://github.com/jacaudi/dras/commit/6a47d287208b35301a3f0a655327669595755513))

## [2.2.0](https://github.com/jacaudi/dras/compare/v2.1.1...v2.2.0) (2026-04-06)

* feat!: migrate from go-semantic-release to JS semantic-release ([1d4b6b7](https://github.com/jacaudi/dras/commit/1d4b6b7fea8bb208988f84d8e0023331fbcdef4b))


### BREAKING CHANGES

* Release tooling switched from go-semantic-release to JS
semantic-release. The .semrelrc config is replaced by .releaserc.json.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

## [v2.1.0](https://github.com/jacaudi/dras/releases/tag/v2.1.0) - 2026-01-16

- [`b184b58`](https://github.com/jacaudi/dras/commit/b184b5817dffda2a83887895ac02975ffe93f2db) fix: specify secrets for GitHub App in on-merge workflow
- [`1133e07`](https://github.com/jacaudi/dras/commit/1133e07f9ffdabd7d38c42baf027abd7603a0a8f) fix: set use-github-app to true in on-merge workflow
- [`85c0a60`](https://github.com/jacaudi/dras/commit/85c0a6075023570b2f40cd6a612208b581b380ba) fix: add conditional use of GitHub App in uplift workflow
- [`53f08d4`](https://github.com/jacaudi/dras/commit/53f08d40237bb9784d64fedc9e66de791dc5fa8a) feat: add uplift auto-tagging after successful docker builds
- [`33bece9`](https://github.com/jacaudi/dras/commit/33bece94e39fa1550276961db8e51b3132b4c262) Merge branch 'auto-claude/002-onboard-central-github-actions'
- [`8125a03`](https://github.com/jacaudi/dras/commit/8125a03f5d2226b0fa3567a25dc953ee3ac3d434) Merge main: incorporate lint fixes for test files
- [`bac39c4`](https://github.com/jacaudi/dras/commit/bac39c474d7a2e70dc60662b51f72a140c99a460) fix(lint): resolve golangci-lint errors in test files
- [`2e783ca`](https://github.com/jacaudi/dras/commit/2e783cae9d2e24bb541f8b121600e1deb0909d54) auto-claude: subtask-1-2 - Fix nil pointer dereferences in mock test files
- [`f1287a9`](https://github.com/jacaudi/dras/commit/f1287a9366331ff53a8e790d551eb84a8e3a40ac) auto-claude: subtask-1-1 - Remove unused fields from benchmark test struct
- [`5ce3cff`](https://github.com/jacaudi/dras/commit/5ce3cff1a55b49dc1602f604ded6186eb6f15efb) auto-claude: subtask-3-1 - Update on-release.yml to use central workflows
- [`ade44b1`](https://github.com/jacaudi/dras/commit/ade44b12dc5ea27b10756f0acc098a5828b40dbb) auto-claude: subtask-2-1 - Update on-merge.yml to use central lint, test, and docker-build workflows
- [`b1e6a7b`](https://github.com/jacaudi/dras/commit/b1e6a7b1d3f4146166adceabf8b6cd53569aa158) auto-claude: subtask-1-2 - Update on-branch-push.yml to use central lint and test workflows
- [`a4f911f`](https://github.com/jacaudi/dras/commit/a4f911f855f2751ea1e0b230241657af82315a5d) auto-claude: subtask-1-1 - Update on-pr.yml to use central lint and test workflows
- [`9aca5b5`](https://github.com/jacaudi/dras/commit/9aca5b5151d5410b3d3b68110a6722009d467726) auto-claude: Merge auto-claude/001-renovate-dashboard
- [`2fd3665`](https://github.com/jacaudi/dras/commit/2fd3665e8b825e606230756a0edd33aec00feafc) auto-claude: subtask-3-2 - Run all unit tests to verify no regressions
- [`9640624`](https://github.com/jacaudi/dras/commit/9640624c201b724b8024a8ab1f433b6b8f7a00d0) auto-claude: subtask-1-1 - Update notify dependency to v1.5.0
- [`21f0527`](https://github.com/jacaudi/dras/commit/21f05273fd830d7a0b42206fa06e5e49805c8602) fix: remove renovate workflow -- now managing with renovate operator
- [`ce6052e`](https://github.com/jacaudi/dras/commit/ce6052e54f3a787e78b7ca859b20199734870b72) feat: enhance versioning in Dockerfile and version package
- [`4b5e5b5`](https://github.com/jacaudi/dras/commit/4b5e5b5796254a4e11928a7c9a1c6e11e0f5e3e7) fix: update test to new message check
- [`5a5dd6e`](https://github.com/jacaudi/dras/commit/5a5dd6ef75158f5abebe8339f87efb52fd255de4) feat: change alert messages that is sent

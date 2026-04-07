# Changelog

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

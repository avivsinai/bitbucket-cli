# Changelog

All notable changes to this project will be documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- Comprehensive Data Center coverage: reviewer groups, auto-merge management,
  diff statistics, PR tasks and suggestions, comment reactions, branch
  permissions, secrets rotation, logging controls.
- Bitbucket Cloud support: authentication, Pipelines run/list/view/log, webhook
  management, `status pipeline`, shared rate-limit telemetry.
- Shared infrastructure upgrades: retrying HTTP client with caching, jq and
  Go-template output, pager integration, interactive prompts, browser helpers.
- Observability: `bkt status rate-limit`, adaptive throttling, HTTP trace mode.
- OSS readiness: Code of Conduct, contributing guide, governance, security
  policy, issue/PR templates, CI workflows, SBOM build, GoReleaser config.

### Changed
- `bkt pr diff` now supports `--stat` and streams via the pager when available.
- `bkt webhook` commands support both Data Center and Cloud instances.

### Fixed
- Improved error handling for context resolution and merge workflows.

## [0.1.0] - 2025-10-26
- Initial public release of `bkt`.

[Unreleased]: https://github.com/avivsinai/bitbucket-cli/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/avivsinai/bitbucket-cli/releases/tag/v0.1.0

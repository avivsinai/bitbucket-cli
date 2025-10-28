# Changelog

All notable changes to this project will be documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.2.0] - 2025-10-28

### Added
- Comprehensive Data Center coverage: reviewer groups, auto-merge management,
  diff statistics, PR tasks and suggestions, comment reactions, branch
  permissions, secrets rotation, logging controls.
- Bitbucket Cloud support: authentication, repository/branch/pull-request
  flows, Pipelines run/list/view/log, webhook management, `status pipeline`,
  shared rate-limit telemetry.
- Raw `bkt api` escape hatch with method/field/header/param support for
  experimentation and automation.
- Extension lifecycle commands (`bkt extension install|list|remove|exec`) with
  automatic cloning into the CLI config directory.
- Shared infrastructure upgrades: retrying HTTP client with caching, jq and
  Go-template output, pager integration, interactive prompts, browser helpers.
- Observability: `bkt status rate-limit`, adaptive throttling, HTTP trace mode.
- OSS readiness: Code of Conduct, contributing guide, governance, security
  policy, issue/PR templates, CI workflows, SBOM build, GoReleaser config.
- Project list command for pre-context discovery.
- Git remote inference for repository defaults.
- Enhanced pagination and retry logic for Cloud API.

### Changed
- `bkt pr diff` now supports `--stat` and streams via the pager when available.
- `bkt webhook` commands support both Data Center and Cloud instances.
- Simplified installation instructions to focus on Go install.

### Fixed
- Added timeout protection for git command execution to prevent hanging.
- Fixed CI workflow to use correct branch name (master).
- Corrected Go version references in CI and release workflows.
- Updated GoReleaser configuration to use modern syntax.
- Added clarifying comments for intentionally ignored errors.
- Improved error handling for context resolution and merge workflows.

## [0.1.0] - 2025-10-26
- Initial public release of `bkt`.

[Unreleased]: https://github.com/avivsinai/bitbucket-cli/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/avivsinai/bitbucket-cli/releases/tag/v0.1.0

# Changelog

All notable changes to this project will be documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Changed
- Clarified Bitbucket Cloud context creation in README, showing that `--host api.bitbucket.org` is required and adding a tip to use `bkt auth status` to discover the correct host value.

## [0.4.1] - 2026-01-17

### Fixed
- Improved error messages for CAPTCHA-locked accounts. When a Bitbucket account
  is locked due to failed authentication attempts, the CLI now displays the
  actual CAPTCHA message instead of a generic "XSRF check failed" error (#16).
- Fixed SSH URL auto-detection for `ssh://host:port/PROJECT/repo.git` format.
  Previously, commands would default to a configured project instead of parsing
  the project from the git remote URL (#17).

### Changed
- **Breaking**: Git remote now takes precedence over context config for
  project/repo detection. If you are in a git repository that matches your
  configured host, the CLI will use the project and repo from the git remote
  URL, overriding any values set in your context config. Use explicit
  `--project` and `--repo` flags to override this behavior.

## [0.4.0] - 2026-01-17

### Added
- New `bkt issue` command group for Bitbucket Cloud issue tracker (Cloud-only).
  - `bkt issue list`: List issues with filtering by state, kind, priority, assignee, milestone.
  - `bkt issue view`: Display issue details with optional comments.
  - `bkt issue create`: Create new issues with title, body, kind, priority, assignee, etc.
  - `bkt issue edit`: Update existing issue fields.
  - `bkt issue close`: Close an issue.
  - `bkt issue reopen`: Reopen a closed issue.
  - `bkt issue delete`: Delete an issue with confirmation prompt.
  - `bkt issue comment`: Add or list comments on an issue.
  - `bkt issue status`: Show issues assigned to or created by the current user.
  - All commands support `--json` and `--yaml` output formats.
- New `bkt pr checks` command to display build/CI status for pull requests.
  - Supports both Bitbucket Data Center and Cloud APIs.
  - Color-coded output: green for success, red for failure, yellow for in-progress.
  - `--wait` flag polls until all builds complete (useful for CI automation).
  - `--timeout` flag sets maximum wait time (default: 30 minutes).
  - `--interval` flag configures initial polling frequency (default: 10 seconds).
  - `--max-interval` flag sets backoff cap (default: 2 minutes).
  - Exponential backoff (1.5x multiplier) to reduce API load during long builds.
  - Random jitter (±15%) prevents thundering herd when multiple clients poll.
  - Graceful handling of Ctrl-C interruption during polling.
  - Automatic retry with backoff on transient errors (up to 3 attempts).
  - Returns non-zero exit code when builds fail (for scripting).
- Shared `CommitStatus` type in `pkg/types` for consistency between API clients.

## [0.2.1] - 2025-11-09

### Security
- Tokens are now persisted in the host OS keychain (Keychain/WinCred/Secret
  Service) instead of `config.yml`, with an opt-in encrypted file fallback
  gated behind `--allow-insecure-store` for legacy hosts.

### Fixed
- Removed plaintext credential writes and aligned CLI output with lint
  expectations (errcheck), keeping tests and release automation green.

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

[Unreleased]: https://github.com/avivsinai/bitbucket-cli/compare/v0.2.1...HEAD
[0.2.1]: https://github.com/avivsinai/bitbucket-cli/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/avivsinai/bitbucket-cli/releases/tag/v0.1.0

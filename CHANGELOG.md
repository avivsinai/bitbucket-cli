# Changelog

All notable changes to this project will be documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.26.2] - 2026-04-24
### Added
- `bkt auth doctor [host]` diagnoses macOS Keychain re-prompt issues without reading the stored secret. Reports the binary's signature, Designated Requirement, and whether a Keychain entry is present for the host (#181).

### Fixed
- macOS Keychain "Always Allow" now survives `brew upgrade bkt`. The release codesign script pins the Designated Requirement to the bundle identifier (`-r='designated => identifier "…"'`) instead of letting it default to the cdhash, which changes on every build. The previous claim in 0.21.0 that the stable identifier alone preserved approvals was incomplete — without an explicit DR, `codesign` still derived one from the cdhash. (#181)
- Token saves on macOS now delete any stale Keychain item before writing, so the new entry is created with the current binary's DR instead of inheriting an ACL from a previous bkt build. Re-run `bkt auth login` once after upgrading to benefit. (#181)
- macOS Keychain items created by bkt now set `KeychainTrustApplication` on darwin, so fresh items trust the calling binary at create time instead of prompting on every access. `KeychainAccessibleWhenUnlocked` is set alongside for forward compatibility (currently a no-op on the file-based Keychain path 99designs/keyring uses). (#181)


## [0.26.1] - 2026-04-22
### Added
- Release `.tar.gz` and `.zip` archives now bundle `skills/bkt/` alongside the CLI binary and docs.

### Fixed
- Bitbucket Cloud 401 auth failures now explain the most common API-token misconfigurations, including the required Atlassian-account email username and `read:user:bitbucket` scope.


## [0.26.0] - 2026-04-18
### Added
- `bkt pr close` as an alias for `bkt pr decline`, matching `gh`-style command expectations (#160).
- `--destination` as a visible alias for `--target` on `bkt pr create`, mutually exclusive with `--target` (#160).
- Global `--format json|yaml` flag as an alternative to `--json` / `--yaml`; validated before any mutating subcommand runs (#160).


## [0.25.0] - 2026-04-16
### Added
- `bkt pr decline` now accepts `--comment` / `-m`, plus `--text` and `--body` aliases, to send a decline reason on Bitbucket Cloud and Data Center (#161).
- Nix flake support for running and installing `bkt` via `nix run` / `nix profile install`, with CI validation on Linux and macOS (#165).

## [0.24.1] - 2026-04-15
### Fixed
- Release workflow attestation subject collection now avoids Bash-4-only `mapfile`, so macOS runners can finish artifact attestation after GoReleaser publishes a release.


## [0.24.0] - 2026-04-15
### Added
- `bkt pr comments --details` now shows file:line context, resolved/complete status, full comment text, and nested reply indentation for Bitbucket Data Center pull request threads (#110).

### Changed
- Bitbucket Cloud OAuth login now layers PKCE (RFC 7636, S256) onto the existing authorization-code flow while preserving the client-secret token exchange Bitbucket still requires (#162).
- Release automation now attests published artifacts and tightens release-commit verification before tags are created (#164).


## [0.23.0] - 2026-04-12
### Added
- `bkt pr edit --with-default-reviewers` now works on Bitbucket Cloud and Data Center, merging effective default reviewers with explicit reviewer edits while preserving mixed Cloud reviewer identities (#150).

### Changed
- Added pre-commit and CI validation that generated skill rule files and the `SKILL.md` References section stay in sync with the Cobra command tree, with `GO` override support in the verification path (#148).

### Fixed
- `bkt pr update` is now an alias for `bkt pr edit`, matching `gh`-style command expectations and `--help` output (#149).


## [0.22.0] - 2026-04-12
### Added
- `bkt auth login --web` browser-based OAuth 2.0 login for Bitbucket Cloud, with keyring-stored short-lived access tokens and automatic refresh on expiry (#152).
- `BKT_HOST` + `BKT_TOKEN` env-var-driven headless authentication for CI and containers, so commands can run without a prior `bkt auth login` or `bkt context create` (#138).
- Pull request JSON output now includes creation and update timestamps for Cloud and Data Center listings.

### Changed
- `bkt pr list` now appends a local creation timestamp column (`YYYY-MM-DD HH:MM`) to text output for Cloud and Data Center listings.


## [0.21.0] - 2026-04-10
### Added
- OAuth 2.0 infrastructure for Cloud login in the new `pkg/oauth` package, plus a `TokenRefresher` hook on `httpx.Client` for transparent 401-driven token refresh. Phase 1 only; login-command wiring is not included yet (#137).
- Unit test coverage for `bkt branch` and `bkt branch protect` subcommands (#143).
- Unit test coverage for `bkt context` subcommands (#143).
- Unit test coverage for `bkt project list` (#144).
- Unit test coverage for `bkt webhook` subcommands (#145).


## [0.20.0] - 2026-04-09
### Added
- Deprecation warning on all `bkt issue` subcommands for Bitbucket Cloud contexts, ahead of the August 20, 2026 Issues/Wikis sunset (#139).
- Generated per-command skill rule files and rewritten `SKILL.md` via `cmd/docgen` pipeline (#133).
- Unit tests for `bkt pr comments` covering Data Center and Cloud code paths, state filtering, and empty results (#140).


## [0.19.0] - 2026-04-07
### Added
- `bkt auth login --auth-method bearer` for Data Center instances that require bearer token authentication instead of basic auth (#97).
- `bkt pr create --with-default-reviewers` now works on Bitbucket Cloud, fetching effective default reviewers via the API with UUID-based self-exclusion and deduplication (#127).
- `cmd/docgen` skill generator and `make generate-skill` target for auto-generating skill documentation from Cobra command definitions (#131).

### Fixed
- Bearer auth propagation to all inline `bbdc.New()` call sites in `repo list`, `repo view`, `status commit`, and `status pr` (#112).


## [0.18.0] - 2026-04-07
### Added
- `bkt pr edit --reviewer` and `--remove-reviewer` flags to add or remove reviewers on existing pull requests for both Data Center and Cloud (#128).
- `bkt pr create --body` / `-b` as an alias for `--description`, matching `gh` CLI conventions (#126).
- `Long` and `Example` docstrings for all Cobra commands, improving `--help` output and enabling future skill auto-generation (#129).


## [0.17.0] - 2026-04-05
### Added
- `bkt pr publish` and `bkt pr unpublish` commands to toggle draft status on Bitbucket Cloud and Data Center 8.18+ pull requests. Alias: `bkt pr ready` (#122).
- `bkt pr create` now outputs the pull request URL on success (#105).
- `bkt pr create` defaults `--source` to the current branch and `--target` to the repository default branch, making both flags optional (#116).
- Bitbucket Pipelines install section in README with validated `curl | tar` snippet.
- Privacy policy at `docs/PRIVACY.md`.
- `.scratch/` convention for local working documents (gitignored).

### Fixed
- `bkt pr create` now selects the git remote matching the active Bitbucket host instead of defaulting to the first remote (#116).
- Deterministic remote name ordering in fallback when multiple remotes match (#116).
- Cloud diff tests (`TestCommitDiffCloud`, `TestCommitDiffEmptyCloud`) no longer fail on Bitbucket Pipelines where the CWD has a `bitbucket.org` remote (#119).
- Stale credential storage description in `docs/SECURITY.md` updated to reflect OS keychain, `BKT_TOKEN`, and encrypted file backend.
- Consolidated skill publishing into the release workflow so it no longer depends on tag-push events that `GITHUB_TOKEN` cannot trigger.
- Pinned all GitHub Actions to commit SHAs across every workflow for supply-chain safety.
- Added missing `timeout-minutes` and `concurrency` blocks to all workflows.
- Standalone publish-skill workflow now accepts `workflow_dispatch` with an explicit `tag` input.
- Bitbucket Pipelines Go image updated from 1.24 to 1.25 to match `go.mod`.

## [0.16.4] - 2026-04-02
### Fixed
- Passed the temp release-notes path directly to GoReleaser so GitHub Actions preserves the `--release-notes` argument during publishing.


## [0.16.3] - 2026-04-02
### Fixed
- Wrote generated GitHub release notes to the runner temp directory so GoReleaser can publish without dirtying the checked-out tree.


## [0.16.2] - 2026-04-02
### Changed
- Switched releases to the shared PR-based `scripts/release.sh` flow, with `CHANGELOG.md` supplying the GitHub release notes and CI creating the version tag only after the merged release commit verifies.

### Fixed
- Removed deprecated release shims so there is exactly one supported release entrypoint.


## [0.16.1] - 2026-04-02

### Added
- Pipeline step state and result details in `bkt pipeline view` and `bkt status pipeline` for Bitbucket Cloud, making in-flight step progress visible without `--json`.

### Fixed
- Switched Data Center `bkt pr comments` to the pull request activities endpoint so general and inline comments can be listed without the invalid `path` query error.

## [0.15.0] - 2026-04-01

### Added
- `bkt pr create --draft` to create draft pull requests on Bitbucket Cloud and on Bitbucket Data Center 8.18+.
- `bkt pr comment --pending` to create pending review comments on Bitbucket Cloud and Bitbucket Data Center.

## [0.14.7] - 2026-04-01

### Fixed
- Keyed manual release rerun concurrency by tag so rerunning one release tag cannot cancel a different release recovery run.

## [0.14.6] - 2026-04-01

### Fixed
- Treated `Version already exists` as success when a skill publish reruns after retrying without an alias, preventing false-negative publish failures on release reruns.

## [0.14.5] - 2026-04-01

### Changed
- Added manual `workflow_dispatch` support for the release workflow so an existing tag can be rerun cleanly when GitHub release automation needs recovery without minting another version.

## [0.14.4] - 2026-04-01

### Fixed
- Serialized macOS keychain reads and writes behind an inter-process lock to prevent prompt storms when multiple `bkt` processes access the same token concurrently.
- Ad-hoc signed macOS binaries with the stable identifier `io.github.avivsinai.bitbucket-cli` in both local builds and GoReleaser artifacts so Keychain approvals survive Homebrew upgrades.

## [0.14.3] - 2026-04-01

### Fixed
- Prevented skill publish failures when a skill alias falls outside registry length limits.
- Ensured release automation stages optional Codex plugin manifests and verifies skill/plugin metadata versions before publishing or tagging.

### Changed
- Added a default-branch marketplace dispatch workflow so plugin updates are announced after merges to `master`.
- Updated the Codecov action to v6.

## [0.14.2] - 2026-03-30

### Changed
- Switched skill publishing to a tag-based release flow so binary and skill releases are driven from the same version tag.

## [0.14.1] - 2026-03-29

### Added
- Codex plugin manifest metadata for `bkt`, including interface metadata for marketplace and CLI consumers.

### Changed
- Consolidated skill packaging around the shared plugin manifest layout.

### Fixed
- Aligned Claude and Codex plugin manifest versions with release tags.

## [0.14.0] - 2026-03-18

### Added
- `bkt pr comment --file <path> --to-line <n>` and `--from-line <n>` flags for inline comments on PR diffs. Targets specific lines in the diff: `--to-line` for added/changed lines (new side), `--from-line` for removed lines (old side). Supports both Bitbucket Cloud (`inline` object) and Data Center (`anchor` object) (#86).
- `Inline` field on Cloud `PullRequestComment` and `Anchor` field on DC `PullRequestComment` structs, exposing file/line location in `--json` output.

### Changed
- Refactored `CommentPullRequest` in both Cloud and DC clients from positional parameters to a `CommentOptions` struct, enabling extensible comment creation.

## [0.13.1] - 2026-03-18

### Fixed
- Rejected half-braced UUID inputs (e.g. `{uuid` or `uuid}`) in `looksLikeUUID`. The regex now requires either both curly braces or neither.

## [0.13.0] - 2026-03-18

### Added
- UUID-based reviewer identification in `bkt pr create --reviewer`. Automatically detects canonical UUIDs and sends them with the `uuid` key to Bitbucket Cloud, while preserving username-based identification for non-UUID values (#87).
- USERNAME column in `bkt repo default-reviewers list` Cloud output, making it easier to copy reviewer identifiers for use with `--reviewer`.

### Fixed
- Tightened `looksLikeUUID` regex to match only canonical UUIDs (8-4-4-4-12 hex segments), preventing false positives on hex-only usernames like `cafe` or `dead`.
- Bitbucket Cloud API conformance: reviewer auto-detection, merge 202 async polling with bounded retries, variable UUID normalization in URL paths, and pipeline pagelen raised to API maximum of 100 (#78).

### Changed
- `--reviewer` flag help text now clarifies that both usernames and `{UUID}` values are accepted.
- Moved `looksLikeUUID` helper alongside `normalizeUUID` in `client.go` for co-location.

## [0.12.0] - 2026-03-17

### Added
- `bkt pr comments <id>` command to list pull request comments with optional `--state` filtering (`resolved`, `unresolved`, `all`). Supports both Cloud and Data Center with paginated API calls. Cloud uses the `resolution` object for client-side state filtering (#75).
- `bkt pr comment --parent <id>` flag for creating threaded replies under existing PR comments. Maps to the `parent.id` API field on both Cloud and Data Center (#76).
- `bkt repo default-reviewers list` command to show effective default reviewers for a repository. Cloud displays reviewer display names and UUIDs; Data Center returns users from default reviewer conditions (#77).

### Changed
- Bumped minimum Go version to 1.25 (required by `golang.org/x/term` v0.41.0).
- CI and release workflows updated to use Go 1.25.

## [0.11.1] - 2026-03-11

### Security
- Rejected path traversal names in `bkt extension remove` and `bkt extension exec`.
- Stopped passing sensitive `BKT_*` credentials into extension subprocess environments.
- Hardened git command invocation by using `git clone --` where supported and rejecting option-like positional arguments where `--` is unavailable.
- Capped `Retry-After` backoff handling in the retrying HTTP client to 60 seconds.
- Added a warning when loading plaintext host tokens from `config.yml`.

### Changed
- `bkt auth login` now requires `--allow-http` for `http://` hosts and warns when insecure HTTP is explicitly allowed.
- `bkt auth login --token` now warns that command-line tokens may be visible to other local users.

### Fixed
- Added regression coverage confirming Bitbucket Cloud `/user` requests preserve the versioned `/2.0` base path.

## [0.11.0] - 2026-03-11

### Added
- `bkt pr create --with-default-reviewers` flag that automatically fetches and merges repository default reviewers (Data Center only). Properly unmarshals `RestPullRequestCondition` responses, flattens nested reviewer groups, normalizes branch refs, and deduplicates across conditions and explicit `--reviewer` values.
- Cloud `GetEffectiveDefaultReviewers` client (gated pending UUID-based reviewer identity migration).
- Generic `mergeReviewers` helper with full deduplication including explicit reviewer duplicates.

## [0.10.0] - 2026-03-01

### Added
- `bkt commit diff <from> <to>` command to stream unified diffs between two refs (commit SHAs, branches, or tags) for both Bitbucket Cloud and Data Center.

### Fixed
- Added missing Cloud HTTP error test and unsupported host kind test for `commit diff` command.
- Documented `..` edge case in Cloud spec parsing for branch names containing double-dot.

## [0.9.0] - 2026-02-24

### Added
- Bitbucket Cloud support for `pr checkout`, `pr diff`, `pr approve`, `pr merge`, and `pr comment` subcommands (#57).
- `pr diff --stat` support for Cloud with per-file diff statistics.
- Fork-aware `pr checkout` on Cloud with automatic protocol inference from existing remotes.

### Fixed
- `pr checkout` now cleans up freshly added fork remotes when the subsequent fetch fails, preventing "remote already exists" errors on re-runs.

## [0.8.2] - 2026-02-24

### Added
- `BKT_TOKEN` environment variable for runtime token injection, bypassing the keyring entirely. Modeled after `GH_TOKEN` in the GitHub CLI. Enables headless/container usage without keyring dependencies (#59).
- `auth status` now shows the token source (`BKT_TOKEN` or `keyring`) for each configured host.

### Fixed
- `--allow-insecure-store` in headless/container environments no longer hangs on an interactive passphrase prompt. Returns an actionable error directing users to set `BKT_KEYRING_PASSPHRASE` or use `BKT_TOKEN` (#59).
- `auth login` and `auth logout` now return a clear error when `BKT_TOKEN` is set, since the token is externally managed.

## [0.8.1] - 2026-02-16

### Changed
- Added Codecov integration for automated coverage tracking and badge in README.
- Moved testing guidance from `docs/TESTING.md` into `CONTRIBUTING.md`.
- Removed static `docs/TESTING.md` coverage audit in favor of dynamic Codecov reporting.

## [0.8.0] - 2026-02-16

### Added
- `bkt pr decline <id>` to decline (reject) a pull request, supporting both Data Center and Cloud (#51).
- `bkt pr reopen <id>` to reopen a previously declined pull request (#51).
- `--delete-source` flag on `bkt pr decline` to delete the source branch after declining (Data Center only).

### Fixed
- `--delete-source` now correctly targets the source branch's own repository for forked pull requests, preventing accidental deletion in the wrong repo.

### Testing
- Comprehensive test coverage improvements across 7 packages: config, httpx, bbcloud, bbdc, cmdutil, format, and TESTING.md.

## [0.7.2] - 2026-02-06

### Fixed
- Made keyring operations more reliable in interactive environments by using a longer default timeout, while keeping a short timeout for headless/SSH/CI to prevent hangs. Added `BKT_KEYRING_TIMEOUT` for configuration (#46).

## [0.7.1] - 2026-02-05

### Fixed
- Prevented auth login hangs in headless/SSH environments when keyring backends block on GUI prompts (#44).
- Fixed skill publish version conflicts in CI.

## [0.7.0] - 2026-02-04

### Added
- Added issue attachment management commands (`bkt issue attachment ...`) (#41).

### Fixed
- Improved attachment handling safety, tests, and documentation (#41).
- Prevented a release race condition in CI with concurrency control.

## [0.6.0] - 2026-02-01

### Added
- `bkt pr list --mine` flag to list your PRs across all repositories in a workspace (Cloud) or project (Data Center) (#35). Thanks @steveardis!

## [0.5.5] - 2026-01-31

### Added
- Support build numbers as input for pipeline commands: `bkt pipeline view 10` (#38).
- Bitbucket Pipelines CI configuration for dogfooding on Bitbucket Cloud mirror.
- Documented `BKT_HTTP_DEBUG` environment variable for API troubleshooting.

### Fixed
- Fixed 400 "unexpected.response.body" error on `bkt pipeline view` and `bkt pipeline logs` commands. Bitbucket Cloud requires UUID braces to be URL-encoded (#38).
- Fixed 406 error on `bkt pipeline logs` by setting correct Accept header for octet-stream response.

## [0.5.4] - 2026-01-30

### Added
- `bkt pipeline list` now displays build number (`#N`) and timestamp for each pipeline run (#36).

### Changed
- Pipeline list now sorts by newest first (`-created_on`) instead of oldest first.

## [0.5.3] - 2026-01-27

### Added
- New `bkt` skill for Claude Code and Codex CLI (#28).

### Fixed
- Preserve base URL path when resolving request paths.
- Update Bitbucket Cloud auth to use Atlassian API tokens.

## [0.5.2] - 2026-01-18

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

[Unreleased]: https://github.com/avivsinai/bitbucket-cli/compare/v0.26.2...HEAD
[0.26.2]: https://github.com/avivsinai/bitbucket-cli/compare/v0.26.1...v0.26.2
[0.26.1]: https://github.com/avivsinai/bitbucket-cli/compare/v0.26.0...v0.26.1
[0.26.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.25.0...v0.26.0
[0.25.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.24.1...v0.25.0
[0.24.1]: https://github.com/avivsinai/bitbucket-cli/compare/v0.24.0...v0.24.1
[0.24.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.23.0...v0.24.0
[0.23.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.22.0...v0.23.0
[0.22.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.21.0...v0.22.0
[0.21.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.20.0...v0.21.0
[0.20.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.19.0...v0.20.0
[0.19.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.18.0...v0.19.0
[0.18.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.17.0...v0.18.0
[0.17.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.16.4...v0.17.0
[0.16.4]: https://github.com/avivsinai/bitbucket-cli/compare/v0.16.3...v0.16.4
[0.16.3]: https://github.com/avivsinai/bitbucket-cli/compare/v0.16.2...v0.16.3
[0.16.2]: https://github.com/avivsinai/bitbucket-cli/compare/v0.16.1...v0.16.2
[0.16.1]: https://github.com/avivsinai/bitbucket-cli/compare/v0.15.0...v0.16.1
[0.15.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.14.7...v0.15.0
[0.14.7]: https://github.com/avivsinai/bitbucket-cli/compare/v0.14.6...v0.14.7
[0.14.6]: https://github.com/avivsinai/bitbucket-cli/compare/v0.14.5...v0.14.6
[0.14.5]: https://github.com/avivsinai/bitbucket-cli/compare/v0.14.4...v0.14.5
[0.14.4]: https://github.com/avivsinai/bitbucket-cli/compare/v0.14.3...v0.14.4
[0.14.3]: https://github.com/avivsinai/bitbucket-cli/compare/v0.14.2...v0.14.3
[0.14.2]: https://github.com/avivsinai/bitbucket-cli/compare/v0.14.1...v0.14.2
[0.14.1]: https://github.com/avivsinai/bitbucket-cli/compare/v0.14.0...v0.14.1
[0.14.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.13.1...v0.14.0
[0.13.1]: https://github.com/avivsinai/bitbucket-cli/compare/v0.13.0...v0.13.1
[0.13.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.12.0...v0.13.0
[0.12.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.11.1...v0.12.0
[0.11.1]: https://github.com/avivsinai/bitbucket-cli/compare/v0.11.0...v0.11.1
[0.11.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.10.0...v0.11.0
[0.10.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.8.2...v0.9.0
[0.8.2]: https://github.com/avivsinai/bitbucket-cli/compare/v0.8.1...v0.8.2
[0.8.1]: https://github.com/avivsinai/bitbucket-cli/compare/v0.8.0...v0.8.1
[0.8.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.7.2...v0.8.0
[0.7.2]: https://github.com/avivsinai/bitbucket-cli/compare/v0.7.1...v0.7.2
[0.7.1]: https://github.com/avivsinai/bitbucket-cli/compare/v0.7.0...v0.7.1
[0.7.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.6.0...v0.7.0
[0.6.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.5.5...v0.6.0
[0.5.5]: https://github.com/avivsinai/bitbucket-cli/compare/v0.5.4...v0.5.5
[0.5.4]: https://github.com/avivsinai/bitbucket-cli/compare/v0.5.3...v0.5.4
[0.5.3]: https://github.com/avivsinai/bitbucket-cli/compare/v0.5.2...v0.5.3
[0.5.2]: https://github.com/avivsinai/bitbucket-cli/compare/v0.4.1...v0.5.2
[0.4.1]: https://github.com/avivsinai/bitbucket-cli/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.2.1...v0.4.0
[0.2.1]: https://github.com/avivsinai/bitbucket-cli/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/avivsinai/bitbucket-cli/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/avivsinai/bitbucket-cli/releases/tag/v0.1.0

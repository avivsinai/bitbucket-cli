<file_tree>
bitbucket-cli/
├── .github/
│   ├── ISSUE_TEMPLATE/
│   │   ├── bug_report.md
│   │   ├── config.yml
│   │   └── feature_request.md
│   ├── workflows/
│   │   ├── ci.yml
│   │   └── release.yml
│   ├── CODEOWNERS
│   ├── dependabot.yml
│   └── PULL_REQUEST_TEMPLATE.md
├── .promptcode/
│   └── presets/
│       └── bitbucket-cli-full.patterns
├── docs/
│   ├── RELEASE.md
│   ├── ROADMAP.md
│   ├── SECURITY.md
│   └── VERSIONING.md
├── internal/
│   ├── bktcmd/
│   │   └── cmd.go
│   ├── config/
│   │   └── config.go
│   └── remote/
│       └── remote.go
├── pkg/
│   ├── bbcloud/
│   │   ├── branches.go
│   │   ├── client_test.go
│   │   ├── client.go
│   │   ├── doc.go
│   │   ├── ping.go
│   │   ├── pullrequests.go
│   │   └── webhooks.go
│   ├── bbdc/
│   │   ├── admin.go
│   │   ├── automerge.go
│   │   ├── branches.go
│   │   ├── branchpermissions.go
│   │   ├── client.go
│   │   ├── diffstat.go
│   │   ├── doc.go
│   │   ├── permissions.go
│   │   ├── ping.go
│   │   ├── pullrequests.go
│   │   ├── reactions.go
│   │   ├── repos.go
│   │   ├── reviewers.go
│   │   ├── suggestions.go
│   │   ├── tasks.go
│   │   └── webhooks.go
│   ├── browser/
│   │   └── browser.go
│   ├── cmd/
│   │   ├── admin/
│   │   │   └── admin.go
│   │   ├── api/
│   │   │   └── api.go
│   │   ├── auth/
│   │   │   └── auth.go
│   │   ├── branch/
│   │   │   ├── branch.go
│   │   │   ├── protect.go
│   │   │   └── rebase.go
│   │   ├── context/
│   │   │   └── context.go
│   │   ├── extension/
│   │   │   └── extension.go
│   │   ├── factory/
│   │   │   └── factory.go
│   │   ├── perms/
│   │   │   └── perms.go
│   │   ├── pipeline/
│   │   │   └── pipeline.go
│   │   ├── pr/
│   │   │   ├── automerge.go
│   │   │   ├── pr.go
│   │   │   ├── reactions.go
│   │   │   ├── reviewergroup.go
│   │   │   ├── suggestions.go
│   │   │   └── tasks.go
│   │   ├── repo/
│   │   │   └── repo.go
│   │   ├── root/
│   │   │   └── root.go
│   │   ├── status/
│   │   │   ├── pipeline_cloud.go
│   │   │   ├── ratelimit.go
│   │   │   └── status.go
│   │   └── webhook/
│   │       └── webhook.go
│   ├── cmdutil/
│   │   ├── client.go
│   │   ├── context.go
│   │   ├── errors.go
│   │   ├── factory.go
│   │   ├── output.go
│   │   └── url.go
│   ├── format/
│   │   ├── doc.go
│   │   └── format.go
│   ├── httpx/
│   │   ├── client_test.go
│   │   ├── client.go
│   │   └── doc.go
│   ├── iostreams/
│   │   └── iostreams.go
│   ├── pager/
│   │   └── pager.go
│   ├── progress/
│   │   └── spinner.go
│   └── prompter/
│       ├── prompter_test.go
│       └── prompter.go
├── .gitignore
├── .goreleaser.yaml
├── AGENTS.md
├── CHANGELOG.md
├── CODE_OF_CONDUCT.md
├── CONTRIBUTING.md
├── go.mod
├── go.sum
├── GOVERNANCE.md
├── LICENSE
├── Makefile
├── README.md
├── SECURITY.md
└── SUPPORT.md</file_tree>

<files>
File: .github/CODEOWNERS (54 tokens)
```
# Default owners
*       @avivsinai

# Client libraries
pkg/bbdc/    @avivsinai
pkg/bbcloud/ @avivsinai

# CLI commands
pkg/cmd/     @avivsinai

```

File: .github/dependabot.yml (81 tokens)
```
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    assignees:
      - avivsinai
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
    assignees:
      - avivsinai

```

File: .github/ISSUE_TEMPLATE/bug_report.md (132 tokens)
```
---
name: Bug report
about: Help us fix a problem in bkt
labels: bug
---

## Describe the bug

<!-- A clear and concise description of what the bug is. -->

## To reproduce

Steps to reproduce the behavior:
1. ...
2. ...

## Expected behavior

<!-- What you expected to happen. -->

## Environment

- bkt version (`bkt --version`):
- OS and architecture:
- Bitbucket deployment (cloud, dc):
- Context configuration (workspace/project):

## Logs / output

```
# paste relevant output
```

## Additional context

<!-- Add any other context about the problem here. -->

```

File: .github/ISSUE_TEMPLATE/config.yml (44 tokens)
```
blank_issues_enabled: false
contact_links:
  - name: Security report
    url: https://github.com/avivsinai/bitbucket-cli/security
    about: Please report security vulnerabilities privately.

```

File: .github/ISSUE_TEMPLATE/feature_request.md (100 tokens)
```
---
name: Feature request
about: Suggest an idea for improving bkt
labels: enhancement
---

## Summary

<!-- A clear and concise description of the feature. -->

## Motivation

<!-- Why do you need this? What problem does it solve? -->

## Proposed solution

<!-- Describe the solution you'd like. Include CLI examples if relevant. -->

## Alternatives

<!-- Describe alternatives you've considered. -->

## Additional context

<!-- Add any other context or screenshots about the feature request here. -->

```

File: .github/PULL_REQUEST_TEMPLATE.md (113 tokens)
```
## Summary

<!-- What does this change do? Why is it needed? -->

## Testing

- [ ] `make fmt`
- [ ] `make test`
- [ ] `make build`
- [ ] Other (please specify):

## Screenshots / recordings

<!-- Attach terminal recordings or screenshots when changing CLI UX. -->

## Checklist

- [ ] Adds or updates documentation
- [ ] Updates `CHANGELOG.md`
- [ ] Signed commits (`git commit -s`)

## Notes for reviewers

<!-- Anything reviewers should pay special attention to. -->

```

File: .github/workflows/ci.yml (393 tokens)
```
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

permissions:
  contents: read
  security-events: write

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v6
        with:
          go-version: '1.25'

      - name: Verify gofmt
        run: |
          files="$(gofmt -l .)"
          if [ -n "$files" ]; then
            echo "The following files need gofmt:" >&2
            echo "$files" >&2
            exit 1
          fi

      - name: Go vet
        run: go vet ./...

      - name: Run tests
        run: go test ./...

      - name: Build CLI
        run: go build ./cmd/bkt

      - name: Generate SBOM
        uses: anchore/syft-action@v0
        with:
          args: dir:. -o cyclonedx-json=sbom.cdx.json

      - name: Upload SBOM artifact
        uses: actions/upload-artifact@v5
        with:
          name: sbom
          path: sbom.cdx.json

  scorecard:
    runs-on: ubuntu-latest
    permissions:
      id-token: write
      contents: read
      security-events: write
    steps:
      - uses: actions/checkout@v5
      - uses: ossf/scorecard-action@v2
        with:
          results_file: results.sarif
          results_format: sarif
      - uses: github/codeql-action/upload-sarif@v4
        with:
          sarif_file: results.sarif

```

File: .github/workflows/release.yml (188 tokens)
```
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  goreleaser:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v6
        with:
          go-version: '1.25'

      - name: Install Syft
        run: |
          curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sudo sh -s -- -b /usr/local/bin

      - name: Set up GoReleaser
        uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

```

File: .gitignore (42 tokens)
```
# Build artifacts
bkt
dist/

# Go
bin/
*.test
*.out

# IDEs
.idea/
.vscode/
*.swp

# macOS
.DS_Store


```

File: .goreleaser.yaml (401 tokens)
```
project_name: bkt

release:
  prerelease: false
  changelog:
    use: git
    sort: asc

builds:
  - id: bkt
    main: ./cmd/bkt
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    env:
      - CGO_ENABLED=0
    ldflags:
      - -s -w -X github.com/avivsinai/bitbucket-cli/internal/build.Version={{.Version}}
    mod_timestamp: '{{ .CommitDate }}'

archives:
  - format: tar.gz
    name_template: '{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}'
    files:
      - README.md
      - LICENSE
      - CHANGELOG.md

checksum:
  name_template: 'bkt_{{ .Version }}_checksums.txt'

sboms:
  - id: syft-json
    artifacts: archive
    cmd: syft dir:${{dir}} -o cyclonedx-json={{ .Name }}.sbom.json

brews:
  - name: bkt
    tap:
      owner: avivsinai
      name: homebrew-tap
    folder: Formula
    description: Bitbucket CLI with gh-style ergonomics
    homepage: https://github.com/avivsinai/bitbucket-cli
    license: MIT
    test: |
      system "#{bin}/bkt", "--version"

nfpms:
  - id: deb
    package_name: bkt
    file_name_template: 'bkt_{{ .Version }}_{{ .Arch }}'
    formats: [deb]
    maintainer: Aviv Sinai <opensource@avivsinai.dev>
    description: Bitbucket CLI
    homepage: https://github.com/avivsinai/bitbucket-cli
    license: MIT

```

File: .promptcode/presets/bitbucket-cli-full.patterns (62 tokens)
```
# bitbucket-cli-full preset
# Generated: 2025-10-27T07:14:50.243Z
# Source: patterns preserved (1)
# Patterns preserved as provided: 1

**/*
!**/node_modules/**
!**/dist/**
!**/build/**

```

File: AGENTS.md (527 tokens)
```
# Repository Guidelines

## Project Structure & Module Organization
- `cmd/bkt/` contains the CLI entry point; the binary executes `internal/bktcmd`.
- `internal/` hosts non-exported wiring, including configuration (`internal/config`) and build metadata.
- `pkg/` holds reusable packages consumed by Cobra commands (for example `pkg/cmd/repo` and `pkg/bbdc`).
- Tests live alongside Go packages (e.g., `pkg/cmd/...`); add `_test.go` files next to the implementation.

## Build, Test, and Development Commands
- `make build` – compile the CLI (`go build ./cmd/bkt`).
- `make test` or `go test ./...` – run all unit tests.
- `make fmt` – format code via `go fmt ./...`.
- `make tidy` – sync module dependencies (`go mod tidy`).
- Run the binary locally with `go run ./cmd/bkt --help` during development.

## Coding Style & Naming Conventions
- Follow Go conventions: tabs for indentation, exported identifiers use PascalCase, private helpers use camelCase.
- Keep package names short and lower-case; command packages sit under `pkg/cmd/<topic>`.
- Prefer structured errors with context (`fmt.Errorf("action: %w", err)`).
- Use `go fmt` before committing; add targeted comments only where logic is non-obvious.

## Testing Guidelines
- Write table-driven tests in `_test.go` files; name tests `TestPackageBehavior`.
- Favor golden files for CLI output snapshots when helpful under `testdata/`.
- Aim for coverage of flag parsing, API adapters, and error paths; mock HTTP interactions where necessary.
- Use `go test ./pkg/...` to focus on library packages during iteration.

## Commit & Pull Request Guidelines
- Commit messages follow the conventional prefix style (`feat:`, `fix:`, `docs:`) as seen in `feat: initial commit`.
- Keep commits focused and descriptive; reference issues with `Refs #123` in the body when applicable.
- Pull requests should include: clear summary, testing notes (`go test ./...`), and any screenshots for CLI output changes.
- Ensure CI (when configured) passes before requesting review; request at least one reviewer familiar with Bitbucket DC flows.

## Security & Configuration Tips
- Never commit real credentials; the CLI reads tokens from `$XDG_CONFIG_HOME/bkt/config.yml`.
- Use environment overrides (e.g., `BKT_CONFIG_DIR`) for sandbox testing without touching primary config.


```

File: CHANGELOG.md (410 tokens)
```
# Changelog

All notable changes to this project will be documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

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

### Changed
- `bkt pr diff` now supports `--stat` and streams via the pager when available.
- `bkt webhook` commands support both Data Center and Cloud instances.

### Fixed
- Improved error handling for context resolution and merge workflows.

## [0.1.0] - 2025-10-26
- Initial public release of `bkt`.

[Unreleased]: https://github.com/avivsinai/bitbucket-cli/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/avivsinai/bitbucket-cli/releases/tag/v0.1.0

```

File: CODE_OF_CONDUCT.md (1073 tokens)
```
# Contributor Covenant Code of Conduct

## Our Pledge

We as members, contributors, and leaders pledge to make participation in the bkt
community a harassment-free experience for everyone, regardless of age, body
size, visible or invisible disability, ethnicity, sex characteristics, gender
identity and expression, level of experience, education, socio-economic status,
nationality, personal appearance, race, religion, or sexual identity and
orientation.

We pledge to act and interact in ways that contribute to an open, welcoming,
diverse, inclusive, and healthy community.

## Our Standards

Examples of behavior that contributes to a positive environment for our
community include:

- Demonstrating empathy and kindness toward other people
- Being respectful of differing opinions, viewpoints, and experiences
- Giving and gracefully accepting constructive feedback
- Accepting responsibility and apologizing to those affected by our mistakes,
  and learning from the experience
- Focusing on what is best not just for us as individuals, but for the overall
  community

Examples of unacceptable behavior include:

- The use of sexualized language or imagery, and sexual attention or advances of
  any kind
- Trolling, insulting or derogatory comments, and personal or political attacks
- Public or private harassment
- Publishing others' private information, such as a physical or email address,
  without their explicit permission
- Other conduct which could reasonably be considered inappropriate in a
  professional setting

## Enforcement Responsibilities

Community leaders are responsible for clarifying and enforcing our standards of
acceptable behavior and will take appropriate and fair corrective action in
response to any behavior that they deem inappropriate, threatening, offensive,
or harmful.

Community leaders have the right and responsibility to remove, edit, or reject
comments, commits, code, wiki edits, issues, and other contributions that are
not aligned to this Code of Conduct, and will communicate reasons for moderation
decisions when appropriate.

## Scope

This Code of Conduct applies within all community spaces, and also applies when
an individual is officially representing the community in public spaces.
Examples of representing our community include using an official e-mail address,
posting via an official social media account, or acting as an appointed
representative at an online or offline event.

## Enforcement

Instances of abusive, harassing, or otherwise unacceptable behavior may be
reported to the project stewards at
[opensource@avivsinai.dev](mailto:opensource@avivsinai.dev). All complaints will
be reviewed and investigated promptly and fairly.

All community leaders are obligated to respect the privacy and security of the
reporter of any incident.

## Enforcement Guidelines

Community leaders will follow these Community Impact Guidelines in determining
the consequences for any action they deem in violation of this Code of Conduct:

1. **Correction**
   - *Community Impact*: Use of inappropriate language or other behavior deemed
     unprofessional or unwelcome in the community.
   - *Consequence*: A private, written warning from community leaders, providing
     clarity around the nature of the violation and an explanation of why the
     behavior was inappropriate. A public apology may be requested.

2. **Warning**
   - *Community Impact*: A violation through a single incident or series of
     actions.
   - *Consequence*: A warning with consequences for continued behavior. No
     interaction with the people involved, including unsolicited interaction
     with those enforcing the Code of Conduct, for a specified period of time.
     This includes avoiding interactions in community spaces as well as external
     channels like social media. Violating these terms may lead to a temporary or
     permanent ban.

3. **Temporary Ban**
   - *Community Impact*: A serious violation of community standards, including
     sustained inappropriate behavior.
   - *Consequence*: A temporary ban from any sort of interaction or public
     communication with the community for a specified period of time. No public
     or private interaction with the people involved, including unsolicited
     interaction with those enforcing the Code of Conduct, is allowed during
     this period. Violating these terms may lead to a permanent ban.

4. **Permanent Ban**
   - *Community Impact*: Demonstrating a pattern of violation of community
     standards, including sustained inappropriate behavior, harassment of an
     individual, or aggression toward or disparagement of classes of individuals.
   - *Consequence*: A permanent ban from any sort of public interaction within
     the community.

## Attribution

This Code of Conduct is adapted from the [Contributor Covenant][homepage],
version 2.1, available at
[https://www.contributor-covenant.org/version/2/1/code_of_conduct.html](https://www.contributor-covenant.org/version/2/1/code_of_conduct.html).

Community Impact Guidelines were inspired by
[Mozilla's code of conduct enforcement ladder](https://github.com/mozilla/diversity).

For answers to common questions about this code of conduct, see the FAQ at
[https://www.contributor-covenant.org/faq](https://www.contributor-covenant.org/faq).
Translations are available at the Contributor Covenant
[translations](https://www.contributor-covenant.org/translations) page.

[homepage]: https://www.contributor-covenant.org

```

File: CONTRIBUTING.md (572 tokens)
```
# Contributing to bkt

Thanks for your interest in making bkt better! We welcome issues, pull
requests, docs fixes, and release automation improvements.

## Ground rules

- Be respectful and follow our [Code of Conduct](CODE_OF_CONDUCT.md).
- We do **not** require a CLA. Instead, by contributing you agree to the
  [Developer Certificate of Origin (DCO)](https://developercertificate.org/).
  Please sign your commits with `git commit -s`.
- Always include tests when you add or change behavior. Table-driven unit tests
  live alongside the package they exercise.
- Run the quality gates before opening a PR:

  ```bash
  make fmt
  make test
  make build
  make sbom   # optional but encouraged if you have syft installed
  ```

- For non-trivial changes, open an issue or discussion first so we can align on
  direction.

## Workflow

1. Fork the repository and create a feature branch.
2. Make your changes with clear, conventional commits (`feat:`, `fix:`, `docs:`,
   etc.).
3. Update documentation and changelog entries when you change user-facing
   behavior.
4. Run the quality gates listed above. `make test` must pass on Linux and macOS.
5. Open a pull request. Include:
   - A concise summary of the change and rationale
   - Testing notes (commands executed, platforms exercised)
   - Screenshots or terminal captures for CLI UX changes
6. Respond to review feedback. We aim to respond within two business days.

## Project structure recap

See [README](README.md#project-layout) for the code layout. In short:

- `pkg/cmd/...` holds Cobra commands
- `pkg/bbdc` and `pkg/bbcloud` encapsulate Bitbucket Data Center and Cloud APIs
- `internal/config` persists contexts/hosts in `$XDG_CONFIG_HOME/bkt`
- `.github/` contains automation, templates, and CI workflows

## Release process (summary)

The detailed steps live in [`docs/RELEASE.md`](docs/RELEASE.md). In short:

1. Bump versions in `internal/build/version.go` (via ldflags) and update
   `CHANGELOG.md`.
2. Tag the release (`git tag vX.Y.Z && git push --tags`).
3. GitHub Actions runs [GoReleaser](goreleaser.yaml) to publish binaries and
   build SBOMs via Syft.

## Community roles

Governance and decision-making guidelines live in [GOVERNANCE.md](GOVERNANCE.md).
If you're interested in becoming a maintainer, open a discussion thread so we
can chat about expectations.

```

File: docs/RELEASE.md (293 tokens)
```
# Release handbook

1. **Prepare**
   - Ensure `main` is green.
   - Update `CHANGELOG.md` with the upcoming version under "Unreleased".
   - Run `make fmt test build`.
   - Regenerate the SBOM locally if possible: `make sbom`.

2. **Version bump**
   - Determine the next semantic version based on the changes.
   - Tag the release: `git tag vX.Y.Z` and `git push origin vX.Y.Z`.

3. **Automation**
   - GitHub Actions (`release.yml`) runs GoReleaser to build:
     - Linux, macOS, and Windows binaries (amd64 + arm64)
     - Checksums (`bkt_${VERSION}_checksums.txt`)
     - SBOMs (`sbom-${VERSION}.cyclonedx.json` via Syft)
   - Artifacts are uploaded to the GitHub Release page.

4. **Post-release**
   - Verify the release artifacts and SBOMs.
   - Announce the release in the `CHANGELOG.md` (already updated) and discussions.
   - Cut a new `Unreleased` section in the changelog for the next cycle.

## Dry runs

Use `goreleaser release --clean --snapshot` to exercise the pipeline without
publishing artifacts.

## Release cadence

We aim for monthly releases, with additional patch releases as needed for
security or regression fixes.

```

File: docs/ROADMAP.md (177 tokens)
```
# Roadmap

## Near term (Q4 2025)

- Data Center: integration tests against Bitbucket 9.x containers.
- Cloud: device authorization flow for managed accounts.
- `bkt status` enhancements for branch protection and audit logging.
- Golden snapshot tests for CLI human output using `testdata/` fixtures.

## Mid term (H1 2026)

- Plugin system for custom Bitbucket workflows.
- Declarative context definitions (`bkt context apply`) sourced from YAML.
- SSO / OAuth client management helpers.
- End-to-end smoke tests that exercise Pipelines via the REST API stub.

## Stretch

- Multi-cloud packaging (Homebrew, Scoop, pkg.go.dev install instructions).
- Native shell completions generation (`bkt completion bash|zsh|fish`).
- Extensible telemetry exporters (OpenTelemetry traces for API calls).

```

File: docs/SECURITY.md (343 tokens)
```
# Security operations playbook

This document expands on the top-level [SECURITY.md](../SECURITY.md) with
implementation details.

## Secrets handling

- bkt never stores plaintext credentials under version control. Tokens are read
  from `$XDG_CONFIG_HOME/bkt/config.yml` with permissions `0600`.
- For development, set `BKT_CONFIG_DIR` to a throwaway directory.
- Never commit test credentials. Use environment variables or the
  `internal/config/testdata` fixtures when unit testing.

## Dependency updates

- Dependabot is enabled for Go modules and GitHub Actions (`.github/dependabot.yml`).
- Run `go list -m -u all` periodically to spot stale modules.
- CI runs [OpenSSF Scorecard](https://github.com/ossf/scorecard) weekly.

## Supply chain

- Release artifacts are built with GoReleaser (`goreleaser.yaml`).
- Each release publishes a checksum manifest and an SBOM generated via Syft.
- Container images (if built) are signed with cosign and accompanied by an SBOM.

## Incident response

1. Triage the report and reproduce the issue.
2. Assign a severity (CVSS) and determine the affected versions.
3. Prepare a patch on a private branch. Request a security review from another
   maintainer.
4. Tag a release with the fix and update `CHANGELOG.md` with mitigation steps.
5. Notify the reporter and disclose publicly within seven days of the fix.

## Contact

Email [security@avivsinai.dev](mailto:security@avivsinai.dev). We prefer
coordinated disclosure.

```

File: docs/VERSIONING.md (88 tokens)
```
# Versioning

bkt follows [Semantic Versioning](https://semver.org/).

- MAJOR versions introduce breaking changes (e.g., CLI flag removals).
- MINOR versions add functionality in a backwards compatible manner.
- PATCH versions include backwards compatible bug fixes.

We tag releases as `vX.Y.Z` and publish binaries via GoReleaser. The changelog
summarises the changes for each release.

```

File: go.mod (155 tokens)
```
module github.com/avivsinai/bitbucket-cli

go 1.25.3

require (
	github.com/itchyny/gojq v0.12.17
	github.com/spf13/cobra v1.10.1
	golang.org/x/term v0.36.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/itchyny/timefmt-go v0.1.6 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	golang.org/x/sys v0.37.0 // indirect
)

```

File: go.sum (977 tokens)
```
github.com/cpuguy83/go-md2man/v2 v2.0.6/go.mod h1:oOW0eioCTA6cOiMLiUPZOpcVxMig6NIQQ7OS05n1F4g=
github.com/inconshreveable/mousetrap v1.1.0 h1:wN+x4NVGpMsO7ErUn/mUI3vEoE6Jt13X2s0bqwp9tc8=
github.com/inconshreveable/mousetrap v1.1.0/go.mod h1:vpF70FUmC8bwa3OWnCshd2FqLfsEA9PFc4w1p2J65bw=
github.com/itchyny/gojq v0.12.17 h1:8av8eGduDb5+rvEdaOO+zQUjA04MS0m3Ps8HiD+fceg=
github.com/itchyny/gojq v0.12.17/go.mod h1:WBrEMkgAfAGO1LUcGOckBl5O726KPp+OlkKug0I/FEY=
github.com/itchyny/timefmt-go v0.1.6 h1:ia3s54iciXDdzWzwaVKXZPbiXzxxnv1SPGFfM/myJ5Q=
github.com/itchyny/timefmt-go v0.1.6/go.mod h1:RRDZYC5s9ErkjQvTvvU7keJjxUYzIISJGxm9/mAERQg=
github.com/russross/blackfriday/v2 v2.1.0/go.mod h1:+Rmxgy9KzJVeS9/2gXHxylqXiyQDYRxCVz55jmeOWTM=
github.com/spf13/cobra v1.10.1 h1:lJeBwCfmrnXthfAupyUTzJ/J4Nc1RsHC/mSRU2dll/s=
github.com/spf13/cobra v1.10.1/go.mod h1:7SmJGaTHFVBY0jW4NXGluQoLvhqFQM+6XSKD+P4XaB0=
github.com/spf13/pflag v1.0.9 h1:9exaQaMOCwffKiiiYk6/BndUBv+iRViNW+4lEMi0PvY=
github.com/spf13/pflag v1.0.9/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=
golang.org/x/sys v0.37.0 h1:fdNQudmxPjkdUTPnLn5mdQv7Zwvbvpaxqs831goi9kQ=
golang.org/x/sys v0.37.0/go.mod h1:OgkHotnGiDImocRcuBABYBEXf8A9a87e/uXjp9XT3ks=
golang.org/x/term v0.36.0 h1:zMPR+aF8gfksFprF/Nc/rd1wRS1EI6nDBGyWAvDzx2Q=
golang.org/x/term v0.36.0/go.mod h1:Qu394IJq6V6dCBRgwqshf3mPF85AqzYEzofzRdZkWss=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405 h1:yhCVgyC4o1eVCa2tZl7eS0r+SDo693bJlVdllGtEeKM=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/yaml.v3 v3.0.1 h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=
gopkg.in/yaml.v3 v3.0.1/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=

```

File: GOVERNANCE.md (307 tokens)
```
# Project governance

## Roles

- **Maintainers**: responsible for roadmap curation, triage, and releases.
  Current maintainers:
  - Aviv Sinai (@avivsinai)
  - Project Stewards group (<opensource@avivsinai.dev>)
- **Reviewers**: contributors with merge rights on specific areas (commands,
  clients, docs). Reviewers can approve PRs but maintainers perform final merges.
- **Contributors**: anyone sending PRs, docs updates, or filing issues.

## Decision making

- Day-to-day decisions happen in issues, PRs, or discussions.
- Significant changes (CLI UX, API surface, security posture) require an
  accepted design proposal or RFC tracked in `docs/rfcs/` (future enhancement).
- Maintainers aim for consensus. If consensus cannot be reached, the maintainer
  quorum decides.

## Meetings

We operate asynchronously. Ad-hoc maintainer syncs occur as needed; notes are
published in `docs/notes/` when applicable.

## Becoming a maintainer

- Demonstrate sustained, high-quality contributions for at least two release
  cycles.
- Show ownership over an area (e.g., pipelines, config, docs).
- Request nomination via GitHub Discussions. Existing maintainers will vote and
  record the outcome in the discussion.

## Stepping down

Maintainers may step down at any time by notifying the team. Access will be
revoked and CODEOWNERS updated accordingly.

```

File: internal/bktcmd/cmd.go (343 tokens)
```
package bktcmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/avivsinai/bitbucket-cli/internal/build"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/factory"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/root"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// Main initialises CLI dependencies and executes the root command.
func Main() int {
	f, err := factory.New(build.Version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialise factory: %v\n", err)
		return 1
	}

	ios, err := f.Streams()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to configure IO: %v\n", err)
		return 1
	}

	rootCmd, err := root.NewCmdRoot(f)
	if err != nil {
		_, _ = fmt.Fprintf(ios.ErrOut, "failed to create root command: %v\n", err)
		return 1
	}

	if err := rootCmd.Execute(); err != nil {
		var exitErr *cmdutil.ExitError
		if errors.As(err, &exitErr) {
			if exitErr.Msg != "" {
				_, _ = fmt.Fprintln(ios.ErrOut, exitErr.Msg)
			}
			return exitErr.Code
		}
		if err != cmdutil.ErrSilent {
			_, _ = fmt.Fprintf(ios.ErrOut, "Error: %v\n", err)
		}
		return 1
	}

	return 0
}

```

File: internal/config/config.go (1435 tokens)
```
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

const currentVersion = 1

var (
	// ErrContextNotFound is returned when a requested context is missing.
	ErrContextNotFound = errors.New("context not found")
	// ErrHostNotFound is returned when a requested host entry is missing.
	ErrHostNotFound = errors.New("host not found")
)

// Config models persisted CLI state.
type Config struct {
	Version       int                 `yaml:"version"`
	ActiveContext string              `yaml:"active_context,omitempty"`
	Contexts      map[string]*Context `yaml:"contexts,omitempty"`
	Hosts         map[string]*Host    `yaml:"hosts,omitempty"`

	path string
	mu   sync.RWMutex
}

// Context captures user-scoped defaults that reference a host.
type Context struct {
	Host        string `yaml:"host"`
	ProjectKey  string `yaml:"project_key,omitempty"`
	Workspace   string `yaml:"workspace,omitempty"`
	DefaultRepo string `yaml:"default_repo,omitempty"`
}

// Host stores connection and credential details for a Bitbucket instance.
type Host struct {
	Kind     string `yaml:"kind"` // dc | cloud
	BaseURL  string `yaml:"base_url"`
	Username string `yaml:"username,omitempty"`
	Token    string `yaml:"token,omitempty"`
}

// Load retrieves configuration from disk, returning default values when the
// file does not exist. Supports config.yml and config.yaml filenames.
func Load() (*Config, error) {
	path, err := resolvePath()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Version:  currentVersion,
		Contexts: make(map[string]*Context),
		Hosts:    make(map[string]*Host),
		path:     path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	if cfg.Contexts == nil {
		cfg.Contexts = make(map[string]*Context)
	}
	if cfg.Hosts == nil {
		cfg.Hosts = make(map[string]*Host)
	}

	return cfg, nil
}

// Save persists the configuration atomically.
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.path == "" {
		path, err := resolvePath()
		if err != nil {
			return err
		}
		c.path = path
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	if c.Version == 0 {
		c.Version = currentVersion
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".config-*.yml")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp config: %w", err)
	}

	if err := tmpFile.Chmod(0o600); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("chmod temp config: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}

	if err := os.Rename(tmpFile.Name(), c.path); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// Path returns the absolute config file path.
func (c *Config) Path() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.path
}

// SetContext upserts a named context.
func (c *Config) SetContext(name string, ctx *Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Contexts == nil {
		c.Contexts = make(map[string]*Context)
	}
	c.Contexts[name] = ctx
}

// Context retrieves a named context.
func (c *Config) Context(name string) (*Context, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ctx, ok := c.Contexts[name]
	if !ok {
		return nil, ErrContextNotFound
	}
	return ctx, nil
}

// DeleteContext removes a named context and clears the active context if needed.
func (c *Config) DeleteContext(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.Contexts, name)
	if c.ActiveContext == name {
		c.ActiveContext = ""
	}
}

// SetActiveContext updates the active context name after verifying it exists.
func (c *Config) SetActiveContext(name string) error {
	if name == "" {
		c.mu.Lock()
		c.ActiveContext = ""
		c.mu.Unlock()
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.Contexts[name]; !ok {
		return ErrContextNotFound
	}
	c.ActiveContext = name
	return nil
}

// SetHost upserts host credentials by key.
func (c *Config) SetHost(key string, host *Host) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Hosts == nil {
		c.Hosts = make(map[string]*Host)
	}
	c.Hosts[key] = host
}

// Host retrieves host credentials by key.
func (c *Config) Host(key string) (*Host, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	h, ok := c.Hosts[key]
	if !ok {
		return nil, ErrHostNotFound
	}
	return h, nil
}

// DeleteHost removes a host entry. Contexts referencing the host should be
// cleaned up by the caller.
func (c *Config) DeleteHost(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Hosts, key)
}

func resolvePath() (string, error) {
	base := os.Getenv("BKT_CONFIG_DIR")
	if base == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("resolve config dir: %w", err)
		}
		base = filepath.Join(dir, "bkt")
	}
	return filepath.Join(base, "config.yml"), nil
}

```

File: internal/remote/remote.go (112 tokens)
```
package remote

import "errors"

// ErrNotImplemented signals that remote detection is not yet implemented.
var ErrNotImplemented = errors.New("remote detection not implemented")

// Locator represents a repository identifier derived from a git remote.
type Locator struct {
	Host       string
	Kind       string // dc | cloud
	Workspace  string
	ProjectKey string
	RepoSlug   string
}

// Detect attempts to infer the locator from git remotes.
func Detect(repoPath string) (Locator, error) {
	return Locator{}, ErrNotImplemented
}

```

File: LICENSE (221 tokens)
```
MIT License

Copyright (c) 2025 Aviv Sinai

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.


```

File: Makefile (132 tokens)
```
.PHONY: build fmt lint test tidy sbom release

build:
	go build ./cmd/bkt

fmt:
	go fmt ./...

lint:
	golangci-lint run

test:
	go test ./...

tidy:
	go mod tidy

sbom:
	@if ! command -v syft >/dev/null 2>&1; then \
		echo "syft not installed; install from https://github.com/anchore/syft" >&2; \
		exit 1; \
	fi
	syft dir:. -o cyclonedx-json=sbom.cdx.json

release:
	goreleaser release --clean

```

File: pkg/bbcloud/branches.go (535 tokens)
```
package bbcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Branch represents a Bitbucket Cloud branch.
type Branch struct {
	Name   string `json:"name"`
	Target struct {
		Hash string `json:"hash"`
		Type string `json:"type"`
	} `json:"target"`
	IsDefault bool `json:"default"`
	Links     struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
}

// BranchListOptions configure branch listings.
type BranchListOptions struct {
	Filter string
	Limit  int
}

// branchListPage wraps paginated branch responses.
type branchListPage struct {
	Values []Branch `json:"values"`
	Next   string   `json:"next"`
}

// ListBranches lists repository branches.
func (c *Client) ListBranches(ctx context.Context, workspace, repoSlug string, opts BranchListOptions) ([]Branch, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	pageLen := opts.Limit
	if pageLen <= 0 || pageLen > 100 {
		pageLen = 30
	}

	var params []string
	params = append(params, fmt.Sprintf("pagelen=%d", pageLen))
	if strings.TrimSpace(opts.Filter) != "" {
		params = append(params, "q="+url.QueryEscape(fmt.Sprintf("name ~ \"%s\"", opts.Filter)))
	}

	path := fmt.Sprintf("/repositories/%s/%s/refs/branches?%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		strings.Join(params, "&"),
	)

	var branches []Branch
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page branchListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		branches = append(branches, page.Values...)

		if opts.Limit > 0 && len(branches) >= opts.Limit {
			branches = branches[:opts.Limit]
			break
		}

		if page.Next == "" {
			break
		}
		nextURL, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		path = nextURL.RequestURI()
	}

	return branches, nil
}

```

File: pkg/bbcloud/client_test.go (705 tokens)
```
package bbcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestListPipelinesPaginates(t *testing.T) {
	var hits int32
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")

		switch count {
		case 1:
			if r.URL.Query().Get("pagelen") == "" {
				t.Fatalf("expected pagelen query in first request")
			}
			payload := PipelinePage{
				Values: []Pipeline{{UUID: "1"}, {UUID: "2"}},
				Next:   serverURL + "/repositories/work/repo/pipelines/?pagelen=20&page=2",
			}
			_ = json.NewEncoder(w).Encode(payload)
		case 2:
			payload := PipelinePage{
				Values: []Pipeline{{UUID: "3"}},
			}
			_ = json.NewEncoder(w).Encode(payload)
		default:
			t.Fatalf("unexpected extra request %d", count)
		}
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	pipelines, err := client.ListPipelines(ctx, "work", "repo", 0)
	if err != nil {
		t.Fatalf("ListPipelines: %v", err)
	}

	if len(pipelines) != 3 {
		t.Fatalf("expected 3 pipelines, got %d", len(pipelines))
	}
	if hits != 2 {
		t.Fatalf("expected 2 requests, got %d", hits)
	}
}

func TestListPipelinesRespectsLimit(t *testing.T) {
	var hits int32
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")

		if count == 1 {
			payload := PipelinePage{
				Values: []Pipeline{{UUID: "1"}, {UUID: "2"}},
				Next:   serverURL + "/repositories/work/repo/pipelines/?pagelen=20&page=2",
			}
			_ = json.NewEncoder(w).Encode(payload)
			return
		}

		t.Fatalf("unexpected second request when limit satisfied")
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	pipelines, err := client.ListPipelines(ctx, "work", "repo", 1)
	if err != nil {
		t.Fatalf("ListPipelines: %v", err)
	}

	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}
	if hits != 1 {
		t.Fatalf("expected 1 request, got %d", hits)
	}
}

```

File: pkg/bbcloud/client.go (2796 tokens)
```
package bbcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

// Options configure the Bitbucket Cloud client.
type Options struct {
	BaseURL     string
	Username    string
	Token       string
	Workspace   string
	EnableCache bool
	Retry       httpx.RetryPolicy
}

// Client wraps Bitbucket Cloud REST endpoints.
type Client struct {
	http *httpx.Client
}

// HTTP exposes the underlying HTTP client for advanced scenarios.
func (c *Client) HTTP() *httpx.Client {
	return c.http
}

// New constructs a Bitbucket Cloud client.
func New(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		opts.BaseURL = "https://api.bitbucket.org/2.0"
	}

	httpClient, err := httpx.New(httpx.Options{
		BaseURL:     opts.BaseURL,
		Username:    opts.Username,
		Password:    opts.Token,
		UserAgent:   "bkt-cli",
		EnableCache: opts.EnableCache,
		Retry:       opts.Retry,
	})
	if err != nil {
		return nil, err
	}

	return &Client{http: httpClient}, nil
}

// User represents a Bitbucket Cloud user profile.
type User struct {
	UUID     string `json:"uuid"`
	Username string `json:"username"`
	Display  string `json:"display_name"`
}

// CurrentUser retrieves the authenticated user.
func (c *Client) CurrentUser(ctx context.Context) (*User, error) {
	req, err := c.http.NewRequest(ctx, "GET", "/user", nil)
	if err != nil {
		return nil, err
	}
	var user User
	if err := c.http.Do(req, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// Repository identifies a Bitbucket Cloud repository.
type Repository struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	SCM       string `json:"scm"`
	IsPrivate bool   `json:"is_private"`
	Links     struct {
		Clone []struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"clone"`
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	Workspace struct {
		Slug string `json:"slug"`
	} `json:"workspace"`
	Project struct {
		Key string `json:"key"`
	} `json:"project"`
}

// Pipeline represents a pipeline execution.
type Pipeline struct {
	UUID  string `json:"uuid"`
	State struct {
		Result struct {
			Name string `json:"name"`
		} `json:"result"`
		Stage struct {
			Name string `json:"name"`
		} `json:"stage"`
		Name string `json:"name"`
	} `json:"state"`
	Target struct {
		Type string `json:"type"`
		Ref  struct {
			Name string `json:"name"`
		} `json:"ref"`
	} `json:"target"`
	CreatedOn   string `json:"created_on"`
	CompletedOn string `json:"completed_on"`
}

// PipelinePage encapsulates paginated pipeline results.
type PipelinePage struct {
	Values []Pipeline `json:"values"`
	Next   string     `json:"next"`
}

// ListPipelines lists recent pipelines.
func (c *Client) ListPipelines(ctx context.Context, workspace, repoSlug string, limit int) ([]Pipeline, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	pageLen := limit
	if pageLen <= 0 || pageLen > 50 {
		pageLen = 20
	}

	path := fmt.Sprintf("/repositories/%s/%s/pipelines/?pagelen=%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		pageLen,
	)

	var pipelines []Pipeline

	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page PipelinePage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		pipelines = append(pipelines, page.Values...)

		if limit > 0 && len(pipelines) >= limit {
			pipelines = pipelines[:limit]
			break
		}

		if page.Next == "" {
			break
		}

		nextURL, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		if nextURL.IsAbs() {
			if uri := nextURL.RequestURI(); uri != "" {
				path = uri
			} else {
				path = nextURL.String()
			}
		} else {
			path = nextURL.String()
		}
	}

	return pipelines, nil
}

// RepositoryListPage encapsulates paginated repository responses.
type repositoryListPage struct {
	Values []Repository `json:"values"`
	Next   string       `json:"next"`
}

// ListRepositories enumerates repositories for the workspace.
func (c *Client) ListRepositories(ctx context.Context, workspace string, limit int) ([]Repository, error) {
	if workspace == "" {
		return nil, fmt.Errorf("workspace is required")
	}

	pageLen := limit
	if pageLen <= 0 || pageLen > 100 {
		pageLen = 20
	}

	path := fmt.Sprintf("/repositories/%s?pagelen=%d",
		url.PathEscape(workspace),
		pageLen,
	)

	var repos []Repository

	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page repositoryListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		repos = append(repos, page.Values...)

		if limit > 0 && len(repos) >= limit {
			repos = repos[:limit]
			break
		}

		if page.Next == "" {
			break
		}

		// Bitbucket returns absolute URLs for next; reuse as-is.
		pathURL, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		path = pathURL.RequestURI()
	}

	return repos, nil
}

// GetRepository retrieves repository details.
func (c *Client) GetRepository(ctx context.Context, workspace, repoSlug string) (*Repository, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	path := fmt.Sprintf("/repositories/%s/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var repo Repository
	if err := c.http.Do(req, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// CreateRepositoryInput describes repository creation parameters.
type CreateRepositoryInput struct {
	Slug        string
	Name        string
	Description string
	IsPrivate   bool
	ProjectKey  string
}

// CreateRepository creates a repository within the workspace.
func (c *Client) CreateRepository(ctx context.Context, workspace string, input CreateRepositoryInput) (*Repository, error) {
	if workspace == "" {
		return nil, fmt.Errorf("workspace is required")
	}
	if input.Slug == "" {
		return nil, fmt.Errorf("repository slug is required")
	}

	body := map[string]any{
		"scm":        "git",
		"is_private": input.IsPrivate,
	}

	if input.Name != "" {
		body["name"] = input.Name
	}
	if input.Description != "" {
		body["description"] = input.Description
	}
	if input.ProjectKey != "" {
		body["project"] = map[string]any{
			"key": input.ProjectKey,
		}
	}

	path := fmt.Sprintf("/repositories/%s/%s",
		url.PathEscape(workspace),
		url.PathEscape(input.Slug),
	)
	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var repo Repository
	if err := c.http.Do(req, &repo); err != nil {
		return nil, err
	}
	return &repo, nil
}

// TriggerPipelineInput configures a pipeline run.
type TriggerPipelineInput struct {
	Ref       string
	Variables map[string]string
}

// TriggerPipeline triggers a new pipeline for the repo.
func (c *Client) TriggerPipeline(ctx context.Context, workspace, repoSlug string, in TriggerPipelineInput) (*Pipeline, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if in.Ref == "" {
		return nil, fmt.Errorf("ref is required")
	}

	body := map[string]any{
		"target": map[string]any{
			"ref_type": "branch",
			"type":     "pipeline_ref_target",
			"ref_name": in.Ref,
		},
	}
	if len(in.Variables) > 0 {
		vars := make([]map[string]any, 0, len(in.Variables))
		for k, v := range in.Variables {
			vars = append(vars, map[string]any{
				"key":     k,
				"value":   v,
				"secured": false,
			})
		}
		body["variables"] = vars
	}

	path := fmt.Sprintf("/repositories/%s/%s/pipelines/",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)

	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var pipeline Pipeline
	if err := c.http.Do(req, &pipeline); err != nil {
		return nil, err
	}
	return &pipeline, nil
}

// GetPipeline fetches pipeline details.
func (c *Client) GetPipeline(ctx context.Context, workspace, repoSlug, uuid string) (*Pipeline, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pipelines/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		strings.Trim(uuid, "{}"),
	)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	var pipeline Pipeline
	if err := c.http.Do(req, &pipeline); err != nil {
		return nil, err
	}
	return &pipeline, nil
}

// PipelineStep represents an individual pipeline step execution.
type PipelineStep struct {
	UUID  string `json:"uuid"`
	Name  string `json:"name"`
	State struct {
		Name string `json:"name"`
	} `json:"state"`
	Result struct {
		Name string `json:"name"`
	} `json:"result"`
}

// ListPipelineSteps enumerates step executions for the pipeline.
func (c *Client) ListPipelineSteps(ctx context.Context, workspace, repoSlug, pipelineUUID string) ([]PipelineStep, error) {
	path := fmt.Sprintf("/repositories/%s/%s/pipelines/%s/steps/",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		strings.Trim(pipelineUUID, "{}"),
	)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values []PipelineStep `json:"values"`
	}
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}
	return resp.Values, nil
}

// PipelineLog represents a step log chunk.
type PipelineLog struct {
	StepUUID string `json:"step_uuid"`
	Type     string `json:"type"`
	Log      string `json:"log"`
}

// GetPipelineLogs fetches logs for a pipeline step.
func (c *Client) GetPipelineLogs(ctx context.Context, workspace, repoSlug, pipelineUUID, stepUUID string) ([]byte, error) {
	pipelineUUID = strings.Trim(pipelineUUID, "{}")
	stepUUID = strings.Trim(stepUUID, "{}")
	path := fmt.Sprintf("/repositories/%s/%s/pipelines/%s/steps/%s/log",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		pipelineUUID,
		stepUUID,
	)

	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var buf strings.Builder
	if err := c.http.Do(req, &buf); err != nil {
		return nil, err
	}

	return []byte(buf.String()), nil
}

```

File: pkg/bbcloud/doc.go (16 tokens)
```
// Package bbcloud contains the Bitbucket Cloud client implementation.
package bbcloud

```

File: pkg/bbcloud/ping.go (124 tokens)
```
package bbcloud

import (
	"context"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

// Ping performs a lightweight request to update rate limit telemetry.
func (c *Client) Ping(ctx context.Context) error {
	req, err := c.http.NewRequest(ctx, "GET", "/user", nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// RateLimit exposes the last recorded rate limit data.
func (c *Client) RateLimit() httpx.RateLimit {
	return c.http.RateLimitState()
}

```

File: pkg/bbcloud/pullrequests.go (1298 tokens)
```
package bbcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// PullRequest models a Bitbucket Cloud pull request.
type PullRequest struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	State  string `json:"state"`
	Author struct {
		DisplayName string `json:"display_name"`
		Username    string `json:"username"`
	} `json:"author"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Commit struct {
			Hash string `json:"hash"`
		} `json:"commit"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
	} `json:"destination"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	Summary struct {
		Raw string `json:"raw"`
	} `json:"summary"`
}

// PullRequestListOptions configure PR listings.
type PullRequestListOptions struct {
	State string
	Limit int
	Mine  string
}

type pullRequestListPage struct {
	Values []PullRequest `json:"values"`
	Next   string        `json:"next"`
}

// ListPullRequests lists pull requests for a repository.
func (c *Client) ListPullRequests(ctx context.Context, workspace, repoSlug string, opts PullRequestListOptions) ([]PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	pageLen := opts.Limit
	if pageLen <= 0 || pageLen > 100 {
		pageLen = 20
	}

	var params []string
	params = append(params, fmt.Sprintf("pagelen=%d", pageLen))
	if state := strings.TrimSpace(opts.State); state != "" && !strings.EqualFold(state, "all") {
		params = append(params, "state="+url.QueryEscape(strings.ToUpper(state)))
	}
	if opts.Mine != "" {
		params = append(params, "q="+url.QueryEscape(fmt.Sprintf("author.username=\"%s\"", opts.Mine)))
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests?%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		strings.Join(params, "&"),
	)

	var prs []PullRequest
	for path != "" {
		req, err := c.http.NewRequest(ctx, "GET", path, nil)
		if err != nil {
			return nil, err
		}

		var page pullRequestListPage
		if err := c.http.Do(req, &page); err != nil {
			return nil, err
		}

		prs = append(prs, page.Values...)

		if opts.Limit > 0 && len(prs) >= opts.Limit {
			prs = prs[:opts.Limit]
			break
		}

		if page.Next == "" {
			break
		}
		nextURL, err := url.Parse(page.Next)
		if err != nil {
			return nil, err
		}
		path = nextURL.RequestURI()
	}

	return prs, nil
}

// GetPullRequest fetches a pull request by ID.
func (c *Client) GetPullRequest(ctx context.Context, workspace, repoSlug string, id int) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		id,
	)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := c.http.Do(req, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// CreatePullRequestInput configures PR creation.
type CreatePullRequestInput struct {
	Title       string
	Description string
	Source      string
	Destination string
	CloseSource bool
	Reviewers   []string
}

// CreatePullRequest creates a new pull request.
func (c *Client) CreatePullRequest(ctx context.Context, workspace, repoSlug string, input CreatePullRequestInput) (*PullRequest, error) {
	if workspace == "" || repoSlug == "" {
		return nil, fmt.Errorf("workspace and repository slug are required")
	}
	if strings.TrimSpace(input.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	if strings.TrimSpace(input.Source) == "" || strings.TrimSpace(input.Destination) == "" {
		return nil, fmt.Errorf("source and destination branches are required")
	}

	body := map[string]any{
		"title":               input.Title,
		"close_source_branch": input.CloseSource,
		"source": map[string]any{
			"branch": map[string]string{"name": input.Source},
		},
		"destination": map[string]any{
			"branch": map[string]string{"name": input.Destination},
		},
	}
	if input.Description != "" {
		body["description"] = input.Description
	}
	if len(input.Reviewers) > 0 {
		var reviewers []map[string]string
		for _, reviewer := range input.Reviewers {
			reviewers = append(reviewers, map[string]string{"username": reviewer})
		}
		body["reviewers"] = reviewers
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)

	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := c.http.Do(req, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

```

File: pkg/bbcloud/webhooks.go (584 tokens)
```
package bbcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Webhook models a Bitbucket Cloud repository webhook.
type Webhook struct {
	UUID        string   `json:"uuid"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	Active      bool     `json:"active"`
}

// WebhookInput configures webhook creation.
type WebhookInput struct {
	Description string
	URL         string
	Events      []string
	Active      bool
}

// ListWebhooks enumerates repository webhooks.
func (c *Client) ListWebhooks(ctx context.Context, workspace, repoSlug string) ([]Webhook, error) {
	path := fmt.Sprintf("/repositories/%s/%s/hooks",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values []Webhook `json:"values"`
	}
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}
	return resp.Values, nil
}

// CreateWebhook creates a new repository webhook.
func (c *Client) CreateWebhook(ctx context.Context, workspace, repoSlug string, input WebhookInput) (*Webhook, error) {
	if input.URL == "" {
		return nil, fmt.Errorf("webhook url is required")
	}
	if len(input.Events) == 0 {
		return nil, fmt.Errorf("at least one event is required")
	}

	body := map[string]any{
		"description": input.Description,
		"url":         input.URL,
		"events":      input.Events,
		"active":      input.Active,
	}

	path := fmt.Sprintf("/repositories/%s/%s/hooks",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)
	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var hook Webhook
	if err := c.http.Do(req, &hook); err != nil {
		return nil, err
	}

	return &hook, nil
}

// DeleteWebhook removes a webhook by uuid.
func (c *Client) DeleteWebhook(ctx context.Context, workspace, repoSlug, uuid string) error {
	path := fmt.Sprintf("/repositories/%s/%s/hooks/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(strings.Trim(uuid, "{}")),
	)
	req, err := c.http.NewRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

```

File: pkg/bbdc/admin.go (299 tokens)
```
package bbdc

import "context"

// RotateSecret triggers encryption key rotation via the secrets manager plugin.
func (c *Client) RotateSecret(ctx context.Context) error {
	req, err := c.http.NewRequest(ctx, "POST", "/rest/secrets-manager/1.0/keys/rotate", nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// LoggingConfig captures logging control settings.
type LoggingConfig struct {
	Level string `json:"level"`
	Async bool   `json:"async"`
}

// GetLoggingConfig fetches the current logging configuration.
func (c *Client) GetLoggingConfig(ctx context.Context) (*LoggingConfig, error) {
	req, err := c.http.NewRequest(ctx, "GET", "/rest/api/1.0/admin/logs/settings", nil)
	if err != nil {
		return nil, err
	}

	var cfg LoggingConfig
	if err := c.http.Do(req, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// UpdateLoggingConfig updates logging level/async mode.
func (c *Client) UpdateLoggingConfig(ctx context.Context, cfg LoggingConfig) error {
	req, err := c.http.NewRequest(ctx, "PUT", "/rest/api/1.0/admin/logs/settings", cfg)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

```

File: pkg/bbdc/automerge.go (573 tokens)
```
package bbdc

import (
	"context"
	"fmt"
	"net/url"
)

// AutoMergeSettings controls automatic merge behaviour when all checks pass.
type AutoMergeSettings struct {
	Enabled       bool   `json:"enabled"`
	StrategyID    string `json:"strategyId,omitempty"`
	CommitMessage string `json:"commitMessage,omitempty"`
	CloseSource   bool   `json:"closeSourceBranch"`
}

// GetAutoMerge retrieves auto-merge settings for a pull request.
func (c *Client) GetAutoMerge(ctx context.Context, projectKey, repoSlug string, prID int) (*AutoMergeSettings, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/auto-merge",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), nil)
	if err != nil {
		return nil, err
	}

	var settings AutoMergeSettings
	if err := c.http.Do(req, &settings); err != nil {
		return nil, err
	}

	return &settings, nil
}

// EnableAutoMerge enables automatic merge for the pull request using the given strategy.
func (c *Client) EnableAutoMerge(ctx context.Context, projectKey, repoSlug string, prID int, settings AutoMergeSettings) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}
	settings.Enabled = true

	req, err := c.http.NewRequest(ctx, "PUT", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/auto-merge",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), settings)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// DisableAutoMerge removes any automatic merge configuration for the pull request.
func (c *Client) DisableAutoMerge(ctx context.Context, projectKey, repoSlug string, prID int) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "DELETE", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/auto-merge",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), nil)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

```

File: pkg/bbdc/branches.go (1128 tokens)
```
package bbdc

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Branch represents a repository branch.
type Branch struct {
	ID           string `json:"id"`
	DisplayID    string `json:"displayId"`
	Type         string `json:"type"`
	LatestCommit string `json:"latestCommit"`
	IsDefault    bool   `json:"isDefault"`
}

// BranchListOptions filters branch listing.
type BranchListOptions struct {
	Filter string
	Limit  int
}

// ListBranches retrieves branches for a repository.
func (c *Client) ListBranches(ctx context.Context, projectKey, repoSlug string, opts BranchListOptions) ([]Branch, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	query := fmt.Sprintf("limit=%d", valueOrPositive(opts.Limit, 25))
	if opts.Filter != "" {
		query += "&filterText=" + url.QueryEscape(opts.Filter)
	}

	start := 0
	var branches []Branch

	for {
		u := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/branches?%s&start=%d",
			url.PathEscape(projectKey),
			url.PathEscape(repoSlug),
			query,
			start,
		)
		req, err := c.http.NewRequest(ctx, "GET", u, nil)
		if err != nil {
			return nil, err
		}

		var resp paged[Branch]
		if err := c.http.Do(req, &resp); err != nil {
			return nil, err
		}

		branches = append(branches, resp.Values...)

		if resp.IsLastPage || len(resp.Values) == 0 || (opts.Limit > 0 && len(branches) >= opts.Limit) {
			if opts.Limit > 0 && len(branches) > opts.Limit {
				branches = branches[:opts.Limit]
			}
			break
		}

		start = resp.NextPageStart
	}

	return branches, nil
}

// CreateBranchInput describes branch creation payload.
type CreateBranchInput struct {
	Name       string
	StartPoint string
	Message    string
}

// CreateBranch creates a new branch within the repository.
func (c *Client) CreateBranch(ctx context.Context, projectKey, repoSlug string, in CreateBranchInput) (*Branch, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}
	if in.Name == "" {
		return nil, fmt.Errorf("branch name is required")
	}
	if in.StartPoint == "" {
		return nil, fmt.Errorf("start point (commit or branch) is required")
	}

	body := map[string]any{
		"name":       ensureRef(in.Name),
		"startPoint": ensureRef(in.StartPoint),
	}
	if in.Message != "" {
		body["message"] = in.Message
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/branch-utils/1.0/projects/%s/repos/%s/branches",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	), body)
	if err != nil {
		return nil, err
	}

	var branch Branch
	if err := c.http.Do(req, &branch); err != nil {
		return nil, err
	}
	return &branch, nil
}

// DeleteBranch removes a branch from the repository.
func (c *Client) DeleteBranch(ctx context.Context, projectKey, repoSlug, branch string, dryRun bool) error {
	if projectKey == "" || repoSlug == "" || branch == "" {
		return fmt.Errorf("project key, repository slug, and branch are required")
	}

	body := map[string]any{
		"name":   ensureRef(branch),
		"dryRun": dryRun,
	}

	req, err := c.http.NewRequest(ctx, "DELETE", fmt.Sprintf("/rest/branch-utils/1.0/projects/%s/repos/%s/branches",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	), body)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// SetDefaultBranch updates the default branch for a repository.
func (c *Client) SetDefaultBranch(ctx context.Context, projectKey, repoSlug, branch string) error {
	if projectKey == "" || repoSlug == "" || branch == "" {
		return fmt.Errorf("project key, repository slug, and branch are required")
	}

	body := map[string]any{
		"id": ensureRef(branch),
	}

	req, err := c.http.NewRequest(ctx, "PUT", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/settings/default-branch",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	), body)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

func ensureRef(ref string) string {
	if strings.HasPrefix(ref, "refs/heads/") || strings.HasPrefix(ref, "refs/tags/") {
		return ref
	}
	return "refs/heads/" + ref
}

func valueOrPositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

```

File: pkg/bbdc/branchpermissions.go (757 tokens)
```
package bbdc

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// BranchRestriction represents a branch permission rule.
type BranchRestriction struct {
	ID      int    `json:"id"`
	Type    string `json:"type"`
	Matcher struct {
		ID   string `json:"id"`
		Type struct {
			ID string `json:"id"`
		} `json:"type"`
		DisplayID string `json:"displayId"`
	} `json:"matcher"`
	Users  []User   `json:"users"`
	Groups []string `json:"groups"`
}

// BranchRestrictionInput controls creation of a branch restriction.
type BranchRestrictionInput struct {
	Type        string
	MatcherID   string
	MatcherType string
	Users       []string
	Groups      []string
}

// ListBranchRestrictions lists restriction rules for the repository.
func (c *Client) ListBranchRestrictions(ctx context.Context, projectKey, repoSlug string) ([]BranchRestriction, error) {
	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/branch-permissions/2.0/projects/%s/repos/%s/restrictions",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values []BranchRestriction `json:"values"`
	}
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}

	return resp.Values, nil
}

// CreateBranchRestriction creates a new restriction.
func (c *Client) CreateBranchRestriction(ctx context.Context, projectKey, repoSlug string, in BranchRestrictionInput) (*BranchRestriction, error) {
	if in.Type == "" {
		return nil, fmt.Errorf("restriction type is required")
	}
	if in.MatcherID == "" {
		return nil, fmt.Errorf("matcher id is required")
	}
	if in.MatcherType == "" {
		in.MatcherType = "BRANCH"
	}

	body := map[string]any{
		"type": map[string]any{"id": strings.ToUpper(in.Type)},
		"matcher": map[string]any{
			"id":        in.MatcherID,
			"displayId": in.MatcherID,
			"type": map[string]any{
				"id": strings.ToUpper(in.MatcherType),
			},
		},
	}

	if len(in.Users) > 0 {
		body["users"] = in.Users
	}
	if len(in.Groups) > 0 {
		body["groups"] = in.Groups
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/branch-permissions/2.0/projects/%s/repos/%s/restrictions",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	), body)
	if err != nil {
		return nil, err
	}

	var restriction BranchRestriction
	if err := c.http.Do(req, &restriction); err != nil {
		return nil, err
	}
	return &restriction, nil
}

// DeleteBranchRestriction deletes a restriction by id.
func (c *Client) DeleteBranchRestriction(ctx context.Context, projectKey, repoSlug string, restrictionID int) error {
	req, err := c.http.NewRequest(ctx, "DELETE", fmt.Sprintf("/rest/branch-permissions/2.0/projects/%s/repos/%s/restrictions/%d",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		restrictionID,
	), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

```

File: pkg/bbdc/client.go (2178 tokens)
```
package bbdc

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

// Options configure the Bitbucket Data Center client.
type Options struct {
	BaseURL     string
	Username    string
	Token       string
	EnableCache bool
	Retry       httpx.RetryPolicy
}

// Client wraps Bitbucket Data Center REST endpoints.
type Client struct {
	http *httpx.Client
}

// HTTP exposes the underlying HTTP client for advanced scenarios.
func (c *Client) HTTP() *httpx.Client {
	return c.http
}

// New constructs a Bitbucket Data Center client.
func New(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}

	httpClient, err := httpx.New(httpx.Options{
		BaseURL:     opts.BaseURL,
		Username:    opts.Username,
		Password:    opts.Token,
		UserAgent:   "bkt-cli",
		EnableCache: opts.EnableCache,
		Retry:       opts.Retry,
	})
	if err != nil {
		return nil, err
	}

	return &Client{http: httpClient}, nil
}

// User represents a Bitbucket user.
type User struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	ID       int    `json:"id"`
	Email    string `json:"emailAddress"`
	Active   bool   `json:"active"`
	FullName string `json:"displayName"`
	Type     string `json:"type"`
}

// Repository represents a Bitbucket repository.
type Repository struct {
	Slug          string   `json:"slug"`
	Name          string   `json:"name"`
	ID            int      `json:"id"`
	Project       *Project `json:"project"`
	DefaultBranch string   `json:"defaultBranch,omitempty"`
	Links         struct {
		Self []struct {
			Href string `json:"href"`
		} `json:"self"`
		Web []struct {
			Href string `json:"href"`
		} `json:"web"`
		Clone []struct {
			Href string `json:"href"`
			Name string `json:"name"`
		} `json:"clone"`
	} `json:"links"`
}

// Project represents a Bitbucket project.
type Project struct {
	Key         string `json:"key"`
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Public      bool   `json:"public"`
}

// PullRequest models a Bitbucket pull request.
type PullRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Version     int    `json:"version"`
	Author      struct {
		User User `json:"user"`
	} `json:"author"`
	FromRef      Ref                      `json:"fromRef"`
	ToRef        Ref                      `json:"toRef"`
	Reviewers    []PullRequestReviewer    `json:"reviewers"`
	Participants []PullRequestParticipant `json:"participants"`
	Links        struct {
		Self []struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

// Ref describes a SCM ref.
type Ref struct {
	ID           string     `json:"id"`
	DisplayID    string     `json:"displayId"`
	LatestCommit string     `json:"latestCommit"`
	Repository   Repository `json:"repository"`
}

// CommitStatus describes build status for a commit.
type CommitStatus struct {
	State       string `json:"state"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

type paged[T any] struct {
	Size          int  `json:"size"`
	Limit         int  `json:"limit"`
	IsLastPage    bool `json:"isLastPage"`
	Start         int  `json:"start"`
	NextPageStart int  `json:"nextPageStart"`
	Values        []T  `json:"values"`
}

// CurrentUser fetches the user identified by slug.
func (c *Client) CurrentUser(ctx context.Context, userSlug string) (*User, error) {
	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/users/%s", url.PathEscape(userSlug)), nil)
	if err != nil {
		return nil, err
	}
	var user User
	if err := c.http.Do(req, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// ListRepositories enumerates repositories for a project, handling pagination.
func (c *Client) ListRepositories(ctx context.Context, projectKey string, limit int) ([]Repository, error) {
	if projectKey == "" {
		return nil, fmt.Errorf("project key is required")
	}

	const defaultPageSize = 25

	var (
		start = 0
		found []Repository
	)

	for {
		pageSize := defaultPageSize
		if limit > 0 {
			remaining := limit - len(found)
			if remaining <= 0 {
				break
			}
			if remaining < pageSize {
				pageSize = remaining
			}
		}

		u := fmt.Sprintf("/rest/api/1.0/projects/%s/repos?limit=%d&start=%d", url.PathEscape(projectKey), pageSize, start)
		req, err := c.http.NewRequest(ctx, "GET", u, nil)
		if err != nil {
			return nil, err
		}

		var resp paged[Repository]
		if err := c.http.Do(req, &resp); err != nil {
			return nil, err
		}

		found = append(found, resp.Values...)

		if limit > 0 && len(found) >= limit {
			found = found[:limit]
			break
		}

		if resp.IsLastPage || len(resp.Values) == 0 {
			break
		}
		start = resp.NextPageStart
	}

	return found, nil
}

// GetRepository fetches details for a repository.
func (c *Client) GetRepository(ctx context.Context, projectKey, repoSlug string) (*Repository, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s", url.PathEscape(projectKey), url.PathEscape(repoSlug)), nil)
	if err != nil {
		return nil, err
	}

	var repo Repository
	if err := c.http.Do(req, &repo); err != nil {
		return nil, err
	}

	return &repo, nil
}

// GetPullRequest fetches a pull request by id.
func (c *Client) GetPullRequest(ctx context.Context, projectKey, repoSlug string, id int) (*PullRequest, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d", url.PathEscape(projectKey), url.PathEscape(repoSlug), id), nil)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := c.http.Do(req, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// ListPullRequests lists pull requests for a repository.
func (c *Client) ListPullRequests(ctx context.Context, projectKey, repoSlug, state string, limit int) ([]PullRequest, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	const defaultPageSize = 25

	var (
		start = 0
		all   []PullRequest
	)

	for {
		pageSize := defaultPageSize
		if limit > 0 {
			remaining := limit - len(all)
			if remaining <= 0 {
				break
			}
			if remaining < pageSize {
				pageSize = remaining
			}
		}

		params := []string{fmt.Sprintf("limit=%d", pageSize)}
		if state != "" {
			params = append(params, "state="+url.QueryEscape(strings.ToUpper(state)))
		}

		u := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests?%s&start=%d",
			url.PathEscape(projectKey),
			url.PathEscape(repoSlug),
			strings.Join(params, "&"),
			start,
		)
		req, err := c.http.NewRequest(ctx, "GET", u, nil)
		if err != nil {
			return nil, err
		}

		var resp paged[PullRequest]
		if err := c.http.Do(req, &resp); err != nil {
			return nil, err
		}

		all = append(all, resp.Values...)

		if resp.IsLastPage || len(resp.Values) == 0 {
			break
		}
		start = resp.NextPageStart
	}

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}

	return all, nil
}

// CommitStatuses returns build statuses for a commit.
func (c *Client) CommitStatuses(ctx context.Context, sha string) ([]CommitStatus, error) {
	if sha == "" {
		return nil, fmt.Errorf("commit SHA is required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/build-status/1.0/commits/%s", sha), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values []CommitStatus `json:"values"`
	}
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}
	return resp.Values, nil
}

```

File: pkg/bbdc/diffstat.go (464 tokens)
```
package bbdc

import (
	"context"
	"fmt"
	"net/url"
)

// DiffStat aggregates additions/deletions for a pull request diff.
type DiffStat struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Files     int `json:"files"`
}

// PullRequestDiffStat retrieves diff statistics for the given pull request.
func (c *Client) PullRequestDiffStat(ctx context.Context, projectKey, repoSlug string, prID int) (*DiffStat, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	u := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/changes?withCounts=true&limit=1000",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	)

	stat := &DiffStat{}

	start := 0
	for {
		req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("%s&start=%d", u, start), nil)
		if err != nil {
			return nil, err
		}

		var resp struct {
			Values []struct {
				Path struct {
					ToString string `json:"toString"`
				} `json:"path"`
				Stats struct {
					Additions int `json:"additions"`
					Deletions int `json:"deletions"`
				} `json:"stats"`
			} `json:"values"`
			IsLastPage    bool `json:"isLastPage"`
			NextPageStart int  `json:"nextPageStart"`
		}

		if err := c.http.Do(req, &resp); err != nil {
			return nil, err
		}

		for _, change := range resp.Values {
			stat.Files++
			stat.Additions += change.Stats.Additions
			stat.Deletions += change.Stats.Deletions
		}

		if resp.IsLastPage || len(resp.Values) == 0 {
			break
		}
		start = resp.NextPageStart
	}

	return stat, nil
}

```

File: pkg/bbdc/doc.go (19 tokens)
```
// Package bbdc contains the Bitbucket Data Center client implementation.
package bbdc

```

File: pkg/bbdc/permissions.go (1062 tokens)
```
package bbdc

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Permission represents a Bitbucket permission assignment.
type Permission struct {
	User       User   `json:"user"`
	Permission string `json:"permission"`
}

// ListRepoPermissions returns repository user permissions.
func (c *Client) ListRepoPermissions(ctx context.Context, projectKey, repoSlug string, limit int) ([]Permission, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}
	return c.listPermissions(ctx, fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/permissions/users",
		url.PathEscape(projectKey), url.PathEscape(repoSlug)), limit)
}

// ListProjectPermissions returns project user permissions.
func (c *Client) ListProjectPermissions(ctx context.Context, projectKey string, limit int) ([]Permission, error) {
	if projectKey == "" {
		return nil, fmt.Errorf("project key is required")
	}
	return c.listPermissions(ctx, fmt.Sprintf("/rest/api/1.0/projects/%s/permissions/users", url.PathEscape(projectKey)), limit)
}

func (c *Client) listPermissions(ctx context.Context, path string, limit int) ([]Permission, error) {
	pageLimit := valueOrPositive(limit, 100)
	start := 0
	var out []Permission

	for {
		u := fmt.Sprintf("%s?limit=%d&start=%d", path, pageLimit, start)
		req, err := c.http.NewRequest(ctx, "GET", u, nil)
		if err != nil {
			return nil, err
		}
		var resp paged[Permission]
		if err := c.http.Do(req, &resp); err != nil {
			return nil, err
		}
		out = append(out, resp.Values...)
		if resp.IsLastPage || len(resp.Values) == 0 || (limit > 0 && len(out) >= limit) {
			if limit > 0 && len(out) > limit {
				out = out[:limit]
			}
			break
		}
		start = resp.NextPageStart
	}
	return out, nil
}

// GrantRepoPermission assigns a permission to a user for a repository.
func (c *Client) GrantRepoPermission(ctx context.Context, projectKey, repoSlug, username, permission string) error {
	if projectKey == "" || repoSlug == "" || username == "" || permission == "" {
		return fmt.Errorf("project, repo, username, and permission are required")
	}

	req, err := c.http.NewRequest(ctx, "PUT", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/permissions/users?name=%s&permission=%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		url.QueryEscape(username),
		url.QueryEscape(strings.ToUpper(permission)),
	), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// GrantProjectPermission assigns a permission to a user for a project.
func (c *Client) GrantProjectPermission(ctx context.Context, projectKey, username, permission string) error {
	if projectKey == "" || username == "" || permission == "" {
		return fmt.Errorf("project key, username, and permission are required")
	}

	req, err := c.http.NewRequest(ctx, "PUT", fmt.Sprintf("/rest/api/1.0/projects/%s/permissions/users?name=%s&permission=%s",
		url.PathEscape(projectKey),
		url.QueryEscape(username),
		url.QueryEscape(strings.ToUpper(permission)),
	), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// RevokeRepoPermission removes a repository permission for a user.
func (c *Client) RevokeRepoPermission(ctx context.Context, projectKey, repoSlug, username string) error {
	if projectKey == "" || repoSlug == "" || username == "" {
		return fmt.Errorf("project, repo, and username are required")
	}

	req, err := c.http.NewRequest(ctx, "DELETE", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/permissions/users?name=%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		url.QueryEscape(username),
	), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// RevokeProjectPermission removes a project permission for a user.
func (c *Client) RevokeProjectPermission(ctx context.Context, projectKey, username string) error {
	if projectKey == "" || username == "" {
		return fmt.Errorf("project key and username are required")
	}

	req, err := c.http.NewRequest(ctx, "DELETE", fmt.Sprintf("/rest/api/1.0/projects/%s/permissions/users?name=%s",
		url.PathEscape(projectKey),
		url.QueryEscape(username),
	), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

```

File: pkg/bbdc/ping.go (134 tokens)
```
package bbdc

import (
	"context"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

// Ping issues a lightweight request to populate telemetry such as rate limits.
func (c *Client) Ping(ctx context.Context) error {
	req, err := c.http.NewRequest(ctx, "GET", "/rest/api/1.0/application-properties", nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// RateLimit returns the last observed rate limit headers.
func (c *Client) RateLimit() httpx.RateLimit {
	return c.http.RateLimitState()
}

```

File: pkg/bbdc/pullrequests.go (1348 tokens)
```
package bbdc

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
)

// PullRequestReviewer represents a reviewer assignment.
type PullRequestReviewer struct {
	User User `json:"user"`
}

// PullRequestParticipant wraps a reviewer/participant entry.
type PullRequestParticipant struct {
	User     User   `json:"user"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	Approved bool   `json:"approved"`
}

// PullRequestComment represents a PR comment.
type PullRequestComment struct {
	ID     int    `json:"id"`
	Text   string `json:"text"`
	Author struct {
		User User `json:"user"`
	} `json:"author"`
}

// CreatePROptions configures pull request creation.
type CreatePROptions struct {
	Title        string
	Description  string
	SourceBranch string
	TargetBranch string
	Reviewers    []string
	CloseSource  bool
}

// CreatePullRequest creates a pull request between branches.
func (c *Client) CreatePullRequest(ctx context.Context, projectKey, repoSlug string, opts CreatePROptions) (*PullRequest, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}
	if opts.SourceBranch == "" || opts.TargetBranch == "" {
		return nil, fmt.Errorf("source and target branches are required")
	}
	if opts.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	body := map[string]any{
		"title":       opts.Title,
		"description": opts.Description,
		"fromRef": map[string]any{
			"id": ensureRef(opts.SourceBranch),
			"repository": map[string]any{
				"slug":    repoSlug,
				"project": map[string]any{"key": strings.ToUpper(projectKey)},
			},
		},
		"toRef": map[string]any{
			"id": ensureRef(opts.TargetBranch),
			"repository": map[string]any{
				"slug":    repoSlug,
				"project": map[string]any{"key": strings.ToUpper(projectKey)},
			},
		},
		"closeSourceBranch": opts.CloseSource,
	}

	if len(opts.Reviewers) > 0 {
		reviewers := make([]map[string]any, 0, len(opts.Reviewers))
		for _, reviewer := range opts.Reviewers {
			reviewers = append(reviewers, map[string]any{
				"user": map[string]string{"name": reviewer},
			})
		}
		body["reviewers"] = reviewers
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	), body)
	if err != nil {
		return nil, err
	}

	var pr PullRequest
	if err := c.http.Do(req, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// MergePROptions controls pull request merges.
type MergePROptions struct {
	Message           string
	Strategy          string
	CloseSourceBranch bool
}

// MergePullRequest merges the pull request.
func (c *Client) MergePullRequest(ctx context.Context, projectKey, repoSlug string, prID int, version int, opts MergePROptions) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}

	body := map[string]any{
		"version":           version,
		"message":           opts.Message,
		"closeSourceBranch": opts.CloseSourceBranch,
	}
	if opts.Strategy != "" {
		body["mergeStrategyId"] = opts.Strategy
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/merge",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), body)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// ApprovePullRequest records an approval for the current token.
func (c *Client) ApprovePullRequest(ctx context.Context, projectKey, repoSlug string, prID int) error {
	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/approve",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// CommentPullRequest adds a comment to the pull request.
func (c *Client) CommentPullRequest(ctx context.Context, projectKey, repoSlug string, prID int, text string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("comment text is required")
	}

	body := map[string]any{"text": text}
	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), body)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// PullRequestDiff streams the diff for the given pull request into w.
func (c *Client) PullRequestDiff(ctx context.Context, projectKey, repoSlug string, id int, w io.Writer) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}
	if w == nil {
		return fmt.Errorf("writer is required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/diff",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		id,
	), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/plain")

	return c.http.Do(req, w)
}

```

File: pkg/bbdc/reactions.go (595 tokens)
```
package bbdc

import (
	"context"
	"fmt"
	"net/url"
)

// Reaction represents an emoji reaction on a pull request comment.
type Reaction struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
}

// ListCommentReactions lists reactions for a given comment.
func (c *Client) ListCommentReactions(ctx context.Context, projectKey, repoSlug string, prID, commentID int) ([]Reaction, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments/%d/reactions",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
		commentID,
	), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values []Reaction `json:"values"`
	}
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}

	return resp.Values, nil
}

// AddCommentReaction adds a reaction to a comment.
func (c *Client) AddCommentReaction(ctx context.Context, projectKey, repoSlug string, prID, commentID int, emoji string) error {
	if projectKey == "" || repoSlug == "" || emoji == "" {
		return fmt.Errorf("project key, repository slug, and emoji are required")
	}

	body := map[string]any{"emoji": emoji}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments/%d/reactions",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
		commentID,
	), body)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// RemoveCommentReaction removes a reaction from a comment.
func (c *Client) RemoveCommentReaction(ctx context.Context, projectKey, repoSlug string, prID, commentID int, emoji string) error {
	if projectKey == "" || repoSlug == "" || emoji == "" {
		return fmt.Errorf("project key, repository slug, and emoji are required")
	}

	req, err := c.http.NewRequest(ctx, "DELETE", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments/%d/reactions/%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
		commentID,
		url.PathEscape(emoji),
	), nil)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

```

File: pkg/bbdc/repos.go (365 tokens)
```
package bbdc

import (
	"context"
	"fmt"
	"net/url"
)

// CreateRepositoryInput describes repository creation parameters.
type CreateRepositoryInput struct {
	Name          string
	SCMID         string
	Forkable      bool
	Public        bool
	Description   string
	DefaultBranch string
}

// CreateRepository creates a repository within the given project.
func (c *Client) CreateRepository(ctx context.Context, projectKey string, in CreateRepositoryInput) (*Repository, error) {
	if projectKey == "" {
		return nil, fmt.Errorf("project key is required")
	}
	if in.Name == "" {
		return nil, fmt.Errorf("repository name is required")
	}

	body := map[string]any{
		"name":        in.Name,
		"scmId":       valueOrDefault(in.SCMID, "git"),
		"forkable":    in.Forkable,
		"public":      in.Public,
		"description": in.Description,
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos", url.PathEscape(projectKey)), body)
	if err != nil {
		return nil, err
	}

	var repo Repository
	if err := c.http.Do(req, &repo); err != nil {
		return nil, err
	}

	if in.DefaultBranch != "" {
		if err := c.SetDefaultBranch(ctx, projectKey, repo.Slug, in.DefaultBranch); err != nil {
			return nil, err
		}
		repo.DefaultBranch = in.DefaultBranch
	}

	return &repo, nil
}

func valueOrDefault(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

```

File: pkg/bbdc/reviewers.go (548 tokens)
```
package bbdc

import (
	"context"
	"fmt"
	"net/url"
)

// ReviewerGroup represents a Bitbucket default reviewer group association.
type ReviewerGroup struct {
	Name string `json:"name"`
	ID   int    `json:"id"`
}

// ListReviewerGroups returns reviewer groups associated with a repository's default reviewers.
func (c *Client) ListReviewerGroups(ctx context.Context, projectKey, repoSlug string) ([]ReviewerGroup, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/default-reviewers/groups",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
	), nil)
	if err != nil {
		return nil, err
	}

	var payload struct {
		Values []ReviewerGroup `json:"values"`
	}
	if err := c.http.Do(req, &payload); err != nil {
		return nil, err
	}

	return payload.Values, nil
}

// AddReviewerGroup adds a reviewer group to the repository default reviewers.
func (c *Client) AddReviewerGroup(ctx context.Context, projectKey, repoSlug, group string) error {
	if projectKey == "" || repoSlug == "" || group == "" {
		return fmt.Errorf("project key, repository slug, and group name are required")
	}

	endpoint := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/default-reviewers/groups?name=%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		url.QueryEscape(group),
	)

	req, err := c.http.NewRequest(ctx, "PUT", endpoint, nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// RemoveReviewerGroup removes a reviewer group association from repository defaults.
func (c *Client) RemoveReviewerGroup(ctx context.Context, projectKey, repoSlug, group string) error {
	if projectKey == "" || repoSlug == "" || group == "" {
		return fmt.Errorf("project key, repository slug, and group name are required")
	}

	endpoint := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/default-reviewers/groups?name=%s",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		url.QueryEscape(group),
	)

	req, err := c.http.NewRequest(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

```

File: pkg/bbdc/suggestions.go (392 tokens)
```
package bbdc

import (
	"context"
	"fmt"
	"net/url"
)

// Suggestion represents a code suggestion attached to a pull request comment.
type Suggestion struct {
	ID        int    `json:"id"`
	Text      string `json:"text"`
	Applied   bool   `json:"applied"`
	CommentID int    `json:"commentId"`
}

// ApplySuggestion applies a code suggestion identified by comment+suggestion id.
func (c *Client) ApplySuggestion(ctx context.Context, projectKey, repoSlug string, prID, commentID, suggestionID int) error {
	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments/%d/suggestions/%d/apply",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
		commentID,
		suggestionID,
	), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// SuggestionPreview fetches the suggestion details for inspection prior to applying.
func (c *Client) SuggestionPreview(ctx context.Context, projectKey, repoSlug string, prID, commentID, suggestionID int) (*Suggestion, error) {
	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments/%d/suggestions/%d",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
		commentID,
		suggestionID,
	), nil)
	if err != nil {
		return nil, err
	}

	var suggestion Suggestion
	if err := c.http.Do(req, &suggestion); err != nil {
		return nil, err
	}
	return &suggestion, nil
}

```

File: pkg/bbdc/tasks.go (827 tokens)
```
package bbdc

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// PullRequestTask models a task attached to a pull request comment or diff.
type PullRequestTask struct {
	ID        int    `json:"id"`
	State     string `json:"state"`
	Text      string `json:"text"`
	Author    User   `json:"author"`
	CreatedAt int64  `json:"createdDate"`
	UpdatedAt int64  `json:"updatedDate"`
}

// ListPullRequestTasks lists tasks for the pull request.
func (c *Client) ListPullRequestTasks(ctx context.Context, projectKey, repoSlug string, prID int) ([]PullRequestTask, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/tasks",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values []PullRequestTask `json:"values"`
	}
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}

	return resp.Values, nil
}

// CreatePullRequestTask creates a new task attached to the pull request.
func (c *Client) CreatePullRequestTask(ctx context.Context, projectKey, repoSlug string, prID int, text string) (*PullRequestTask, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("task text is required")
	}

	body := map[string]any{
		"text": text,
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/tasks",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), body)
	if err != nil {
		return nil, err
	}

	var task PullRequestTask
	if err := c.http.Do(req, &task); err != nil {
		return nil, err
	}

	return &task, nil
}

// CompletePullRequestTask marks a task as resolved.
func (c *Client) CompletePullRequestTask(ctx context.Context, projectKey, repoSlug string, prID, taskID int) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/tasks/%d/resolve",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
		taskID,
	), nil)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

// ReopenPullRequestTask reopens a resolved task.
func (c *Client) ReopenPullRequestTask(ctx context.Context, projectKey, repoSlug string, prID, taskID int) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/tasks/%d/reopen",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
		taskID,
	), nil)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}

```

File: pkg/bbdc/webhooks.go (799 tokens)
```
package bbdc

import (
	"context"
	"fmt"
	"net/url"
)

// Webhook represents a Bitbucket webhook configuration.
type Webhook struct {
	ID            int            `json:"id"`
	Name          string         `json:"name"`
	URL           string         `json:"url"`
	Active        bool           `json:"active"`
	Events        []string       `json:"events"`
	Configuration map[string]any `json:"configuration,omitempty"`
}

// ListWebhooks retrieves repository webhooks.
func (c *Client) ListWebhooks(ctx context.Context, projectKey, repoSlug string) ([]Webhook, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}
	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/webhooks",
		url.PathEscape(projectKey), url.PathEscape(repoSlug)), nil)
	if err != nil {
		return nil, err
	}
	var resp struct {
		Values []Webhook `json:"values"`
	}
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}
	return resp.Values, nil
}

// CreateWebhookInput describes webhook creation request.
type CreateWebhookInput struct {
	Name   string
	URL    string
	Events []string
	Active bool
}

// CreateWebhook registers a webhook for the repository.
func (c *Client) CreateWebhook(ctx context.Context, projectKey, repoSlug string, in CreateWebhookInput) (*Webhook, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}
	if in.Name == "" || in.URL == "" || len(in.Events) == 0 {
		return nil, fmt.Errorf("name, url, and at least one event are required")
	}

	body := map[string]any{
		"name":   in.Name,
		"url":    in.URL,
		"events": in.Events,
		"active": in.Active,
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/webhooks",
		url.PathEscape(projectKey), url.PathEscape(repoSlug)), body)
	if err != nil {
		return nil, err
	}

	var hook Webhook
	if err := c.http.Do(req, &hook); err != nil {
		return nil, err
	}
	return &hook, nil
}

// DeleteWebhook removes a webhook by ID.
func (c *Client) DeleteWebhook(ctx context.Context, projectKey, repoSlug string, id int) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}
	req, err := c.http.NewRequest(ctx, "DELETE", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/webhooks/%d",
		url.PathEscape(projectKey), url.PathEscape(repoSlug), id), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// TestWebhook triggers a test delivery for the webhook.
func (c *Client) TestWebhook(ctx context.Context, projectKey, repoSlug string, id int) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}
	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/webhooks/%d/test",
		url.PathEscape(projectKey), url.PathEscape(repoSlug), id), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

```

File: pkg/browser/browser.go (224 tokens)
```
package browser

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
)

// Browser opens URLs using the host operating system facilities.
type Browser interface {
	Open(url string) error
}

type system struct{}

// NewSystem returns a Browser using the platform default opener.
func NewSystem() Browser {
	return &system{}
}

// Open launches the user's browser for the provided URL. Falls back to a
// descriptive error when the platform helper is unavailable.
func (s *system) Open(url string) error {
	if url == "" {
		return errors.New("url is required")
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("launch browser: %w", err)
	}
	return cmd.Wait()
}

```

File: pkg/cmd/admin/admin.go (1171 tokens)
```
package admin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdAdmin provides administrative operations for Bitbucket Data Center.
func NewCmdAdmin(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Administrative operations for Bitbucket",
	}

	cmd.AddCommand(newSecretsCmd(f))
	cmd.AddCommand(newLoggingCmd(f))

	return cmd
}

func newSecretsCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage secrets manager operations",
	}

	cmd.AddCommand(newSecretsRotateCmd(f))
	return cmd
}

func newSecretsRotateCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotate",
		Short: "Rotate encryption keys via the Secrets Manager plugin",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSecretsRotate(cmd, f)
		},
	}
	return cmd
}

func runSecretsRotate(cmd *cobra.Command, f *cmdutil.Factory) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, _, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("secrets rotation is only supported for Data Center contexts")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	spinner := f.ProgressSpinner()
	spinner.Start("rotating secrets")
	if err := client.RotateSecret(ctx); err != nil {
		spinner.Fail("secret rotation failed")
		return err
	}
	spinner.Stop("secret rotation complete")

	fmt.Fprintf(ios.Out, "✓ Secrets rotated successfully\n")
	return nil
}

func newLoggingCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logging",
		Short: "Inspect or update logging settings",
	}

	cmd.AddCommand(newLoggingGetCmd(f))
	cmd.AddCommand(newLoggingSetCmd(f))

	return cmd
}

func newLoggingGetCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show current logging configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLoggingGet(cmd, f)
		},
	}
	return cmd
}

func newLoggingSetCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &struct {
		Level string
		Async bool
	}{}

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Update logging configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLoggingSet(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Level, "level", "", "Logging level: TRACE, DEBUG, INFO, WARN, ERROR")
	cmd.Flags().BoolVar(&opts.Async, "async", false, "Enable asynchronous logging")

	return cmd
}

func runLoggingGet(cmd *cobra.Command, f *cmdutil.Factory) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, _, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("logging inspection is only supported for Data Center contexts")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	cfg, err := client.GetLoggingConfig(ctx)
	if err != nil {
		return err
	}

	return cmdutil.WriteOutput(cmd, ios.Out, cfg, func() error {
		fmt.Fprintf(ios.Out, "Level: %s\nAsync: %t\n", cfg.Level, cfg.Async)
		return nil
	})
}

func runLoggingSet(cmd *cobra.Command, f *cmdutil.Factory, opts *struct {
	Level string
	Async bool
}) error {
	_, _, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("logging configuration is only supported for Data Center contexts")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	cfg := bbdc.LoggingConfig{}
	if opts.Level != "" {
		cfg.Level = strings.ToUpper(opts.Level)
	}
	cfg.Async = opts.Async

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	if err := client.UpdateLoggingConfig(ctx, cfg); err != nil {
		return err
	}

	ios, err := f.Streams()
	if err != nil {
		return err
	}
	fmt.Fprintf(ios.Out, "✓ Updated logging configuration\n")
	return nil
}

```

File: pkg/cmd/api/api.go (1169 tokens)
```
package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type apiOptions struct {
	Method  string
	Input   string
	Fields  []string
	Headers []string
	Params  []string
}

// NewCmdAPI exposes a raw REST escape hatch akin to gh api.
func NewCmdAPI(f *cmdutil.Factory) *cobra.Command {
	opts := &apiOptions{}
	cmd := &cobra.Command{
		Use:   "api <path>",
		Short: "Make raw Bitbucket API requests",
		Long: `Call Bitbucket REST APIs directly for endpoints that do not yet have first-class commands.

Examples:
  bkt api /rest/api/1.0/projects
  bkt api /2.0/repositories --workspace my-team --param pagelen=50
  bkt api /rest/api/1.0/projects/ABC/repos --method POST --field name=demo --field scmId=git`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAPI(cmd, f, opts, args[0])
		},
	}

	cmd.Flags().StringVarP(&opts.Method, "method", "X", "", "HTTP method (default GET, or POST when a body is supplied)")
	cmd.Flags().StringVarP(&opts.Input, "input", "d", "", "JSON string to use as the request body")
	cmd.Flags().StringArrayVarP(&opts.Fields, "field", "F", nil, "Add JSON body field (key=value, repeatable)")
	cmd.Flags().StringArrayVarP(&opts.Headers, "header", "H", nil, "Add an HTTP request header (Key: Value)")
	cmd.Flags().StringArrayVarP(&opts.Params, "param", "P", nil, "Append query parameter (key=value)")

	return cmd
}

func runAPI(cmd *cobra.Command, f *cmdutil.Factory, opts *apiOptions, path string) error {
	method := strings.ToUpper(strings.TrimSpace(opts.Method))

	var body any
	if len(opts.Fields) > 0 && opts.Input != "" {
		return fmt.Errorf("--field and --input flags cannot be combined")
	}

	if len(opts.Fields) > 0 {
		payload := make(map[string]any, len(opts.Fields))
		for _, field := range opts.Fields {
			key, value, err := parseKeyValue(field)
			if err != nil {
				return fmt.Errorf("parse field %q: %w", field, err)
			}
			if key == "" {
				return fmt.Errorf("field %q is missing a key", field)
			}
			payload[key] = inferJSONValue(value)
		}
		body = payload
	} else if strings.TrimSpace(opts.Input) != "" {
		raw := json.RawMessage(opts.Input)
		body = raw
	}

	if method == "" {
		if body != nil {
			method = "POST"
		} else {
			method = "GET"
		}
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, _, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	httpClient, err := cmdutil.NewHTTPClient(host)
	if err != nil {
		return err
	}

	req, err := httpClient.NewRequest(cmd.Context(), method, path, body)
	if err != nil {
		return err
	}

	for _, header := range opts.Headers {
		key, value, err := parseHeader(header)
		if err != nil {
			return err
		}
		if key == "" {
			return fmt.Errorf("invalid header %q", header)
		}
		req.Header.Set(key, value)
	}

	if len(opts.Params) > 0 {
		query := req.URL.Query()
		for _, param := range opts.Params {
			key, value, err := parseKeyValue(param)
			if err != nil {
				return fmt.Errorf("parse param %q: %w", param, err)
			}
			if key == "" {
				return fmt.Errorf("param %q is missing a key", param)
			}
			query.Add(key, value)
		}
		req.URL.RawQuery = query.Encode()
	}

	ios, err := f.Streams()
	if err != nil {
		return err
	}

	if err := httpClient.Do(req, ios.Out); err != nil {
		return err
	}

	return nil
}

func parseKeyValue(input string) (string, string, error) {
	parts := strings.SplitN(input, "=", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("expected key=value format")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func parseHeader(input string) (string, string, error) {
	parts := strings.SplitN(input, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("header must be in \"Key: Value\" format")
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), nil
}

func inferJSONValue(raw string) any {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	var v any
	if err := json.Unmarshal([]byte(trimmed), &v); err == nil {
		return v
	}
	return raw
}

```

File: pkg/cmd/auth/auth.go (3044 tokens)
```
package auth

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// NewCmdAuth returns the root auth command.
func NewCmdAuth(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Bitbucket authentication credentials",
	}

	cmd.AddCommand(newLoginCmd(f))
	cmd.AddCommand(newStatusCmd(f))
	cmd.AddCommand(newLogoutCmd(f))

	return cmd
}

type loginOptions struct {
	Kind     string
	Host     string
	Username string
	Token    string
}

func newLoginCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &loginOptions{
		Kind: "dc",
	}

	cmd := &cobra.Command{
		Use:   "login [host]",
		Short: "Authenticate against a Bitbucket Data Center or Cloud host",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Host = args[0]
			}
			return runLogin(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Kind, "kind", opts.Kind, "Bitbucket deployment kind (dc or cloud)")
	cmd.Flags().StringVar(&opts.Username, "username", "", "Username for authentication (PAT owner or x-token-auth for HTTP tokens)")
	cmd.Flags().StringVar(&opts.Token, "token", "", "Personal access token or HTTP access token")

	return cmd
}

func runLogin(cmd *cobra.Command, f *cmdutil.Factory, opts *loginOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(ios.In)

	if opts.Host == "" {
		if !isTerminal(ios.In) {
			return fmt.Errorf("host is required when not running in a TTY")
		}
		opts.Host, err = promptString(reader, ios.Out, "Bitbucket base URL (e.g. https://bitbucket.example.com)")
		if err != nil {
			return err
		}
	}

	baseURL, err := cmdutil.NormalizeBaseURL(opts.Host)
	if err != nil {
		return err
	}

	kind := strings.ToLower(opts.Kind)
	if kind == "" {
		kind = "dc"
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	var hostKey string

	switch kind {
	case "dc":
		hostKey, err = cmdutil.HostKeyFromURL(baseURL)
		if err != nil {
			return err
		}
		if opts.Username == "" {
			if !isTerminal(ios.In) {
				return fmt.Errorf("username is required when not running in a TTY")
			}
			opts.Username, err = promptString(reader, ios.Out, "Username (use x-token-auth for project/repo tokens)")
			if err != nil {
				return err
			}
		}

		if opts.Token == "" {
			if !isTerminal(ios.In) {
				return fmt.Errorf("token is required when not running in a TTY")
			}
			opts.Token, err = promptSecret(ios, "Token")
			if err != nil {
				return err
			}
		}

		client, err := bbdc.New(bbdc.Options{
			BaseURL:  baseURL,
			Username: opts.Username,
			Token:    opts.Token,
		})
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		user, err := client.CurrentUser(ctx, opts.Username)
		if err != nil {
			return fmt.Errorf("verify credentials: %w", err)
		}

		cfg.SetHost(hostKey, &config.Host{
			Kind:     "dc",
			BaseURL:  baseURL,
			Username: opts.Username,
			Token:    opts.Token,
		})

		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Fprintf(ios.Out, "✓ Logged in to %s as %s (%s)\n", baseURL, user.FullName, user.Name)
	case "cloud":
		if opts.Username == "" {
			if !isTerminal(ios.In) {
				return fmt.Errorf("username is required when not running in a TTY")
			}
			opts.Username, err = promptString(reader, ios.Out, "Bitbucket username")
			if err != nil {
				return err
			}
		}

		if opts.Token == "" {
			if !isTerminal(ios.In) {
				return fmt.Errorf("token is required when not running in a TTY")
			}
			opts.Token, err = promptSecret(ios, "App password")
			if err != nil {
				return err
			}
		}

		apiURL := baseURL
		if strings.Contains(baseURL, "bitbucket.org") && !strings.Contains(baseURL, "api.bitbucket.org") {
			apiURL = "https://api.bitbucket.org/2.0"
		}

		hostKey, err = cmdutil.HostKeyFromURL(apiURL)
		if err != nil {
			return err
		}

		client, err := bbcloud.New(bbcloud.Options{
			BaseURL:     apiURL,
			Username:    opts.Username,
			Token:       opts.Token,
			EnableCache: true,
			Retry: httpx.RetryPolicy{
				MaxAttempts:    4,
				InitialBackoff: 200 * time.Millisecond,
				MaxBackoff:     2 * time.Second,
			},
		})
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		user, err := client.CurrentUser(ctx)
		if err != nil {
			return fmt.Errorf("verify credentials: %w", err)
		}

		cfg.SetHost(hostKey, &config.Host{
			Kind:     "cloud",
			BaseURL:  apiURL,
			Username: opts.Username,
			Token:    opts.Token,
		})

		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Fprintf(ios.Out, "✓ Logged in to Bitbucket Cloud as %s (%s)\n", user.Display, user.Username)
	default:
		return fmt.Errorf("unsupported deployment kind %q", opts.Kind)
	}

	return nil
}

func newStatusCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show authentication status for configured hosts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd, f)
		},
	}
	return cmd
}

func runStatus(cmd *cobra.Command, f *cmdutil.Factory) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	type hostSummary struct {
		Key      string `json:"key"`
		Kind     string `json:"kind"`
		BaseURL  string `json:"base_url"`
		Username string `json:"username,omitempty"`
	}

	type contextSummary struct {
		Name        string `json:"name"`
		Host        string `json:"host"`
		ProjectKey  string `json:"project_key,omitempty"`
		Workspace   string `json:"workspace,omitempty"`
		DefaultRepo string `json:"default_repo,omitempty"`
		Active      bool   `json:"active"`
	}

	var hostKeys []string
	for key := range cfg.Hosts {
		hostKeys = append(hostKeys, key)
	}
	sort.Strings(hostKeys)

	var hosts []hostSummary
	for _, key := range hostKeys {
		h := cfg.Hosts[key]
		hosts = append(hosts, hostSummary{
			Key:      key,
			Kind:     h.Kind,
			BaseURL:  h.BaseURL,
			Username: h.Username,
		})
	}

	var contextNames []string
	for name := range cfg.Contexts {
		contextNames = append(contextNames, name)
	}
	sort.Strings(contextNames)

	var contexts []contextSummary
	for _, name := range contextNames {
		ctx := cfg.Contexts[name]
		contexts = append(contexts, contextSummary{
			Name:        name,
			Host:        ctx.Host,
			ProjectKey:  ctx.ProjectKey,
			Workspace:   ctx.Workspace,
			DefaultRepo: ctx.DefaultRepo,
			Active:      cfg.ActiveContext == name,
		})
	}

	payload := struct {
		ActiveContext string           `json:"active_context,omitempty"`
		Hosts         []hostSummary    `json:"hosts"`
		Contexts      []contextSummary `json:"contexts"`
	}{
		ActiveContext: cfg.ActiveContext,
		Hosts:         hosts,
		Contexts:      contexts,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(hosts) == 0 {
			fmt.Fprintln(ios.Out, "No hosts configured. Run `bkt auth login` to add one.")
			return nil
		}

		fmt.Fprintln(ios.Out, "Hosts:")
		for _, h := range hosts {
			fmt.Fprintf(ios.Out, "  %s (%s)\n", h.BaseURL, h.Kind)
			if h.Username != "" {
				fmt.Fprintf(ios.Out, "    user: %s\n", h.Username)
			}
		}

		if len(contexts) == 0 {
			fmt.Fprintf(ios.Out, "\nNo contexts configured. Use `%s context create` to add one.\n", f.ExecutableName)
			return nil
		}

		fmt.Fprintln(ios.Out, "\nContexts:")
		for _, ctx := range contexts {
			activeMarker := " "
			if ctx.Active {
				activeMarker = "*"
			}
			fmt.Fprintf(ios.Out, "  %s %s (host: %s)\n", activeMarker, ctx.Name, ctx.Host)
			if ctx.ProjectKey != "" {
				fmt.Fprintf(ios.Out, "    project: %s\n", ctx.ProjectKey)
			}
			if ctx.Workspace != "" {
				fmt.Fprintf(ios.Out, "    workspace: %s\n", ctx.Workspace)
			}
			if ctx.DefaultRepo != "" {
				fmt.Fprintf(ios.Out, "    repo: %s\n", ctx.DefaultRepo)
			}
		}
		return nil
	})
}

type logoutOptions struct {
	Host string
}

func newLogoutCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &logoutOptions{}

	cmd := &cobra.Command{
		Use:   "logout [host]",
		Short: "Remove stored credentials for a host",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Host = args[0]
			}
			return runLogout(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Host, "host", "", "Host key or base URL to remove")

	return cmd
}

func runLogout(cmd *cobra.Command, f *cmdutil.Factory, opts *logoutOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	hostIdentifier := strings.TrimSpace(opts.Host)
	if hostIdentifier == "" {
		return fmt.Errorf("host is required")
	}

	key := hostIdentifier
	if _, ok := cfg.Hosts[key]; !ok {
		baseURL, err := cmdutil.NormalizeBaseURL(hostIdentifier)
		if err != nil {
			return fmt.Errorf("unknown host %q", hostIdentifier)
		}
		key, err = cmdutil.HostKeyFromURL(baseURL)
		if err != nil {
			return err
		}
		if _, ok := cfg.Hosts[key]; !ok {
			return fmt.Errorf("host %q not found in configuration", hostIdentifier)
		}
	}

	cfg.DeleteHost(key)

	for name, ctx := range cfg.Contexts {
		if ctx.Host == key {
			cfg.DeleteContext(name)
		}
	}

	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Removed credentials for %s\n", key)
	return nil
}

func promptString(reader *bufio.Reader, out io.Writer, label string) (string, error) {
	fmt.Fprintf(out, "%s: ", label)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func promptSecret(ios *iostreams.IOStreams, label string) (string, error) {
	file, ok := ios.In.(*os.File)
	if ok && term.IsTerminal(int(file.Fd())) {
		fmt.Fprintf(ios.Out, "%s: ", label)
		bytes, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(ios.Out)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(bytes)), nil
	}

	reader := bufio.NewReader(ios.In)
	return promptString(reader, ios.Out, label)
}

func isTerminal(in io.Reader) bool {
	file, ok := in.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

```

File: pkg/cmd/branch/branch.go (2517 tokens)
```
package branch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdBranch exposes branch operations.
func NewCmdBranch(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "branch",
		Short: "Inspect and manage branches",
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newDeleteCmd(f))
	cmd.AddCommand(newSetDefaultCmd(f))
	cmd.AddCommand(newProtectCmd(f))
	cmd.AddCommand(newRebaseCmd(f))

	return cmd
}

type listOptions struct {
	Project   string
	Workspace string
	Repo      string
	Filter    string
	Limit     int
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{Limit: 50}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List branches",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Filter, "filter", "", "Filter branches by text")
	cmd.Flags().IntVar(&opts.Limit, "limit", opts.Limit, "Maximum branches to list (0 for all)")

	return cmd
}

func runList(cmd *cobra.Command, f *cmdutil.Factory, opts *listOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	switch host.Kind {
	case "dc":
		projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		branches, err := client.ListBranches(ctx, projectKey, repoSlug, bbdc.BranchListOptions{Filter: opts.Filter, Limit: opts.Limit})
		if err != nil {
			return err
		}

		payload := map[string]any{
			"project":  projectKey,
			"repo":     repoSlug,
			"branches": branches,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(branches) == 0 {
				fmt.Fprintln(ios.Out, "No branches found.")
				return nil
			}

			for _, branch := range branches {
				marker := " "
				if branch.IsDefault {
					marker = "*"
				}
				fmt.Fprintf(ios.Out, "%s %s\t%s\n", marker, branch.DisplayID, branch.LatestCommit)
			}
			return nil
		})

	case "cloud":
		workspace := firstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		branches, err := client.ListBranches(ctx, workspace, repoSlug, bbcloud.BranchListOptions{Filter: opts.Filter, Limit: opts.Limit})
		if err != nil {
			return err
		}

		payload := map[string]any{
			"workspace": workspace,
			"repo":      repoSlug,
			"branches":  branches,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(branches) == 0 {
				fmt.Fprintln(ios.Out, "No branches found.")
				return nil
			}

			for _, branch := range branches {
				marker := " "
				if branch.IsDefault {
					marker = "*"
				}
				hash := branch.Target.Hash
				if len(hash) > 12 {
					hash = hash[:12]
				}
				fmt.Fprintf(ios.Out, "%s %s\t%s\n", marker, branch.Name, hash)
			}
			return nil
		})

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type createOptions struct {
	Project string
	Repo    string
	Source  string
	Message string
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{}
	cmd := &cobra.Command{
		Use:   "create <branch>",
		Short: "Create a new branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, f, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Source, "from", "", "Branch or commit to start from (required)")
	cmd.Flags().StringVar(&opts.Message, "message", "", "Optional branch creation message")
	_ = cmd.MarkFlagRequired("from")

	return cmd
}

func runCreate(cmd *cobra.Command, f *cmdutil.Factory, name string, opts *createOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("branch create currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	branch, err := client.CreateBranch(ctx, projectKey, repoSlug, bbdc.CreateBranchInput{
		Name:       name,
		StartPoint: opts.Source,
		Message:    opts.Message,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Created branch %s (%s)\n", branch.DisplayID, branch.LatestCommit)
	return nil
}

type deleteOptions struct {
	Project string
	Repo    string
	DryRun  bool
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &deleteOptions{}
	cmd := &cobra.Command{
		Use:     "delete <branch>",
		Aliases: []string{"rm"},
		Short:   "Delete a branch",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd, f, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "Perform a dry run without deleting")

	return cmd
}

func runDelete(cmd *cobra.Command, f *cmdutil.Factory, name string, opts *deleteOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("branch delete currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.DeleteBranch(ctx, projectKey, repoSlug, name, opts.DryRun); err != nil {
		return err
	}

	action := "Deleted"
	if opts.DryRun {
		action = "Validated"
	}
	fmt.Fprintf(ios.Out, "✓ %s branch %s\n", action, name)
	return nil
}

func newSetDefaultCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-default <branch>",
		Short: "Set the default branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetDefault(cmd, f, args[0])
		},
	}
	return cmd
}

func runSetDefault(cmd *cobra.Command, f *cmdutil.Factory, name string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("branch set-default currently supports Data Center contexts only")
	}

	projectKey := ctxCfg.ProjectKey
	repoSlug := ctxCfg.DefaultRepo
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.SetDefaultBranch(ctx, projectKey, repoSlug, name); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Set default branch to %s\n", name)
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

```

File: pkg/cmd/branch/protect.go (1814 tokens)
```
package branch

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type protectOptions struct {
	Project string
	Repo    string
	Branch  string
	Type    string
	Users   []string
	Groups  []string
	ID      int
}

func newProtectCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "protect",
		Short: "Manage branch protection rules",
	}

	cmd.AddCommand(newProtectListCmd(f))
	cmd.AddCommand(newProtectAddCmd(f))
	cmd.AddCommand(newProtectRemoveCmd(f))

	return cmd
}

func newProtectListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &protectOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List branch restrictions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProtectList(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func newProtectAddCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &protectOptions{}
	cmd := &cobra.Command{
		Use:   "add <branch>",
		Short: "Add a branch restriction",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Branch = args[0]
			return runProtectAdd(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Type, "type", "no-creates", "Restriction type (no-creates, no-deletes, fast-forward-only, require-approvals)")
	cmd.Flags().StringSliceVar(&opts.Users, "user", nil, "Usernames to apply the restriction to (repeatable)")
	cmd.Flags().StringSliceVar(&opts.Groups, "group", nil, "Group names to apply the restriction to (repeatable)")

	return cmd
}

func newProtectRemoveCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &protectOptions{}
	cmd := &cobra.Command{
		Use:   "remove <restriction-id>",
		Short: "Remove a branch restriction",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid restriction id %q", args[0])
			}
			opts.ID = id
			return runProtectRemove(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func runProtectList(cmd *cobra.Command, f *cmdutil.Factory, opts *protectOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("branch protect list currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	restrictions, err := client.ListBranchRestrictions(ctx, projectKey, repoSlug)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"project":      projectKey,
		"repo":         repoSlug,
		"restrictions": restrictions,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(restrictions) == 0 {
			fmt.Fprintln(ios.Out, "No branch restrictions configured.")
			return nil
		}
		for _, res := range restrictions {
			fmt.Fprintf(ios.Out, "%d	%s	%s\n", res.ID, res.Type, res.Matcher.DisplayID)
		}
		return nil
	})
}

func runProtectAdd(cmd *cobra.Command, f *cmdutil.Factory, opts *protectOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("branch protect add currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	typeID := mapProtectType(opts.Type)
	if typeID == "" {
		return fmt.Errorf("unsupported restriction type %q", opts.Type)
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	restriction, err := client.CreateBranchRestriction(ctx, projectKey, repoSlug, bbdc.BranchRestrictionInput{
		Type:        typeID,
		MatcherID:   ensureBranchRef(opts.Branch),
		MatcherType: "BRANCH",
		Users:       opts.Users,
		Groups:      opts.Groups,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Added restriction %d (%s) on %s\n", restriction.ID, restriction.Type, restriction.Matcher.DisplayID)
	return nil
}

func runProtectRemove(cmd *cobra.Command, f *cmdutil.Factory, opts *protectOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("branch protect remove currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.DeleteBranchRestriction(ctx, projectKey, repoSlug, opts.ID); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Removed restriction %d\n", opts.ID)
	return nil
}

func mapProtectType(t string) string {
	switch strings.ToLower(t) {
	case "no-creates":
		return "NO_CREATES"
	case "no-deletes":
		return "NO_DELETES"
	case "fast-forward-only":
		return "FAST_FORWARD_ONLY"
	case "require-approvals":
		return "PULL_REQUEST"
	default:
		return ""
	}
}

func ensureBranchRef(branch string) string {
	if branch == "" {
		return "refs/heads/*"
	}
	if strings.HasPrefix(branch, "refs/") {
		return branch
	}
	return "refs/heads/" + branch
}

```

File: pkg/cmd/branch/rebase.go (470 tokens)
```
package branch

import (
	"fmt"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type rebaseOptions struct {
	Onto        string
	Interactive bool
	NoFetch     bool
}

func newRebaseCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &rebaseOptions{}
	cmd := &cobra.Command{
		Use:   "rebase <branch>",
		Short: "Rebase the current branch onto another branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Onto = args[0]
			return runRebase(cmd, f, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.Interactive, "interactive", false, "Run rebase in interactive mode")
	cmd.Flags().BoolVar(&opts.NoFetch, "no-fetch", false, "Skip fetching before rebase")

	return cmd
}

func runRebase(cmd *cobra.Command, f *cmdutil.Factory, opts *rebaseOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	if !opts.NoFetch {
		fetch := exec.CommandContext(cmd.Context(), "git", "fetch", "--all")
		fetch.Stdout = ios.Out
		fetch.Stderr = ios.ErrOut
		fetch.Stdin = ios.In
		if err := fetch.Run(); err != nil {
			return fmt.Errorf("git fetch: %w", err)
		}
	}

	args := []string{"rebase"}
	if opts.Interactive {
		args = append(args, "-i")
	}
	args = append(args, opts.Onto)

	rebase := exec.CommandContext(cmd.Context(), "git", args...)
	rebase.Stdout = ios.Out
	rebase.Stderr = ios.ErrOut
	rebase.Stdin = ios.In
	if err := rebase.Run(); err != nil {
		return fmt.Errorf("git rebase: %w", err)
	}

	fmt.Fprintf(ios.Out, "✓ Rebasing onto %s complete\n", opts.Onto)
	return nil
}

```

File: pkg/cmd/context/context.go (1853 tokens)
```
package context

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdContext returns the context management command tree.
func NewCmdContext(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage Bitbucket CLI contexts",
	}

	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newUseCmd(f))
	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newDeleteCmd(f))

	return cmd
}

type createOptions struct {
	Host      string
	Project   string
	Workspace string
	Repo      string
	SetActive bool
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{}
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new CLI context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, f, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.Host, "host", "", "Host key or base URL (required)")
	cmd.Flags().StringVar(&opts.Project, "project", "", "Default Bitbucket project key (Data Center)")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Default Bitbucket workspace (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Default repository slug")
	cmd.Flags().BoolVar(&opts.SetActive, "set-active", false, "Set the new context as active")

	return cmd
}

func runCreate(cmd *cobra.Command, f *cmdutil.Factory, name string, opts *createOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	hostKey := strings.TrimSpace(opts.Host)
	if hostKey == "" {
		return fmt.Errorf("--host is required")
	}

	host, ok := cfg.Hosts[hostKey]
	if !ok {
		baseURL, err := cmdutil.NormalizeBaseURL(hostKey)
		if err != nil {
			return fmt.Errorf("host %q not found; run `%s auth login` first", hostKey, f.ExecutableName)
		}
		hostKey, err = cmdutil.HostKeyFromURL(baseURL)
		if err != nil {
			return err
		}
		host, ok = cfg.Hosts[hostKey]
		if !ok {
			return fmt.Errorf("host %q not found; run `%s auth login` first", opts.Host, f.ExecutableName)
		}
	}

	ctx := &config.Context{
		Host:        hostKey,
		DefaultRepo: strings.TrimSpace(opts.Repo),
	}

	switch host.Kind {
	case "dc":
		if opts.Project == "" {
			return fmt.Errorf("--project is required for Data Center contexts")
		}
		ctx.ProjectKey = strings.ToUpper(opts.Project)
	case "cloud":
		if opts.Workspace == "" {
			return fmt.Errorf("--workspace is required for Bitbucket Cloud contexts")
		}
		ctx.Workspace = opts.Workspace
	default:
		return fmt.Errorf("unknown host kind %q", host.Kind)
	}

	cfg.SetContext(name, ctx)

	if opts.SetActive || cfg.ActiveContext == "" {
		if err := cfg.SetActiveContext(name); err != nil {
			return err
		}
	}

	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Created context %q (host: %s)\n", name, hostKey)
	if cfg.ActiveContext == name {
		fmt.Fprintf(ios.Out, "✓ Context %q is now active\n", name)
	}
	return nil
}

func newUseCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use <name>",
		Short: "Activate an existing context",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUse(cmd, f, args[0])
		},
	}
	return cmd
}

func runUse(cmd *cobra.Command, f *cmdutil.Factory, name string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	if err := cfg.SetActiveContext(name); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Activated context %q\n", name)
	return nil
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, f)
		},
	}
	return cmd
}

func runList(cmd *cobra.Command, f *cmdutil.Factory) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	type summary struct {
		Name        string `json:"name"`
		Host        string `json:"host"`
		ProjectKey  string `json:"project_key,omitempty"`
		Workspace   string `json:"workspace,omitempty"`
		DefaultRepo string `json:"default_repo,omitempty"`
		Active      bool   `json:"active"`
	}

	var names []string
	for name := range cfg.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)

	var contexts []summary
	for _, name := range names {
		ctx := cfg.Contexts[name]
		contexts = append(contexts, summary{
			Name:        name,
			Host:        ctx.Host,
			ProjectKey:  ctx.ProjectKey,
			Workspace:   ctx.Workspace,
			DefaultRepo: ctx.DefaultRepo,
			Active:      cfg.ActiveContext == name,
		})
	}

	payload := struct {
		Active   string    `json:"active_context,omitempty"`
		Contexts []summary `json:"contexts"`
	}{
		Active:   cfg.ActiveContext,
		Contexts: contexts,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(contexts) == 0 {
			fmt.Fprintf(ios.Out, "No contexts configured. Use `%s context create` to add one.\n", f.ExecutableName)
			return nil
		}

		for _, ctx := range contexts {
			marker := " "
			if ctx.Active {
				marker = "*"
			}
			fmt.Fprintf(ios.Out, "%s %s (host: %s)\n", marker, ctx.Name, ctx.Host)
			if ctx.ProjectKey != "" {
				fmt.Fprintf(ios.Out, "    project: %s\n", ctx.ProjectKey)
			}
			if ctx.Workspace != "" {
				fmt.Fprintf(ios.Out, "    workspace: %s\n", ctx.Workspace)
			}
			if ctx.DefaultRepo != "" {
				fmt.Fprintf(ios.Out, "    repo: %s\n", ctx.DefaultRepo)
			}
		}
		return nil
	})
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm"},
		Short:   "Delete a context",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd, f, args[0])
		},
	}
	return cmd
}

func runDelete(cmd *cobra.Command, f *cmdutil.Factory, name string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	if _, err := cfg.Context(name); err != nil {
		return err
	}

	cfg.DeleteContext(name)

	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Deleted context %q\n", name)
	return nil
}

```

File: pkg/cmd/extension/extension.go (2260 tokens)
```
package extension

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdExtension manages external bkt extensions.
func NewCmdExtension(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "extension",
		Short: "Manage bkt CLI extensions",
	}

	cmd.AddCommand(newInstallCmd(f))
	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newRemoveCmd(f))
	cmd.AddCommand(newExecCmd(f))

	return cmd
}

func newInstallCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <repository>",
		Short: "Install an extension from a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExtensionInstall(cmd, f, args[0])
		},
	}
	return cmd
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List installed extensions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExtensionList(cmd, f)
		},
	}
	return cmd
}

func newRemoveCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Short:   "Remove an installed extension",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExtensionRemove(cmd, f, args[0])
		},
	}
	return cmd
}

func newExecCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec <name> [args...]",
		Short: "Execute an installed extension",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExtensionExec(cmd, f, args[0], args[1:])
		},
	}
	return cmd
}

func runExtensionInstall(cmd *cobra.Command, f *cmdutil.Factory, repo string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	root, err := ensureExtensionRoot(f)
	if err != nil {
		return err
	}

	name := inferExtensionName(repo)
	if name == "" {
		return fmt.Errorf("unable to infer extension name from %q", repo)
	}

	destination := filepath.Join(root, name)
	if _, err := os.Stat(destination); err == nil {
		return fmt.Errorf("extension %q is already installed", name)
	}

	args := []string{"clone", repo, destination}
	gitCmd := exec.CommandContext(cmd.Context(), "git", args...)
	gitCmd.Stdout = ios.Out
	gitCmd.Stderr = ios.ErrOut
	gitCmd.Stdin = ios.In

	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	execPath, err := findExtensionExecutable(destination, name)
	if err != nil {
		fmt.Fprintf(ios.ErrOut, "warning: %v\n", err)
	}

	fmt.Fprintf(ios.Out, "✓ Installed extension %s\n", name)
	if execPath != "" {
		rel, _ := filepath.Rel(root, execPath)
		fmt.Fprintf(ios.Out, "  binary: %s\n", rel)
	}
	return nil
}

func runExtensionList(cmd *cobra.Command, f *cmdutil.Factory) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	root, err := extensionRoot(f)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		entries = nil
	} else if err != nil {
		return err
	}

	type extensionSummary struct {
		Name       string `json:"name"`
		Path       string `json:"path"`
		Executable string `json:"executable,omitempty"`
	}

	var summaries []extensionSummary
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		dir := filepath.Join(root, name)
		execPath, _ := findExtensionExecutable(dir, name)
		rel := ""
		if execPath != "" {
			rel, _ = filepath.Rel(root, execPath)
		}
		summaries = append(summaries, extensionSummary{
			Name:       name,
			Path:       dir,
			Executable: rel,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})

	data := struct {
		Extensions []extensionSummary `json:"extensions"`
	}{Extensions: summaries}

	return cmdutil.WriteOutput(cmd, ios.Out, data, func() error {
		if len(summaries) == 0 {
			fmt.Fprintln(ios.Out, "No extensions installed. Use `bkt extension install <repository>` to add one.")
			return nil
		}
		for _, ext := range summaries {
			line := ext.Name
			if ext.Executable != "" {
				line = fmt.Sprintf("%s\t%s", ext.Name, ext.Executable)
			}
			fmt.Fprintln(ios.Out, line)
		}
		return nil
	})
}

func runExtensionRemove(cmd *cobra.Command, f *cmdutil.Factory, name string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	root, err := extensionRoot(f)
	if err != nil {
		return err
	}

	dir := filepath.Join(root, name)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("extension %q is not installed", name)
	} else if err != nil {
		return err
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove extension: %w", err)
	}

	fmt.Fprintf(ios.Out, "✓ Removed extension %s\n", name)
	return nil
}

func runExtensionExec(cmd *cobra.Command, f *cmdutil.Factory, name string, args []string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	root, err := extensionRoot(f)
	if err != nil {
		return err
	}

	dir := filepath.Join(root, name)
	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("extension %q is not installed", name)
		}
		return err
	}

	execPath, err := findExtensionExecutable(dir, name)
	if err != nil {
		return err
	}

	cmdExec := exec.CommandContext(cmd.Context(), execPath, args...)
	cmdExec.Stdout = ios.Out
	cmdExec.Stderr = ios.ErrOut
	cmdExec.Stdin = ios.In
	cmdExec.Dir = dir
	cmdExec.Env = append(os.Environ(),
		fmt.Sprintf("BKT_EXTENSION_DIR=%s", dir),
		fmt.Sprintf("BKT_EXTENSION_NAME=%s", name),
	)

	return cmdExec.Run()
}

func extensionRoot(f *cmdutil.Factory) (string, error) {
	cfg, err := f.ResolveConfig()
	if err != nil {
		return "", err
	}
	path := cfg.Path()
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("configuration path unknown; run `bkt auth login` first")
	}
	return filepath.Join(filepath.Dir(path), "extensions"), nil
}

func ensureExtensionRoot(f *cmdutil.Factory) (string, error) {
	root, err := extensionRoot(f)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	return root, nil
}

func inferExtensionName(repo string) string {
	trimmed := strings.TrimSpace(repo)
	trimmed = strings.TrimSuffix(trimmed, ".git")
	trimmed = strings.TrimSuffix(trimmed, "/")

	delim := strings.LastIndexAny(trimmed, "/:")
	if delim != -1 {
		trimmed = trimmed[delim+1:]
	}

	trimmed = strings.TrimPrefix(trimmed, "bkt-")
	return strings.TrimSpace(trimmed)
}

func findExtensionExecutable(dir, name string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var candidates []string
	prefix := fmt.Sprintf("bkt-%s", name)
	for _, entry := range entries {
		if entry.IsDir() {
			// consider bin/ subdirectory
			if entry.Name() == "bin" {
				subEntries, err := os.ReadDir(filepath.Join(dir, "bin"))
				if err != nil {
					continue
				}
				for _, sub := range subEntries {
					if sub.IsDir() {
						continue
					}
					if strings.HasPrefix(sub.Name(), prefix) && isExecutable(sub) {
						candidates = append(candidates, filepath.Join(dir, "bin", sub.Name()))
					}
				}
			}
			continue
		}
		if strings.HasPrefix(entry.Name(), prefix) && isExecutable(entry) {
			candidates = append(candidates, filepath.Join(dir, entry.Name()))
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no executable matching %q found in %s", prefix, dir)
	}

	sort.Strings(candidates)
	return candidates[0], nil
}

func isExecutable(entry os.DirEntry) bool {
	info, err := entry.Info()
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}
	mode := info.Mode()
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(info.Name()))
		return ext == ".exe" || ext == ".bat" || ext == ".cmd" || ext == ".ps1"
	}
	return mode&0o111 != 0
}

```

File: pkg/cmd/factory/factory.go (254 tokens)
```
package factory

import (
	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/browser"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
	"github.com/avivsinai/bitbucket-cli/pkg/pager"
	"github.com/avivsinai/bitbucket-cli/pkg/progress"
	"github.com/avivsinai/bitbucket-cli/pkg/prompter"
)

// New constructs a command factory following gh/jk idioms.
func New(appVersion string) (*cmdutil.Factory, error) {
	ios := iostreams.System()

	f := &cmdutil.Factory{
		AppVersion:     appVersion,
		ExecutableName: "bkt",
		IOStreams:      ios,
	}

	f.Browser = browser.NewSystem()
	f.Pager = pager.NewSystem(ios)
	f.Prompter = prompter.New(ios)
	f.Spinner = progress.NewSpinner(ios)

	f.Config = func() (*config.Config, error) {
		return config.Load()
	}

	return f, nil
}

```

File: pkg/cmd/perms/perms.go (2788 tokens)
```
package perms

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCommand manages repository and project permissions.
func NewCommand(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "perms",
		Short: "Manage Bitbucket permissions",
	}

	cmd.AddCommand(newProjectCmd(f))
	cmd.AddCommand(newRepoCmd(f))

	return cmd
}

type projectListOptions struct {
	Project string
	Limit   int
}

type projectGrantOptions struct {
	Project    string
	Username   string
	Permission string
}

type projectRevokeOptions struct {
	Project  string
	Username string
}

func newProjectCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage project-level permissions",
	}

	listOpts := &projectListOptions{Limit: 100}
	list := &cobra.Command{
		Use:   "list",
		Short: "List project permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectList(cmd, f, listOpts)
		},
	}
	list.Flags().StringVar(&listOpts.Project, "project", "", "Bitbucket project key (required)")
	list.Flags().IntVar(&listOpts.Limit, "limit", listOpts.Limit, "Maximum entries to display (0 for all)")
	_ = list.MarkFlagRequired("project")

	grantOpts := &projectGrantOptions{}
	grant := &cobra.Command{
		Use:   "grant",
		Short: "Grant project permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectGrant(cmd, f, grantOpts)
		},
	}
	grant.Flags().StringVar(&grantOpts.Project, "project", "", "Bitbucket project key (required)")
	grant.Flags().StringVar(&grantOpts.Username, "user", "", "Username to grant (required)")
	grant.Flags().StringVar(&grantOpts.Permission, "perm", "PROJECT_READ", "Permission (PROJECT_READ, PROJECT_WRITE, PROJECT_ADMIN)")
	_ = grant.MarkFlagRequired("project")
	_ = grant.MarkFlagRequired("user")

	revokeOpts := &projectRevokeOptions{}
	revoke := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke project permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectRevoke(cmd, f, revokeOpts)
		},
	}
	revoke.Flags().StringVar(&revokeOpts.Project, "project", "", "Bitbucket project key (required)")
	revoke.Flags().StringVar(&revokeOpts.Username, "user", "", "Username to revoke (required)")
	_ = revoke.MarkFlagRequired("project")
	_ = revoke.MarkFlagRequired("user")

	cmd.AddCommand(list, grant, revoke)
	return cmd
}

type repoListOptions struct {
	Project string
	Repo    string
	Limit   int
}

type repoGrantOptions struct {
	Project    string
	Repo       string
	Username   string
	Permission string
}

type repoRevokeOptions struct {
	Project  string
	Repo     string
	Username string
}

func newRepoCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage repository-level permissions",
	}

	listOpts := &repoListOptions{Limit: 100}
	list := &cobra.Command{
		Use:   "list",
		Short: "List repository permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRepoList(cmd, f, listOpts)
		},
	}
	list.Flags().StringVar(&listOpts.Project, "project", "", "Bitbucket project key (required)")
	list.Flags().StringVar(&listOpts.Repo, "repo", "", "Repository slug (required)")
	list.Flags().IntVar(&listOpts.Limit, "limit", listOpts.Limit, "Maximum entries to display (0 for all)")
	_ = list.MarkFlagRequired("project")
	_ = list.MarkFlagRequired("repo")

	grantOpts := &repoGrantOptions{}
	grant := &cobra.Command{
		Use:   "grant",
		Short: "Grant repository permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRepoGrant(cmd, f, grantOpts)
		},
	}
	grant.Flags().StringVar(&grantOpts.Project, "project", "", "Bitbucket project key (required)")
	grant.Flags().StringVar(&grantOpts.Repo, "repo", "", "Repository slug (required)")
	grant.Flags().StringVar(&grantOpts.Username, "user", "", "Username to grant (required)")
	grant.Flags().StringVar(&grantOpts.Permission, "perm", "REPO_READ", "Permission (REPO_READ, REPO_WRITE, REPO_ADMIN)")
	_ = grant.MarkFlagRequired("project")
	_ = grant.MarkFlagRequired("repo")
	_ = grant.MarkFlagRequired("user")

	revokeOpts := &repoRevokeOptions{}
	revoke := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke repository permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRepoRevoke(cmd, f, revokeOpts)
		},
	}
	revoke.Flags().StringVar(&revokeOpts.Project, "project", "", "Bitbucket project key (required)")
	revoke.Flags().StringVar(&revokeOpts.Repo, "repo", "", "Repository slug (required)")
	revoke.Flags().StringVar(&revokeOpts.Username, "user", "", "Username to revoke (required)")
	_ = revoke.MarkFlagRequired("project")
	_ = revoke.MarkFlagRequired("repo")
	_ = revoke.MarkFlagRequired("user")

	cmd.AddCommand(list, grant, revoke)
	return cmd
}

func runProjectList(cmd *cobra.Command, f *cmdutil.Factory, opts *projectListOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, _, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("perms project list currently supports Data Center contexts only")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	perms, err := client.ListProjectPermissions(ctx, opts.Project, opts.Limit)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"project":     opts.Project,
		"permissions": perms,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		for _, p := range perms {
			fmt.Fprintf(ios.Out, "%s\t%s\n", firstNonEmpty(p.User.FullName, p.User.Name), p.Permission)
		}
		if len(perms) == 0 {
			fmt.Fprintln(ios.Out, "No permissions found.")
		}
		return nil
	})
}

func runProjectGrant(cmd *cobra.Command, f *cmdutil.Factory, opts *projectGrantOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, _, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("perms project grant currently supports Data Center contexts only")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.GrantProjectPermission(ctx, opts.Project, opts.Username, opts.Permission); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Granted %s on project %s to %s\n", strings.ToUpper(opts.Permission), opts.Project, opts.Username)
	return nil
}

func runProjectRevoke(cmd *cobra.Command, f *cmdutil.Factory, opts *projectRevokeOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, _, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("perms project revoke currently supports Data Center contexts only")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.RevokeProjectPermission(ctx, opts.Project, opts.Username); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Revoked project permission for %s on %s\n", opts.Username, opts.Project)
	return nil
}

func runRepoList(cmd *cobra.Command, f *cmdutil.Factory, opts *repoListOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, _, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("perms repo list currently supports Data Center contexts only")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	perms, err := client.ListRepoPermissions(ctx, opts.Project, opts.Repo, opts.Limit)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"project":     opts.Project,
		"repo":        opts.Repo,
		"permissions": perms,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		for _, p := range perms {
			fmt.Fprintf(ios.Out, "%s\t%s\n", firstNonEmpty(p.User.FullName, p.User.Name), p.Permission)
		}
		if len(perms) == 0 {
			fmt.Fprintln(ios.Out, "No permissions found.")
		}
		return nil
	})
}

func runRepoGrant(cmd *cobra.Command, f *cmdutil.Factory, opts *repoGrantOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, _, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("perms repo grant currently supports Data Center contexts only")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.GrantRepoPermission(ctx, opts.Project, opts.Repo, opts.Username, opts.Permission); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Granted %s on %s/%s to %s\n", strings.ToUpper(opts.Permission), opts.Project, opts.Repo, opts.Username)
	return nil
}

func runRepoRevoke(cmd *cobra.Command, f *cmdutil.Factory, opts *repoRevokeOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, _, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("perms repo revoke currently supports Data Center contexts only")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.RevokeRepoPermission(ctx, opts.Project, opts.Repo, opts.Username); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Revoked repository permission for %s on %s/%s\n", opts.Username, opts.Project, opts.Repo)
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

```

File: pkg/cmd/pipeline/pipeline.go (2256 tokens)
```
package pipeline

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdPipeline interacts with Bitbucket Cloud pipelines.
func NewCmdPipeline(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pipeline",
		Short: "Run and inspect Bitbucket Cloud pipelines",
		Long:  "Interact with Bitbucket Cloud Pipelines. Commands are no-ops for Data Center contexts.",
	}

	cmd.AddCommand(newRunCmd(f))
	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newViewCmd(f))
	cmd.AddCommand(newLogsCmd(f))

	return cmd
}

type baseOptions struct {
	Workspace string
	Repo      string
}

type runOptions struct {
	baseOptions
	Ref       string
	Variables []string
}

type listOptions struct {
	baseOptions
	Limit int
}

type viewOptions struct {
	baseOptions
	UUID string
}

type logsOptions struct {
	baseOptions
	UUID string
	Step string
}

func newRunCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &runOptions{}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Trigger a new pipeline run",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipelineRun(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Ref, "ref", "main", "Git ref to run the pipeline on")
	cmd.Flags().StringSliceVar(&opts.Variables, "var", nil, "Pipeline variable in KEY=VALUE form (repeatable)")

	return cmd
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{Limit: 20}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List recent pipeline runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPipelineList(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().IntVar(&opts.Limit, "limit", opts.Limit, "Maximum pipelines to display")

	return cmd
}

func newViewCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &viewOptions{}
	cmd := &cobra.Command{
		Use:   "view <uuid>",
		Short: "Show details for a pipeline run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.UUID = args[0]
			return runPipelineView(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func newLogsCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &logsOptions{}
	cmd := &cobra.Command{
		Use:   "logs <uuid>",
		Short: "Fetch logs for a pipeline run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.UUID = args[0]
			return runPipelineLogs(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Step, "step", "", "Specific step UUID to fetch logs for")

	return cmd
}

func runPipelineRun(cmd *cobra.Command, f *cmdutil.Factory, opts *runOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	workspace, repo, host, err := resolveCloudRepo(cmd, f, opts.Workspace, opts.Repo)
	if err != nil {
		return err
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	vars := make(map[string]string)
	for _, v := range opts.Variables {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid variable %q, expected KEY=VALUE", v)
		}
		vars[strings.TrimSpace(parts[0])] = parts[1]
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	pipeline, err := client.TriggerPipeline(ctx, workspace, repo, bbcloud.TriggerPipelineInput{
		Ref:       opts.Ref,
		Variables: vars,
	})
	if err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Triggered pipeline %s on %s/%s (%s)\n", pipeline.UUID, workspace, repo, pipeline.State.Name)
	return nil
}

func runPipelineList(cmd *cobra.Command, f *cmdutil.Factory, opts *listOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	workspace, repo, host, err := resolveCloudRepo(cmd, f, opts.Workspace, opts.Repo)
	if err != nil {
		return err
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	pipelines, err := client.ListPipelines(ctx, workspace, repo, opts.Limit)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"workspace": workspace,
		"repo":      repo,
		"pipelines": pipelines,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(pipelines) == 0 {
			fmt.Fprintln(ios.Out, "No pipelines found.")
			return nil
		}
		for _, p := range pipelines {
			fmt.Fprintf(ios.Out, "%s\t%-12s\t%s\t%s\n", p.UUID, p.State.Name, p.Target.Ref.Name, p.State.Result.Name)
		}
		return nil
	})
}

func runPipelineView(cmd *cobra.Command, f *cmdutil.Factory, opts *viewOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	workspace, repo, host, err := resolveCloudRepo(cmd, f, opts.Workspace, opts.Repo)
	if err != nil {
		return err
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	pipeline, err := client.GetPipeline(ctx, workspace, repo, opts.UUID)
	if err != nil {
		return err
	}

	steps, err := client.ListPipelineSteps(ctx, workspace, repo, opts.UUID)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"pipeline": pipeline,
		"steps":    steps,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		fmt.Fprintf(ios.Out, "%s\t%s\t%s\n", pipeline.UUID, pipeline.State.Name, pipeline.State.Result.Name)
		if len(steps) > 0 {
			fmt.Fprintln(ios.Out, "Steps:")
			for _, step := range steps {
				fmt.Fprintf(ios.Out, "  %s\t%s\t%s\n", step.UUID, step.Name, step.Result.Name)
			}
		}
		return nil
	})
}

func runPipelineLogs(cmd *cobra.Command, f *cmdutil.Factory, opts *logsOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	workspace, repo, host, err := resolveCloudRepo(cmd, f, opts.Workspace, opts.Repo)
	if err != nil {
		return err
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	stepID := opts.Step
	if stepID == "" {
		steps, err := client.ListPipelineSteps(ctx, workspace, repo, opts.UUID)
		if err != nil {
			return err
		}
		if len(steps) == 0 {
			return fmt.Errorf("pipeline %s has no steps yet", opts.UUID)
		}
		stepID = steps[len(steps)-1].UUID
	}

	logs, err := client.GetPipelineLogs(ctx, workspace, repo, opts.UUID, stepID)
	if err != nil {
		return err
	}

	if _, err := ios.Out.Write(logs); err != nil {
		return err
	}
	return nil
}

func resolveCloudRepo(cmd *cobra.Command, f *cmdutil.Factory, workspaceOverride, repoOverride string) (string, string, *config.Host, error) {
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return "", "", nil, err
	}
	if host.Kind != "cloud" {
		return "", "", nil, fmt.Errorf("command supports Bitbucket Cloud contexts only")
	}

	workspace := firstNonEmpty(workspaceOverride, ctxCfg.Workspace)
	repo := firstNonEmpty(repoOverride, ctxCfg.DefaultRepo)
	if workspace == "" || repo == "" {
		return "", "", nil, fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
	}

	return workspace, repo, host, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

```

File: pkg/cmd/pr/automerge.go (1782 tokens)
```
package pr

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type autoMergeOptions struct {
	Project     string
	Repo        string
	ID          int
	Strategy    string
	Message     string
	CloseSource bool
}

func newAutoMergeCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auto-merge",
		Short: "Manage pull request auto-merge",
	}

	cmd.AddCommand(newAutoMergeEnableCmd(f))
	cmd.AddCommand(newAutoMergeDisableCmd(f))
	cmd.AddCommand(newAutoMergeStatusCmd(f))

	return cmd
}

func newAutoMergeEnableCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &autoMergeOptions{CloseSource: true}
	cmd := &cobra.Command{
		Use:   "enable <id>",
		Short: "Enable auto-merge for a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runAutoMergeEnable(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Strategy, "strategy", "", "Merge strategy ID (leave empty for default)")
	cmd.Flags().StringVar(&opts.Message, "message", "", "Custom merge commit message")
	cmd.Flags().BoolVar(&opts.CloseSource, "close-source", opts.CloseSource, "Close source branch when auto-merge completes")

	return cmd
}

func newAutoMergeDisableCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &autoMergeOptions{}
	cmd := &cobra.Command{
		Use:   "disable <id>",
		Short: "Disable auto-merge",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runAutoMergeDisable(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func newAutoMergeStatusCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &autoMergeOptions{}
	cmd := &cobra.Command{
		Use:   "status <id>",
		Short: "Show auto-merge configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runAutoMergeStatus(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func runAutoMergeEnable(cmd *cobra.Command, f *cmdutil.Factory, opts *autoMergeOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("auto-merge enable currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	settings := bbdc.AutoMergeSettings{
		StrategyID:    opts.Strategy,
		CommitMessage: opts.Message,
		CloseSource:   opts.CloseSource,
	}

	if err := client.EnableAutoMerge(ctx, projectKey, repoSlug, opts.ID, settings); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Auto-merge enabled for pull request #%d\n", opts.ID)
	return nil
}

func runAutoMergeDisable(cmd *cobra.Command, f *cmdutil.Factory, opts *autoMergeOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("auto-merge disable currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.DisableAutoMerge(ctx, projectKey, repoSlug, opts.ID); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Auto-merge disabled for pull request #%d\n", opts.ID)
	return nil
}

func runAutoMergeStatus(cmd *cobra.Command, f *cmdutil.Factory, opts *autoMergeOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("auto-merge status currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	settings, err := client.GetAutoMerge(ctx, projectKey, repoSlug, opts.ID)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"project":      projectKey,
		"repo":         repoSlug,
		"pull_request": opts.ID,
		"auto_merge":   settings,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if settings == nil || !settings.Enabled {
			fmt.Fprintf(ios.Out, "Auto-merge disabled for pull request #%d\n", opts.ID)
			return nil
		}
		fmt.Fprintf(ios.Out, "Auto-merge enabled using strategy %s\n", settings.StrategyID)
		if settings.CommitMessage != "" {
			fmt.Fprintf(ios.Out, "Message: %s\n", settings.CommitMessage)
		}
		fmt.Fprintf(ios.Out, "Close source branch: %t\n", settings.CloseSource)
		return nil
	})
}

```

File: pkg/cmd/pr/pr.go (6469 tokens)
```
package pr

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdPR returns the pull request command tree.
func NewCmdPR(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pr",
		Short: "Manage pull requests",
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newViewCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newCheckoutCmd(f))
	cmd.AddCommand(newDiffCmd(f))
	cmd.AddCommand(newApproveCmd(f))
	cmd.AddCommand(newMergeCmd(f))
	cmd.AddCommand(newCommentCmd(f))
	cmd.AddCommand(newReviewerGroupCmd(f))
	cmd.AddCommand(newAutoMergeCmd(f))
	cmd.AddCommand(newTaskCmd(f))
	cmd.AddCommand(newReactionCmd(f))
	cmd.AddCommand(newSuggestionCmd(f))

	return cmd
}

type listOptions struct {
	Project   string
	Workspace string
	Repo      string
	State     string
	Limit     int
	Mine      bool
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{State: "OPEN", Limit: 20}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List pull requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.State, "state", opts.State, "Filter by state (OPEN, MERGED, DECLINED)")
	cmd.Flags().IntVar(&opts.Limit, "limit", opts.Limit, "Maximum pull requests to list (0 for all)")
	cmd.Flags().BoolVar(&opts.Mine, "mine", false, "Show pull requests authored by the authenticated user")

	return cmd
}

func runList(cmd *cobra.Command, f *cmdutil.Factory, opts *listOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	switch host.Kind {
	case "dc":
		projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		prs, err := client.ListPullRequests(ctx, projectKey, repoSlug, opts.State, opts.Limit)
		if err != nil {
			return err
		}

		if opts.Mine && host.Username != "" {
			filtered := prs[:0]
			current := strings.ToLower(host.Username)
			for _, pr := range prs {
				author := strings.ToLower(firstNonEmpty(pr.Author.User.Name, pr.Author.User.Slug))
				if author == current {
					filtered = append(filtered, pr)
				}
			}
			prs = filtered
		}

		payload := map[string]any{
			"project":       projectKey,
			"repo":          repoSlug,
			"pull_requests": prs,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(prs) == 0 {
				fmt.Fprintf(ios.Out, "No pull requests (%s).\n", strings.ToUpper(opts.State))
				return nil
			}

			for _, pr := range prs {
				author := firstNonEmpty(pr.Author.User.FullName, pr.Author.User.Name)
				fmt.Fprintf(ios.Out, "#%d\t%-8s\t%s\n", pr.ID, pr.State, pr.Title)
				fmt.Fprintf(ios.Out, "    %s -> %s\tby %s\n", pr.FromRef.DisplayID, pr.ToRef.DisplayID, author)
			}
			return nil
		})

	case "cloud":
		workspace := firstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		mine := ""
		if opts.Mine && host.Username != "" {
			mine = host.Username
		}

		prs, err := client.ListPullRequests(ctx, workspace, repoSlug, bbcloud.PullRequestListOptions{
			State: opts.State,
			Limit: opts.Limit,
			Mine:  mine,
		})
		if err != nil {
			return err
		}

		payload := map[string]any{
			"workspace":     workspace,
			"repo":          repoSlug,
			"pull_requests": prs,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(prs) == 0 {
				fmt.Fprintf(ios.Out, "No pull requests (%s).\n", strings.ToUpper(opts.State))
				return nil
			}

			for _, pr := range prs {
				author := firstNonEmpty(pr.Author.DisplayName, pr.Author.Username)
				fmt.Fprintf(ios.Out, "#%d\t%-8s\t%s\n", pr.ID, pr.State, pr.Title)
				fmt.Fprintf(ios.Out, "    %s -> %s\tby %s\n", pr.Source.Branch.Name, pr.Destination.Branch.Name, author)
			}
			return nil
		})

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type viewOptions struct {
	Project   string
	Workspace string
	Repo      string
	ID        int
	Web       bool
}

func newViewCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &viewOptions{}
	cmd := &cobra.Command{
		Use:   "view <id>",
		Short: "Show details for a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runView(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().BoolVar(&opts.Web, "web", false, "Open the pull request in your browser")

	return cmd
}

func runView(cmd *cobra.Command, f *cmdutil.Factory, opts *viewOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	switch host.Kind {
	case "dc":
		projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		pr, err := client.GetPullRequest(ctx, projectKey, repoSlug, opts.ID)
		if err != nil {
			return err
		}

		payload := map[string]any{
			"project":      projectKey,
			"repo":         repoSlug,
			"pull_request": pr,
		}

		if opts.Web {
			if link := firstPRLinkDC(pr, "self"); link != "" {
				if err := f.BrowserOpener().Open(link); err != nil {
					return fmt.Errorf("open browser: %w", err)
				}
			} else {
				return fmt.Errorf("pull request does not expose a web URL")
			}
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			fmt.Fprintf(ios.Out, "Pull Request #%d: %s\n", pr.ID, pr.Title)
			fmt.Fprintf(ios.Out, "State: %s\n", pr.State)
			fmt.Fprintf(ios.Out, "Author: %s\n", firstNonEmpty(pr.Author.User.FullName, pr.Author.User.Name))
			fmt.Fprintf(ios.Out, "From: %s\nTo:   %s\n", pr.FromRef.DisplayID, pr.ToRef.DisplayID)
			if strings.TrimSpace(pr.Description) != "" {
				fmt.Fprintf(ios.Out, "\n%s\n", pr.Description)
			}

			if len(pr.Reviewers) > 0 {
				fmt.Fprintln(ios.Out, "\nReviewers:")
				for _, reviewer := range pr.Reviewers {
					fmt.Fprintf(ios.Out, "  %s\n", firstNonEmpty(reviewer.User.FullName, reviewer.User.Name))
				}
			}
			return nil
		})

	case "cloud":
		workspace := firstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		pr, err := client.GetPullRequest(ctx, workspace, repoSlug, opts.ID)
		if err != nil {
			return err
		}

		payload := map[string]any{
			"workspace":    workspace,
			"repo":         repoSlug,
			"pull_request": pr,
		}

		if opts.Web {
			if link := firstPRLinkCloud(pr); link != "" {
				if err := f.BrowserOpener().Open(link); err != nil {
					return fmt.Errorf("open browser: %w", err)
				}
			} else {
				return fmt.Errorf("pull request does not expose a web URL")
			}
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			fmt.Fprintf(ios.Out, "Pull Request #%d: %s\n", pr.ID, pr.Title)
			fmt.Fprintf(ios.Out, "State: %s\n", pr.State)
			fmt.Fprintf(ios.Out, "Author: %s\n", firstNonEmpty(pr.Author.DisplayName, pr.Author.Username))
			fmt.Fprintf(ios.Out, "From: %s\nTo:   %s\n", pr.Source.Branch.Name, pr.Destination.Branch.Name)
			if strings.TrimSpace(pr.Summary.Raw) != "" {
				fmt.Fprintf(ios.Out, "\n%s\n", pr.Summary.Raw)
			}
			return nil
		})

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

func firstPRLinkDC(pr *bbdc.PullRequest, kind string) string {
	if pr == nil {
		return ""
	}
	switch kind {
	case "self":
		for _, link := range pr.Links.Self {
			if strings.TrimSpace(link.Href) != "" {
				return link.Href
			}
		}
	}
	return ""
}

func firstPRLinkCloud(pr *bbcloud.PullRequest) string {
	if pr == nil {
		return ""
	}
	if pr.Links.HTML.Href != "" {
		return pr.Links.HTML.Href
	}
	return ""
}

type createOptions struct {
	Project     string
	Workspace   string
	Repo        string
	Title       string
	Source      string
	Target      string
	Description string
	Reviewers   []string
	CloseSource bool
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new pull request",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Title, "title", "", "Pull request title (required)")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Pull request description")
	cmd.Flags().StringVar(&opts.Source, "source", "", "Source branch (required)")
	cmd.Flags().StringVar(&opts.Target, "target", "", "Target branch (required)")
	cmd.Flags().StringSliceVar(&opts.Reviewers, "reviewer", nil, "Reviewers to request (repeatable)")
	cmd.Flags().BoolVar(&opts.CloseSource, "close-source", false, "Close source branch on merge")

	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("source")
	_ = cmd.MarkFlagRequired("target")

	return cmd
}

func runCreate(cmd *cobra.Command, f *cmdutil.Factory, opts *createOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	switch host.Kind {
	case "dc":
		projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		pr, err := client.CreatePullRequest(ctx, projectKey, repoSlug, bbdc.CreatePROptions{
			Title:        opts.Title,
			Description:  opts.Description,
			SourceBranch: opts.Source,
			TargetBranch: opts.Target,
			Reviewers:    opts.Reviewers,
			CloseSource:  opts.CloseSource,
		})
		if err != nil {
			return err
		}

		fmt.Fprintf(ios.Out, "✓ Created pull request #%d\n", pr.ID)
		return nil

	case "cloud":
		workspace := firstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		pr, err := client.CreatePullRequest(ctx, workspace, repoSlug, bbcloud.CreatePullRequestInput{
			Title:       opts.Title,
			Description: opts.Description,
			Source:      opts.Source,
			Destination: opts.Target,
			CloseSource: opts.CloseSource,
			Reviewers:   opts.Reviewers,
		})
		if err != nil {
			return err
		}

		fmt.Fprintf(ios.Out, "✓ Created pull request #%d\n", pr.ID)
		return nil

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type checkoutOptions struct {
	Project string
	Repo    string
	ID      int
	Branch  string
	Remote  string
}

func newCheckoutCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &checkoutOptions{Remote: "origin"}
	cmd := &cobra.Command{
		Use:   "checkout <id>",
		Short: "Check out the pull request branch",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runCheckout(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Branch, "branch", "", "Local branch name (defaults to pr/<id>)")
	cmd.Flags().StringVar(&opts.Remote, "remote", opts.Remote, "Git remote name to fetch from")

	return cmd
}

func runCheckout(cmd *cobra.Command, f *cmdutil.Factory, opts *checkoutOptions) error {
	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("pr checkout currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	branchName := opts.Branch
	if branchName == "" {
		branchName = fmt.Sprintf("pr/%d", opts.ID)
	}

	ref := fmt.Sprintf("refs/pull-requests/%d/from", opts.ID)
	fetchArgs := []string{"fetch", opts.Remote, fmt.Sprintf("%s:%s", ref, branchName)}
	if err := runGit(cmd.Context(), fetchArgs...); err != nil {
		return err
	}

	if err := runGit(cmd.Context(), "checkout", branchName); err != nil {
		return err
	}
	return nil
}

type diffOptions struct {
	Project string
	Repo    string
	ID      int
	Stat    bool
}

func newDiffCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &diffOptions{}
	cmd := &cobra.Command{
		Use:   "diff <id>",
		Short: "Show the diff for a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runDiff(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().BoolVar(&opts.Stat, "stat", false, "Show diff statistics instead of full patch")

	return cmd
}

func runDiff(cmd *cobra.Command, f *cmdutil.Factory, opts *diffOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("pr diff currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	if opts.Stat {
		stat, err := client.PullRequestDiffStat(ctx, projectKey, repoSlug, opts.ID)
		if err != nil {
			return err
		}
		payload := map[string]any{
			"project":      projectKey,
			"repo":         repoSlug,
			"pull_request": opts.ID,
			"stats":        stat,
		}
		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			fmt.Fprintf(ios.Out, "Files: %d\nAdditions: %d\nDeletions: %d\n", stat.Files, stat.Additions, stat.Deletions)
			return nil
		})
	}

	pager := f.PagerManager()
	if pager.Enabled() {
		w, err := pager.Start()
		if err == nil {
			defer pager.Stop()
			return client.PullRequestDiff(ctx, projectKey, repoSlug, opts.ID, w)
		}
	}

	return client.PullRequestDiff(ctx, projectKey, repoSlug, opts.ID, ios.Out)
}

func newApproveCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "approve <id>",
		Short: "Approve a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			return runApprove(cmd, f, id)
		},
	}
	return cmd
}

func runApprove(cmd *cobra.Command, f *cmdutil.Factory, id int) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("pr approve currently supports Data Center contexts only")
	}

	projectKey := ctxCfg.ProjectKey
	repoSlug := ctxCfg.DefaultRepo
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.ApprovePullRequest(ctx, projectKey, repoSlug, id); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Approved pull request #%d\n", id)
	return nil
}

type mergeOptions struct {
	Message     string
	Strategy    string
	CloseSource bool
	Project     string
	Repo        string
}

func newMergeCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &mergeOptions{}
	cmd := &cobra.Command{
		Use:   "merge <id>",
		Short: "Merge a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			return runMerge(cmd, f, id, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Message, "message", "", "Merge commit message override")
	cmd.Flags().StringVar(&opts.Strategy, "strategy", "", "Merge strategy ID (e.g., fast-forward)")
	cmd.Flags().BoolVar(&opts.CloseSource, "close-source", true, "Close source branch on merge")

	return cmd
}

func runMerge(cmd *cobra.Command, f *cmdutil.Factory, id int, opts *mergeOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("pr merge currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	pr, err := client.GetPullRequest(ctx, projectKey, repoSlug, id)
	if err != nil {
		return err
	}

	if err := client.MergePullRequest(ctx, projectKey, repoSlug, id, pr.Version, bbdc.MergePROptions{
		Message:           opts.Message,
		Strategy:          opts.Strategy,
		CloseSourceBranch: opts.CloseSource,
	}); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Merged pull request #%d\n", id)
	return nil
}

type commentOptions struct {
	Project string
	Repo    string
	Text    string
}

func newCommentCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &commentOptions{}
	cmd := &cobra.Command{
		Use:   "comment <id> --text <message>",
		Short: "Comment on a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			return runComment(cmd, f, id, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Text, "text", "", "Comment text")
	_ = cmd.MarkFlagRequired("text")

	return cmd
}

func runComment(cmd *cobra.Command, f *cmdutil.Factory, id int, opts *commentOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("pr comment currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	if err := client.CommentPullRequest(ctx, projectKey, repoSlug, id, opts.Text); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Commented on pull request #%d\n", id)
	return nil
}

func runGit(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

```

File: pkg/cmd/pr/reactions.go (1790 tokens)
```
package pr

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type reactionOptions struct {
	Project string
	Repo    string
	ID      int
	Comment int
	Emoji   string
}

func newReactionCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reaction",
		Short: "Manage comment reactions",
	}

	cmd.AddCommand(newReactionListCmd(f))
	cmd.AddCommand(newReactionAddCmd(f))
	cmd.AddCommand(newReactionRemoveCmd(f))

	return cmd
}

func newReactionListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &reactionOptions{}
	cmd := &cobra.Command{
		Use:   "list <id> <comment-id>",
		Short: "List comment reactions",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			commentID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid comment id %q", args[1])
			}
			opts.ID = prID
			opts.Comment = commentID
			return runReactionList(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func newReactionAddCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &reactionOptions{}
	cmd := &cobra.Command{
		Use:   "add <id> <comment-id>",
		Short: "Add a reaction to a comment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			commentID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid comment id %q", args[1])
			}
			opts.ID = prID
			opts.Comment = commentID
			return runReactionAdd(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Emoji, "emoji", "", "Emoji to add (e.g. :thumbsup:)")
	_ = cmd.MarkFlagRequired("emoji")
	return cmd
}

func newReactionRemoveCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &reactionOptions{}
	cmd := &cobra.Command{
		Use:   "remove <id> <comment-id>",
		Short: "Remove a reaction",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			commentID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid comment id %q", args[1])
			}
			opts.ID = prID
			opts.Comment = commentID
			return runReactionRemove(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Emoji, "emoji", "", "Emoji to remove")
	_ = cmd.MarkFlagRequired("emoji")
	return cmd
}

func runReactionList(cmd *cobra.Command, f *cmdutil.Factory, opts *reactionOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("reaction list currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	reactions, err := client.ListCommentReactions(ctx, projectKey, repoSlug, opts.ID, opts.Comment)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"project":   projectKey,
		"repo":      repoSlug,
		"reactions": reactions,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(reactions) == 0 {
			fmt.Fprintf(ios.Out, "No reactions for comment %d\n", opts.Comment)
			return nil
		}
		for _, reaction := range reactions {
			fmt.Fprintf(ios.Out, "%s x%d\n", reaction.Emoji, reaction.Count)
		}
		return nil
	})
}

func runReactionAdd(cmd *cobra.Command, f *cmdutil.Factory, opts *reactionOptions) error {
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("reaction add currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	if err := client.AddCommentReaction(ctx, projectKey, repoSlug, opts.ID, opts.Comment, opts.Emoji); err != nil {
		return err
	}

	ios, err := f.Streams()
	if err != nil {
		return err
	}
	fmt.Fprintf(ios.Out, "✓ Added %s to comment %d\n", opts.Emoji, opts.Comment)
	return nil
}

func runReactionRemove(cmd *cobra.Command, f *cmdutil.Factory, opts *reactionOptions) error {
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("reaction remove currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	if err := client.RemoveCommentReaction(ctx, projectKey, repoSlug, opts.ID, opts.Comment, opts.Emoji); err != nil {
		return err
	}

	ios, err := f.Streams()
	if err != nil {
		return err
	}
	fmt.Fprintf(ios.Out, "✓ Removed %s from comment %d\n", opts.Emoji, opts.Comment)
	return nil
}

```

File: pkg/cmd/pr/reviewergroup.go (1434 tokens)
```
package pr

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

func newReviewerGroupCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reviewer-group",
		Short: "Manage default reviewer groups",
	}

	cmd.AddCommand(newReviewerGroupListCmd(f))
	cmd.AddCommand(newReviewerGroupAddCmd(f))
	cmd.AddCommand(newReviewerGroupRemoveCmd(f))

	return cmd
}

type reviewerGroupOptions struct {
	Project string
	Repo    string
	Name    string
}

func newReviewerGroupListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &reviewerGroupOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List default reviewer groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReviewerGroupList(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func newReviewerGroupAddCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &reviewerGroupOptions{}
	cmd := &cobra.Command{
		Use:   "add <group>",
		Short: "Add a default reviewer group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			return runReviewerGroupAdd(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func newReviewerGroupRemoveCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &reviewerGroupOptions{}
	cmd := &cobra.Command{
		Use:   "remove <group>",
		Short: "Remove a default reviewer group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			return runReviewerGroupRemove(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func runReviewerGroupList(cmd *cobra.Command, f *cmdutil.Factory, opts *reviewerGroupOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("reviewer-group list currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	groups, err := client.ListReviewerGroups(ctx, projectKey, repoSlug)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"project":         projectKey,
		"repo":            repoSlug,
		"reviewer_groups": groups,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(groups) == 0 {
			fmt.Fprintln(ios.Out, "No reviewer groups configured.")
			return nil
		}
		for _, g := range groups {
			fmt.Fprintf(ios.Out, "%s\n", g.Name)
		}
		return nil
	})
}

func runReviewerGroupAdd(cmd *cobra.Command, f *cmdutil.Factory, opts *reviewerGroupOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("reviewer-group add currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.AddReviewerGroup(ctx, projectKey, repoSlug, opts.Name); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Added reviewer group %s\n", opts.Name)
	return nil
}

func runReviewerGroupRemove(cmd *cobra.Command, f *cmdutil.Factory, opts *reviewerGroupOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("reviewer-group remove currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.RemoveReviewerGroup(ctx, projectKey, repoSlug, opts.Name); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Removed reviewer group %s\n", opts.Name)
	return nil
}

```

File: pkg/cmd/pr/suggestions.go (770 tokens)
```
package pr

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type suggestionOptions struct {
	Project      string
	Repo         string
	ID           int
	CommentID    int
	SuggestionID int
	Preview      bool
}

func newSuggestionCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &suggestionOptions{}
	cmd := &cobra.Command{
		Use:   "suggestion <id> <comment-id> <suggestion-id>",
		Short: "Apply or preview a code suggestion",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			commentID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid comment id %q", args[1])
			}
			suggestionID, err := strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid suggestion id %q", args[2])
			}
			opts.ID = prID
			opts.CommentID = commentID
			opts.SuggestionID = suggestionID
			return runSuggestion(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().BoolVar(&opts.Preview, "preview", false, "Preview suggestion without applying")

	return cmd
}

func runSuggestion(cmd *cobra.Command, f *cmdutil.Factory, opts *suggestionOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("suggestions currently support Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if opts.Preview {
		suggestion, err := client.SuggestionPreview(ctx, projectKey, repoSlug, opts.ID, opts.CommentID, opts.SuggestionID)
		if err != nil {
			return err
		}

		payload := map[string]any{
			"project":    projectKey,
			"repo":       repoSlug,
			"suggestion": suggestion,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			fmt.Fprintf(ios.Out, "%s\n", suggestion.Text)
			return nil
		})
	}

	if err := client.ApplySuggestion(ctx, projectKey, repoSlug, opts.ID, opts.CommentID, opts.SuggestionID); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Applied suggestion %d\n", opts.SuggestionID)
	return nil
}

```

File: pkg/cmd/pr/tasks.go (2033 tokens)
```
package pr

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type taskOptions struct {
	Project string
	Repo    string
	ID      int
	TaskID  int
	Text    string
}

func newTaskCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Manage pull request tasks",
	}

	cmd.AddCommand(newTaskListCmd(f))
	cmd.AddCommand(newTaskCreateCmd(f))
	cmd.AddCommand(newTaskCompleteCmd(f))
	cmd.AddCommand(newTaskReopenCmd(f))

	return cmd
}

func newTaskListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &taskOptions{}
	cmd := &cobra.Command{
		Use:   "list <id>",
		Short: "List tasks for a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runTaskList(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func newTaskCreateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &taskOptions{}
	cmd := &cobra.Command{
		Use:   "create <id>",
		Short: "Create a task on a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			opts.ID = id
			return runTaskCreate(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Text, "text", "", "Task text")
	_ = cmd.MarkFlagRequired("text")

	return cmd
}

func newTaskCompleteCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &taskOptions{}
	cmd := &cobra.Command{
		Use:   "complete <id> <task-id>",
		Short: "Complete a pull request task",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			taskID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid task id %q", args[1])
			}
			opts.ID = prID
			opts.TaskID = taskID
			return runTaskComplete(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func newTaskReopenCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &taskOptions{}
	cmd := &cobra.Command{
		Use:   "reopen <id> <task-id>",
		Short: "Reopen a resolved task",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			taskID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid task id %q", args[1])
			}
			opts.ID = prID
			opts.TaskID = taskID
			return runTaskReopen(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func runTaskList(cmd *cobra.Command, f *cmdutil.Factory, opts *taskOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("task list currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	tasks, err := client.ListPullRequestTasks(ctx, projectKey, repoSlug, opts.ID)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"project": projectKey,
		"repo":    repoSlug,
		"tasks":   tasks,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(tasks) == 0 {
			fmt.Fprintf(ios.Out, "No tasks on pull request #%d\n", opts.ID)
			return nil
		}
		for _, task := range tasks {
			fmt.Fprintf(ios.Out, "[%s] %d %s\n", strings.ToUpper(task.State), task.ID, task.Text)
		}
		return nil
	})
}

func runTaskCreate(cmd *cobra.Command, f *cmdutil.Factory, opts *taskOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("task create currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	task, err := client.CreatePullRequestTask(ctx, projectKey, repoSlug, opts.ID, opts.Text)
	if err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Created task %d\n", task.ID)
	return nil
}

func runTaskComplete(cmd *cobra.Command, f *cmdutil.Factory, opts *taskOptions) error {
	return toggleTaskState(cmd, f, opts, true)
}

func runTaskReopen(cmd *cobra.Command, f *cmdutil.Factory, opts *taskOptions) error {
	return toggleTaskState(cmd, f, opts, false)
}

func toggleTaskState(cmd *cobra.Command, f *cmdutil.Factory, opts *taskOptions, resolve bool) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("task management currently supports Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if resolve {
		if err := client.CompletePullRequestTask(ctx, projectKey, repoSlug, opts.ID, opts.TaskID); err != nil {
			return err
		}
		fmt.Fprintf(ios.Out, "✓ Completed task %d\n", opts.TaskID)
		return nil
	}

	if err := client.ReopenPullRequestTask(ctx, projectKey, repoSlug, opts.ID, opts.TaskID); err != nil {
		return err
	}
	fmt.Fprintf(ios.Out, "✓ Reopened task %d\n", opts.TaskID)
	return nil
}

```

File: pkg/cmd/repo/repo.go (5736 tokens)
```
package repo

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdRepo wires repository subcommands.
func NewCmdRepo(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Work with Bitbucket repositories",
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newViewCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newCloneCmd(f))
	cmd.AddCommand(newBrowseCmd(f))

	return cmd
}

type listOptions struct {
	Project   string
	Workspace string
	Limit     int
}

type createOptions struct {
	Project       string
	Workspace     string
	CloudProject  string
	Description   string
	Public        bool
	Forkable      bool
	DefaultBranch string
	SCM           string
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{
		Limit: 30,
	}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List repositories within the active scope",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().IntVar(&opts.Limit, "limit", opts.Limit, "Maximum repositories to display (0 for all)")
	return cmd
}

func runList(cmd *cobra.Command, f *cmdutil.Factory, opts *listOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	switch host.Kind {
	case "dc":
		projectKey := strings.TrimSpace(opts.Project)
		if projectKey == "" {
			projectKey = ctxCfg.ProjectKey
		}
		if projectKey == "" {
			return fmt.Errorf("project key required; set with --project or configure the context default")
		}
		projectKey = strings.ToUpper(projectKey)

		client, err := bbdc.New(bbdc.Options{
			BaseURL:  host.BaseURL,
			Username: host.Username,
			Token:    host.Token,
		})
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		repos, err := client.ListRepositories(ctx, projectKey, opts.Limit)
		if err != nil {
			return err
		}

		type repoSummary struct {
			Project string   `json:"project"`
			Slug    string   `json:"slug"`
			Name    string   `json:"name"`
			ID      int      `json:"id"`
			WebURL  string   `json:"web_url,omitempty"`
			Clone   []string `json:"clone_urls,omitempty"`
		}

		var summaries []repoSummary
		for _, repo := range repos {
			summaries = append(summaries, repoSummary{
				Project: repo.Project.Key,
				Slug:    repo.Slug,
				Name:    repo.Name,
				ID:      repo.ID,
				WebURL:  firstLinkDC(repo, "web"),
				Clone:   cloneLinksDC(repo),
			})
		}

		payload := struct {
			Project string        `json:"project"`
			Repos   []repoSummary `json:"repositories"`
		}{
			Project: projectKey,
			Repos:   summaries,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(summaries) == 0 {
				fmt.Fprintf(ios.Out, "No repositories found in project %s.\n", projectKey)
				return nil
			}

			for _, r := range summaries {
				fmt.Fprintf(ios.Out, "%s/%s\t%s\n", r.Project, r.Slug, r.Name)
				if r.WebURL != "" {
					fmt.Fprintf(ios.Out, "    web:   %s\n", r.WebURL)
				}
				if len(r.Clone) > 0 {
					fmt.Fprintf(ios.Out, "    clone: %s\n", strings.Join(r.Clone, ", "))
				}
			}
			return nil
		})

	case "cloud":
		workspace := strings.TrimSpace(opts.Workspace)
		if workspace == "" {
			workspace = ctxCfg.Workspace
		}
		if workspace == "" {
			return fmt.Errorf("workspace required; set with --workspace or configure the context default")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		repos, err := client.ListRepositories(ctx, workspace, opts.Limit)
		if err != nil {
			return err
		}

		type repoSummary struct {
			Workspace string   `json:"workspace"`
			Slug      string   `json:"slug"`
			Name      string   `json:"name"`
			UUID      string   `json:"uuid"`
			WebURL    string   `json:"web_url,omitempty"`
			Clone     []string `json:"clone_urls,omitempty"`
		}

		var summaries []repoSummary
		for _, repo := range repos {
			summaries = append(summaries, repoSummary{
				Workspace: workspace,
				Slug:      repo.Slug,
				Name:      repo.Name,
				UUID:      strings.Trim(repo.UUID, "{}"),
				WebURL:    firstLinkCloud(repo),
				Clone:     cloneLinksCloud(repo),
			})
		}

		payload := struct {
			Workspace string        `json:"workspace"`
			Repos     []repoSummary `json:"repositories"`
		}{
			Workspace: workspace,
			Repos:     summaries,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(summaries) == 0 {
				fmt.Fprintf(ios.Out, "No repositories found in workspace %s.\n", workspace)
				return nil
			}

			for _, r := range summaries {
				fmt.Fprintf(ios.Out, "%s/%s\t%s\n", r.Workspace, r.Slug, r.Name)
				if r.WebURL != "" {
					fmt.Fprintf(ios.Out, "    web:   %s\n", r.WebURL)
				}
				if len(r.Clone) > 0 {
					fmt.Fprintf(ios.Out, "    clone: %s\n", strings.Join(r.Clone, ", "))
				}
			}
			return nil
		})

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type viewOptions struct {
	Project   string
	Workspace string
	Repo      string
}

type cloneOptions struct {
	Project   string
	Workspace string
	Repo      string
	UseSSH    bool
	Dest      string
}

func newViewCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &viewOptions{}
	cmd := &cobra.Command{
		Use:   "view [repository]",
		Short: "Display details for a repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Repo = args[0]
			}
			return runView(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func runView(cmd *cobra.Command, f *cmdutil.Factory, opts *viewOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	switch host.Kind {
	case "dc":
		projectKey := strings.TrimSpace(opts.Project)
		if projectKey == "" {
			projectKey = ctxCfg.ProjectKey
		}
		if projectKey == "" {
			return fmt.Errorf("project key required; set with --project or configure the context default")
		}
		projectKey = strings.ToUpper(projectKey)

		repoSlug := strings.TrimSpace(opts.Repo)
		if repoSlug == "" {
			repoSlug = ctxCfg.DefaultRepo
		}
		if repoSlug == "" {
			return fmt.Errorf("repository slug required; pass --repo or set the context default")
		}

		client, err := bbdc.New(bbdc.Options{
			BaseURL:  host.BaseURL,
			Username: host.Username,
			Token:    host.Token,
		})
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		repo, err := client.GetRepository(ctx, projectKey, repoSlug)
		if err != nil {
			return err
		}

		type repoDetails struct {
			Project string   `json:"project"`
			Slug    string   `json:"slug"`
			Name    string   `json:"name"`
			ID      int      `json:"id"`
			WebURL  string   `json:"web_url,omitempty"`
			Clone   []string `json:"clone_urls,omitempty"`
		}

		details := repoDetails{
			Project: repo.Project.Key,
			Slug:    repo.Slug,
			Name:    repo.Name,
			ID:      repo.ID,
			WebURL:  firstLinkDC(*repo, "web"),
			Clone:   cloneLinksDC(*repo),
		}

		return cmdutil.WriteOutput(cmd, ios.Out, details, func() error {
			fmt.Fprintf(ios.Out, "%s/%s (%d)\n", details.Project, details.Slug, details.ID)
			fmt.Fprintf(ios.Out, "Name: %s\n", details.Name)
			if details.WebURL != "" {
				fmt.Fprintf(ios.Out, "Web:  %s\n", details.WebURL)
			}
			if len(details.Clone) > 0 {
				for _, url := range details.Clone {
					fmt.Fprintf(ios.Out, "Clone: %s\n", url)
				}
			}
			return nil
		})

	case "cloud":
		workspace := strings.TrimSpace(opts.Workspace)
		if workspace == "" {
			workspace = ctxCfg.Workspace
		}
		if workspace == "" {
			return fmt.Errorf("workspace required; set with --workspace or configure the context default")
		}

		repoSlug := strings.TrimSpace(opts.Repo)
		if repoSlug == "" {
			repoSlug = ctxCfg.DefaultRepo
		}
		if repoSlug == "" {
			return fmt.Errorf("repository slug required; pass --repo or set the context default")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		repo, err := client.GetRepository(ctx, workspace, repoSlug)
		if err != nil {
			return err
		}

		type repoDetails struct {
			Workspace string   `json:"workspace"`
			Slug      string   `json:"slug"`
			Name      string   `json:"name"`
			UUID      string   `json:"uuid"`
			WebURL    string   `json:"web_url,omitempty"`
			Clone     []string `json:"clone_urls,omitempty"`
		}

		details := repoDetails{
			Workspace: workspace,
			Slug:      repo.Slug,
			Name:      repo.Name,
			UUID:      strings.Trim(repo.UUID, "{}"),
			WebURL:    firstLinkCloud(*repo),
			Clone:     cloneLinksCloud(*repo),
		}

		return cmdutil.WriteOutput(cmd, ios.Out, details, func() error {
			fmt.Fprintf(ios.Out, "%s/%s (%s)\n", details.Workspace, details.Slug, details.UUID)
			fmt.Fprintf(ios.Out, "Name: %s\n", details.Name)
			if details.WebURL != "" {
				fmt.Fprintf(ios.Out, "Web:  %s\n", details.WebURL)
			}
			if len(details.Clone) > 0 {
				for _, url := range details.Clone {
					fmt.Fprintf(ios.Out, "Clone: %s\n", url)
				}
			}
			return nil
		})

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

func runClone(cmd *cobra.Command, f *cmdutil.Factory, opts *cloneOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	switch host.Kind {
	case "dc":
		projectKey := strings.TrimSpace(opts.Project)
		if projectKey == "" {
			projectKey = ctxCfg.ProjectKey
		}
		if projectKey == "" {
			return fmt.Errorf("project key required; set with --project or configure the context default")
		}

		repoSlug := strings.TrimSpace(opts.Repo)
		if repoSlug == "" {
			repoSlug = ctxCfg.DefaultRepo
		}
		if repoSlug == "" {
			return fmt.Errorf("repository slug required; pass argument or set the context default")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		repo, err := client.GetRepository(ctx, projectKey, repoSlug)
		if err != nil {
			return err
		}

		cloneURL, err := selectCloneURLDC(*repo, opts.UseSSH)
		if err != nil {
			return err
		}

		return runGitClone(cmd, ios.Out, ios.ErrOut, ios.In, cloneURL, opts.Dest)

	case "cloud":
		workspace := strings.TrimSpace(opts.Workspace)
		if workspace == "" {
			workspace = ctxCfg.Workspace
		}
		if workspace == "" {
			return fmt.Errorf("workspace required; set with --workspace or configure the context default")
		}

		repoSlug := strings.TrimSpace(opts.Repo)
		if repoSlug == "" {
			repoSlug = ctxCfg.DefaultRepo
		}
		if repoSlug == "" {
			return fmt.Errorf("repository slug required; pass argument or set the context default")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		repo, err := client.GetRepository(ctx, workspace, repoSlug)
		if err != nil {
			return err
		}

		cloneURL, err := selectCloneURLCloud(*repo, opts.UseSSH)
		if err != nil {
			return err
		}

		return runGitClone(cmd, ios.Out, ios.ErrOut, ios.In, cloneURL, opts.Dest)

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

func runBrowse(cmd *cobra.Command, f *cmdutil.Factory) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	switch host.Kind {
	case "dc":
		projectKey := ctxCfg.ProjectKey
		repoSlug := ctxCfg.DefaultRepo
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must define project and default repo")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()
		repo, err := client.GetRepository(ctx, projectKey, repoSlug)
		if err != nil {
			return err
		}

		if link := firstLinkDC(*repo, "web"); link != "" {
			fmt.Fprintln(ios.Out, link)
			return nil
		}

		return fmt.Errorf("repository does not expose a web URL")

	case "cloud":
		workspace := ctxCfg.Workspace
		repoSlug := ctxCfg.DefaultRepo
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must define workspace and default repo")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()
		repo, err := client.GetRepository(ctx, workspace, repoSlug)
		if err != nil {
			return err
		}

		if link := firstLinkCloud(*repo); link != "" {
			fmt.Fprintln(ios.Out, link)
			return nil
		}

		return fmt.Errorf("repository does not expose a web URL")

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var opts createOptions

	cmd := &cobra.Command{
		Use:   "create <repository>",
		Short: "Create a new repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoSlug := args[0]
			return runCreate(cmd, f, repoSlug, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.CloudProject, "cloud-project", "", "Bitbucket Cloud project key")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Repository description")
	cmd.Flags().BoolVar(&opts.Public, "public", false, "Create repository as public")
	cmd.Flags().BoolVar(&opts.Forkable, "forkable", false, "Allow forking of the repository")
	cmd.Flags().StringVar(&opts.DefaultBranch, "default-branch", "", "Default branch to set after creation")
	cmd.Flags().StringVar(&opts.SCM, "scm", "git", "SCM type (git)")

	return cmd
}

func runCreate(cmd *cobra.Command, f *cmdutil.Factory, slug string, opts createOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	switch host.Kind {
	case "dc":
		projectKey := strings.TrimSpace(opts.Project)
		if projectKey == "" {
			projectKey = ctxCfg.ProjectKey
		}
		if projectKey == "" {
			return fmt.Errorf("project key required; set with --project or configure the context default")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		input := bbdc.CreateRepositoryInput{
			Name:          slug,
			SCMID:         opts.SCM,
			Description:   opts.Description,
			Public:        opts.Public,
			Forkable:      opts.Forkable,
			DefaultBranch: opts.DefaultBranch,
		}

		repo, err := client.CreateRepository(ctx, projectKey, input)
		if err != nil {
			return err
		}

		fmt.Fprintf(ios.Out, "✓ Created %s/%s\n", repo.Project.Key, repo.Slug)
		if repo.DefaultBranch != "" {
			fmt.Fprintf(ios.Out, "  default branch: %s\n", repo.DefaultBranch)
		}
		for _, clone := range cloneLinksDC(*repo) {
			fmt.Fprintf(ios.Out, "  clone: %s\n", clone)
		}
		return nil

	case "cloud":
		workspace := strings.TrimSpace(opts.Workspace)
		if workspace == "" {
			workspace = ctxCfg.Workspace
		}
		if workspace == "" {
			return fmt.Errorf("workspace required; set with --workspace or configure the context default")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		input := bbcloud.CreateRepositoryInput{
			Slug:        slug,
			Name:        slug,
			Description: opts.Description,
			IsPrivate:   !opts.Public,
			ProjectKey:  strings.TrimSpace(opts.CloudProject),
		}

		repo, err := client.CreateRepository(ctx, workspace, input)
		if err != nil {
			return err
		}

		fmt.Fprintf(ios.Out, "✓ Created %s/%s\n", workspace, repo.Slug)
		for _, clone := range cloneLinksCloud(*repo) {
			fmt.Fprintf(ios.Out, "  clone: %s\n", clone)
		}
		return nil

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

func newCloneCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &cloneOptions{}
	cmd := &cobra.Command{
		Use:   "clone <repository>",
		Short: "Clone a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Repo = args[0]
			return runClone(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().BoolVar(&opts.UseSSH, "ssh", false, "Use SSH clone URL")
	cmd.Flags().StringVar(&opts.Dest, "dest", "", "Destination directory")

	return cmd
}

func newBrowseCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Print the repository web URL",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBrowse(cmd, f)
		},
	}
	return cmd
}

func firstLinkDC(repo bbdc.Repository, kind string) string {
	switch kind {
	case "web":
		if len(repo.Links.Web) > 0 {
			return repo.Links.Web[0].Href
		}
		if len(repo.Links.Self) > 0 {
			return repo.Links.Self[0].Href
		}
	}
	return ""
}

func cloneLinksDC(repo bbdc.Repository) []string {
	var urls []string
	for _, link := range repo.Links.Clone {
		if strings.TrimSpace(link.Href) == "" {
			continue
		}
		urls = append(urls, fmt.Sprintf("%s (%s)", link.Href, link.Name))
	}
	return urls
}

func firstLinkCloud(repo bbcloud.Repository) string {
	if repo.Links.HTML.Href != "" {
		return repo.Links.HTML.Href
	}
	for _, c := range repo.Links.Clone {
		if strings.EqualFold(c.Name, "https") {
			return c.Href
		}
	}
	return ""
}

func cloneLinksCloud(repo bbcloud.Repository) []string {
	var urls []string
	for _, link := range repo.Links.Clone {
		if strings.TrimSpace(link.Href) == "" {
			continue
		}
		urls = append(urls, fmt.Sprintf("%s (%s)", link.Href, link.Name))
	}
	return urls
}

func selectCloneURLDC(repo bbdc.Repository, useSSH bool) (string, error) {
	desired := "http"
	if useSSH {
		desired = "ssh"
	}
	for _, link := range repo.Links.Clone {
		if strings.EqualFold(link.Name, desired) {
			return link.Href, nil
		}
	}
	return "", fmt.Errorf("no %s clone URL available", desired)
}

func selectCloneURLCloud(repo bbcloud.Repository, useSSH bool) (string, error) {
	desired := "https"
	if useSSH {
		desired = "ssh"
	}
	for _, link := range repo.Links.Clone {
		name := strings.ToLower(link.Name)
		if name == desired {
			return link.Href, nil
		}
		if desired == "https" && name == "http" {
			return link.Href, nil
		}
	}
	return "", fmt.Errorf("no %s clone URL available", desired)
}

func runGitClone(cmd *cobra.Command, out, errOut io.Writer, in io.Reader, cloneURL, dest string) error {
	args := []string{"clone", cloneURL}
	if dest != "" {
		args = append(args, dest)
	}

	gitCmd := exec.CommandContext(cmd.Context(), "git", args...)
	gitCmd.Stdout = out
	gitCmd.Stderr = errOut
	gitCmd.Stdin = in

	return gitCmd.Run()
}

```

File: pkg/cmd/root/root.go (645 tokens)
```
package root

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmd/admin"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/api"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/auth"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/branch"
	contextcmd "github.com/avivsinai/bitbucket-cli/pkg/cmd/context"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/extension"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/perms"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/pipeline"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/pr"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/repo"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/status"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/webhook"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdRoot assembles the root Cobra command using shared dependencies.
func NewCmdRoot(f *cmdutil.Factory) (*cobra.Command, error) {
	ios, err := f.Streams()
	if err != nil {
		return nil, err
	}

	root := &cobra.Command{
		Use:   f.ExecutableName,
		Short: "Bitbucket CLI with gh-style ergonomics.",
		Long: `Work seamlessly with Bitbucket Data Center and Cloud from the command line.

Common flows:
  bkt auth login https://bitbucket.example.com
  bkt pr list --mine
  bkt status pr 123 --json`,
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	root.SetContext(context.Background())

	root.PersistentFlags().StringP("context", "c", "", "Active Bitbucket context name")
	root.PersistentFlags().Bool("json", false, "Output in JSON format when supported")
	root.PersistentFlags().Bool("yaml", false, "Output in YAML format when supported")
	root.PersistentFlags().String("jq", "", "Apply a jq expression to JSON output (requires --json)")
	root.PersistentFlags().String("template", "", "Render output using Go templates")

	root.AddCommand(
		admin.NewCmdAdmin(f),
		auth.NewCmdAuth(f),
		contextcmd.NewCmdContext(f),
		repo.NewCmdRepo(f),
		pr.NewCmdPR(f),
		branch.NewCmdBranch(f),
		perms.NewCommand(f),
		webhook.NewCommand(f),
		status.NewCmdStatus(f),
		pipeline.NewCmdPipeline(f),
		api.NewCmdAPI(f),
		extension.NewCmdExtension(f),
	)

	root.Version = f.AppVersion
	root.SetIn(ios.In)
	root.SetOut(ios.Out)
	root.SetErr(ios.ErrOut)

	return root, nil
}

```

File: pkg/cmd/status/pipeline_cloud.go (833 tokens)
```
package status

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type cloudStatusOptions struct {
	Workspace string
	Repo      string
	UUID      string
}

func newCloudPipelineCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &cloudStatusOptions{}
	cmd := &cobra.Command{
		Use:   "pipeline <uuid>",
		Short: "Show Bitbucket Cloud pipeline status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.UUID = args[0]
			return runCloudPipelineStatus(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func runCloudPipelineStatus(cmd *cobra.Command, f *cmdutil.Factory, opts *cloudStatusOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	workspace, repo, host, err := resolveCloudStatusContext(cmd, f, opts.Workspace, opts.Repo)
	if err != nil {
		return err
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	pipeline, err := client.GetPipeline(ctx, workspace, repo, opts.UUID)
	if err != nil {
		return err
	}

	steps, err := client.ListPipelineSteps(ctx, workspace, repo, opts.UUID)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"pipeline": pipeline,
		"steps":    steps,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		fmt.Fprintf(ios.Out, "%s\t%s\t%s\n", pipeline.UUID, pipeline.State.Name, pipeline.State.Result.Name)
		fmt.Fprintf(ios.Out, "Ref: %s\n", pipeline.Target.Ref.Name)
		if pipeline.CreatedOn != "" {
			fmt.Fprintf(ios.Out, "Created: %s\n", pipeline.CreatedOn)
		}
		if pipeline.CompletedOn != "" {
			fmt.Fprintf(ios.Out, "Completed: %s\n", pipeline.CompletedOn)
		}
		if len(steps) > 0 {
			fmt.Fprintln(ios.Out, "Steps:")
			for _, step := range steps {
				fmt.Fprintf(ios.Out, "  %s\t%s\t%s\n", step.UUID, step.Name, step.Result.Name)
			}
		}
		return nil
	})
}

func resolveCloudStatusContext(cmd *cobra.Command, f *cmdutil.Factory, workspaceOverride, repoOverride string) (string, string, *config.Host, error) {
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return "", "", nil, err
	}
	if host.Kind != "cloud" {
		return "", "", nil, fmt.Errorf("command supports Bitbucket Cloud contexts only")
	}

	workspace := firstNonEmpty(workspaceOverride, ctxCfg.Workspace)
	repo := firstNonEmpty(repoOverride, ctxCfg.DefaultRepo)
	if workspace == "" || repo == "" {
		return "", "", nil, fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
	}

	return workspace, repo, host, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

```

File: pkg/cmd/status/ratelimit.go (580 tokens)
```
package status

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

type rateLimitOptions struct{}

func newRateLimitCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rate-limit",
		Short: "Show API rate limit telemetry for the active context",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRateLimit(cmd, f)
		},
	}
	return cmd
}

func runRateLimit(cmd *cobra.Command, f *cmdutil.Factory) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, _, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	switch host.Kind {
	case "dc":
		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}
		if err := client.Ping(ctx); err != nil {
			return err
		}
		rl := client.RateLimit()
		return renderRateLimit(cmd, ios.Out, rl)
	case "cloud":
		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}
		if err := client.Ping(ctx); err != nil {
			return err
		}
		rl := client.RateLimit()
		return renderRateLimit(cmd, ios.Out, rl)
	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

func renderRateLimit(cmd *cobra.Command, out io.Writer, rl httpx.RateLimit) error {
	payload := map[string]any{
		"limit":     rl.Limit,
		"remaining": rl.Remaining,
		"reset":     rl.Reset,
		"source":    rl.Source,
	}

	return cmdutil.WriteOutput(cmd, out, payload, func() error {
		fmt.Fprintf(out, "Limit: %d\n", rl.Limit)
		fmt.Fprintf(out, "Remaining: %d\n", rl.Remaining)
		if !rl.Reset.IsZero() {
			fmt.Fprintf(out, "Resets At: %s\n", rl.Reset.Format(time.RFC3339))
		}
		if rl.Source != "" {
			fmt.Fprintf(out, "Source: %s\n", rl.Source)
		}
		return nil
	})
}

```

File: pkg/cmd/status/status.go (1557 tokens)
```
package status

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdStatus exposes commit and PR status commands.
func NewCmdStatus(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Inspect commit and pull request statuses",
	}

	cmd.AddCommand(newCommitCmd(f))
	cmd.AddCommand(newPullRequestCmd(f))
	cmd.AddCommand(newCloudPipelineCmd(f))
	cmd.AddCommand(newRateLimitCmd(f))

	return cmd
}

func newCommitCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "commit <sha>",
		Short: "Show the build statuses for a commit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCommit(cmd, f, args[0])
		},
	}
	return cmd
}

func runCommit(cmd *cobra.Command, f *cmdutil.Factory, sha string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, _, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	if host.Kind != "dc" {
		return fmt.Errorf("status commit currently supports Data Center contexts only")
	}

	client, err := bbdc.New(bbdc.Options{
		BaseURL:  host.BaseURL,
		Username: host.Username,
		Token:    host.Token,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	statuses, err := client.CommitStatuses(ctx, sha)
	if err != nil {
		return err
	}

	return renderStatuses(cmd, f, ios.Out, sha, statuses, nil)
}

type prOptions struct {
	Project string
	Repo    string
}

func newPullRequestCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &prOptions{}
	cmd := &cobra.Command{
		Use:   "pr <id>",
		Short: "Show the build statuses for a pull request head commit",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			return runPullRequest(cmd, f, id, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func runPullRequest(cmd *cobra.Command, f *cmdutil.Factory, prID int, opts *prOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	if host.Kind != "dc" {
		return fmt.Errorf("status pr currently supports Data Center contexts only")
	}

	projectKey := strings.TrimSpace(opts.Project)
	if projectKey == "" {
		projectKey = ctxCfg.ProjectKey
	}
	if projectKey == "" {
		return fmt.Errorf("project key required; set with --project or configure the context default")
	}
	projectKey = strings.ToUpper(projectKey)

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; pass --repo or set the context default")
	}

	client, err := bbdc.New(bbdc.Options{
		BaseURL:  host.BaseURL,
		Username: host.Username,
		Token:    host.Token,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	pr, err := client.GetPullRequest(ctx, projectKey, repoSlug, prID)
	if err != nil {
		return err
	}

	statuses, err := client.CommitStatuses(ctx, pr.FromRef.LatestCommit)
	if err != nil {
		return err
	}

	info := map[string]any{
		"pull_request": map[string]any{
			"id":    pr.ID,
			"title": pr.Title,
		},
		"context": map[string]any{
			"project": projectKey,
			"repo":    repoSlug,
		},
		"commit": pr.FromRef.LatestCommit,
	}

	return renderStatuses(cmd, f, ios.Out, pr.FromRef.LatestCommit, statuses, info)
}

func renderStatuses(cmd *cobra.Command, f *cmdutil.Factory, out io.Writer, commit string, statuses []bbdc.CommitStatus, metadata map[string]any) error {
	type statusSummary struct {
		State       string `json:"state"`
		Key         string `json:"key"`
		Name        string `json:"name"`
		URL         string `json:"url,omitempty"`
		Description string `json:"description,omitempty"`
	}

	var summaries []statusSummary
	for _, s := range statuses {
		summaries = append(summaries, statusSummary{
			State:       s.State,
			Key:         s.Key,
			Name:        s.Name,
			URL:         s.URL,
			Description: s.Description,
		})
	}

	payload := map[string]any{
		"commit":   commit,
		"statuses": summaries,
	}
	for k, v := range metadata {
		payload[k] = v
	}

	return cmdutil.WriteOutput(cmd, out, payload, func() error {
		if metadata != nil {
			if pr, ok := metadata["pull_request"].(map[string]any); ok {
				fmt.Fprintf(out, "Pull request #%d: %s\n", pr["id"], pr["title"])
			}
			if ctx, ok := metadata["context"].(map[string]any); ok {
				fmt.Fprintf(out, "Project %s / Repo %s\n", ctx["project"], ctx["repo"])
			}
		}

		fmt.Fprintf(out, "Commit %s\n", commit)
		if len(summaries) == 0 {
			fmt.Fprintln(out, "No statuses reported.")
			return nil
		}

		for _, s := range summaries {
			line := fmt.Sprintf("%-10s %-20s %s", s.State, s.Key, s.Name)
			if s.Description != "" {
				line = fmt.Sprintf("%s — %s", line, s.Description)
			}
			fmt.Fprintln(out, line)
			if s.URL != "" {
				fmt.Fprintf(out, "    %s\n", s.URL)
			}
		}
		return nil
	})
}

```

File: pkg/cmd/webhook/webhook.go (3024 tokens)
```
package webhook

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCommand returns the webhook command.
func NewCommand(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manage Bitbucket webhooks",
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newDeleteCmd(f))
	cmd.AddCommand(newTestCmd(f))

	return cmd
}

type listOptions struct {
	Project   string
	Workspace string
	Repo      string
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List configured webhooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override (Data Center)")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func runList(cmd *cobra.Command, f *cmdutil.Factory, opts *listOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	switch host.Kind {
	case "dc":
		projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		hooks, err := client.ListWebhooks(ctx, projectKey, repoSlug)
		if err != nil {
			return err
		}

		payload := map[string]any{
			"project":  projectKey,
			"repo":     repoSlug,
			"webhooks": hooks,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(hooks) == 0 {
				fmt.Fprintln(ios.Out, "No webhooks configured.")
				return nil
			}

			for _, hook := range hooks {
				status := "disabled"
				if hook.Active {
					status = "active"
				}
				fmt.Fprintf(ios.Out, "%d\t%s\t%s (%s)\n", hook.ID, status, hook.Name, hook.URL)
			}
			return nil
		})
	case "cloud":
		workspace := firstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		hooks, err := client.ListWebhooks(ctx, workspace, repoSlug)
		if err != nil {
			return err
		}

		payload := map[string]any{
			"workspace": workspace,
			"repo":      repoSlug,
			"webhooks":  hooks,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			if len(hooks) == 0 {
				fmt.Fprintln(ios.Out, "No webhooks configured.")
				return nil
			}
			for _, hook := range hooks {
				status := "disabled"
				if hook.Active {
					status = "active"
				}
				fmt.Fprintf(ios.Out, "%s\t%s\t%s\n", hook.UUID, status, hook.URL)
			}
			return nil
		})
	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type createOptions struct {
	Project   string
	Workspace string
	Repo      string
	Name      string
	URL       string
	Events    []string
	Active    bool
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{Active: true}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new webhook",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override (Data Center)")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Name, "name", "", "Webhook name (required)")
	cmd.Flags().StringVar(&opts.URL, "url", "", "Webhook callback URL (required)")
	cmd.Flags().StringSliceVar(&opts.Events, "event", nil, "Events to subscribe to (repeatable)")
	cmd.Flags().BoolVar(&opts.Active, "active", opts.Active, "Whether the webhook starts active")

	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("url")
	_ = cmd.MarkFlagRequired("event")

	return cmd
}

func runCreate(cmd *cobra.Command, f *cmdutil.Factory, opts *createOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	switch host.Kind {
	case "dc":
		projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		hook, err := client.CreateWebhook(ctx, projectKey, repoSlug, bbdc.CreateWebhookInput{
			Name:   opts.Name,
			URL:    opts.URL,
			Events: opts.Events,
			Active: opts.Active,
		})
		if err != nil {
			return err
		}

		fmt.Fprintf(ios.Out, "✓ Created webhook #%d (%s)\n", hook.ID, hook.Name)
		return nil
	case "cloud":
		workspace := firstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		hook, err := client.CreateWebhook(ctx, workspace, repoSlug, bbcloud.WebhookInput{
			Description: opts.Name,
			URL:         opts.URL,
			Events:      opts.Events,
			Active:      opts.Active,
		})
		if err != nil {
			return err
		}

		fmt.Fprintf(ios.Out, "✓ Created webhook %s\n", hook.UUID)
		return nil
	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

type deleteOptions struct {
	Project    string
	Workspace  string
	Repo       string
	Identifier string
}

type testOptions struct {
	Project string
	Repo    string
	ID      string
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &deleteOptions{}
	cmd := &cobra.Command{
		Use:     "delete <id|uuid>",
		Aliases: []string{"rm"},
		Short:   "Delete a webhook",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Identifier = args[0]
			return runDelete(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override (Data Center)")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace override (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func runDelete(cmd *cobra.Command, f *cmdutil.Factory, opts *deleteOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	switch host.Kind {
	case "dc":
		projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		id, err := strconv.Atoi(opts.Identifier)
		if err != nil {
			return fmt.Errorf("invalid webhook id %q", opts.Identifier)
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		if err := client.DeleteWebhook(ctx, projectKey, repoSlug, id); err != nil {
			return err
		}

		fmt.Fprintf(ios.Out, "✓ Deleted webhook #%d\n", id)
		return nil
	case "cloud":
		workspace := firstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		if err := client.DeleteWebhook(ctx, workspace, repoSlug, opts.Identifier); err != nil {
			return err
		}

		fmt.Fprintf(ios.Out, "✓ Deleted webhook %s\n", opts.Identifier)
		return nil
	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

func newTestCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &testOptions{}
	cmd := &cobra.Command{
		Use:   "test <id>",
		Short: "Trigger a webhook test delivery",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.ID = args[0]
			return runTest(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func runTest(cmd *cobra.Command, f *cmdutil.Factory, opts *testOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("webhook test is supported for Data Center contexts only")
	}

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	id, err := strconv.Atoi(opts.ID)
	if err != nil {
		return fmt.Errorf("invalid webhook id %q", opts.ID)
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.TestWebhook(ctx, projectKey, repoSlug, id); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Triggered test delivery for webhook #%d\n", id)
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

```

File: pkg/cmdutil/client.go (563 tokens)
```
package cmdutil

import (
	"fmt"
	"time"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

// NewDCClient constructs a Bitbucket Data Center client using the supplied host.
func NewDCClient(host *config.Host) (*bbdc.Client, error) {
	if host == nil {
		return nil, fmt.Errorf("missing host configuration")
	}
	if host.BaseURL == "" {
		return nil, fmt.Errorf("host %q has no base URL configured", host.Kind)
	}
	opts := bbdc.Options{
		BaseURL:     host.BaseURL,
		Username:    host.Username,
		Token:       host.Token,
		EnableCache: true,
		Retry: httpx.RetryPolicy{
			MaxAttempts:    4,
			InitialBackoff: 250 * time.Millisecond,
			MaxBackoff:     2 * time.Second,
		},
	}
	return bbdc.New(opts)
}

// NewCloudClient constructs a Bitbucket Cloud client using the supplied host.
func NewCloudClient(host *config.Host) (*bbcloud.Client, error) {
	if host == nil {
		return nil, fmt.Errorf("missing host configuration")
	}
	if host.BaseURL == "" {
		host.BaseURL = "https://api.bitbucket.org/2.0"
	}
	opts := bbcloud.Options{
		BaseURL:     host.BaseURL,
		Username:    host.Username,
		Token:       host.Token,
		EnableCache: true,
		Retry: httpx.RetryPolicy{
			MaxAttempts:    4,
			InitialBackoff: 250 * time.Millisecond,
			MaxBackoff:     2 * time.Second,
		},
	}
	return bbcloud.New(opts)
}

// NewHTTPClient constructs a raw HTTP client for the configured host.
func NewHTTPClient(host *config.Host) (*httpx.Client, error) {
	if host == nil {
		return nil, fmt.Errorf("missing host configuration")
	}

	switch host.Kind {
	case "dc":
		client, err := NewDCClient(host)
		if err != nil {
			return nil, err
		}
		return client.HTTP(), nil
	case "cloud":
		client, err := NewCloudClient(host)
		if err != nil {
			return nil, err
		}
		return client.HTTP(), nil
	default:
		return nil, fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

```

File: pkg/cmdutil/context.go (326 tokens)
```
package cmdutil

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
)

// ResolveContext fetches the context and host configuration given an optional
// override name (typically provided via --context). When the override is empty
// the active context from the config file is used.
func ResolveContext(f *Factory, cmd *cobra.Command, override string) (string, *config.Context, *config.Host, error) {
	cfg, err := f.ResolveConfig()
	if err != nil {
		return "", nil, nil, err
	}

	contextName := override
	if contextName == "" {
		contextName = cfg.ActiveContext
	}

	if contextName == "" {
		return "", nil, nil, fmt.Errorf("no active context; run `%s context use <name>`", f.ExecutableName)
	}

	ctx, err := cfg.Context(contextName)
	if err != nil {
		return "", nil, nil, err
	}

	if ctx.Host == "" {
		return "", nil, nil, fmt.Errorf("context %q has no host configured", contextName)
	}

	host, err := cfg.Host(ctx.Host)
	if err != nil {
		return "", nil, nil, err
	}

	return contextName, ctx, host, nil
}

// FlagValue returns the value for the named flag if it exists.
func FlagValue(cmd *cobra.Command, name string) string {
	flag := cmd.Flags().Lookup(name)
	if flag == nil {
		return ""
	}
	return flag.Value.String()
}

```

File: pkg/cmdutil/errors.go (124 tokens)
```
package cmdutil

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// ErrSilent mirrors gh's sentinel used to suppress error printing.
	ErrSilent = errors.New("silent")
)

// ExitError wraps an exit code and optional message.
type ExitError struct {
	Code int
	Msg  string
}

func (e *ExitError) Error() string {
	return e.Msg
}

// NotImplemented returns a helpful placeholder error for unfinished commands.
func NotImplemented(cmd *cobra.Command) error {
	return fmt.Errorf("%s not yet implemented", cmd.CommandPath())
}

```

File: pkg/cmdutil/factory.go (639 tokens)
```
package cmdutil

import (
	"sync"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/browser"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
	"github.com/avivsinai/bitbucket-cli/pkg/pager"
	"github.com/avivsinai/bitbucket-cli/pkg/progress"
	"github.com/avivsinai/bitbucket-cli/pkg/prompter"
)

// Factory wires together shared services used by Cobra commands.
type Factory struct {
	AppVersion     string
	ExecutableName string

	IOStreams *iostreams.IOStreams

	Config func() (*config.Config, error)

	// Lazy-initialised platform helpers.
	Browser  browser.Browser
	Pager    pager.Manager
	Prompter prompter.Interface
	Spinner  progress.Spinner

	once struct {
		cfg sync.Once
	}
	cfg    *config.Config
	cfgErr error
	ioOnce sync.Once
	ios    *iostreams.IOStreams
}

// ResolveConfig loads configuration, caching the result.
func (f *Factory) ResolveConfig() (*config.Config, error) {
	f.once.cfg.Do(func() {
		if f.Config == nil {
			f.cfg, f.cfgErr = config.Load()
			return
		}
		f.cfg, f.cfgErr = f.Config()
	})
	return f.cfg, f.cfgErr
}

// Streams returns process IO streams, initialising them lazily.
func (f *Factory) Streams() (*iostreams.IOStreams, error) {
	f.ioOnce.Do(func() {
		if f.IOStreams != nil {
			f.ios = f.IOStreams
			return
		}
		f.ios = iostreams.System()
	})
	return f.ios, nil
}

// BrowserOpener returns a Browser, initialising the default system implementation
// when necessary.
func (f *Factory) BrowserOpener() browser.Browser {
	if f.Browser == nil {
		f.Browser = browser.NewSystem()
	}
	return f.Browser
}

// PagerManager returns the pager manager, defaulting to a system-backed
// instance bound to the factory streams.
func (f *Factory) PagerManager() pager.Manager {
	if f.Pager == nil {
		ios, _ := f.Streams()
		f.Pager = pager.NewSystem(ios)
	}
	return f.Pager
}

// Prompt returns the prompter helper for interactive input.
func (f *Factory) Prompt() prompter.Interface {
	if f.Prompter == nil {
		ios, _ := f.Streams()
		f.Prompter = prompter.New(ios)
	}
	return f.Prompter
}

// ProgressSpinner exposes a spinner helper for long-running operations.
func (f *Factory) ProgressSpinner() progress.Spinner {
	if f.Spinner == nil {
		ios, _ := f.Streams()
		f.Spinner = progress.NewSpinner(ios)
	}
	return f.Spinner
}

```

File: pkg/cmdutil/output.go (470 tokens)
```
package cmdutil

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/format"
)

// OutputSettings captures structured output preferences.
type OutputSettings struct {
	Format   string
	JQ       string
	Template string
}

// OutputSettings extracts flags from the command hierarchy with validation.
func ResolveOutputSettings(cmd *cobra.Command) (OutputSettings, error) {
	root := cmd.Root()

	lookup := func(name string) string {
		flag := root.PersistentFlags().Lookup(name)
		if flag == nil {
			return ""
		}
		return flag.Value.String()
	}

	jsonEnabled := lookup("json") == "true"
	yamlEnabled := lookup("yaml") == "true"
	jqExpr := lookup("jq")
	tmpl := lookup("template")

	if jsonEnabled && yamlEnabled {
		return OutputSettings{}, fmt.Errorf("cannot use --json and --yaml simultaneously")
	}

	if jqExpr != "" && tmpl != "" {
		return OutputSettings{}, fmt.Errorf("cannot use --jq and --template simultaneously")
	}

	if jqExpr != "" && !jsonEnabled {
		return OutputSettings{}, fmt.Errorf("--jq requires --json")
	}

	format := ""
	if jsonEnabled {
		format = "json"
	} else if yamlEnabled {
		format = "yaml"
	}

	return OutputSettings{
		Format:   format,
		JQ:       jqExpr,
		Template: tmpl,
	}, nil
}

// OutputFormat preserves backwards compatibility for callers needing only the
// format value.
func OutputFormat(cmd *cobra.Command) (string, error) {
	settings, err := ResolveOutputSettings(cmd)
	if err != nil {
		return "", err
	}
	return settings.Format, nil
}

// WriteOutput writes structured output according to user preferences and runs
// fallback when no structured output is requested.
func WriteOutput(cmd *cobra.Command, w io.Writer, data any, fallback func() error) error {
	settings, err := ResolveOutputSettings(cmd)
	if err != nil {
		return err
	}
	opts := format.Options{Format: settings.Format, JQ: settings.JQ, Template: settings.Template}
	return format.Write(w, opts, data, fallback)
}

```

File: pkg/cmdutil/url.go (268 tokens)
```
package cmdutil

import (
	"fmt"
	"net/url"
	"strings"
)

// NormalizeBaseURL ensures the Bitbucket base URL includes a scheme and has no
// trailing slash.
func NormalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("host is required")
	}
	if !strings.HasPrefix(raw, "http://") && !strings.HasPrefix(raw, "https://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse URL: %w", err)
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""
	return strings.TrimRight(u.String(), "/"), nil
}

// HostKeyFromURL resolves the host component used as the configuration key.
func HostKeyFromURL(baseURL string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid base URL %q", baseURL)
	}
	return u.Host, nil
}

```

File: pkg/format/doc.go (14 tokens)
```
// Package format renders structured and human friendly CLI output.
package format

```

File: pkg/format/format.go (603 tokens)
```
package format

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"text/template"

	"github.com/itchyny/gojq"
	"gopkg.in/yaml.v3"
)

// Options configures structured output rendering.
type Options struct {
	Format   string
	JQ       string
	Template string
}

// Write serializes data according to the chosen options. When no structured
// output is requested the fallback function is invoked to render human-friendly
// output.
func Write(w io.Writer, opts Options, data any, fallback func() error) error {
	if opts.Format == "" && opts.JQ == "" && opts.Template == "" {
		if fallback == nil {
			return nil
		}
		return fallback()
	}

	value := data

	if opts.JQ != "" {
		var err error
		value, err = applyJQ(opts.JQ, value)
		if err != nil {
			return err
		}
	}

	if opts.Template != "" {
		tmpl, err := template.New("output").Parse(opts.Template)
		if err != nil {
			return fmt.Errorf("parse template: %w", err)
		}
		return tmpl.Execute(w, value)
	}

	switch opts.Format {
	case "", "json":
		enc := json.NewEncoder(w)
		if opts.Format != "" {
			enc.SetIndent("", "  ")
		}
		if err := enc.Encode(value); err != nil {
			return fmt.Errorf("encode json: %w", err)
		}
		return nil
	case "yaml":
		out, err := yaml.Marshal(value)
		if err != nil {
			return fmt.Errorf("encode yaml: %w", err)
		}
		_, err = w.Write(out)
		return err
	default:
		return fmt.Errorf("unsupported format %q", opts.Format)
	}
}

func applyJQ(expression string, value any) (any, error) {
	query, err := gojq.Parse(expression)
	if err != nil {
		return nil, fmt.Errorf("parse jq expression: %w", err)
	}
	code, err := gojq.Compile(query)
	if err != nil {
		return nil, fmt.Errorf("compile jq expression: %w", err)
	}

	iter := code.Run(value)
	var results []any
	for {
		v, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := v.(error); isErr {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("jq evaluation failed: %w", err)
		}
		results = append(results, v)
	}

	if len(results) == 0 {
		return nil, nil
	}
	if len(results) == 1 {
		return results[0], nil
	}
	return results, nil
}

```

File: pkg/httpx/client_test.go (1476 tokens)
```
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type payload struct {
	Message string `json:"message"`
}

func TestClientCachingWithETag(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", "etag-123")
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", "42")
		if r.Header.Get("If-None-Match") == "etag-123" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_ = json.NewEncoder(w).Encode(payload{Message: "hello"})
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL, EnableCache: true})
	if err != nil {
		t.Fatalf("New client: %v", err)
	}

	req1, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	var out payload
	if err := client.Do(req1, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if out.Message != "hello" {
		t.Fatalf("expected hello, got %q", out.Message)
	}

	req2, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	out = payload{}
	if err := client.Do(req2, &out); err != nil {
		t.Fatalf("Do cache: %v", err)
	}
	if out.Message != "hello" {
		t.Fatalf("expected cached hello, got %q", out.Message)
	}

	if hits != 2 {
		t.Fatalf("expected 2 hits (initial + 304), got %d", hits)
	}

	rate := client.RateLimitState()
	if rate.Remaining != 42 {
		t.Fatalf("expected remaining 42, got %d", rate.Remaining)
	}
}

func TestClientRetriesOnServerError(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		if count == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload{Message: "ok"})
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{
		BaseURL:     server.URL,
		EnableCache: false,
		Retry: RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     20 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	var out payload
	if err := client.Do(req, &out); err != nil {
		t.Fatalf("Do with retry: %v", err)
	}
	if out.Message != "ok" {
		t.Fatalf("expected ok, got %q", out.Message)
	}

	if hits != 2 {
		t.Fatalf("expected 2 attempts, got %d", hits)
	}
}

func TestClientNewRequestPreservesQuery(t *testing.T) {
	client, err := New(Options{BaseURL: "https://example.com/api"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/rest/projects?limit=25&start=0", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if got := req.URL.String(); got != "https://example.com/rest/projects?limit=25&start=0" {
		t.Fatalf("unexpected URL: %s", got)
	}
	if req.URL.RawQuery != "limit=25&start=0" {
		t.Fatalf("expected raw query preserved, got %q", req.URL.RawQuery)
	}
}

func TestClientNewRequestHandlesRelativeWithoutSlash(t *testing.T) {
	client, err := New(Options{BaseURL: "https://example.com/api"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "rest/repos", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if got := req.URL.String(); got != "https://example.com/rest/repos" {
		t.Fatalf("unexpected URL: %s", got)
	}
}

func TestClientBackoffRespectsContextCancellation(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{
		BaseURL: server.URL,
		Retry: RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 500 * time.Millisecond,
			MaxBackoff:     time.Second,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	req, err := client.NewRequest(ctx, http.MethodGet, "/fail", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	var once sync.Once
	time.AfterFunc(50*time.Millisecond, func() {
		once.Do(cancel)
	})

	start := time.Now()
	err = client.Do(req, nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected error from cancelled context")
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context cancellation error, got %v", err)
	}
	if elapsed >= 400*time.Millisecond {
		t.Fatalf("expected cancellation to interrupt backoff, took %v", elapsed)
	}
	if hits != 1 {
		t.Fatalf("expected single request, got %d", hits)
	}
}

```

File: pkg/httpx/client.go (3275 tokens)
```
package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client wraps HTTP access with Bitbucket-aware defaults.
type Client struct {
	baseURL   *url.URL
	username  string
	password  string
	userAgent string

	httpClient *http.Client

	enableCache bool
	cacheMu     sync.RWMutex
	cache       map[string]*cacheEntry

	rateMu sync.RWMutex
	rate   RateLimit

	retry RetryPolicy

	debug bool
}

// Options configures a Client.
type Options struct {
	BaseURL   string
	Username  string
	Password  string
	UserAgent string
	Timeout   time.Duration

	EnableCache bool
	Retry       RetryPolicy
	Debug       bool
}

// RetryPolicy defines exponential backoff characteristics for retries.
type RetryPolicy struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// RateLimit captures headers advertised by Bitbucket for throttling.
type RateLimit struct {
	Limit     int
	Remaining int
	Reset     time.Time
	Source    string
}

type cacheEntry struct {
	etag     string
	body     []byte
	storedAt time.Time
}

// New constructs a Client from options.
func New(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	base, err := url.Parse(opts.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	if base.Scheme == "" {
		return nil, fmt.Errorf("base URL must include scheme (e.g. https)")
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	client := &Client{
		baseURL:  base,
		username: strings.TrimSpace(opts.Username),
		password: opts.Password,
		userAgent: func() string {
			if opts.UserAgent != "" {
				return opts.UserAgent
			}
			return "bkt-cli"
		}(),
		httpClient: &http.Client{
			Timeout: timeout,
		},
		enableCache: opts.EnableCache,
		cache:       make(map[string]*cacheEntry),
	}

	if opts.Debug || os.Getenv("BKT_HTTP_DEBUG") != "" {
		client.debug = true
	}

	policy := opts.Retry
	if policy.MaxAttempts == 0 {
		policy.MaxAttempts = 3
	}
	if policy.InitialBackoff == 0 {
		policy.InitialBackoff = 200 * time.Millisecond
	}
	if policy.MaxBackoff == 0 {
		policy.MaxBackoff = 2 * time.Second
	}
	client.retry = policy

	return client, nil
}

// NewRequest builds an HTTP request relative to the base URL. Body values are
// JSON encoded when non-nil.
func (c *Client) NewRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("path is required")
	}

	var rel *url.URL
	var err error

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		rel, err = url.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("parse request URL: %w", err)
		}
	} else {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		rel, err = url.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("parse request path: %w", err)
		}
	}

	if rel.Path == "" {
		rel.Path = "/"
	}

	u := c.baseURL.ResolveReference(rel)

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
	}

	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(payload))
		data := payload
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(data)), nil
		}
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	return req, nil
}

// Do executes the HTTP request and decodes the response into v when provided.
func (c *Client) Do(req *http.Request, v any) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	attempts := 0
	for {
		attemptReq, err := cloneRequest(req)
		if err != nil {
			return err
		}

		if c.enableCache && attemptReq.Method == http.MethodGet {
			if etag := c.cachedETag(attemptReq); etag != "" {
				attemptReq.Header.Set("If-None-Match", etag)
			}
		}

		if c.debug {
			fmt.Fprintf(os.Stderr, "--> %s %s\n", attemptReq.Method, attemptReq.URL.String())
		}

        resp, err := c.httpClient.Do(attemptReq)
        if err != nil {
            if !c.shouldRetry(attempts, 0) {
                if c.debug {
                    fmt.Fprintf(os.Stderr, "<-- network error: %v\n", err)
                }
                return err
            }
            attempts++
            continueRetry, waitErr := c.backoff(req.Context(), attempts, resp)
            if waitErr != nil {
                return waitErr
            }
            if !continueRetry {
                if c.debug {
                    fmt.Fprintf(os.Stderr, "<-- retry abort after error: %v\n", err)
                }
                return err
            }
            continue
        }

		c.updateRateLimit(resp)
		c.applyAdaptiveThrottle()

		if c.debug {
			fmt.Fprintf(os.Stderr, "<-- %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
		}

		if resp.StatusCode == http.StatusNotModified && c.enableCache && attemptReq.Method == http.MethodGet {
			resp.Body.Close()
			if err := c.applyCachedResponse(attemptReq, v); err != nil {
				return err
			}
			return nil
		}

        if shouldRetryStatus(resp.StatusCode) {
            bodyBytes, _ := io.ReadAll(resp.Body)
            resp.Body.Close()
            if !c.shouldRetry(attempts, resp.StatusCode) {
                if len(bodyBytes) > 0 {
                    resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
                }
                return decodeError(resp)
            }
            attempts++
            continueRetry, waitErr := c.backoff(req.Context(), attempts, resp)
            if waitErr != nil {
                return waitErr
            }
            if !continueRetry {
                if len(bodyBytes) > 0 {
                    resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
                }
                return decodeError(resp)
            }
            continue
        }

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			defer resp.Body.Close()
			return decodeError(resp)
		}

		if v == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if c.enableCache && attemptReq.Method == http.MethodGet {
				c.storeCache(attemptReq, nil, resp.Header.Get("ETag"))
			}
			return nil
		}

		if writer, ok := v.(io.Writer); ok {
			_, err := io.Copy(writer, resp.Body)
			resp.Body.Close()
			return err
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}

		if c.enableCache && attemptReq.Method == http.MethodGet && resp.Header.Get("ETag") != "" {
			c.storeCache(attemptReq, bodyBytes, resp.Header.Get("ETag"))
		}

		if len(bodyBytes) == 0 {
			return nil
		}

		if err := json.Unmarshal(bodyBytes, v); err != nil {
			return err
		}
		return nil
	}
}

func decodeError(resp *http.Response) error {
	type apiErr struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	var payload apiErr
	data, err := io.ReadAll(resp.Body)
	if err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &payload)
	}

	if len(payload.Errors) > 0 {
		return fmt.Errorf("%s: %s", resp.Status, payload.Errors[0].Message)
	}

	if err == nil && len(data) > 0 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(data)))
	}

	return fmt.Errorf("%s", resp.Status)
}

func cloneRequest(req *http.Request) (*http.Request, error) {
	newReq := req.Clone(req.Context())
	newReq.Header = req.Header.Clone()
	if req.Body != nil {
		if req.GetBody == nil {
			return nil, fmt.Errorf("request body cannot be replayed")
		}
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		newReq.Body = body
	}
	return newReq, nil
}

func shouldRetryStatus(code int) bool {
	if code == http.StatusTooManyRequests {
		return true
	}
	return code >= 500 && code <= 599
}

func (c *Client) shouldRetry(attempts int, status int) bool {
	return attempts+1 < c.retry.MaxAttempts
}

func (c *Client) backoff(ctx context.Context, attempts int, resp *http.Response) (bool, error) {
    if attempts >= c.retry.MaxAttempts {
        return false, nil
    }

    delay := c.retry.InitialBackoff
    if attempts > 1 {
        delay = delay * time.Duration(1<<(attempts-1))
    }
    if delay > c.retry.MaxBackoff {
        delay = c.retry.MaxBackoff
    }

    if resp != nil {
        if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
            if secs, err := strconv.Atoi(retryAfter); err == nil {
                delay = time.Duration(secs) * time.Second
            }
        }
    }

    if delay <= 0 {
        select {
        case <-ctx.Done():
            return false, ctx.Err()
        default:
            return true, nil
        }
    }

    timer := time.NewTimer(delay)
    defer timer.Stop()

    select {
    case <-ctx.Done():
        return false, ctx.Err()
    case <-timer.C:
        return true, nil
    }
}

func (c *Client) cacheKey(req *http.Request) string {
	return req.Method + " " + req.URL.String()
}

func (c *Client) cachedETag(req *http.Request) string {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	if entry, ok := c.cache[c.cacheKey(req)]; ok {
		return entry.etag
	}
	return ""
}

func (c *Client) storeCache(req *http.Request, body []byte, etag string) {
	if etag == "" || len(body) == 0 {
		return
	}
	c.cacheMu.Lock()
	c.cache[c.cacheKey(req)] = &cacheEntry{etag: etag, body: append([]byte(nil), body...), storedAt: time.Now()}
	c.cacheMu.Unlock()
}

func (c *Client) applyCachedResponse(req *http.Request, v any) error {
	if v == nil {
		return nil
	}
	c.cacheMu.RLock()
	entry, ok := c.cache[c.cacheKey(req)]
	c.cacheMu.RUnlock()
	if !ok {
		return fmt.Errorf("cached response missing for %s", req.URL)
	}

	if writer, ok := v.(io.Writer); ok {
		_, err := writer.Write(entry.body)
		return err
	}
	if len(entry.body) == 0 {
		return nil
	}
	return json.Unmarshal(entry.body, v)
}

// RateLimitState returns the last observed rate limit headers.
func (c *Client) RateLimitState() RateLimit {
	c.rateMu.RLock()
	defer c.rateMu.RUnlock()
	return c.rate
}

func (c *Client) updateRateLimit(resp *http.Response) {
	headers := resp.Header

	readHeader := func(key string) int {
		val := headers.Get(key)
		if val == "" {
			return 0
		}
		n, err := strconv.Atoi(val)
		if err != nil {
			return 0
		}
		return n
	}

	limit := readHeader("X-RateLimit-Limit")
	remaining := readHeader("X-RateLimit-Remaining")
	resetHeader := headers.Get("X-RateLimit-Reset")

	var reset time.Time
	if resetHeader != "" {
		if epoch, err := strconv.ParseInt(resetHeader, 10, 64); err == nil {
			if epoch > 0 {
				reset = time.Unix(epoch, 0)
			}
		} else {
			if parsed, err := time.Parse(time.RFC1123, resetHeader); err == nil {
				reset = parsed
			}
		}
	}

	source := ""
	if limit != 0 || remaining != 0 {
		source = "bitbucket"
	}

	if limit == 0 && remaining == 0 {
		// Some endpoints expose Atlassian-RateLimit prefixed headers.
		limit = readHeader("X-Attempt-RateLimit-Limit")
		remaining = readHeader("X-Attempt-RateLimit-Remaining")
		if limit == 0 && remaining == 0 {
			limit = readHeader("X-RateLimit-Capacity")
			remaining = readHeader("X-RateLimit-Available")
		}
		if limit != 0 || remaining != 0 {
			source = "atlassian"
		}
	}

	if limit == 0 && remaining == 0 {
		return
	}

	c.rateMu.Lock()
	c.rate = RateLimit{Limit: limit, Remaining: remaining, Reset: reset, Source: source}
	c.rateMu.Unlock()
}

func (c *Client) applyAdaptiveThrottle() {
	c.rateMu.RLock()
	rl := c.rate
	c.rateMu.RUnlock()

	if rl.Remaining > 1 || rl.Reset.IsZero() {
		return
	}

	sleep := time.Until(rl.Reset)
	if sleep <= 0 {
		return
	}
	if sleep > 5*time.Second {
		sleep = 5 * time.Second
	}
	time.Sleep(sleep)
}

```

File: pkg/httpx/doc.go (20 tokens)
```
// Package httpx centralizes HTTP client behavior such as retries and rate limiting.
package httpx

```

File: pkg/iostreams/iostreams.go (537 tokens)
```
package iostreams

import (
	"io"
	"os"
	"sync"

	"golang.org/x/term"
)

// IOStreams collects input and output streams for command execution.
//
// The structure mirrors gh/jk ergonomics by exposing terminal metadata and
// lazy colour profile detection. Commands can inspect the terminal
// capabilities to decide when to render ANSI colours, tables, or spinner
// widgets.
type IOStreams struct {
	In     io.ReadCloser
	Out    io.Writer
	ErrOut io.Writer

	isStdinTTY  bool
	isStdoutTTY bool
	isStderrTTY bool

	colorEnabled bool
	once         sync.Once
}

// System returns IOStreams bound to the current process standard streams and
// captures terminal metadata so downstream components can make ergonomic
// decisions (colours, paging, prompts, etc.).
func System() *IOStreams {
	isTTY := func(f *os.File) bool {
		if f == nil {
			return false
		}
		return term.IsTerminal(int(f.Fd()))
	}

	return &IOStreams{
		In:          os.Stdin,
		Out:         os.Stdout,
		ErrOut:      os.Stderr,
		isStdinTTY:  isTTY(os.Stdin),
		isStdoutTTY: isTTY(os.Stdout),
		isStderrTTY: isTTY(os.Stderr),
	}
}

// CanPrompt reports whether stdin is a TTY and therefore suitable for
// interactive prompts.
func (s *IOStreams) CanPrompt() bool {
	return s != nil && s.isStdinTTY
}

// ColorEnabled returns true when ANSI colour output should be rendered. The
// decision is cached so repeated checks are inexpensive.
func (s *IOStreams) ColorEnabled() bool {
	if s == nil {
		return false
	}
	s.once.Do(func() {
		s.colorEnabled = s.isStdoutTTY
	})
	return s.colorEnabled
}

// SetColorEnabled allows callers (e.g. tests) to force colour behaviour.
func (s *IOStreams) SetColorEnabled(enabled bool) {
	if s == nil {
		return
	}
	s.once.Do(func() {})
	s.colorEnabled = enabled
}

// IsStdoutTTY reports whether stdout is attached to a terminal.
func (s *IOStreams) IsStdoutTTY() bool {
	return s != nil && s.isStdoutTTY
}

// IsStderrTTY reports whether stderr is attached to a terminal.
func (s *IOStreams) IsStderrTTY() bool {
	return s != nil && s.isStderrTTY
}

```

File: pkg/pager/pager.go (525 tokens)
```
package pager

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// Manager coordinates optional pager processes (e.g. less) used for long
// command output.
type Manager interface {
	Enabled() bool
	Start() (io.WriteCloser, error)
	Stop() error
}

type system struct {
	ios    *iostreams.IOStreams
	cmd    *exec.Cmd
	writer io.WriteCloser
}

// NewSystem returns a pager manager backed by the user's $PAGER when stdout is
// a TTY. When stdout is redirected a no-op manager is returned instead.
func NewSystem(ios *iostreams.IOStreams) Manager {
	if ios == nil || !ios.IsStdoutTTY() {
		return noop{}
	}
	return &system{ios: ios}
}

func (p *system) Enabled() bool { return true }

func (p *system) Start() (io.WriteCloser, error) {
	if p.writer != nil {
		return p.writer, nil
	}

	pagerCmd := strings.Fields(resolvePager())
	cmd := exec.Command(pagerCmd[0], pagerCmd[1:]...)
	cmd.Stdout = p.ios.Out
	cmd.Stderr = p.ios.ErrOut

	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		_ = in.Close()
		return nil, err
	}

	p.cmd = cmd
	p.writer = in
	return in, nil
}

func (p *system) Stop() error {
	if p.writer != nil {
		_ = p.writer.Close()
		p.writer = nil
	}
	if p.cmd != nil {
		err := p.cmd.Wait()
		p.cmd = nil
		return err
	}
	return nil
}

type noop struct{}

func (noop) Enabled() bool { return false }
func (noop) Start() (io.WriteCloser, error) {
	return nopWriteCloser{Writer: os.Stdout}, nil
}
func (noop) Stop() error { return nil }

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

func resolvePager() string {
	if cmd := os.Getenv("BKT_PAGER"); cmd != "" {
		return cmd
	}
	if cmd := os.Getenv("PAGER"); cmd != "" {
		return cmd
	}
	return "less -R"
}

```

File: pkg/progress/spinner.go (589 tokens)
```
package progress

import (
	"fmt"
	"sync"
	"time"

	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// Spinner renders a simple textual indicator while a background task runs.
// Commands can use the spinner to provide user feedback in long-running
// operations without taking a dependency on external packages.
type Spinner interface {
	Start(msg string)
	Stop(msg string)
	Fail(msg string)
}

type noopSpinner struct {
	ios *iostreams.IOStreams
}

// NewSpinner constructs a terminal spinner when stdout is a TTY. Otherwise a
// newline-based fallback is returned.
func NewSpinner(ios *iostreams.IOStreams) Spinner {
	if ios != nil && ios.IsStdoutTTY() {
		return newTTYSpinner(ios)
	}
	return &noopSpinner{ios: ios}
}

func (s *noopSpinner) Start(msg string) { s.write(msg) }
func (s *noopSpinner) Stop(msg string)  { s.write(msg) }
func (s *noopSpinner) Fail(msg string)  { s.write(msg) }

func (s *noopSpinner) write(msg string) {
	if s.ios == nil || msg == "" {
		return
	}
	fmt.Fprintln(s.ios.Out, msg)
}

type ttySpinner struct {
	ios    *iostreams.IOStreams
	stopCh chan struct{}
	mux    sync.Mutex
}

func newTTYSpinner(ios *iostreams.IOStreams) Spinner {
	return &ttySpinner{ios: ios}
}

func (s *ttySpinner) Start(msg string) {
	s.mux.Lock()
	if s.stopCh != nil {
		close(s.stopCh)
	}
	s.stopCh = make(chan struct{})
	stop := s.stopCh
	s.mux.Unlock()

	frames := []rune{'|', '/', '-', '\\'}

	go func() {
		idx := 0
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				fmt.Fprintf(s.ios.Out, "\r%c %s", frames[idx], msg)
				idx = (idx + 1) % len(frames)
			}
		}
	}()
}

func (s *ttySpinner) Stop(msg string) {
	s.endWithPrefix("[OK]", msg)
}

func (s *ttySpinner) Fail(msg string) {
	s.endWithPrefix("[ERR]", msg)
}

func (s *ttySpinner) endWithPrefix(prefix, msg string) {
	s.mux.Lock()
	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}
	s.mux.Unlock()

	if msg == "" {
		return
	}
	fmt.Fprintf(s.ios.Out, "\r%s %s\n", prefix, msg)
}

```

File: pkg/prompter/prompter_test.go (297 tokens)
```
package prompter

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func TestConfirmRetriesOnInvalidInput(t *testing.T) {
	input := "maybe\ny\n"

	ios := &iostreams.IOStreams{
		In:     io.NopCloser(strings.NewReader(input)),
		Out:    &bytes.Buffer{},
		ErrOut: &bytes.Buffer{},
	}
	forceTTY(ios)

	prompt := New(ios)
	got, err := prompt.Confirm("Proceed?", false)
	if err != nil {
		t.Fatalf("Confirm returned error: %v", err)
	}
	if !got {
		t.Fatalf("expected confirmation to be true after invalid input")
	}

	if !strings.Contains(ios.ErrOut.(*bytes.Buffer).String(), "Please respond") {
		t.Fatalf("expected error prompt after invalid input")
	}
}

func forceTTY(ios *iostreams.IOStreams) {
	setBoolField := func(name string) {
		field := reflect.ValueOf(ios).Elem().FieldByName(name)
		ptr := unsafe.Pointer(field.UnsafeAddr())
		reflect.NewAt(field.Type(), ptr).Elem().SetBool(true)
	}

	setBoolField("isStdinTTY")
	setBoolField("isStdoutTTY")
	setBoolField("isStderrTTY")
}

```

File: pkg/prompter/prompter.go (573 tokens)
```
package prompter

import (
	"bufio"
	"errors"
	"fmt"
	"strings"

	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// Interface exposes interactive prompt helpers used by commands.
type Interface interface {
	Input(prompt, defaultValue string) (string, error)
	Confirm(prompt string, defaultYes bool) (bool, error)
}

type system struct {
	ios *iostreams.IOStreams
}

// New creates a prompter bound to the provided IO streams. When prompts are
// not possible (stdin not a TTY) the helper returns errors so commands can
// fallback to non-interactive flows.
func New(ios *iostreams.IOStreams) Interface {
	return &system{ios: ios}
}

func (p *system) reader() (*bufio.Reader, error) {
	if p.ios == nil || !p.ios.CanPrompt() {
		return nil, errors.New("interactive prompts require a TTY")
	}
	return bufio.NewReader(p.ios.In), nil
}

func (p *system) Input(prompt, defaultValue string) (string, error) {
	r, err := p.reader()
	if err != nil {
		return "", err
	}

	question := prompt
	if defaultValue != "" {
		question = fmt.Sprintf("%s [%s]", prompt, defaultValue)
	}

	if _, err := fmt.Fprint(p.ios.Out, question+": "); err != nil {
		return "", err
	}

	line, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return defaultValue, nil
	}
	return line, nil
}

func (p *system) Confirm(prompt string, defaultYes bool) (bool, error) {
	r, err := p.reader()
	if err != nil {
		return false, err
	}

	var suffix string
	if defaultYes {
		suffix = "[Y/n]"
	} else {
		suffix = "[y/N]"
	}

	for {
		if _, err := fmt.Fprintf(p.ios.Out, "%s %s: ", prompt, suffix); err != nil {
			return false, err
		}

		line, err := r.ReadString('\n')
		if err != nil {
			return false, err
		}

		switch strings.ToLower(strings.TrimSpace(line)) {
		case "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		case "":
			return defaultYes, nil
		default:
			fmt.Fprintln(p.ios.ErrOut, "Please respond with 'y' or 'n'.")
		}
	}
}

```

File: README.md (1134 tokens)
```
# bkt – Bitbucket CLI

[![CI](https://github.com/avivsinai/bitbucket-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/avivsinai/bitbucket-cli/actions/workflows/ci.yml)
[![OpenSSF Scorecard](https://img.shields.io/ossf-scorecard/github.com/avivsinai/bitbucket-cli?label=openssf%20scorecard)](https://scorecard.dev/viewer/?uri=github.com/avivsinai/bitbucket-cli)
[![Go Reference](https://pkg.go.dev/badge/github.com/avivsinai/bitbucket-cli.svg)](https://pkg.go.dev/github.com/avivsinai/bitbucket-cli)

`bkt` is a stand-alone Bitbucket command-line interface that targets Bitbucket Data Center **and** Bitbucket Cloud. It mirrors the ergonomics of `gh` while remaining provider-pure (no Jenkins coupling) and delivers a consistent JSON/YAML contract for automation.

## Project layout

```
cmd/bkt/             # CLI entry point
internal/bktcmd/     # Main() wiring (factory + root command)
internal/build/      # Version metadata (overridden via ldflags)
internal/config/     # Context and host configuration
internal/remote/     # Git remote parsing utilities
pkg/cmd/             # Cobra command implementations (auth, repo, pr, ...)
pkg/cmdutil/         # Shared command helpers and factory wiring
pkg/iostreams/       # IO stream abstractions
pkg/bbdc/            # Bitbucket Data Center client implementation
pkg/bbcloud/         # Bitbucket Cloud client implementation
pkg/format/          # Output rendering helpers
pkg/httpx/           # Shared HTTP client and retry logic
```

## Getting started

```bash
go build ./cmd/bkt
./bkt --help
```

### 1. Authenticate against Bitbucket Data Center or Cloud

```bash
bkt auth login https://bitbucket.mycorp.example --username alice --token <PAT>
```

Add `--kind cloud` when targeting Bitbucket Cloud. Credentials are stored in
`$XDG_CONFIG_HOME/bkt/config.yml`.

### 2. Create and activate a context

```bash
bkt context create dc-prod --host bitbucket.mycorp.example --project ABC --set-active
bkt context list
```

Contexts capture the host mapping, default project/workspace, and optional default repository for commands.

### 3. Work with repositories

```bash
bkt repo list --limit 20
bkt repo list --workspace myteam --limit 10   # Cloud workspace override
bkt repo view platform-api
bkt repo create data-pipeline --description "Data ingestion" --project DATA
bkt repo clone platform-api --project DATA --ssh
```

`repo list`/`repo view` automatically target the right REST API for your active context: Data Center uses `/rest/api/1.0/projects/{projectKey}/repos`, while Cloud uses `/2.0/repositories/{workspace}`.

### 4. Pull request workflows

```bash
bkt pr list --state OPEN --limit 10
bkt pr create --title "feat: cache" --source feature/cache --target main --reviewer alice
bkt pr merge 42 --message "merge: feature/cache"
```

The CLI wraps Bitbucket pull-request endpoints for creation, listing, review, and merge operations.

### 5. Branch, permission, webhook, pipeline, and extension management

```bash
bkt branch list --workspace myteam           # Cloud branch listing
bkt branch create release/1.9 --from main    # Data Center branch utils
bkt perms repo list --project DATA --repo platform-api
bkt webhook create --name "CI" --url https://ci.example.com/hook --event repo:refs_changed
bkt pipeline run --workspace myteam --repo api --ref main --var ENV=staging
bkt extension install https://github.com/example/bkt-hello.git
bkt extension exec hello -- --flag=1
bkt status pipeline {pipeline-uuid}
bkt status rate-limit
```

Branch utilities use Bitbucket's Branch Utils REST API for listing, creation, deletion, and default updates. Permission and webhook commands map to their respective REST endpoints for consistent automation.

Extensions are cloned into `$XDG_CONFIG_HOME/bkt/extensions` (or the directory configured via `BKT_CONFIG_DIR`) and executed in-place. Binaries should follow the `bkt-<name>` naming convention so the CLI can discover them automatically.

### Structured output & raw API access

Every command supports the global `--json` and `--yaml` flags for automation-ready output.

For endpoints that are not yet wrapped, reach directly for the API escape hatch:

```bash
bkt api /rest/api/1.0/projects --param limit=100 --json
bkt api /2.0/repositories --param workspace=myteam --field pagelen=50
```

## Support

See [SUPPORT.md](SUPPORT.md) for available support channels and response times.

## Roadmap highlights

- Device authorization flow for Bitbucket Cloud workspaces.
- Declarative context apply (`bkt context apply`) from YAML manifests.
- Native shell completions and plugin discovery.
- Extended telemetry exporters (OpenTelemetry traces for API calls).

```

File: SECURITY.md (274 tokens)
```
# Security Policy

## Supported versions

| Version | Supported |
| ------- | --------- |
| main    | ✅        |
| tags    | ✅        |

We support the latest release and the `main` branch. Older tags are archived as
read-only snapshots.

## Reporting a vulnerability

Please email [security@avivsinai.dev](mailto:security@avivsinai.dev) with
"[bkt]" in the subject. Include:

- A detailed description of the issue and the potential impact
- Steps to reproduce (or proof of concept)
- Any temporary mitigations you are aware of

We will acknowledge receipt within two business days and provide regular status
updates until the issue is resolved.

If you require encrypted communication, request our PGP key in your initial
email.

## Disclosure policy

1. We investigate and confirm the vulnerability.
2. We coordinate a fix and target release date.
3. We publish a patched release and update the CHANGELOG with mitigation
   details.
4. Once a fix is available, we disclose the issue publicly.

## Dependencies

We rely on GitHub Dependabot, the OpenSSF Scorecard workflow, and the CI
workflows under `.github/workflows/` to keep dependencies fresh. See
[`docs/SECURITY.md`](docs/SECURITY.md) for deeper operational guidance.

```

File: SUPPORT.md (121 tokens)
```
# Support

- **Questions / ideas**: open a GitHub Discussion or start a thread in
  `#bkt-cli` on the community Slack (invite in the README).
- **Bugs**: file an issue using the "Bug report" template.
- **Security reports**: email [security@avivsinai.dev](mailto:security@avivsinai.dev).
- **Commercial support**: not currently offered. If this becomes critical for
  your organization, open a discussion so we can evaluate options.

We strive to respond to issues within two business days.

```

</files>


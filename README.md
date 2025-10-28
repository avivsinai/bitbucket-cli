# bkt – Bitbucket CLI

[![CI](https://github.com/avivsinai/bitbucket-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/avivsinai/bitbucket-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/avivsinai/bitbucket-cli)](https://github.com/avivsinai/bitbucket-cli/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/avivsinai/bitbucket-cli)](https://goreportcard.com/report/github.com/avivsinai/bitbucket-cli)
[![OpenSSF Scorecard](https://img.shields.io/ossf-scorecard/github.com/avivsinai/bitbucket-cli?label=openssf%20scorecard)](https://scorecard.dev/viewer/?uri=github.com/avivsinai/bitbucket-cli)
[![Go Reference](https://pkg.go.dev/badge/github.com/avivsinai/bitbucket-cli.svg)](https://pkg.go.dev/github.com/avivsinai/bitbucket-cli)
[![License](https://img.shields.io/github/license/avivsinai/bitbucket-cli)](LICENSE)

`bkt` is a stand-alone Bitbucket command-line interface that targets Bitbucket Data Center **and** Bitbucket Cloud. It mirrors the ergonomics of `gh` while remaining provider-pure (no Jenkins coupling) and delivers a consistent JSON/YAML contract for automation.

## Installation

```bash
go install github.com/avivsinai/bitbucket-cli/cmd/bkt@latest
```

Or download pre-built binaries from the [releases page](https://github.com/avivsinai/bitbucket-cli/releases/latest).

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

## Testing

`go test ./...` runs fast smoke coverage that wires the CLI against an in-memory Bitbucket mock (see `pkg/cmd/smoke/cli_smoke_test.go`). Extend that harness as new regression scenarios emerge.

## Support

- **Questions / Ideas**: Open a [GitHub Discussion](https://github.com/avivsinai/bitbucket-cli/discussions)
- **Bug Reports**: File an [issue](https://github.com/avivsinai/bitbucket-cli/issues/new?template=bug_report.md)
- **Security Reports**: Email [security@avivsinai.dev](mailto:security@avivsinai.dev)

## Roadmap highlights

- Device authorization flow for Bitbucket Cloud workspaces.
- Declarative context apply (`bkt context apply`) from YAML manifests.
- Native shell completions and plugin discovery.
- Extended telemetry exporters (OpenTelemetry traces for API calls).

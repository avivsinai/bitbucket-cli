# bkt – Bitbucket CLI

<p align="center"><em>Bitbucket Cloud & Data Center workflows for developers, coding agents, and automation-first teams.</em></p>

[![CI](https://github.com/avivsinai/bitbucket-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/avivsinai/bitbucket-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/avivsinai/bitbucket-cli?cache=none)](https://github.com/avivsinai/bitbucket-cli/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/avivsinai/bitbucket-cli?cache=none)](https://goreportcard.com/report/github.com/avivsinai/bitbucket-cli)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/avivsinai/bitbucket-cli/badge)](https://scorecard.dev/viewer/?uri=github.com/avivsinai/bitbucket-cli)
[![Go Reference](https://pkg.go.dev/badge/github.com/avivsinai/bitbucket-cli.svg)](https://pkg.go.dev/github.com/avivsinai/bitbucket-cli)
[![License](https://img.shields.io/github/license/avivsinai/bitbucket-cli?cache=none)](LICENSE)

`bkt` is a stand-alone Bitbucket command-line interface that targets Bitbucket Data Center **and** Bitbucket Cloud. It mirrors the ergonomics of `gh` and delivers a consistent JSON/YAML contract for automation.

**Built for AI & automation:** Drop `bkt` into Claude Code, Codex and other coding agents, or shell scripts and they inherit structured output, predictable flags, and safe defaults—no glue code required.

## Installation

```bash
go install github.com/avivsinai/bitbucket-cli/cmd/bkt@latest
```

This installs `bkt` to `$GOPATH/bin` (or `$HOME/go/bin` by default). Ensure the directory is in your `$PATH`.

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

#### Bitbucket Data Center

```bash
# Guided flow: opens browser to create token
bkt auth login https://bitbucket.mycorp.example --web

# Or provide credentials directly
bkt auth login https://bitbucket.mycorp.example --username alice --token <PAT>
```

Create a **Personal Access Token (PAT)** in Bitbucket Data Center:
1. Go to **Profile picture → Manage account → Personal access tokens**
2. Click **Create a token**
3. Grant permissions: **Repository Read**, **Repository Write**, **Project Read**
4. Copy the token (you won't see it again)

#### Bitbucket Cloud

```bash
# Guided flow: opens browser to create token
bkt auth login https://bitbucket.org --kind cloud --web

# Or provide credentials directly
bkt auth login https://bitbucket.org --kind cloud --username <email> --token <api-token>
```

Create an **API token** in Bitbucket Cloud:
1. Go to **Personal settings → API tokens** ([direct link](https://bitbucket.org/account/settings/api-tokens/))
2. Click **Create token**
3. Grant scopes:
   - **Repositories**: Read (and Write if creating branches)
   - **Pull requests**: Read, Write
   - **Account**: Read (for commands that fetch user info)
4. Copy the generated token

> **Important:** Use your **Atlassian account email** as the username (not your Bitbucket username).

<details>
<summary>Legacy: App passwords (deprecated)</summary>

App passwords are deprecated. New app passwords cannot be created since September 2025, and existing ones will stop working June 2026. If you have an existing app password:

```bash
bkt auth login https://bitbucket.org --kind cloud --username <bitbucket-username> --token <app-password>
```

Note: For app passwords, use your **Bitbucket username** (not email).

</details>

#### Credential storage

Access tokens are stored in your OS keychain (Keychain Access on macOS, Windows Credential Manager, or
Secret Service/KWallet on Linux) while host metadata lives in
`$XDG_CONFIG_HOME/bkt/config.yml`. Pass `--allow-insecure-store` (or set
`BKT_ALLOW_INSECURE_STORE=1`) to permit the encrypted file backend on systems
without a native keychain.

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
bkt repo browse --project DATA --repo platform-api
bkt repo clone platform-api --project DATA --ssh
```

`repo list`/`repo view` automatically target the right REST API for your active context: Data Center uses `/rest/api/1.0/projects/{projectKey}/repos`, while Cloud uses `/2.0/repositories/{workspace}`.

### 4. Pull request workflows

```bash
bkt pr list --state OPEN --limit 10
bkt pr create --title "feat: cache" --source feature/cache --target main --reviewer alice
bkt pr merge 42 --message "merge: feature/cache"
bkt pr checks 42                              # Show build/CI status
bkt pr checks 42 --wait                       # Wait for builds to complete
bkt pr checks 42 --wait --timeout 5m          # Wait with timeout
bkt pr checks 42 --wait --max-interval 1m     # Custom backoff cap
```

The CLI wraps Bitbucket pull-request endpoints for creation, listing, review, and merge operations. The `checks` command displays build status with color-coded output (green for success, red for failure, yellow for in-progress) and supports polling until all builds complete. Polling uses exponential backoff with jitter to avoid overwhelming the API during long builds.

### 5. Issue tracking (Bitbucket Cloud only)

```bash
bkt issue list --state open --kind bug           # List open bugs
bkt issue view 42 --comments                     # View issue with comments
bkt issue create -t "Login broken" -k bug -p major
bkt issue edit 42 --assignee "{abc-123}" --priority critical
bkt issue close 42                               # Close an issue
bkt issue reopen 42                              # Reopen a closed issue
bkt issue comment 42 -b "Fixed in v1.2.0"        # Add a comment
bkt issue status                                 # Show your assigned/created issues
```

Note: The issue tracker is only available for Bitbucket Cloud. Bitbucket Data Center uses Jira for issue tracking.

### 6. Branch, permission, webhook, pipeline, and extension management

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

- **Questions / Ideas**: File an [issue](https://github.com/avivsinai/bitbucket-cli/issues/new?template=feature_request.md)
- **Bug Reports**: File an [issue](https://github.com/avivsinai/bitbucket-cli/issues/new?template=bug_report.md)

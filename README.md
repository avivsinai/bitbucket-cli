# bkt – Bitbucket CLI

[![CI](https://github.com/avivsinai/bitbucket-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/avivsinai/bitbucket-cli/actions/workflows/ci.yml)
[![OpenSSF Scorecard](https://img.shields.io/ossf-scorecard/github.com/avivsinai/bitbucket-cli?label=openssf%20scorecard)](https://scorecard.dev/viewer/?uri=github.com/avivsinai/bitbucket-cli)
[![Go Reference](https://pkg.go.dev/badge/github.com/avivsinai/bitbucket-cli.svg)](https://pkg.go.dev/github.com/avivsinai/bitbucket-cli)

`bkt` is a stand-alone Bitbucket command-line interface that targets Bitbucket Data Center first and is ready for Bitbucket Cloud. It mirrors the ergonomics of `gh` while remaining provider-pure (no Jenkins coupling) and delivers a consistent JSON/YAML contract for automation.

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
bkt repo view platform-api
bkt repo create data-pipeline --description "Data ingestion" --project DATA
bkt repo clone platform-api --project DATA --ssh
```

`repo list`/`repo view` call the Bitbucket Data Center REST API (`/rest/api/1.0/projects/{projectKey}/repos`) to enumerate repositories and surface clone/web URLs for scripting.

### 4. Pull request workflows

```bash
bkt pr list --state OPEN --limit 10
bkt pr create --title "feat: cache" --source feature/cache --target main --reviewer alice
bkt pr merge 42 --message "merge: feature/cache"
```

The CLI wraps Bitbucket pull-request endpoints for creation, listing, review, and merge operations.

### 5. Branch, permission, webhook, and pipeline management

```bash
bkt branch list
bkt branch create release/1.9 --from main
bkt perms repo list --project DATA --repo platform-api
bkt webhook create --name "CI" --url https://ci.example.com/hook --event repo:refs_changed
bkt pipeline run --workspace myteam --repo api --ref main --var ENV=staging
bkt status pipeline {pipeline-uuid}
bkt status rate-limit
```

Branch utilities use Bitbucket's Branch Utils REST API for listing, creation, deletion, and default updates. Permission and webhook commands map to their respective REST endpoints for consistent automation.

### Structured output

Every command supports the global `--json` and `--yaml` flags for automation-ready output.

## Support

See [SUPPORT.md](SUPPORT.md) for available support channels and response times.

## Roadmap highlights

- Device authorization flow for Bitbucket Cloud workspaces.
- Declarative context apply (`bkt context apply`) from YAML manifests.
- Native shell completions and plugin discovery.
- Extended telemetry exporters (OpenTelemetry traces for API calls).

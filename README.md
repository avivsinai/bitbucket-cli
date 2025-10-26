# bkt – Bitbucket CLI

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

### 1. Authenticate against Bitbucket Data Center

```bash
bkt auth login https://bitbucket.mycorp.example --username alice --token <PAT>
```

The login flow verifies credentials via the Data Center REST API and persists them in `$XDG_CONFIG_HOME/bkt/config.yml`.

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
```

`repo list` and `repo view` use the Bitbucket Data Center REST API (`/rest/api/1.0/projects/{projectKey}/repos`) to enumerate repositories and surface clone/web URLs for scripting.

### 4. Inspect build statuses

```bash
bkt status commit <sha>
bkt status pr 42 --repo platform-api
```

Commit and pull request status inspection reads from `/rest/build-status/1.0/commits/{sha}` so you can see CI results without leaving the terminal.

### Structured output

Every command supports the global `--json` and `--yaml` flags for automation-ready output.

## Roadmap highlights

- Bitbucket Cloud support (API token auth, workspaces, pipelines).
- Repository creation, clone helpers, and `repo browse` deep links.
- Pull request management (`list`, `view`, `create`, `merge`).
- Webhook administration and pipeline orchestration.

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

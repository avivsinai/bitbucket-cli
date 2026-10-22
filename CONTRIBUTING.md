# Contributing to bkt

Thanks for your interest in making bkt better! We welcome issues, pull
requests, docs fixes, and release automation improvements.

## Ground rules

- Be respectful and follow our [Code of Conduct](CODE_OF_CONDUCT.md).
- We do **not** require a CLA. By contributing you agree to the
  [Developer Certificate of Origin (DCO)](https://developercertificate.org/);
  sign your commits with `git commit -s`.
- Include tests when you add or change behavior.
- For non-trivial changes, open an issue first so we can agree on direction.

## Build and test

Go 1.26 or later. The Make targets also work on Windows under GNU Make.

```bash
make build                 # ./bin/bkt
make test                  # go test ./...
make fmt                   # go fmt ./...
make lint                  # golangci-lint
make tidy                  # go mod tidy
make generate-skill        # regenerate skills/bkt/rules from command help
make sync-skills           # copy skills/bkt to the .claude/ and .agents/ mirrors
make check-skills          # verify the mirrors match
make nix-update-vendor-hash  # after any go.mod/go.sum change
```

`go run ./cmd/bkt --help` is the quickest smoke test. `go test ./...` also runs
smoke tests that drive the CLI against an in-memory Bitbucket mock
(`pkg/cmd/smoke/cli_smoke_test.go`). Use `BKT_CONFIG_DIR` pointing at a scratch
directory when you try commands, so your real configuration stays untouched.

### Test conventions

- Tests live next to the code they exercise (`foo.go` → `foo_test.go`).
- Prefer table-driven tests for functions with several input/output cases.
- Use `httptest.NewServer` for API clients; tests make no real network calls.
- Use `t.TempDir()`, `t.Setenv()`, and `t.Cleanup()` for isolation.
- Name tests after the scenario, not only the function.

## Project layout

```
cmd/bkt/          CLI entry point
cmd/docgen/       generates skills/bkt/rules from command help
internal/bktcmd/  Main() wiring (factory + root command)
internal/build/   version metadata (set via ldflags)
internal/config/  contexts and hosts in $XDG_CONFIG_HOME/bkt
internal/remote/  git remote parsing
internal/secret/  OS keychain and encrypted file store
pkg/cmd/          Cobra commands (auth, repo, pr, ...)
pkg/cmdutil/      shared command helpers and factory
pkg/bbdc/         Bitbucket Data Center client
pkg/bbcloud/      Bitbucket Cloud client
pkg/oauth/        Bitbucket Cloud OAuth flow and consumer
pkg/httpx/        shared HTTP client, retries, token refresh
pkg/format/       output rendering
skills/bkt/       canonical agent skill (mirrored to .claude/ and .agents/)
```

`skills/bkt/` is canonical. After you edit it, or change command help, run
`make generate-skill sync-skills` and commit the result; CI fails when the
mirrors or generated rules drift.

## Pull requests

1. Fork the repository and create a feature branch.
2. Use conventional commits (`feat:`, `fix:`, `docs:`, `ci:`, `chore:`).
3. Update docs and add a `CHANGELOG.md` entry under `Unreleased` for
   user-facing changes.
4. Run `make fmt test build`. CI runs on Linux and Windows.
5. Open a pull request with a short summary, testing notes, and a terminal
   capture for CLI output changes.

We aim to respond within two business days.

## Releases

Maintainers release with `./scripts/release.sh X.Y.Z`, which opens a release
PR. After it merges, CI tags the commit and publishes binaries, packages, and
SBOMs. See [`docs/RELEASE.md`](docs/RELEASE.md).

## Community roles

Governance lives in [GOVERNANCE.md](GOVERNANCE.md). If you want to become a
maintainer, open an issue so we can talk about expectations.

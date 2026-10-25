# bitbucket-cli

This is the master agent instruction file for this repository. Keep repository policy here. `AGENTS.md` exists only as a Codex compatibility shim and should contain only Codex-specific notes.

## Project Structure & Module Organization

- `cmd/bkt/` contains the CLI entry point; the binary executes `internal/bktcmd`.
- `internal/` hosts non-exported wiring, including configuration (`internal/config`) and build metadata.
- `pkg/` holds reusable packages consumed by Cobra commands such as `pkg/cmd/repo` and `pkg/bbdc`.
- Tests live alongside Go packages; add `_test.go` files next to the implementation.

## Build, Test, and Development Commands

- `make build` compiles the CLI with `go build ./cmd/bkt`.
- `make test` or `go test ./...` runs the full unit test suite.
- `make fmt` formats the codebase with `go fmt ./...`.
- `make tidy` syncs module dependencies with `go mod tidy`.
- `go run ./cmd/bkt --help` is the quick local smoke test.
- Windows-native development is supported: the same Make targets run under GNU
  Make with cmd.exe-pinned recipes, and the `build-windows` CI job is the
  source of truth for what works on Windows.

## Coding Style & Naming Conventions

- Follow standard Go conventions: tabs for indentation, PascalCase for exported identifiers, camelCase for private helpers.
- Keep package names short and lowercase; command packages live under `pkg/cmd/<topic>`.
- Prefer contextual errors such as `fmt.Errorf("action: %w", err)`.
- Run `go fmt` before committing; add comments only when the logic is not obvious.

## Testing Guidelines

- Prefer table-driven tests in `_test.go` files named `Test<Subject>`.
- Use golden files under `testdata/` when CLI output snapshots add value.
- Cover flag parsing, API adapters, and error paths; mock HTTP interactions where needed.
- Use `go test ./pkg/...` for faster package-focused iteration.

## Release Contract

- Release from `master` only through `./scripts/release.sh X.Y.Z` and the resulting release PR; do not create manual tags or GitHub releases.
- A push to `master` updates the AvivSinai marketplace immediately for the `bkt` skill.
- Keep `CHANGELOG.md` and skill/plugin metadata on one version in the release commit; after the release PR merges, CI validates the merged commit, creates the tag with the release GitHub App, publishes GitHub/Homebrew artifacts, and uses the committed changelog entry as the GitHub release notes. Releases are immutable: a failed publish is fixed with the next patch, never by re-running a tag.
- `master` is ruleset-protected (PR only, required checks, squash/linear). Never merge past red or pending checks, or push a tag, unless the maintainer declares a crisis.
- Bitbucket Cloud browser OAuth (`bkt auth login --kind cloud --web`) is
  zero-config in every build — release binaries, Homebrew core, `go install`,
  Nix — because bkt's OAuth consumer lives in source (`pkg/oauth/cloud.go`), as
  the GitHub CLI and Git Credential Manager ship theirs. A native app's
  consumer secret is public-client material (RFC 8252 §8.5), not a leak.
  Moving it out of source or into CI secrets is a breaking auth change (#179
  broke it, #190 restored it for release binaries only), never generic
  hardening. Keep the Bitbucket consumer marked public (no
  `client_credentials` grant) and keep the same consumer: refresh tokens are
  bound to the consumer that issued them.
- See `docs/RELEASE.md` for the full release handbook.

## Commit & Pull Request Guidelines

- Use conventional commits such as `feat:`, `fix:`, `docs:`, `ci:`, or `chore:`.
- Keep commits focused and descriptive; reference issues in the body when useful.
- Pull requests should include a short summary plus testing notes such as `go test ./...`.

## Security & Configuration Tips

- Never commit real credentials; the CLI reads tokens from `$XDG_CONFIG_HOME/bkt/config.yml`.
- Use environment overrides such as `BKT_CONFIG_DIR` for sandbox testing without touching primary config.

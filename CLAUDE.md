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
- Keep `CHANGELOG.md` and skill/plugin metadata on one version in the release commit; after the release PR merges, CI validates the merged commit, creates the tag, publishes GitHub/Homebrew artifacts, and uses the committed changelog entry as the GitHub release notes.
- See `docs/RELEASE.md` for the full release handbook.

## Commit & Pull Request Guidelines

- Use conventional commits such as `feat:`, `fix:`, `docs:`, `ci:`, or `chore:`.
- Keep commits focused and descriptive; reference issues in the body when useful.
- Pull requests should include a short summary plus testing notes such as `go test ./...`.

## Security & Configuration Tips

- Never commit real credentials; the CLI reads tokens from `$XDG_CONFIG_HOME/bkt/config.yml`.
- Use environment overrides such as `BKT_CONFIG_DIR` for sandbox testing without touching primary config.

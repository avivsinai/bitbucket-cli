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

## Releasing
- See [docs/RELEASE.md](./docs/RELEASE.md) for the full release handbook.
- **Do not** manually create GitHub releases—CI handles this automatically when you push a tag.
- Process: update `CHANGELOG.md`, commit, `git tag vX.Y.Z`, `git push origin master --tags`.

## Commit & Pull Request Guidelines
- Commit messages follow the conventional prefix style (`feat:`, `fix:`, `docs:`) as seen in `feat: initial commit`.
- Keep commits focused and descriptive; reference issues with `Refs #123` in the body when applicable.
- Pull requests should include: clear summary, testing notes (`go test ./...`), and any screenshots for CLI output changes.
- Ensure CI (when configured) passes before requesting review; request at least one reviewer familiar with Bitbucket DC flows.

## Security & Configuration Tips
- Never commit real credentials; the CLI reads tokens from `$XDG_CONFIG_HOME/bkt/config.yml`.
- Use environment overrides (e.g., `BKT_CONFIG_DIR`) for sandbox testing without touching primary config.


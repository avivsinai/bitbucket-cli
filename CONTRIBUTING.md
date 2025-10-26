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

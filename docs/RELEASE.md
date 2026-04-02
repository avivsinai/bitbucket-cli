# Release handbook

1. **Prepare**
   - Ensure `master` is green and up to date.
   - Update `CHANGELOG.md` under `Unreleased`.
   - Do not tag manually. Use the release PR script so changelog + skill/plugin metadata move together.
   - Run `make release RELEASE_VERSION=X.Y.Z` or `./scripts/release.sh X.Y.Z` from `master`.
     - This verifies the worktree is clean and at `origin/master`.
     - It creates `release/vX.Y.Z`.
     - It turns `Unreleased` into `## [X.Y.Z] - YYYY-MM-DD`.
     - It bumps `skills/*/SKILL.md`, `.claude-plugin/plugin.json`, and `.codex-plugin/plugin.json`.
     - It runs `./scripts/check-release-version.sh`, `make fmt`, `make test`, and `make build`.
     - It commits `chore(release): vX.Y.Z`, pushes the release branch, opens a PR, and enables squash auto-merge by default.

2. **Merge**
   - Let the release PR merge via auto-merge after checks pass.
   - The merge commit title must stay `chore(release): vX.Y.Z` so CI can detect it.

3. **Automation** (all handled by `release.yml` after the release PR merges)
   - The workflow validates the merged release commit on `master`.
   - It creates `vX.Y.Z` only after the merged commit passes verification.
   - GoReleaser builds:
     - Linux, macOS, and Windows binaries (amd64 + arm64)
     - Checksums (`bkt_${VERSION}_checksums.txt`)
     - SBOMs (`sbom-${VERSION}.cyclonedx.json` via Syft)
   - Artifacts are uploaded to the GitHub Release page.
   - The `bkt` skill publish workflow runs from the CI-created tag.

4. **Post-release**
   - Verify the release artifacts and SBOMs.
   - Announce the release in the `CHANGELOG.md` (already updated) and discussions.
   - `CHANGELOG.md` already contains a fresh empty `Unreleased` section because the release script preserves it.

## Guardrails

- Treat `scripts/release.sh` as the only supported release entrypoint.
- `scripts/check-release-version.sh vX.Y.Z` now validates both metadata versions and the matching `CHANGELOG.md` heading.
- If a release PR merges but the tag publish fails verification, fix forward with the next patch version instead of rewriting the failed tag.

## Dry runs

Use `goreleaser release --clean --snapshot` to exercise the publishing pipeline without
publishing artifacts. For local release-prep rehearsal without pushing, run:

```bash
make release RELEASE_VERSION=X.Y.Z RELEASE_NO_AUTO_MERGE=1
```

## Release cadence

We aim for monthly releases, with additional patch releases as needed for
security or regression fixes.

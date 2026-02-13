# Release handbook

1. **Prepare**
   - Ensure `master` is green.
   - Update `CHANGELOG.md` with the upcoming version under "Unreleased".
   - Run `make fmt test build`.
   - Regenerate the SBOM locally if possible: `make sbom`.

2. **Version bump**
   - Determine the next semantic version based on the changes.
   - Tag the release: `git tag vX.Y.Z` and `git push origin vX.Y.Z`.

3. **Automation** (all handled by `release.yml` on tag push)
   - GoReleaser builds:
     - Linux, macOS, and Windows binaries (amd64 + arm64)
     - Checksums (`bkt_${VERSION}_checksums.txt`)
     - SBOMs (`sbom-${VERSION}.cyclonedx.json` via Syft)
   - Artifacts are uploaded to the GitHub Release page.
   - The `bkt` skill is published to the skild registry (auto-incremented version).
   - A PR is created and auto-merged to bump `SKILL.md` to match the published version.

4. **Post-release**
   - Verify the release artifacts and SBOMs.
   - Announce the release in the `CHANGELOG.md` (already updated) and discussions.
   - Cut a new `Unreleased` section in the changelog for the next cycle.

## Dry runs

Use `goreleaser release --clean --snapshot` to exercise the pipeline without
publishing artifacts.

## Release cadence

We aim for monthly releases, with additional patch releases as needed for
security or regression fixes.

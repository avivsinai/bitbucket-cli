# Release handbook

1. **Prepare**
   - Ensure `master` is green and up to date.
   - Update `CHANGELOG.md` under `Unreleased`.
   - Do not tag manually. Use the release PR script so changelog + skill/plugin metadata move together.
   - Run `make release RELEASE_VERSION=X.Y.Z` or `./scripts/release.sh X.Y.Z` from `master`.
     - This verifies the worktree is clean and at `origin/master`.
     - It creates `release/vX.Y.Z`.
     - It turns `Unreleased` into `## [X.Y.Z] - YYYY-MM-DD`.
     - It bumps `skills/*/SKILL.md`, syncs the committed `.claude/skills/` and
       `.agents/skills/` mirrors, and updates `.claude-plugin/plugin.json` and
       `.codex-plugin/plugin.json`.
     - It runs `./scripts/check-release-version.sh`, `make check-skills`, `gofmt`, `go vet`, `make test`, and `make build`.
     - It commits `chore(release): vX.Y.Z`, pushes the release branch, opens a PR, and enables squash auto-merge by default.

2. **Merge**
   - Let the release PR merge via auto-merge after checks pass.
   - The merge commit title must stay `chore(release): vX.Y.Z` so CI can detect it.

3. **Automation** (all handled by `release.yml` after the release PR merges)
   - The workflow validates the merged release commit on `master`.
   - It creates `vX.Y.Z` only after the merged commit passes verification,
     using a token from the release GitHub App (`vars.RELEASE_APP_CLIENT_ID`,
     `secrets.RELEASE_APP_PRIVATE_KEY`). The `v*` tag ruleset lets only that
     App (and the admin, in a declared crisis) create tags.
   - GoReleaser builds:
     - Linux, macOS, and Windows binaries (amd64 + arm64)
     - Checksums (`bkt_${VERSION}_checksums.txt`)
     - SBOMs (`sbom-${VERSION}.cyclonedx.json` via Syft)
   - Artifacts are uploaded to the GitHub Release page.
   - GoReleaser generates WinGet manifests. After the initial package version is
     accepted into the catalog, it attempts to open an update PR from
     `avivsinai/winget-pkgs` to `microsoft/winget-pkgs` when the publishing
     secret is configured.
   - The workflow emits GitHub build provenance attestations for the released files.
   - The `Publish skills` job in `release.yml` publishes the `bkt` skill as part
     of the same release run. Separately, `notify-marketplace.yml` refreshes the
     marketplace on every push to `master`. `publish-skill.yml` is a manual
     `workflow_dispatch` fallback for re-publishing an existing tag's skill.

4. **Post-release**
   - Verify the release artifacts and SBOMs.
   - Verify that the WinGet update PR was opened and passed upstream validation.
     GoReleaser reports WinGet publishing errors without failing the release, so
     a green release workflow is not proof that the package update landed.
   - Announce the release via the `CHANGELOG.md` entry (already updated), which
     CI uses as the GitHub Release notes.
   - `CHANGELOG.md` already contains a fresh empty `Unreleased` section because the release script preserves it.

## Guardrails

- Treat `scripts/release.sh` as the only supported release entrypoint.
- bkt's Bitbucket Cloud OAuth consumer lives in `pkg/oauth/cloud.go`, so every
  build, including Homebrew core's source build, supports
  `bkt auth login --kind cloud --web`. Releases need no OAuth secrets.
- Homebrew core autobumps `bitbucket-cli` a few hours after each tag. Do not
  open a manual bump PR; if no BrewTestBot PR appears within a day, use
  `brew bump-formula-pr` and follow homebrew-core's AI-disclosure rules (no AI
  co-author trailers).
- Refresh `flake.nix`'s `vendorHash` whenever `go.mod` or `go.sum` changes; do not wait until release time because Nix CI runs on pull requests. Use `make nix-update-vendor-hash` after dependency bumps.
- Submit the first package version manually with `wingetcreate new` and wait for
  its `microsoft/winget-pkgs` PR to be accepted before advertising the
  `winget install AvivSinai.Bitbucket-CLI` command. GoReleaser then handles
  updates; it cannot create the initial catalog entry.
- WinGet publishing requires an `avivsinai/winget-pkgs` fork and a classic PAT
  stored only as the `WINGET_GITHUB_TOKEN` GitHub Actions repository secret.
  Never put its value in repository files, workflow commands, arguments, or
  logs. When the secret is absent, the workflow emits a warning and skips only
  the WinGet upload; the rest of the release continues.
- macOS binaries are signed with a Developer ID and notarized by
  `scripts/codesign-macos.sh` when these repository secrets are set:
  `APPLE_TEAM_ID`, `APPLE_DEVELOPER_ID_P12_BASE64` (a "Developer ID
  Application" certificate, its key and the Developer ID intermediate, base64
  of the .p12),
  `APPLE_DEVELOPER_ID_P12_PASSWORD`, `APPLE_NOTARY_ISSUER_ID`,
  `APPLE_NOTARY_KEY_ID`, and `APPLE_NOTARY_KEY_P8_BASE64` (an App Store Connect
  API key, base64 of the .p8). Set all six or none: with none, the release
  warns and signs ad-hoc; with some, it fails. Ad-hoc binaries get a new
  Keychain partition ID (`cdhash:`) per build, so users must approve Keychain
  access again after each upgrade; Developer ID binaries share one
  `teamid:` partition ID. Keep the signing identifier
  `io.github.avivsinai.bitbucket-cli`: a changed identifier or team makes every
  user approve again.
- `scripts/check-release-version.sh vX.Y.Z` now validates both metadata versions and the matching `CHANGELOG.md` heading.
- Releases are immutable: a published release and its tag cannot change. If any
  publish step fails (GoReleaser, tap, WinGet), fix forward with the next patch
  version; there is no re-run path for an existing tag.
- `master` is protected by a ruleset: PR only, required checks, squash/linear,
  no force-push. The admin bypass (PR merges only) is a crisis escape hatch;
  agents never merge past red or pending checks unless the maintainer declares
  a crisis.

## Dry runs

Use `goreleaser release --clean --snapshot` to exercise the publishing pipeline without
publishing artifacts. For local release-prep rehearsal without pushing, run:

```bash
make release RELEASE_VERSION=X.Y.Z RELEASE_NO_AUTO_MERGE=1
```

## Release cadence

We aim for monthly releases, with additional patch releases as needed for
security or regression fixes.

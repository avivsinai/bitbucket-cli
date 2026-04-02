# Versioning

bkt follows [Semantic Versioning](https://semver.org/).

- MAJOR versions introduce breaking changes (e.g., CLI flag removals).
- MINOR versions add functionality in a backwards compatible manner.
- PATCH versions include backwards compatible bug fixes.

Release PRs carry one version across `CHANGELOG.md` and skill/plugin metadata.
After the release PR merges, CI creates the `vX.Y.Z` tag and publishes binaries
via GoReleaser. The changelog summarises the changes for each release.

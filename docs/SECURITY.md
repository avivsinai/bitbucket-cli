# Security operations playbook

This document expands on the top-level [SECURITY.md](../SECURITY.md) with
implementation details.

## Secrets handling

- bkt never stores plaintext credentials under version control. Tokens are
  stored in the OS keychain (macOS Keychain Access, Windows Credential Manager,
  Linux Secret Service) or an encrypted local file backend. Host metadata lives
  in `$XDG_CONFIG_HOME/bkt/config.yml` with permissions `0600`. The `BKT_TOKEN`
  environment variable provides runtime-only injection for CI/headless use.
- For development, set `BKT_CONFIG_DIR` to a throwaway directory.
- Never commit test credentials. Use environment variables or the
  `internal/config/testdata` fixtures when unit testing.

## Dependency updates

- Dependabot is enabled for Go modules and GitHub Actions (`.github/dependabot.yml`).
- Run `go list -m -u all` periodically to spot stale modules.
- CI runs [OpenSSF Scorecard](https://github.com/ossf/scorecard) weekly.

## Supply chain

- Release artifacts are built with GoReleaser (`goreleaser.yaml`).
- Each release publishes a checksum manifest and an SBOM generated via Syft.
- Binaries are signed ad-hoc on macOS via the GoReleaser post-build hook.

## Incident response

1. Triage the report and reproduce the issue.
2. Assign a severity (CVSS) and determine the affected versions.
3. Prepare a patch on a private branch. Request a security review from another
   maintainer.
4. Cut a release PR with the fix, update `CHANGELOG.md` with mitigation steps,
   and let CI create the release tag after the merge validates.
5. Notify the reporter and disclose publicly within seven days of the fix.

## Contact

Email **avivsinai@gmail.com**. We prefer coordinated disclosure.

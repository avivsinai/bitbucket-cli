# Security Statement for bkt for Bitbucket

**Effective date**: 2026-04-04

## Overview

`bkt` is a standalone command-line interface for Bitbucket Cloud. It runs
locally on the user's machine or CI runner. `bkt` does not operate a
vendor-hosted backend and does not proxy Bitbucket API traffic through
provider-operated infrastructure.

## Architecture and data flow

`bkt` communicates directly with Bitbucket Cloud over HTTPS.

Normal request flow:

1. The end user runs `bkt` locally or in CI.
2. `bkt` reads credentials from the local environment or local secret store.
3. `bkt` sends HTTPS requests directly to Bitbucket Cloud REST endpoints.
4. API responses are rendered locally in terminal output or structured
   machine-readable output (JSON/YAML).

The provider does not receive or store Bitbucket API payloads during normal
product operation.

## Authentication and secret handling

For Bitbucket Cloud, `bkt` uses Atlassian API tokens with scopes created by
the end user at id.atlassian.com.

Credential sources (in order of precedence):

- `BKT_TOKEN` environment variable for runtime-only injection
- OS keychain integrations: macOS Keychain Access, Windows Credential Manager,
  Linux Secret Service / compatible keyring backends
- Encrypted local file backend when explicitly enabled via
  `BKT_KEYRING_PASSPHRASE` for environments without a native keychain

Credential handling properties:

- Credentials are stored only on the end user's device or CI runner.
- Credentials are never sent to provider-operated systems.
- The CLI supports least-privilege token usage by allowing users to create
  tokens with only the scopes needed for the commands they intend to run.
- Command-line token flags should be avoided where possible because shell
  history and process lists may expose them. The CLI warns when `--token` is
  used on the command line.

Required scopes (minimum):

- Account: Read (`read:user:bitbucket`) — required for authentication
- Repositories: Read, Write — for repo and branch commands
- Pull requests: Read, Write — for PR commands
- Issues: Read, Write — for issue commands (optional)

## Local storage

`bkt` stores host/context metadata locally in the user configuration directory
(`$XDG_CONFIG_HOME/bkt/` or OS equivalent). Secrets are stored separately
using the local secret storage mechanisms described above.

`bkt` does not persist Bitbucket repository content or customer data on
provider-operated systems.

## Telemetry, analytics, and third parties

`bkt` does not include telemetry, analytics, advertising, or tracking services.

`bkt` does not send usage data, command history, repository metadata, or
customer content to any third-party analytics or observability provider.

## Transport security

All API communication uses HTTPS (TLS 1.2+) via Go's standard `net/http`
library. Certificate verification is enforced by default. `bkt` does not
require plaintext HTTP for normal cloud operation.

## Logging

`bkt` does not operate a vendor-hosted logging pipeline. Any command output,
redirected output, or CI logs are created and controlled by the end user in
their own environment.

## Supply chain

- Release binaries are built with GoReleaser in GitHub Actions with full
  provenance.
- Each release includes SHA-256 checksums and a CycloneDX SBOM generated
  by Syft.
- Dependencies are monitored by Dependabot and the OpenSSF Scorecard.

## Open source

`bkt` is 100% open-source under the MIT License. The complete source code is
available for public review and security auditing at:
https://github.com/avivsinai/bitbucket-cli

## Vulnerability reporting

Please report security vulnerabilities according to:
https://github.com/avivsinai/bitbucket-cli/blob/master/SECURITY.md

Contact: avivsinai@gmail.com

Security reports are acknowledged within two business days. Confirmed
vulnerabilities are triaged, fixed in a release, and disclosed after a fix
is available.

## Important note on Atlassian credentials

`bkt` currently relies on user-supplied Atlassian credentials for Bitbucket
Cloud access. Atlassian's cloud security requirements place restrictions on
apps that request or store user API tokens. Marketplace approval may require
additional review or an explicit exception from Atlassian.

# Privacy & Security Tab — Prepared Answers

Answers for the Atlassian Marketplace Privacy & Security tab during submission.

## Contact

**Security contact email**: avivsinai@gmail.com

**Security policy URL**: https://github.com/avivsinai/bitbucket-cli/blob/master/SECURITY.md

**Privacy policy URL**: https://github.com/avivsinai/bitbucket-cli/blob/master/docs/marketplace/PRIVACY.md

## Data Storage

**Does your app store customer data?**
> No. bkt is a local CLI tool. No data is stored outside the user's machine.

**Does your app store data in any databases?**
> No. bkt has no server component and no databases.

**Where is customer data stored?**
> N/A — client-side application only. Authentication tokens are stored in the
> user's native OS keychain (macOS Keychain, Windows Credential Manager, or
> Linux Secret Service).

## Data Transmission

**Does your app transmit customer data?**
> bkt transmits API requests directly from the user's machine to the Bitbucket
> Cloud API (api.bitbucket.org) over HTTPS. No data is transmitted to any
> third-party servers.

**Does your app transmit data to third parties?**
> No. All communication is between the user's machine and Bitbucket Cloud.

## Analytics & Tracking

**Does your app collect analytics or usage data?**
> No. bkt has zero telemetry, zero analytics, and zero tracking.

**Does your app use cookies or local storage?**
> No. bkt is a CLI tool and does not use a browser context.

## Hosting & Infrastructure

**Where is your app hosted?**
> N/A — bkt is a client-side CLI binary installed on the user's machine.
> There is no hosted infrastructure.

**What cloud provider do you use?**
> None. bkt has no server component.

**Are backups performed?**
> N/A. There is no server-side data to back up.

## Authentication & Authorization

**Does your app access Atlassian Personal Access Tokens (PATs), user account
passwords, or another type of shared secret?**
> Yes. End users provide Bitbucket-scoped Atlassian API tokens. Tokens are
> stored locally only and are not sent to vendor infrastructure. Legacy app
> passwords may still exist for some users until Atlassian fully sunsets them.

**Provide a justification for the scopes requested by the app.**
> bkt is a general-purpose Bitbucket Cloud CLI. Required scopes depend on the
> commands the customer intends to run. The minimum recommended scopes are:
> - `read:user:bitbucket` for identity verification
> - Repository read/write for repository operations
> - Pull request read/write for PR operations
> - Issue read/write only when issue commands are used
>
> Customers are instructed to create least-privileged tokens matching the
> commands they intend to use.

## End-User Data Processing

**Does your app process End-User Data outside of Atlassian products and
services or outside the end-user's browser?**
> Yes. The CLI processes requested Bitbucket API responses locally on the
> end-user's machine or CI runner. There is no vendor-hosted backend. Data
> processed may include repository/workspace metadata, branch names, pull
> request metadata and content, issue metadata and content, pipeline metadata,
> and user/account identifiers returned by the Bitbucket APIs for commands
> explicitly invoked by the user.

**Does your app store End-User Data outside of Atlassian products?**
> No. The provider does not persist customer Bitbucket content on
> provider-operated systems. Local credential/config storage on the end-user
> device is not vendor-side End-User Data storage.

**Does your app log End-User Data?**
> No.

**Does your app share End-User Data with any third party entities?**
> No.

**Does your app process or store End-User Data in logs outside Atlassian?**
> No.

**Does your app share logs that include End-User Data with third parties?**
> No.

## Legal & Compliance

**Is your app a "service provider" as defined under the CCPA?**
> No. The provider does not operate a backend that receives or processes
> customer End-User Data on the customer's behalf.

**Does your app have a Data Processing Agreement (DPA) for customers?**
> No. The provider does not process or store customer End-User Data on
> provider-operated systems.

**Does your app transfer EEA residents' End-User Data outside of the EEA?**
> No. There is no provider-hosted End-User Data storage or processing.

**Have you completed a CAIQ Lite Questionnaire?**
> No.

**Does your app use full disk encryption at-rest for End-User Data stored
outside of Atlassian or the user's browser?**
> bkt does not store customer End-User Data on provider-operated systems.
> Credentials are stored locally using OS keychain facilities (which leverage
> OS-level encryption) or an encrypted local file backend. Device-level full
> disk encryption is the responsibility of the end user.

**Does your app have any compliance certifications?**
> No. Compliance certifications are not applicable to a client-side CLI tool
> with no server infrastructure.

## Open Source

**Is your app open source?**
> Yes. MIT License. Full source code at
> https://github.com/avivsinai/bitbucket-cli

**How are dependencies managed?**
> Dependabot monitors Go modules and GitHub Actions. OpenSSF Scorecard runs
> weekly. Each release includes SHA-256 checksums and a CycloneDX SBOM.

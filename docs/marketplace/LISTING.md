# Atlassian Marketplace Listing

Submission path: **"My app isn't directly installable"** — informational listing
pointing to GitHub releases. **Cloud only** (new Data Center Marketplace
submissions stopped December 16, 2025).

## App Identity

- **App name**: bkt for Bitbucket
- **Tagline**: Automate PRs, pipelines, and repos from the terminal — or let your AI agent do it
- **Summary**: bkt gives developers and AI agents a single CLI for Bitbucket
  Cloud. Create PRs, trigger pipelines, manage repos, and query issues with
  structured JSON/YAML output. Works in terminals, CI pipelines, and coding
  agents. Free and open-source.

## Highlights

### 1. Automate Bitbucket Without Leaving the Terminal

**Summary**: Create pull requests, merge branches, trigger pipelines, and
manage repos with short, memorable commands. Stop clicking through the UI for
tasks you repeat every day.

**Description**: bkt mirrors the ergonomics of GitHub CLI (`gh`) for Bitbucket
Cloud. Every command maps to the Bitbucket REST API with sensible defaults.
Create a context per workspace and switch between them with `bkt context use`.
PRs, repos, branches, webhooks, and pipelines all work the same way.

### 2. Built for AI Agents and Automation

**Summary**: Every command supports --json and --yaml output. Drop bkt into
Claude Code, Codex, shell scripts, or CI pipelines and get predictable,
parseable results with safe defaults — no wrapper scripts needed.

**Description**: bkt is designed for machine consumption first, human
readability second. All list and view commands emit structured JSON or YAML
when flagged. The `bkt api` escape hatch gives raw access to any Bitbucket
REST endpoint with the same structured output contract. Coding agents use bkt
as their Bitbucket interface without parsing human-readable tables.

### 3. Runs Anywhere — Terminals, Pipelines, Agents

**Summary**: Install on macOS, Linux, or Windows via Homebrew, Scoop, or a
single binary download. Add bkt to Bitbucket Pipelines with one curl command.
No Docker images or custom containers required.

**Description**: Add bkt to Bitbucket Pipelines without Docker wrappers or
custom images. Download the pre-built Linux binary in your script block and
run any bkt command. Combine with Bitbucket's built-in pipeline variables
(BITBUCKET_BRANCH, BITBUCKET_COMMIT, BITBUCKET_TAG) for automated workflows
like release PRs, branch cleanup, and status checks.

## Metadata

- **Categories**: Developer Tools, Utilities
- **Pricing**: Free
- **License**: MIT
- **Compatibility**: Bitbucket Cloud

## Links

- **Documentation**: https://github.com/avivsinai/bitbucket-cli#readme
- **Source code**: https://github.com/avivsinai/bitbucket-cli
- **Support**: https://github.com/avivsinai/bitbucket-cli/issues
- **Security contact**: avivsinai@gmail.com
- **Privacy policy**: PRIVACY.md
- **Provider-specific terms**: PROVIDER_SPECIFIC_TERMS.md
- **Security statement**: SECURITY_STATEMENT.md

## Visual Assets (manual upload required)

- [ ] Logo: 144x144 PNG, transparent background
- [ ] Banner: 1120x548 PNG
- [ ] Screenshots: 1840x900 PNG (minimum 3)
  - `bkt pr list` output with color formatting
  - `bkt pipeline run` and status output
  - `bitbucket-pipelines.yml` showing bkt in a pipeline step

## Authentication Approach

> **Risk**: Atlassian's cloud security requirements state that approved apps
> must not collect or store user API tokens. bkt relies on API tokens with
> scopes stored in the local OS keychain. Before submitting, open an ECOHELP
> request to confirm that a non-installable CLI listing is permitted to use
> this auth model, or whether an exemption is needed.

bkt authenticates using Atlassian API tokens with scopes for Bitbucket Cloud.
These are created by the user at id.atlassian.com and stored locally in the OS
keychain (Keychain Access on macOS, Windows Credential Manager, or Secret
Service on Linux). Tokens are transmitted only to the Bitbucket Cloud API over
HTTPS. bkt does not proxy, relay, or store tokens on any third-party server.

The CLI also supports Data Center Personal Access Tokens for users who connect
to self-hosted instances outside the Marketplace listing scope.

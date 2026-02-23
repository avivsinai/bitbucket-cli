# Issue #59: Headless Auth Fix Design

Refs #59

## Problem

`bkt auth login --allow-insecure-store` fails in headless/container environments because:
1. No way to inject a token at runtime without the keyring
2. The encrypted file backend falls through to `keyring.TerminalPrompt` when no passphrase env var is set, which fails without a TTY

## Design

Two changes, modeled after the GitHub CLI (`GH_TOKEN` pattern):

### 1. `BKT_TOKEN` env var (bypass keyring)

Add a `BKT_TOKEN` environment variable that provides a token at runtime with highest precedence, completely bypassing the keyring.

**Token resolution order (new):**
1. `BKT_TOKEN` env var (if set, skip keyring entirely)
2. Keyring lookup (existing behavior)

**Behavior when `BKT_TOKEN` is set:**
- `loadHostToken()` returns the env var value without opening the keyring
- `auth login` / `auth logout` print a warning and exit with error when `BKT_TOKEN` is active (token is externally managed)
- `auth status` shows `BKT_TOKEN` as the token source

**Location:** `pkg/cmdutil/context.go:loadHostToken()`, `pkg/cmd/auth/auth.go`

### 2. Headless guard for file backend (fail fast)

When the file backend is selected, no passphrase is available, and the environment is headless/non-TTY, return an actionable error instead of falling through to `keyring.TerminalPrompt`.

**Error message:**
```
file backend requires a passphrase in headless environments; set BKT_KEYRING_PASSPHRASE (or KEYRING_FILE_PASSWORD) or use BKT_TOKEN to bypass the keyring entirely
```

**Location:** `internal/secret/store.go:configureFileBackend()`

### What we explicitly chose NOT to do

- **Auto-generate passphrase:** Creates hidden state, portability surprises, and weakens security semantics
- **Plaintext file backend:** Unnecessary given BKT_TOKEN provides the bypass path

## File Changes

| File | Change | Owner |
|------|--------|-------|
| `internal/secret/store.go` | Add `envToken` const, headless guard in `configureFileBackend`, export `IsHeadless()` | Claude |
| `internal/secret/store_test.go` | Test headless guard returns error (no TerminalPrompt) | Claude |
| `pkg/cmdutil/context.go` | Check `BKT_TOKEN` in `loadHostToken()` before keyring | Codex |
| `pkg/cmdutil/context_test.go` | Test BKT_TOKEN precedence, test BKT_TOKEN bypasses keyring | Codex |
| `pkg/cmd/auth/auth.go` | Block login/logout when BKT_TOKEN active, show source in status | Codex |
| `pkg/cmd/auth/auth_test.go` | Test login blocked with BKT_TOKEN | Codex |

## Test Plan

- `configureFileBackend` returns error when headless + no passphrase
- `loadHostToken` returns `BKT_TOKEN` value without opening keyring
- `loadHostToken` prefers `BKT_TOKEN` over stored token
- `auth login` errors when `BKT_TOKEN` is set
- `auth status` displays token source as `BKT_TOKEN`

# Testing Guide

This document outlines testing priorities and patterns for expanding test coverage in the bkt project.

## Current State

Overall statement coverage is **25.1%** across the codebase with **26 test files**. Run `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -1` to get current numbers.

### Per-Package Coverage

| Package | Coverage | Notes |
|---|---|---|
| `pkg/format` | 83.8% | JQ, YAML, templates, fallback, normaliseForJQ |
| `internal/remote` | 80.7% | Cloud/DC URL detection |
| `internal/config` | 79.8% | Load/Save round-trip, token stripping, CRUD |
| `pkg/httpx` | 78.0% | Caching, retries, error decoding, rate limits, 429 retry |
| `internal/build` | 77.3% | ldflags injection |
| `pkg/cmdutil` | 49.2% | Host resolution, output flags, URL normalization |
| `pkg/cmd/api` | 44.2% | YAML, JSON, streaming |
| `pkg/bbcloud` | 43.7% | Repos, PRs, pipelines, decline/reopen, UUID normalization |
| `pkg/cmd/pr` | 40.2% | Decline, reopen, delete-source, merge, list, view |
| `pkg/iostreams` | 34.6% | IO stream abstractions |
| `pkg/prompter` | 30.0% | Confirm retries |
| `pkg/bbdc` | 16.4% | Pagination, limits, state filtering, decline/reopen, auth |
| `pkg/cmd/issue` | 11.3% | Basic issue operations |
| `pkg/cmd/repo` | 7.7% | Clone URL selection |
| `internal/secret` | 7.6% | Keyring backend selection |
| `pkg/cmd/variable` | 7.3% | Variable operations |
| `pkg/cmd/auth` | 2.9% | Credential flow |
| Others | 0% | admin, branch, context, extension, factory, perms, pipeline, project, status, webhook, browser, pager, progress |

### What Tests Cover Well

- **HTTP client infrastructure** (`pkg/httpx`, 78%): ETag caching, retry with exponential backoff, context cancellation, error decoding (structured, plain text, empty), 429 retry with Retry-After, body encoding with GetBody, io.Writer passthrough, Atlassian rate limit headers.
- **Configuration** (`internal/config`, 80%): Load/Save round-trip, MarshalYAML token stripping, CRUD operations, nil map initialization, env var overrides.
- **Output formatting** (`pkg/format`, 84%): JQ on structs/slices, large integer preservation, YAML rendering, Go template rendering, fallback invocation, normaliseForJQ type handling.
- **Git remote detection** (`internal/remote`, 81%): Cloud HTTPS/SSH URLs, DC `/scm/` and `/projects/` paths, missing remote error.
- **API clients** (`pkg/bbcloud` 44%, `pkg/bbdc` 16%): Pagination, limit enforcement, input validation, decline/reopen, UUID normalization, CreateRepository, TriggerPipeline, CurrentUser.
- **Command utilities** (`pkg/cmdutil`, 49%): Host resolution, output flag mutual exclusion, URL normalization.
- **PR commands** (`pkg/cmd/pr`, 40%): Decline, reopen, delete-source (including fork-safe deletion), merge, CLI integration tests.
- **Smoke tests** (`pkg/cmd/smoke`): Full CLI execution for `repo list` (text + JSON) and `project list` with a mock Bitbucket server.

### Remaining Gaps

- **`pkg/httpx`**: `applyAdaptiveThrottle()` partial coverage.
- **`pkg/bbdc`**: Only 16% — needs tests for UpdatePullRequest, CreatePullRequest, PullRequestDiff, branches, webhooks.
- **`pkg/cmd/auth`**: Only 3% — credential flow, token lifecycle largely untested.
- **`pkg/cmd/branch`**, **`pkg/cmd/status`**, **`pkg/cmd/pipeline`**: 0% — all command packages.

## Recommended Testing Priorities

### Priority 1 — API Clients (target 60%+)

- **`pkg/bbdc`** (16%): Needs tests for `UpdatePullRequest`, `CreatePullRequest`, `PullRequestDiff`, `DeleteBranch`, branch protection, webhooks.
- **`pkg/bbcloud`** (44%): Needs tests for `MergePullRequest`, `CreatePullRequest`, commit statuses, webhook operations.

### Priority 2 — Command Packages (target 40%+)

- **`pkg/cmd/auth`** (3%): Token lifecycle, `runStatus()`, URL normalization, non-interactive mode.
- **`pkg/cmd/branch`** (0%): `mapProtectType()`, `ensureBranchRef()`, dual-provider listing.
- **`pkg/cmd/status`** (0%): `renderStatuses()`, Cloud-specific context resolution.
- **`pkg/cmd/pipeline`** (0%): Run, view, logs commands.

### Priority 3 — Secrets and Remaining Utilities

- **`internal/secret`** (8%): `parseBackendList()`, `IsNoKeyringError()`, file backend with `t.TempDir()`.
- **`pkg/httpx`** (78%): `applyAdaptiveThrottle()` remaining paths.

## Testing Patterns

### 1. Table-Driven Tests

Preferred for functions with multiple input/output cases:

```go
func TestNormalizeBaseURL(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid https", "https://example.com", "https://example.com", false},
        {"trailing slash", "https://example.com/", "https://example.com", false},
        {"missing scheme", "example.com", "https://example.com", false},
        {"with path", "https://example.com/bitbucket", "https://example.com/bitbucket", false},
        {"empty string", "", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := NormalizeBaseURL(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("NormalizeBaseURL() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("NormalizeBaseURL() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 2. Mock HTTP Servers

For testing API clients (see `pkg/httpx/client_test.go` for full examples):

```go
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // Verify request
    if r.Method != http.MethodGet {
        t.Errorf("expected GET, got %s", r.Method)
    }
    // Send mock response
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(mockResponse)
}))
defer server.Close()
```

### 3. Smoke Tests with Full CLI Execution

The `pkg/cmd/smoke` pattern is ideal for end-to-end command tests:

```go
func TestPRListDataCenter(t *testing.T) {
    mock := newBitbucketMock(t, "user", "token")
    defer mock.Close()
    mock.StubPRList("PROJ", "repo", prListResponse{...})

    cfg := configForMock(mock.URL(), "user", "token", "test", "PROJ", "")
    stdout, stderr, err := runCLI(t, cfg, "pr", "list", "--limit", "10")
    // ... assertions
}
```

### 4. Golden Files (Future)

For complex CLI output, consider golden file testing:

```go
// Update with: go test -update
got := captureOutput(cmd)
golden := filepath.Join("testdata", t.Name()+".golden")
if *update {
    os.WriteFile(golden, []byte(got), 0644)
}
want, _ := os.ReadFile(golden)
if got != string(want) {
    t.Errorf("output mismatch")
}
```

## Running Tests

```bash
# Run all tests
make test

# Run specific package
go test ./pkg/bbdc/...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Per-function coverage report
go tool cover -func=coverage.out

# Run specific test
go test -run TestListPullRequests ./pkg/bbdc/
```

## Contributing Tests

When adding tests:

1. Place test files alongside the code they test (`foo.go` → `foo_test.go`)
2. Use descriptive test names that explain the scenario
3. Include both happy path and error cases
4. Keep tests fast (avoid real network calls, sleep, etc.)
5. Clean up resources (use `t.Cleanup()` or `defer`)

See [CONTRIBUTING.md](../CONTRIBUTING.md) for the full contribution workflow.

## Test Coverage Goals

**Short-term** (next few PRs):
- Raise `pkg/bbdc` from 16% to 40%+ (UpdatePullRequest, CreatePullRequest, branches)
- Add `pkg/cmd/auth` tests (currently 3%)
- Add `pkg/cmd/branch` tests (currently 0%)
- Target: raise overall coverage from 25% to ~35%

**Medium-term**:
- 60%+ coverage in `pkg/bbdc` and `pkg/bbcloud`
- 40%+ coverage in `pkg/cmd/auth`, `pkg/cmd/branch`, `pkg/cmd/status`
- Add smoke tests for `auth status`, `branch list`
- Add golden file tests for complex formatting
- Target: raise overall coverage to ~45%

**Long-term**:
- 60%+ overall coverage
- Integration tests against real Bitbucket test instances (optional)
- Benchmarks for performance-critical paths (HTTP retry, pagination)
- Fuzzing for input parsing functions (`parseLocator`, `NormalizeBaseURL`, `parseBackendList`)

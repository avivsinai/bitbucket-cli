# Testing Guide

This document outlines testing priorities and patterns for expanding test coverage in the bkt project.

## Current State

As of the February 2026 coverage audit, overall statement coverage is **9.8%** across the codebase. The project has **10 test files** covering 10 of 32 packages.

### Per-Package Coverage

| Package | Coverage | Test File | Notes |
|---|---|---|---|
| `internal/build` | 77.3% | `info_test.go` | Covers ldflags injection |
| `internal/remote` | 79.3% | `remote_test.go` | Covers Cloud/DC URL detection |
| `pkg/httpx` | 59.3% | `client_test.go` | Caching, retries, URL building tested; `decodeError` (0%), `applyAdaptiveThrottle` (45%) missing |
| `pkg/format` | 54.1% | `format_test.go` | JQ and large integer tests; YAML/template paths untested |
| `pkg/cmd/api` | 44.2% | `api_test.go` | YAML, JSON, streaming tested; `--jq`, `--template`, HTTP methods untested |
| `pkg/prompter` | 39.5% | `prompter_test.go` | Confirm retries tested; `Input()` untested |
| `pkg/cmdutil` | 23.2% | `context_test.go` | `ResolveHost` tested; `ResolveContext`, output helpers, URL utils untested |
| `pkg/bbcloud` | 34.1% | `client_test.go`, `pullrequests_test.go` | Pagination for pipelines, repos, PRs tested; decline/reopen tested; GetPullRequest tested |
| `pkg/bbdc` | 14.0% | `pullrequests_test.go` | ListRepositories/ListPullRequests pagination, GetPullRequest path escaping, decline/reopen tested |
| `pkg/cmd/pr` | 22.8% | `pr_test.go` | Decline, reopen, delete-source smoke tests; argument validation |
| `pkg/cmd/repo` | 7.7% | `repo_test.go` | Clone URL selection tested; list/create/browse untested |
| `pkg/cmd/smoke` | n/a | `cli_smoke_test.go` | End-to-end smoke tests for repo list, project list |
| `internal/config` | **0%** | none | 255 lines, all CRUD operations untested |
| `internal/secret` | **0%** | none | 270 lines, keyring backend selection untested |
| `pkg/cmd/auth` | **0%** | none | 537 lines, credential flow untested |
| `pkg/cmd/branch` | **0%** | none | 700 lines across 3 files |
| `pkg/cmd/status` | **0%** | none | 462 lines across 3 files |
| `pkg/cmd/factory` | **0%** | none | Factory wiring |
| `pkg/iostreams` | **0%** | none | IO stream abstractions |
| Others | **0%** | none | admin, context, extension, perms, pipeline, project, webhook, browser, pager, progress |

### What Existing Tests Cover Well

- **HTTP client infrastructure** (`pkg/httpx`): ETag caching, retry with exponential backoff, context cancellation during backoff, URL query preservation, relative path handling.
- **Git remote detection** (`internal/remote`): Cloud HTTPS/SSH URLs, DC `/scm/` and `/projects/` paths, missing remote error.
- **Output formatting** (`pkg/format`): JQ on structs and slices, large integer preservation through json.Number.
- **API command** (`pkg/cmd/api`): YAML output, invalid JSON rejection, plain text streaming, large integer passthrough.
- **Host resolution** (`pkg/cmdutil`): By key, by URL, via context, single-host fallback, multi-host error.
- **Smoke tests** (`pkg/cmd/smoke`): Full CLI execution for `repo list` (text + JSON) and `project list` with a mock Bitbucket server including auth verification.

### What Existing Tests Miss

Even in tested packages, significant gaps remain:

- **`pkg/httpx`**: `decodeError()` at 0%, `applyAdaptiveThrottle()` at 45%, `Do()` at 50% — error response parsing, rate limit throttling, 429 handling, `Retry-After` header, request body serialization, and `io.Writer` passthrough are all untested.
- **`pkg/format`**: YAML output path, Go template rendering, format validation, and fallback-to-caller logic are untested.
- **`pkg/cmdutil`**: `ResolveContext()`, `ResolveOutputSettings()`, `WriteOutput()`, `NormalizeBaseURL()`, `HostKeyFromURL()`, token loading from keyring, and git remote default application are all untested.
- **`pkg/bbcloud`**: Only `ListPipelines` pagination is tested. `ListRepositories`, `CreateRepository`, `TriggerPipeline`, `GetPipelineLogs`, `CurrentUser`, and the Cloud-specific URL parsing logic lack tests.

## Recommended Testing Priorities

### Priority 1 — Core API Clients (highest impact)

These packages own all external API interactions. Bugs here silently corrupt data or break every command.

#### `pkg/bbdc` (0% → target 60%+, ~322 lines)

The entire Bitbucket Data Center client has zero tests. Key areas:

| Function | Lines | Why It Matters |
|---|---|---|
| `ListRepositories` | ~30 | Pagination loop with limit arithmetic; off-by-one bugs silently truncate results |
| `ListPullRequests` | ~35 | Pagination + optional state query parameter; same limit risk |
| `GetPullRequest` | ~15 | URL path escaping for project/repo slugs with special characters |
| `CommitStatuses` | ~15 | Different API path structure; used by `status commit` command |
| `New` | ~20 | Client construction with retry policy; validates wiring |

**Approach**: Use `httptest.NewServer` with request assertions (path, query params, auth header). The `pkg/cmd/smoke/cli_smoke_test.go` `bitbucketMock` pattern is a good reference but tests should be at the unit level here.

```go
func TestListRepositoriesPaginates(t *testing.T) {
    var hits int32
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        count := atomic.AddInt32(&hits, 1)
        w.Header().Set("Content-Type", "application/json")
        switch count {
        case 1:
            json.NewEncoder(w).Encode(map[string]any{
                "values":        []map[string]any{{"slug": "repo1"}},
                "isLastPage":    false,
                "nextPageStart": 1,
            })
        case 2:
            json.NewEncoder(w).Encode(map[string]any{
                "values":     []map[string]any{{"slug": "repo2"}},
                "isLastPage": true,
            })
        }
    }))
    t.Cleanup(server.Close)
    // ... assert 2 repos returned, 2 requests made
}
```

#### `pkg/bbcloud` (10.2% → target 60%+, ~445 lines)

Only `ListPipelines` pagination is tested. Missing coverage for:

- `ListRepositories` — has its own pagination with Cloud's `next` URL pattern
- `CreateRepository` — POST with JSON body serialization
- `TriggerPipeline` — variable construction with `secured` flag
- `GetPipelineLogs` — returns raw bytes, not JSON
- UUID brace trimming in `GetPipeline`, `ListPipelineSteps`, `GetPipelineLogs`
- URL parsing fallback logic (`RequestURI()` vs `String()`)

#### `pkg/httpx` (59.3% → target 80%+, ~542 lines)

The existing tests are good but leave critical paths uncovered:

- **`decodeError()`** (0%): API error response parsing is completely untested. A bug here means every error message is wrong or panics.
- **`applyAdaptiveThrottle()`** (45%): Rate limit sleep logic; untested paths could cause unnecessary delays or missed throttling.
- **`Do()` response handling** (50%): The `io.Writer` passthrough path (used for streaming raw output), request body serialization with `GetBody`, and 429 `Retry-After` header handling are untested.
- **`NewRequest()` body encoding** (53%): JSON body marshaling with `GetBody` for retries is not exercised.

### Priority 2 — Configuration and Secrets (data integrity)

#### `internal/config` (0% → target 70%+, ~255 lines)

This package manages all persistent CLI state. It has no tests despite containing:

- `Load()` / `Save()` — YAML round-trip with atomic file writes; a serialization bug loses all user config
- `MarshalYAML()` — token stripping; if broken, credentials leak to disk
- `SetContext` / `Context` / `DeleteContext` / `SetActiveContext` — CRUD with thread safety
- `SetHost` / `Host` / `DeleteHost` — same pattern
- `resolvePath()` — `BKT_CONFIG_DIR` environment override

**Approach**: Use `t.TempDir()` with `t.Setenv("BKT_CONFIG_DIR", ...)` for filesystem isolation. Test YAML round-trips, nil map initialization, and the token-stripping marshal.

```go
func TestMarshalYAMLStripsToken(t *testing.T) {
    h := &config.Host{Kind: "dc", BaseURL: "https://example.com", Token: "secret"}
    data, err := yaml.Marshal(h)
    if err != nil {
        t.Fatal(err)
    }
    if strings.Contains(string(data), "secret") {
        t.Fatal("token was not stripped during marshaling")
    }
}
```

#### `internal/secret` (0% → target 50%+, ~270 lines)

The keyring backend selection logic is complex and platform-dependent:

- `resolveAllowedBackends()` — reads `BKT_KEYRING_BACKENDS` env var, has platform-specific defaults
- `parseBackendList()` — string-to-enum mapping for backend names
- `IsNoKeyringError()` — error classification
- `Set` / `Get` / `Delete` — nil pointer guards

**Approach**: Unit-test `parseBackendList()` and `IsNoKeyringError()` directly. For `Open()`, use the file backend with `WithFileDir(t.TempDir())` to avoid needing a real keyring.

### Priority 3 — Command Packages (user-facing behavior)

#### `pkg/cmd/pr` (0%, ~1,966 lines — largest untested package)

This is the most complex command package. Recommended test targets:

- **`pr.go` runList/runView**: Test the dual-provider branching (DC vs Cloud), author filtering, and output formatting using the smoke test pattern with `runCLI()`.
- **`tasks.go` toggleTaskState**: Test the complete/reopen state toggle logic.
- **`automerge.go`**: Test settings struct construction and status rendering.
- **`reactions.go`**: Test argument parsing for the triple (PR ID, comment ID, emoji).

#### `pkg/cmd/auth` (0%, ~537 lines)

Auth is security-critical. Key testable areas:

- **`storeHostToken()` / `deleteHostToken()`**: Token lifecycle with mocked secret store.
- **`runStatus()`**: Output formatting for host/context display.
- **URL normalization**: The `bitbucket.org` detection logic and Cloud API URL construction.
- **Non-interactive mode**: Test flag-based auth without TTY.

#### `pkg/cmd/branch` (0%, ~700 lines)

- **`protect.go` `mapProtectType()`**: Pure function mapping protection type strings — table-driven test.
- **`protect.go` `ensureBranchRef()`**: Ref prefix handling with wildcards.
- **`branch.go` runList**: Dual-provider branch listing with commit hash truncation.

#### `pkg/cmd/status` (0%, ~462 lines)

- **`renderStatuses()`**: Pure rendering function; test output format.
- **`resolveCloudStatusContext()`**: Cloud-specific context resolution.

### Priority 4 — Utility Packages

#### `pkg/cmdutil/url.go` (0%, ~42 lines)

Small but used everywhere:

- `NormalizeBaseURL()` — scheme addition, trailing slash removal, query/fragment stripping
- `HostKeyFromURL()` — hostname extraction

Table-driven tests would cover this in a single test function.

#### `pkg/cmdutil/output.go` (0%, ~81 lines)

- `ResolveOutputSettings()` — flag mutual exclusion validation (json vs yaml, jq vs template, jq requires json)
- `WriteOutput()` — structured output dispatch

#### `pkg/format` (54.1% → target 80%+)

Missing: YAML rendering, Go template rendering, invalid format rejection, fallback function invocation.

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
- Add `pkg/bbdc/client_test.go` covering `ListRepositories`, `ListPullRequests`, `GetPullRequest` pagination
- Add `internal/config/config_test.go` covering Load/Save round-trips and `MarshalYAML` token stripping
- Add `pkg/cmdutil/url_test.go` covering `NormalizeBaseURL` and `HostKeyFromURL`
- Expand `pkg/httpx/client_test.go` with `decodeError`, 429 handling, and body serialization tests
- Target: raise overall coverage from 9.8% to ~20%

**Medium-term** (next release):
- 60%+ coverage in `pkg/bbdc` and `pkg/bbcloud`
- 40%+ coverage in `pkg/cmd/pr`, `pkg/cmd/auth`, `pkg/cmd/branch`
- Add smoke tests for `pr list`, `pr view`, `auth status`, `branch list`
- Add golden file tests for complex formatting
- Target: raise overall coverage to ~40%

**Long-term**:
- 60%+ overall coverage
- Integration tests against real Bitbucket test instances (optional)
- Benchmarks for performance-critical paths (HTTP retry, pagination)
- Fuzzing for input parsing functions (`parseLocator`, `NormalizeBaseURL`, `parseBackendList`)

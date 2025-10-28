# Testing Guide

This document outlines testing priorities and patterns for expanding test coverage in the bkt project.

## Current State

As of the last coverage review:

- **Excellent**: `pkg/httpx`, `pkg/prompter`, `internal/remote`, `pkg/cmdutil` have good test coverage
- **Good**: Smoke tests cover critical CLI workflows
- **Needs Work**: `pkg/bbdc` (0 test files), most `pkg/cmd/*` subdirectories (limited coverage)

## Testing Priorities

### Priority 1: Core API Clients

Focus on `pkg/bbdc` (Bitbucket Data Center client) and expand `pkg/bbcloud`:

**Why**: These packages encapsulate all external API interactions. Tests here prevent regressions and document expected API contracts.

**Approach**: Use `httptest.NewServer` to mock Bitbucket responses (see `pkg/httpx/client_test.go` for examples).

**Example targets**:
- `pkg/bbdc/client.go` - test client initialization, base URL handling
- `pkg/bbdc/pullrequests.go` - test PR listing, creation, merging
- `pkg/bbdc/repos.go` - test repository operations
- `pkg/bbcloud/client.go` - expand existing coverage

**Example pattern**:
```go
func TestListPullRequests(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/rest/api/1.0/projects/PROJ/repos/repo/pull-requests" {
            t.Errorf("unexpected path: %s", r.URL.Path)
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "values": []map[string]interface{}{
                {"id": 1, "title": "Test PR"},
            },
        })
    }))
    defer server.Close()

    client := bbdc.NewClient(bbdc.Options{BaseURL: server.URL})
    prs, err := client.ListPullRequests(context.Background(), "PROJ", "repo")
    if err != nil {
        t.Fatalf("ListPullRequests: %v", err)
    }
    if len(prs) != 1 || prs[0].ID != 1 {
        t.Errorf("unexpected PR list: %+v", prs)
    }
}
```

### Priority 2: Command Logic

Add tests for higher-traffic Cobra commands in `pkg/cmd/*`:

**Why**: These packages contain the user-facing logic. Tests ensure CLI behavior stays consistent.

**Approach**: Test helper functions and data transformations. For commands that need IO, consider using `io.Pipe` or `bytes.Buffer` for stdin/stdout.

**Example targets**:
- `pkg/cmd/pr/*.go` - PR formatting, filtering, status checks
- `pkg/cmd/repo/*.go` - expand existing `repo_test.go`
- `pkg/cmd/pipeline/*.go` - pipeline state parsing

**Example pattern** (table-driven tests):
```go
func TestFormatPRStatus(t *testing.T) {
    tests := []struct {
        name   string
        pr     bbdc.PullRequest
        want   string
    }{
        {
            name: "open PR",
            pr:   bbdc.PullRequest{State: "OPEN"},
            want: "OPEN",
        },
        {
            name: "merged PR",
            pr:   bbdc.PullRequest{State: "MERGED"},
            want: "MERGED",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := formatStatus(tt.pr)
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

### Priority 3: Edge Cases and Error Paths

Expand coverage for error handling:

**Example targets**:
- API retry logic failures
- Malformed JSON responses
- Network timeouts
- Missing configuration scenarios

## Testing Patterns

### 1. Table-Driven Tests

Preferred for functions with multiple input/output cases:

```go
func TestParseURL(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {"valid https", "https://example.com", "https://example.com", false},
        {"missing scheme", "example.com", "", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := parseURL(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("parseURL() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("parseURL() = %v, want %v", got, tt.want)
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

### 3. Golden Files (Future)

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
- Add at least 3-5 test files to `pkg/bbdc/`
- Expand coverage in `pkg/cmd/pr/`, `pkg/cmd/pipeline/`

**Medium-term** (next release):
- 60%+ coverage in `pkg/bbdc` and `pkg/bbcloud`
- 40%+ coverage in `pkg/cmd/*` packages
- Add golden file tests for complex formatting

**Long-term**:
- Integration tests against real Bitbucket test instances (optional)
- Benchmarks for performance-critical paths
- Fuzzing for input parsing functions

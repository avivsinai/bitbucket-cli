# Inline PR Comments — Design Spec

## Problem

`bkt pr comment` only supports general PR-level comments (with optional `--parent` for threading). Users cannot leave inline comments on specific file lines in a PR diff without resorting to verbose `bkt api` calls with raw JSON payloads.

## Decision Record

GitHub CLI (`gh`) intentionally omits inline comment support because GitHub's API ties inline comments to a review-thread lifecycle (pending reviews, thread resolution, diff positions) — a fundamentally more complex domain (`cli/cli#359`, open since 2020).

Bitbucket's model is simpler: inline comments are PR comments with optional location metadata (`inline`/`anchor` objects) on the same endpoint. No review lifecycle, no pending state. `bkt pr comment` is the natural home for this feature.

## Scope — v1

**In scope:**
- Line-targeted inline comments on PR diffs via `--file`, `--from-line`, `--to-line`
- Cloud and Data Center support
- Add `Inline`/`Anchor` fields to comment response structs for JSON output

**Out of scope (deferred):**
- File-level comments without line targeting (DC needs diff metadata we don't resolve)
- Combining `--parent` with inline flags (reply threading already carries location)
- Both `--from-line` and `--to-line` together (Cloud supports it, DC doesn't map cleanly)
- Review lifecycle commands (approve/request-changes with inline comments)
- Thread resolution

## CLI Interface

```bash
# Comment on a new/changed line (addition side)
bkt pr comment 42 --text "Missing auth check" --file src/handler.go --to-line 25

# Comment on a removed line (deletion side)
bkt pr comment 42 --text "Was this intentional?" --file src/handler.go --from-line 10
```

### New Flags

| Flag | Type | Description |
|------|------|-------------|
| `--file` | `string` | File path as shown in the diff (repo-relative, forward slashes, no normalization) |
| `--from-line` | `int` | Line number in the old file (deletion/source side) |
| `--to-line` | `int` | Line number in the new file (addition/destination side) |

### Validation Rules

1. `--file` is required when `--from-line` or `--to-line` is used
2. `--file` alone is rejected (no file-level comments in v1)
3. Exactly one of `--from-line` or `--to-line` must be specified (not both)
4. `--parent` is mutually exclusive with `--file`/`--from-line`/`--to-line`
5. Line numbers must be positive integers

### Error Messages

```
--file is required when --from-line or --to-line is specified
--file must be used with either --from-line or --to-line (file-level comments not yet supported)
--from-line and --to-line are mutually exclusive
--parent cannot be combined with inline comment flags (--file, --from-line, --to-line)
```

## API Mapping

### Bitbucket Cloud

Endpoint: `POST /repositories/{workspace}/{repo_slug}/pullrequests/{id}/comments`

**`--to-line N`** (addition/new side):
```json
{
  "content": {"raw": "Comment text"},
  "inline": {"path": "src/handler.go", "to": 25}
}
```

**`--from-line N`** (deletion/old side):
```json
{
  "content": {"raw": "Comment text"},
  "inline": {"path": "src/handler.go", "from": 10}
}
```

### Bitbucket Data Center

Endpoint: `POST /rest/api/1.0/projects/{project}/repos/{repo}/pull-requests/{id}/comments`

**`--to-line N`** (addition/new side):
```json
{
  "text": "Comment text",
  "anchor": {"path": "src/handler.go", "line": 25, "lineType": "ADDED", "fileType": "TO"}
}
```

**`--from-line N`** (deletion/old side):
```json
{
  "text": "Comment text",
  "anchor": {"path": "src/handler.go", "line": 10, "lineType": "REMOVED", "fileType": "FROM"}
}
```

DC notes: `diffType` and hash fields are omitted; the API uses EFFECTIVE defaults when absent.

## Implementation Design

### 1. Comment Options Struct (`pkg/cmd/pr/pr.go`)

Add fields to `commentOptions`:

```go
type commentOptions struct {
    Workspace string
    Project   string
    Repo      string
    Text      string
    ParentID  int
    File      string  // new
    FromLine  int     // new
    ToLine    int     // new
}
```

### 2. API Client Refactor

Replace positional params with an options struct in both clients.

**Cloud** (`pkg/bbcloud/pullrequests.go`):

```go
type CommentOptions struct {
    Text     string
    ParentID int
    File     string
    FromLine int
    ToLine   int
}

func (c *Client) CommentPullRequest(ctx context.Context, workspace, repoSlug string, prID int, opts CommentOptions) error
```

Body construction: existing `content` + `parent` logic, plus conditionally add `inline` object when `File` is set.

**DC** (`pkg/bbdc/pullrequests.go`):

```go
type CommentOptions struct {
    Text     string
    ParentID int
    File     string
    FromLine int
    ToLine   int
}

func (c *Client) CommentPullRequest(ctx context.Context, projectKey, repoSlug string, prID int, opts CommentOptions) error
```

Body construction: existing `text` + `parent` logic, plus conditionally add `anchor` object when `File` is set.

### 3. Comment Response Structs

Add location fields to comment structs for JSON output:

**Cloud** — add `Inline` to `PullRequestComment`:
```go
Inline *struct {
    Path string `json:"path"`
    From *int   `json:"from"`
    To   *int   `json:"to"`
} `json:"inline,omitempty"`
```

**DC** — add `Anchor` to `PullRequestComment`:
```go
Anchor *struct {
    Path     string `json:"path"`
    Line     int    `json:"line"`
    LineType string `json:"lineType"`
    FileType string `json:"fileType"`
} `json:"anchor,omitempty"`
```

### 4. Comment List Display (deferred to follow-up)

Enhancing `bkt pr comments` to display inline location data is deferred. The struct changes (Section 3) ensure `--json` output includes location fields immediately. Human-readable table formatting for inline comments will be addressed in a separate change.

## Call Site Audit

`CommentPullRequest` is called from exactly two sites:
- `pkg/cmd/pr/pr.go:1749` (DC path in `runComment`)
- `pkg/cmd/pr/pr.go:1773` (Cloud path in `runComment`)

Both test files (`pullrequests_test.go` in each package) also call the function. All call sites must be updated.

The create call continues to return `error` only (no response body). The struct enhancements (Section 3) benefit the list path's JSON output, not the create path.

## Files Changed

| File | Change |
|------|--------|
| `pkg/cmd/pr/pr.go` | Add `File`, `FromLine`, `ToLine` to `commentOptions`; register flags; add validation; pass to client |
| `pkg/bbcloud/pullrequests.go` | Refactor `CommentPullRequest` to accept `CommentOptions`; add `inline` body logic; add `Inline` to `PullRequestComment` |
| `pkg/bbdc/pullrequests.go` | Refactor `CommentPullRequest` to accept `CommentOptions`; add `anchor` body logic; add `Anchor` to `PullRequestComment` |
| `pkg/bbcloud/pullrequests_test.go` | Add tests for inline comment creation (Cloud) |
| `pkg/bbdc/pullrequests_test.go` | Add tests for anchor comment creation (DC) |
## Testing Strategy

- Table-driven tests for both Cloud and DC `CommentPullRequest` with inline options
- Validation tests: all error cases (missing file, both lines, parent+file, non-positive lines)
- HTTP mock tests verifying correct JSON payload structure for each host
- Existing comment tests must continue to pass (backward compatibility)

## Risks

- **DC `anchor` without diff hashes**: The DC API may require `diffType`/`fromHash`/`toHash` in some edge cases. If integration testing reveals failures, we'll need to fetch diff metadata first. Mitigation: the docs indicate EFFECTIVE defaults apply when hashes are absent.
- **Line number validity**: Users may specify lines that don't exist in the diff. Both APIs return errors for this, which we pass through. No client-side diff validation needed.
- **DC context lines**: When a user targets an unchanged (context) line with `--to-line`, DC receives `lineType: "ADDED"` which may not be semantically correct. The API may reject or misplace the comment. This is a known v1 limitation — supporting `CONTEXT` lines would require diff analysis. Document in `--help` that `--to-line` targets added/changed lines and `--from-line` targets removed lines.

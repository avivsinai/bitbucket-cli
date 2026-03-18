# Inline PR Comments Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--file`, `--from-line`, and `--to-line` flags to `bkt pr comment` for inline/file-level comments on PR diffs, supporting both Cloud and Data Center.

**Architecture:** Refactor `CommentPullRequest` in both API clients from positional params to a `CommentOptions` struct. Add inline payload construction (Cloud `inline`, DC `anchor`). Add response struct fields for JSON output. All validation lives in the CLI layer.

**Tech Stack:** Go, Cobra (CLI), httptest (mocking)

**Spec:** `docs/superpowers/specs/2026-03-18-inline-pr-comments-design.md`

---

### Task 1: Refactor Cloud CommentPullRequest to CommentOptions

**Files:**
- Modify: `pkg/bbcloud/pullrequests.go:354-384`
- Modify: `pkg/bbcloud/pullrequests_test.go:311-404`

- [ ] **Step 1: Define CommentOptions and update signature**

In `pkg/bbcloud/pullrequests.go`, replace the existing `CommentPullRequest` function. Add the `CommentOptions` type just before it:

```go
// CommentOptions configures a pull request comment.
type CommentOptions struct {
	Text     string
	ParentID int
	File     string
	FromLine int
	ToLine   int
}

// CommentPullRequest adds a comment to the pull request.
// When ParentID > 0, the comment is a threaded reply.
// When File is set with FromLine or ToLine, the comment targets a specific diff line.
func (c *Client) CommentPullRequest(ctx context.Context, workspace, repoSlug string, prID int, opts CommentOptions) error {
	if workspace == "" || repoSlug == "" {
		return fmt.Errorf("workspace and repository slug are required")
	}
	if strings.TrimSpace(opts.Text) == "" {
		return fmt.Errorf("comment text is required")
	}

	body := map[string]any{
		"content": map[string]string{
			"raw": opts.Text,
		},
	}
	if opts.ParentID > 0 {
		body["parent"] = map[string]int{"id": opts.ParentID}
	}
	if opts.File != "" {
		inline := map[string]any{"path": opts.File}
		if opts.ToLine > 0 {
			inline["to"] = opts.ToLine
		}
		if opts.FromLine > 0 {
			inline["from"] = opts.FromLine
		}
		body["inline"] = inline
	}

	path := fmt.Sprintf("/repositories/%s/%s/pullrequests/%d/comments",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		prID,
	)
	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return err
	}

	return c.http.Do(req, nil)
}
```

- [ ] **Step 2: Update existing Cloud tests to use CommentOptions**

Update all callers in `pkg/bbcloud/pullrequests_test.go`. Replace every `client.CommentPullRequest(ctx, ws, repo, id, text, parentID)` with `client.CommentPullRequest(ctx, ws, repo, id, bbcloud.CommentOptions{Text: text, ParentID: parentID})`.

Affected tests:
- `TestCommentPullRequest` (line 321): `bbcloud.CommentOptions{Text: "LGTM"}`
- `TestCommentPullRequestValidation` (line 361): `bbcloud.CommentOptions{Text: tt.text}`
- `TestCommentPullRequestWithParent` (line 375): `bbcloud.CommentOptions{Text: "reply", ParentID: 42}`
- `TestCommentPullRequestWithoutParent` (line 396): `bbcloud.CommentOptions{Text: "top-level"}`

- [ ] **Step 3: Run tests to verify refactor is backward-compatible**

Run: `go test ./pkg/bbcloud/ -run TestCommentPullRequest -v`
Expected: All 4 existing tests PASS.

- [ ] **Step 4: Add inline comment tests for Cloud**

Add to `pkg/bbcloud/pullrequests_test.go` after `TestCommentPullRequestWithoutParent`:

```go
func TestCommentPullRequestInlineToLine(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))

	err := client.CommentPullRequest(context.Background(), "myworkspace", "my-repo", 7, bbcloud.CommentOptions{
		Text:   "needs fix",
		File:   "src/handler.go",
		ToLine: 25,
	})
	if err != nil {
		t.Fatalf("CommentPullRequest inline to-line: %v", err)
	}

	inline, ok := gotBody["inline"].(map[string]any)
	if !ok {
		t.Fatal("request body missing inline object")
	}
	if inline["path"] != "src/handler.go" {
		t.Errorf("inline.path = %v, want src/handler.go", inline["path"])
	}
	if to, ok := inline["to"].(float64); !ok || int(to) != 25 {
		t.Errorf("inline.to = %v, want 25", inline["to"])
	}
	if _, ok := inline["from"]; ok {
		t.Error("expected no from field when only to-line is set")
	}
}

func TestCommentPullRequestInlineFromLine(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))

	err := client.CommentPullRequest(context.Background(), "myworkspace", "my-repo", 7, bbcloud.CommentOptions{
		Text:     "was this intentional?",
		File:     "src/handler.go",
		FromLine: 10,
	})
	if err != nil {
		t.Fatalf("CommentPullRequest inline from-line: %v", err)
	}

	inline, ok := gotBody["inline"].(map[string]any)
	if !ok {
		t.Fatal("request body missing inline object")
	}
	if inline["path"] != "src/handler.go" {
		t.Errorf("inline.path = %v, want src/handler.go", inline["path"])
	}
	if from, ok := inline["from"].(float64); !ok || int(from) != 10 {
		t.Errorf("inline.from = %v, want 10", inline["from"])
	}
	if _, ok := inline["to"]; ok {
		t.Error("expected no to field when only from-line is set")
	}
}

func TestCommentPullRequestNoInlineWhenFileEmpty(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))

	err := client.CommentPullRequest(context.Background(), "myworkspace", "my-repo", 7, bbcloud.CommentOptions{
		Text: "general comment",
	})
	if err != nil {
		t.Fatalf("CommentPullRequest: %v", err)
	}

	if _, ok := gotBody["inline"]; ok {
		t.Error("expected no inline field for general comment")
	}
}
```

- [ ] **Step 5: Run all Cloud comment tests**

Run: `go test ./pkg/bbcloud/ -run TestCommentPullRequest -v`
Expected: All 7 tests PASS (4 existing + 3 new).

- [ ] **Step 6: Commit**

```bash
git add pkg/bbcloud/pullrequests.go pkg/bbcloud/pullrequests_test.go
git commit -m "feat(cloud): refactor CommentPullRequest to CommentOptions with inline support"
```

---

### Task 2: Refactor DC CommentPullRequest to CommentOptions

**Files:**
- Modify: `pkg/bbdc/pullrequests.go:189-209`
- Modify: `pkg/bbdc/pullrequests_test.go:454-534`

- [ ] **Step 1: Define CommentOptions and update signature**

In `pkg/bbdc/pullrequests.go`, add `CommentOptions` type and replace the existing function:

```go
// CommentOptions configures a pull request comment.
type CommentOptions struct {
	Text     string
	ParentID int
	File     string
	FromLine int
	ToLine   int
}

// CommentPullRequest adds a comment to the pull request.
// When ParentID > 0, the comment is a threaded reply.
// When File is set with FromLine or ToLine, the comment targets a specific diff line.
func (c *Client) CommentPullRequest(ctx context.Context, projectKey, repoSlug string, prID int, opts CommentOptions) error {
	if strings.TrimSpace(opts.Text) == "" {
		return fmt.Errorf("comment text is required")
	}

	body := map[string]any{"text": opts.Text}
	if opts.ParentID > 0 {
		body["parent"] = map[string]int{"id": opts.ParentID}
	}
	if opts.File != "" {
		anchor := map[string]any{"path": opts.File}
		if opts.ToLine > 0 {
			anchor["line"] = opts.ToLine
			anchor["lineType"] = "ADDED"
			anchor["fileType"] = "TO"
		}
		if opts.FromLine > 0 {
			anchor["line"] = opts.FromLine
			anchor["lineType"] = "REMOVED"
			anchor["fileType"] = "FROM"
		}
		body["anchor"] = anchor
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/comments",
		url.PathEscape(projectKey),
		url.PathEscape(repoSlug),
		prID,
	), body)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}
```

- [ ] **Step 2: Update existing DC tests to use CommentOptions**

Update all callers in `pkg/bbdc/pullrequests_test.go`. Replace every `client.CommentPullRequest(ctx, proj, repo, id, text, parentID)` with `client.CommentPullRequest(ctx, proj, repo, id, bbdc.CommentOptions{Text: text, ParentID: parentID})`.

Affected tests:
- `TestCommentPullRequest` (line 464): `bbdc.CommentOptions{Text: "LGTM"}`
- `TestCommentPullRequestWithParent` (line 486): `bbdc.CommentOptions{Text: "reply", ParentID: 42}`
- `TestCommentPullRequestWithoutParent` (line 507): `bbdc.CommentOptions{Text: "top-level"}`
- `TestCommentPullRequestValidation` (line 533): `bbdc.CommentOptions{Text: tt.text}`

- [ ] **Step 3: Run tests to verify refactor is backward-compatible**

Run: `go test ./pkg/bbdc/ -run TestCommentPullRequest -v`
Expected: All 4 existing tests PASS.

- [ ] **Step 4: Add inline comment tests for DC**

Add to `pkg/bbdc/pullrequests_test.go` after `TestCommentPullRequestValidation`:

```go
func TestCommentPullRequestInlineToLine(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))

	err := client.CommentPullRequest(context.Background(), "PROJ", "my-repo", 7, bbdc.CommentOptions{
		Text:   "needs fix",
		File:   "src/handler.go",
		ToLine: 25,
	})
	if err != nil {
		t.Fatalf("CommentPullRequest inline to-line: %v", err)
	}

	anchor, ok := gotBody["anchor"].(map[string]any)
	if !ok {
		t.Fatal("request body missing anchor object")
	}
	if anchor["path"] != "src/handler.go" {
		t.Errorf("anchor.path = %v, want src/handler.go", anchor["path"])
	}
	if line, ok := anchor["line"].(float64); !ok || int(line) != 25 {
		t.Errorf("anchor.line = %v, want 25", anchor["line"])
	}
	if anchor["lineType"] != "ADDED" {
		t.Errorf("anchor.lineType = %v, want ADDED", anchor["lineType"])
	}
	if anchor["fileType"] != "TO" {
		t.Errorf("anchor.fileType = %v, want TO", anchor["fileType"])
	}
}

func TestCommentPullRequestInlineFromLine(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))

	err := client.CommentPullRequest(context.Background(), "PROJ", "my-repo", 7, bbdc.CommentOptions{
		Text:     "was this intentional?",
		File:     "src/handler.go",
		FromLine: 10,
	})
	if err != nil {
		t.Fatalf("CommentPullRequest inline from-line: %v", err)
	}

	anchor, ok := gotBody["anchor"].(map[string]any)
	if !ok {
		t.Fatal("request body missing anchor object")
	}
	if anchor["path"] != "src/handler.go" {
		t.Errorf("anchor.path = %v, want src/handler.go", anchor["path"])
	}
	if line, ok := anchor["line"].(float64); !ok || int(line) != 10 {
		t.Errorf("anchor.line = %v, want 10", anchor["line"])
	}
	if anchor["lineType"] != "REMOVED" {
		t.Errorf("anchor.lineType = %v, want REMOVED", anchor["lineType"])
	}
	if anchor["fileType"] != "FROM" {
		t.Errorf("anchor.fileType = %v, want FROM", anchor["fileType"])
	}
}

func TestCommentPullRequestNoAnchorWhenFileEmpty(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))

	err := client.CommentPullRequest(context.Background(), "PROJ", "my-repo", 7, bbdc.CommentOptions{
		Text: "general comment",
	})
	if err != nil {
		t.Fatalf("CommentPullRequest: %v", err)
	}

	if _, ok := gotBody["anchor"]; ok {
		t.Error("expected no anchor field for general comment")
	}
}
```

- [ ] **Step 5: Run all DC comment tests**

Run: `go test ./pkg/bbdc/ -run TestCommentPullRequest -v`
Expected: All 7 tests PASS (4 existing + 3 new).

- [ ] **Step 6: Commit**

```bash
git add pkg/bbdc/pullrequests.go pkg/bbdc/pullrequests_test.go
git commit -m "feat(dc): refactor CommentPullRequest to CommentOptions with anchor support"
```

---

### Task 3: Add CLI flags, validation, and wire to clients

**Files:**
- Modify: `pkg/cmd/pr/pr.go:1685-1785`

- [ ] **Step 1: Add fields to commentOptions struct**

In `pkg/cmd/pr/pr.go`, update `commentOptions` (line 1685):

```go
type commentOptions struct {
	Workspace string
	Project   string
	Repo      string
	Text      string
	ParentID  int
	File      string
	FromLine  int
	ToLine    int
}
```

- [ ] **Step 2: Register new flags in newCommentCmd**

In `newCommentCmd` (line 1693), add flags before the `return cmd` and after the existing flags:

```go
cmd.Flags().StringVar(&opts.File, "file", "", "File path in the diff (requires --from-line or --to-line)")
cmd.Flags().IntVar(&opts.FromLine, "from-line", 0, "Line in the old file (removed/source side)")
cmd.Flags().IntVar(&opts.ToLine, "to-line", 0, "Line in the new file (added/destination side)")
```

- [ ] **Step 3: Add validation in RunE**

In `newCommentCmd`, add validation after the existing `--parent` check (line 1706) and before `return runComment(...)`:

```go
hasFile := opts.File != ""
hasFromLine := cmd.Flags().Changed("from-line")
hasToLine := cmd.Flags().Changed("to-line")
hasInline := hasFile || hasFromLine || hasToLine

if (hasFromLine || hasToLine) && !hasFile {
	return fmt.Errorf("--file is required when --from-line or --to-line is specified")
}
if hasFile && !hasFromLine && !hasToLine {
	return fmt.Errorf("--file must be used with either --from-line or --to-line (file-level comments not yet supported)")
}
if hasFromLine && hasToLine {
	return fmt.Errorf("--from-line and --to-line are mutually exclusive")
}
if cmd.Flags().Changed("parent") && hasInline {
	return fmt.Errorf("--parent cannot be combined with inline comment flags (--file, --from-line, --to-line)")
}
if hasFromLine && opts.FromLine <= 0 {
	return fmt.Errorf("--from-line must be a positive integer")
}
if hasToLine && opts.ToLine <= 0 {
	return fmt.Errorf("--to-line must be a positive integer")
}
```

- [ ] **Step 4: Update runComment to pass CommentOptions to both clients**

In `runComment`, update the DC call site (line 1749):

```go
if err := client.CommentPullRequest(ctx, projectKey, repoSlug, id, bbdc.CommentOptions{
	Text:     opts.Text,
	ParentID: opts.ParentID,
	File:     opts.File,
	FromLine: opts.FromLine,
	ToLine:   opts.ToLine,
}); err != nil {
```

Update the Cloud call site (line 1773):

```go
if err := client.CommentPullRequest(ctx, workspace, repoSlug, id, bbcloud.CommentOptions{
	Text:     opts.Text,
	ParentID: opts.ParentID,
	File:     opts.File,
	FromLine: opts.FromLine,
	ToLine:   opts.ToLine,
}); err != nil {
```

- [ ] **Step 5: Run full test suite**

Run: `go test ./...`
Expected: All tests PASS. No compilation errors.

- [ ] **Step 6: Commit**

```bash
git add pkg/cmd/pr/pr.go
git commit -m "feat(pr): add --file, --from-line, --to-line flags for inline comments"
```

---

### Task 4: Add Inline/Anchor fields to comment response structs

**Files:**
- Modify: `pkg/bbcloud/pullrequests.go:508-523` (Cloud PullRequestComment)
- Modify: `pkg/bbdc/pullrequests.go:24-31` (DC PullRequestComment)

- [ ] **Step 1: Add Inline field to Cloud PullRequestComment**

In `pkg/bbcloud/pullrequests.go`, update the struct (after the `Parent` field):

```go
type PullRequestComment struct {
	ID      int `json:"id"`
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
	User       *Account `json:"user"`
	CreatedOn  string   `json:"created_on"`
	UpdatedOn  string   `json:"updated_on"`
	Resolution *struct {
		User      *Account `json:"user"`
		CreatedOn string   `json:"created_on"`
	} `json:"resolution"`
	Parent *struct {
		ID int `json:"id"`
	} `json:"parent"`
	Inline *struct {
		Path string `json:"path"`
		From *int   `json:"from"`
		To   *int   `json:"to"`
	} `json:"inline,omitempty"`
}
```

- [ ] **Step 2: Add Anchor field to DC PullRequestComment**

In `pkg/bbdc/pullrequests.go`, update the struct:

```go
type PullRequestComment struct {
	ID     int    `json:"id"`
	Text   string `json:"text"`
	Author struct {
		User User `json:"user"`
	} `json:"author"`
	Anchor *struct {
		Path     string `json:"path"`
		Line     int    `json:"line"`
		LineType string `json:"lineType"`
		FileType string `json:"fileType"`
	} `json:"anchor,omitempty"`
}
```

- [ ] **Step 3: Build and test**

Run: `go build ./... && go test ./...`
Expected: All tests PASS, no compilation errors.

- [ ] **Step 4: Commit**

```bash
git add pkg/bbcloud/pullrequests.go pkg/bbdc/pullrequests.go
git commit -m "feat: add Inline/Anchor fields to comment response structs for JSON output"
```

---

### Task 5: Final verification and format check

- [ ] **Step 1: Run gofmt**

Run: `make fmt`
Expected: No files reformatted (or auto-fixes applied).

- [ ] **Step 2: Run full test suite**

Run: `make test`
Expected: All tests PASS.

- [ ] **Step 3: Build binary**

Run: `make build`
Expected: Binary builds successfully.

- [ ] **Step 4: Verify --help output**

Run: `go run ./cmd/bkt pr comment --help`
Expected: Shows `--file`, `--from-line`, `--to-line` flags with descriptions.

- [ ] **Step 5: Final commit if any formatting changes**

```bash
git add -A && git diff --staged --quiet || git commit -m "style: gofmt"
```

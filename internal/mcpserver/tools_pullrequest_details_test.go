package mcpserver

import (
	"context"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type fakeC2BBackend struct {
	fakeC2ABackend

	getPRLocator RepositoryRef
	getPRID      int
	getPRResult  PullRequest
	getPRErr     error
	getPRCalls   int

	diffLocator RepositoryRef
	diffID      int
	diffResult  Diff
	diffErr     error
	diffCalls   int

	commentLocator RepositoryRef
	commentID      int
	commentLimit   int
	commentItems   []Comment
	commentMore    bool
	commentErr     error
	commentCalls   int

	checksLocator RepositoryRef
	checksID      int
	checksItems   []Check
	checksMore    bool
	checksErr     error
	checksCalls   int
}

func (f *fakeC2BBackend) getPullRequest(_ context.Context, locator RepositoryRef, id int) (PullRequest, error) {
	f.getPRCalls++
	f.getPRLocator = locator
	f.getPRID = id
	return f.getPRResult, f.getPRErr
}

func (f *fakeC2BBackend) getPullRequestDiff(_ context.Context, locator RepositoryRef, id int) (Diff, error) {
	f.diffCalls++
	f.diffLocator = locator
	f.diffID = id
	return f.diffResult, f.diffErr
}

func (f *fakeC2BBackend) listPullRequestComments(_ context.Context, locator RepositoryRef, id, limit int) ([]Comment, bool, error) {
	f.commentCalls++
	f.commentLocator = locator
	f.commentID = id
	f.commentLimit = limit
	return f.commentItems, f.commentMore, f.commentErr
}

func (f *fakeC2BBackend) getPullRequestChecks(_ context.Context, locator RepositoryRef, id int) ([]Check, bool, error) {
	f.checksCalls++
	f.checksLocator = locator
	f.checksID = id
	return f.checksItems, f.checksMore, f.checksErr
}

func TestPullRequestDetailToolsRoundTrip(t *testing.T) {
	description := boundBitbucketText("full description", PullRequestDescriptionLimit)
	backend := &fakeC2BBackend{
		getPRResult: PullRequest{ID: 7, Title: "Detail", Description: &description, Reviewers: []Reviewer{}},
		diffResult:  adaptDiff("diff --git a/a b/a\n", "source-sha", "target-sha"),
		commentItems: []Comment{{
			ID:   11,
			Body: boundBitbucketText("review note", CommentBodyLimit),
		}},
		commentMore: true,
		checksItems: []Check{{Key: "ci", Name: "CI", State: CheckSuccessful}},
		checksMore:  true,
	}
	snap := &Snapshot{Platform: "cloud", HostLabel: "cloud", DefaultScope: "team", DefaultRepo: "api"}
	session := connectPair(t, newFullServer(snap, "test", backend))

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools.Tools) != 9 {
		t.Fatalf("tool count = %d, want 9 after C.2b registration", len(tools.Tools))
	}
	c2bTools := map[string]bool{
		"bkt_get_pull_request":           false,
		"bkt_get_pull_request_diff":      false,
		"bkt_list_pull_request_comments": false,
		"bkt_get_pull_request_checks":    false,
	}
	for _, tool := range tools.Tools {
		if _, ok := c2bTools[tool.Name]; !ok {
			continue
		}
		c2bTools[tool.Name] = true
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Fatalf("tool %q is missing the read-only annotation", tool.Name)
		}
	}
	for name, found := range c2bTools {
		if !found {
			t.Fatalf("tool %q was not registered", name)
		}
	}

	prRes, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "bkt_get_pull_request",
		Arguments: map[string]any{"id": 7},
	})
	if err != nil {
		t.Fatalf("get pull request: %v", err)
	}
	var pr PullRequest
	decodeStructuredContent(t, prRes, &pr)
	if pr.ID != 7 || pr.Description == nil || pr.Description.Text != "full description" {
		t.Fatalf("pull request = %+v", pr)
	}
	if backend.getPRLocator != (RepositoryRef{Scope: "team", Slug: "api"}) || backend.getPRID != 7 {
		t.Fatalf("get pull request args = locator:%+v id:%d", backend.getPRLocator, backend.getPRID)
	}

	diffRes, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "bkt_get_pull_request_diff",
		Arguments: map[string]any{
			"id":      8,
			"locator": map[string]any{"scope": "other", "slug": "repo"},
		},
	})
	if err != nil {
		t.Fatalf("get pull request diff: %v", err)
	}
	var diff Diff
	decodeStructuredContent(t, diffRes, &diff)
	if diff.SourceCommit != "source-sha" || diff.TargetCommit != "target-sha" || diff.Content.Provenance.Trust != ProvenanceTrustUntrusted {
		t.Fatalf("diff = %+v", diff)
	}
	if backend.diffLocator != (RepositoryRef{Scope: "other", Slug: "repo"}) || backend.diffID != 8 {
		t.Fatalf("diff args = locator:%+v id:%d", backend.diffLocator, backend.diffID)
	}

	commentsRes, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "bkt_list_pull_request_comments",
		Arguments: map[string]any{"id": 9, "limit": 3},
	})
	if err != nil {
		t.Fatalf("list pull request comments: %v", err)
	}
	var comments ListEnvelope[Comment]
	decodeStructuredContent(t, commentsRes, &comments)
	if comments.Count != 1 || comments.Limit != 3 || !comments.Truncated || comments.Items[0].Body.Text != "review note" {
		t.Fatalf("comments = %+v", comments)
	}
	if backend.commentLimit != 3 || backend.commentID != 9 {
		t.Fatalf("comment args = limit:%d id:%d", backend.commentLimit, backend.commentID)
	}

	checksRes, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "bkt_get_pull_request_checks",
		Arguments: map[string]any{"id": 10},
	})
	if err != nil {
		t.Fatalf("get pull request checks: %v", err)
	}
	var checks ListEnvelope[Check]
	decodeStructuredContent(t, checksRes, &checks)
	if checks.Count != 1 || checks.Limit != MaxListLimit || !checks.Truncated || checks.Items[0].State != CheckSuccessful {
		t.Fatalf("checks = %+v", checks)
	}
	if backend.checksID != 10 || backend.checksLocator != (RepositoryRef{Scope: "team", Slug: "api"}) {
		t.Fatalf("checks args = locator:%+v id:%d", backend.checksLocator, backend.checksID)
	}
}

func TestPullRequestDetailToolsRejectInvalidInputBeforeBackend(t *testing.T) {
	backend := &fakeC2BBackend{}
	session := connectPair(t, newFullServer(
		&Snapshot{Platform: "dc", HostLabel: "dc", DefaultScope: "PROJ", DefaultRepo: "api"},
		"test",
		backend,
	))

	tests := []struct {
		name string
		tool string
		args map[string]any
	}{
		{name: "get missing id", tool: "bkt_get_pull_request", args: map[string]any{}},
		{name: "diff negative id", tool: "bkt_get_pull_request_diff", args: map[string]any{"id": -1}},
		{name: "comments zero id", tool: "bkt_list_pull_request_comments", args: map[string]any{"id": 0}},
		{name: "checks partial locator", tool: "bkt_get_pull_request_checks", args: map[string]any{"id": 1, "locator": map[string]any{"scope": "PROJ"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tt.tool, Arguments: tt.args})
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}
			got := decodeStructuredToolError(t, res)
			if got.Code != ErrorInvalidInput || got.Retryable {
				t.Fatalf("error = %+v", got)
			}
		})
	}
	if backend.getPRCalls != 0 || backend.diffCalls != 0 || backend.commentCalls != 0 || backend.checksCalls != 0 {
		t.Fatalf("backend calls = get:%d diff:%d comments:%d checks:%d, want zero", backend.getPRCalls, backend.diffCalls, backend.commentCalls, backend.checksCalls)
	}
}

func TestPullRequestDetailToolsMapBackendErrors(t *testing.T) {
	backend := &fakeC2BBackend{diffErr: errors.New("secret upstream detail")}
	session := connectPair(t, newFullServer(
		&Snapshot{Platform: "cloud", HostLabel: "cloud", DefaultScope: "team", DefaultRepo: "api"},
		"test",
		backend,
	))

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "bkt_get_pull_request_diff",
		Arguments: map[string]any{"id": 1},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	got := decodeStructuredToolError(t, res)
	if got.Code != ErrorUpstream || got.Message != "Bitbucket request failed" || got.Retryable {
		t.Fatalf("error = %+v", got)
	}
}

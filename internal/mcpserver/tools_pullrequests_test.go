package mcpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

type fakeC2ABackend struct {
	fakeRepositoryBackend

	listPRLocator RepositoryRef
	listPRState   string
	listPRRole    string
	listPRLimit   int
	listPRItems   []PullRequest
	listPRMore    bool
	listPRErr     error
	listPRCalls   int

	listMyScope string
	listMyState string
	listMyRole  string
	listMyLimit int
	listMyItems []PullRequest
	listMyMore  bool
	listMyErr   error
	listMyCalls int
}

func (f *fakeC2ABackend) listPullRequests(_ context.Context, locator RepositoryRef, state, role string, limit int) ([]PullRequest, bool, error) {
	f.listPRCalls++
	f.listPRLocator = locator
	f.listPRState = state
	f.listPRRole = role
	f.listPRLimit = limit
	return f.listPRItems, f.listPRMore, f.listPRErr
}

func (f *fakeC2ABackend) listMyPullRequests(_ context.Context, scope, state, role string, limit int) ([]PullRequest, bool, error) {
	f.listMyCalls++
	f.listMyScope = scope
	f.listMyState = state
	f.listMyRole = role
	f.listMyLimit = limit
	return f.listMyItems, f.listMyMore, f.listMyErr
}

func TestListPullRequestsToolPassesValidatedFiltersBeforeLimit(t *testing.T) {
	backend := &fakeC2ABackend{
		listPRItems: []PullRequest{{ID: 7, Title: "Review me", Reviewers: []Reviewer{}}},
		listPRMore:  true,
	}
	snap := &Snapshot{Platform: "cloud", HostLabel: "cloud", DefaultScope: "team", DefaultRepo: "api"}
	session := connectPair(t, newServer(snap, "test", backend))

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "bkt_list_pull_requests",
		Arguments: map[string]any{
			"state": "merged",
			"role":  "reviewer",
			"limit": 2,
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool error: %+v", res.Content)
	}
	if backend.listPRLocator != (RepositoryRef{Scope: "team", Slug: "api"}) || backend.listPRState != "MERGED" || backend.listPRRole != "REVIEWER" || backend.listPRLimit != 2 {
		t.Fatalf("backend args = locator:%+v state:%q role:%q limit:%d", backend.listPRLocator, backend.listPRState, backend.listPRRole, backend.listPRLimit)
	}
	var got ListEnvelope[PullRequest]
	decodeStructuredContent(t, res, &got)
	if got.Count != 1 || got.Limit != 2 || !got.Truncated || len(got.Items) != 1 {
		t.Fatalf("result = %+v", got)
	}
}

func TestPullRequestListToolsPreserveEmptyUpstreamContinuation(t *testing.T) {
	tests := []struct {
		name string
		tool string
		args map[string]any
	}{
		{name: "repository", tool: "bkt_list_pull_requests", args: map[string]any{}},
		{name: "cross repository", tool: "bkt_list_my_pull_requests", args: map[string]any{"role": "author"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := &fakeC2ABackend{listPRMore: true, listMyMore: true}
			session := connectPair(t, newServer(
				&Snapshot{Platform: "dc", HostLabel: "dc", DefaultScope: "PROJ", DefaultRepo: "api"},
				"test",
				backend,
			))

			res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: tt.tool, Arguments: tt.args})
			if err != nil {
				t.Fatalf("CallTool: %v", err)
			}
			var got ListEnvelope[PullRequest]
			decodeStructuredContent(t, res, &got)
			if got.Items == nil || got.Count != 0 || !got.Truncated {
				t.Fatalf("result = %#v, want non-nil empty items with continuation", got)
			}
		})
	}
}

func TestListPullRequestsToolRejectsInvalidFilterBeforeBackend(t *testing.T) {
	backend := &fakeC2ABackend{}
	snap := &Snapshot{Platform: "dc", HostLabel: "dc", DefaultScope: "PROJ", DefaultRepo: "api"}
	session := connectPair(t, newServer(snap, "test", backend))

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "bkt_list_pull_requests",
		Arguments: map[string]any{
			"state": "deleted",
			"role":  "owner",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	got := decodeStructuredToolError(t, res)
	if got.Code != ErrorInvalidInput || backend.listPRCalls != 0 {
		t.Fatalf("error = %+v, backend calls = %d", got, backend.listPRCalls)
	}
}

func TestListPullRequestsToolDCBearerOnlyRoleIsInvalidInputBeforeHTTP(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"values":[],"isLastPage":true}`))
	}))
	t.Cleanup(server.Close)
	client, err := bbdc.New(bbdc.Options{
		BaseURL: server.URL, AuthMethod: "bearer", Token: "token",
		Retry: httpx.RetryPolicy{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	backend := &dcBackend{client: client}
	snap := &Snapshot{Platform: "dc", HostLabel: "dc", DefaultScope: "PROJ", DefaultRepo: "api"}
	session := connectPair(t, newServer(snap, "test", backend))

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "bkt_list_pull_requests",
		Arguments: map[string]any{
			"role": "reviewer",
		},
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	got := decodeStructuredToolError(t, res)
	if got.Code != ErrorInvalidInput || got.Retryable {
		t.Fatalf("error = %+v", got)
	}
	if requests.Load() != 0 {
		t.Fatalf("HTTP requests = %d, want zero", requests.Load())
	}

	res, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "bkt_list_pull_requests",
		Arguments: map[string]any{"role": "all"},
	})
	if err != nil {
		t.Fatalf("role=all CallTool: %v", err)
	}
	if res.IsError || requests.Load() != 1 {
		t.Fatalf("role=all result=%+v HTTP requests=%d, want success and one request", res, requests.Load())
	}
}

func TestListMyPullRequestsToolCloudAuthorAndReviewerCapability(t *testing.T) {
	backend := &fakeC2ABackend{
		listMyItems: []PullRequest{{ID: 8, Title: "Mine", Reviewers: []Reviewer{}}},
	}
	snap := &Snapshot{Platform: "cloud", HostLabel: "cloud", DefaultScope: "team"}
	session := connectPair(t, newServer(snap, "test", backend))

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "bkt_list_my_pull_requests",
		Arguments: map[string]any{
			"role": "author",
		},
	})
	if err != nil {
		t.Fatalf("author CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("author tool error: %+v", res.Content)
	}
	if backend.listMyScope != "team" || backend.listMyState != "OPEN" || backend.listMyRole != "AUTHOR" || backend.listMyLimit != DefaultListLimit {
		t.Fatalf("author backend args = scope:%q state:%q role:%q limit:%d", backend.listMyScope, backend.listMyState, backend.listMyRole, backend.listMyLimit)
	}

	res, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "bkt_list_my_pull_requests",
		Arguments: map[string]any{
			"role": "reviewer",
		},
	})
	if err != nil {
		t.Fatalf("reviewer CallTool: %v", err)
	}
	toolErr := decodeStructuredToolError(t, res)
	if toolErr.Code != ErrorUnsupportedOnPlatform || backend.listMyCalls != 1 {
		t.Fatalf("reviewer error = %+v, backend calls = %d", toolErr, backend.listMyCalls)
	}
}

func TestListMyPullRequestsToolRequiresRoleAndCloudScope(t *testing.T) {
	backend := &fakeC2ABackend{}
	session := connectPair(t, newServer(&Snapshot{Platform: "cloud", HostLabel: "cloud"}, "test", backend))

	for _, args := range []map[string]any{
		{},
		{"role": "author"},
	} {
		res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "bkt_list_my_pull_requests", Arguments: args})
		if err != nil {
			t.Fatalf("CallTool(%v): %v", args, err)
		}
		got := decodeStructuredToolError(t, res)
		if got.Code != ErrorInvalidInput {
			t.Fatalf("CallTool(%v) error = %+v", args, got)
		}
	}
	if backend.listMyCalls != 0 {
		t.Fatalf("backend calls = %d, want zero", backend.listMyCalls)
	}
}

func TestCapabilitiesAdvertiseOnlyCrossRepoReviewerDifference(t *testing.T) {
	dc := capabilities(&Snapshot{Platform: "dc"})
	if len(dc) != 1 || dc[0] != "my_prs.role.reviewer" {
		t.Fatalf("DC capabilities = %v", dc)
	}
	cloud := capabilities(&Snapshot{Platform: "cloud"})
	if cloud == nil || len(cloud) != 0 {
		t.Fatalf("Cloud capabilities = %#v, want non-nil empty", cloud)
	}
}

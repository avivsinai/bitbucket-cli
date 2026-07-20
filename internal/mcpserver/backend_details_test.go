package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

func dcPullRequestPayload() map[string]any {
	return map[string]any{
		"id":          7,
		"title":       "Detail",
		"description": "full description",
		"state":       "OPEN",
		"createdDate": int64(1704067200000),
		"updatedDate": int64(1704153600000),
		"author":      map[string]any{"user": map[string]any{"name": "alice", "displayName": "Alice"}},
		"fromRef": map[string]any{
			"displayId":    "feature",
			"latestCommit": "source-sha",
			"repository":   map[string]any{"slug": "api", "project": map[string]any{"key": "PROJ"}},
		},
		"toRef": map[string]any{
			"displayId":    "main",
			"latestCommit": "target-sha",
			"repository":   map[string]any{"slug": "api", "project": map[string]any{"key": "PROJ"}},
		},
		"reviewers": []any{},
	}
}

func cloudPullRequestPayload() map[string]any {
	return map[string]any{
		"id":          7,
		"title":       "Detail",
		"description": "full description",
		"state":       "OPEN",
		"created_on":  "2024-01-01T00:00:00Z",
		"updated_on":  "2024-01-02T00:00:00Z",
		"author":      map[string]any{"nickname": "alice", "display_name": "Alice"},
		"source": map[string]any{
			"branch":     map[string]any{"name": "feature"},
			"commit":     map[string]any{"hash": "source-sha"},
			"repository": map[string]any{"slug": "api", "full_name": "team/api"},
		},
		"destination": map[string]any{
			"branch":     map[string]any{"name": "main"},
			"commit":     map[string]any{"hash": "target-sha"},
			"repository": map[string]any{"slug": "api", "full_name": "team/api"},
		},
		"reviewers":    []any{},
		"participants": []any{},
	}
}

func TestDCBackendPullRequestDetailAndBoundedDiff(t *testing.T) {
	diffText := strings.Repeat("x", DiffContentLimit+17)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/1.0/projects/PROJ/repos/api/pull-requests/7":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(dcPullRequestPayload())
		case "/rest/api/1.0/projects/PROJ/repos/api/pull-requests/7/diff":
			if r.Header.Get("Accept") != "text/plain" {
				t.Fatalf("Accept = %q", r.Header.Get("Accept"))
			}
			_, _ = w.Write([]byte(diffText))
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	client, err := bbdc.New(bbdc.Options{BaseURL: server.URL, Token: "token", AuthMethod: "bearer", Retry: httpx.RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatal(err)
	}
	backend := &dcBackend{client: client}
	locator := RepositoryRef{Scope: "PROJ", Slug: "api"}

	pr, err := backend.getPullRequest(context.Background(), locator, 7)
	if err != nil {
		t.Fatalf("getPullRequest: %v", err)
	}
	if pr.Description == nil || pr.Description.Text != "full description" || pr.SourceBranch != "feature" {
		t.Fatalf("pull request = %+v", pr)
	}

	diff, err := backend.getPullRequestDiff(context.Background(), locator, 7)
	if err != nil {
		t.Fatalf("getPullRequestDiff: %v", err)
	}
	if len(diff.Content.Text) != DiffContentLimit || !diff.Content.Truncated || diff.Content.OriginalSize == nil || *diff.Content.OriginalSize != len(diffText) {
		t.Fatalf("bounded diff = %+v", diff.Content)
	}
	if diff.SourceCommit != "source-sha" || diff.TargetCommit != "target-sha" {
		t.Fatalf("diff commits = source:%q target:%q", diff.SourceCommit, diff.TargetCommit)
	}
}

func TestDCBackendCommentsScanActivitiesUntilTruncationIsKnown(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("start") {
		case "0":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"action": "APPROVED"},
					{"action": "COMMENTED", "comment": map[string]any{"id": 1, "text": "one"}},
				},
				"isLastPage": false, "nextPageStart": 2,
			})
		case "2":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"action": "COMMENTED", "comment": map[string]any{"id": 2, "text": "two"}},
					{"action": "COMMENTED", "comment": map[string]any{"id": 3, "text": "three", "createdDate": -1}},
				},
				"isLastPage": true,
			})
		default:
			t.Fatalf("unexpected query %q", r.URL.RawQuery)
		}
	}))
	t.Cleanup(server.Close)
	client, err := bbdc.New(bbdc.Options{BaseURL: server.URL, Token: "token", AuthMethod: "bearer", Retry: httpx.RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatal(err)
	}

	items, hasMore, err := (&dcBackend{client: client}).listPullRequestComments(
		context.Background(), RepositoryRef{Scope: "PROJ", Slug: "api"}, 7, 2,
	)
	if err != nil {
		t.Fatalf("listPullRequestComments: %v", err)
	}
	if len(items) != 2 || items[0].ID != 1 || items[1].ID != 2 || !hasMore {
		t.Fatalf("items=%+v hasMore=%v", items, hasMore)
	}
	if len(requests) != 2 || !strings.Contains(requests[0], "limit=3") || !strings.Contains(requests[1], "start=2") {
		t.Fatalf("requests = %v", requests)
	}
}

func TestDCBackendChecksResolveSourceCommitAndPreserveContinuation(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/api/1.0/projects/PROJ/repos/api/pull-requests/7":
			_ = json.NewEncoder(w).Encode(dcPullRequestPayload())
		case "/rest/build-status/1.0/commits/source-sha":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values":     []map[string]any{{"key": "ci", "name": "CI", "state": "SUCCESSFUL"}},
				"isLastPage": false, "nextPageStart": 100,
			})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	client, err := bbdc.New(bbdc.Options{BaseURL: server.URL, Token: "token", AuthMethod: "bearer", Retry: httpx.RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatal(err)
	}

	checks, hasMore, err := (&dcBackend{client: client}).getPullRequestChecks(
		context.Background(), RepositoryRef{Scope: "PROJ", Slug: "api"}, 7,
	)
	if err != nil {
		t.Fatalf("getPullRequestChecks: %v", err)
	}
	if len(checks) != 1 || checks[0].State != CheckSuccessful || !hasMore {
		t.Fatalf("checks=%+v hasMore=%v", checks, hasMore)
	}
	if len(requests) != 2 || !strings.Contains(requests[1], "limit=100") || !strings.Contains(requests[1], "start=0") {
		t.Fatalf("requests = %v", requests)
	}
}

func TestDCBackendChecksReportTerminalHundredItemCapAsTruncated(t *testing.T) {
	statuses := make([]map[string]any, MaxListLimit)
	for i := range statuses {
		statuses[i] = map[string]any{"key": "ci", "state": "SUCCESSFUL"}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/api/1.0/projects/PROJ/repos/api/pull-requests/7":
			_ = json.NewEncoder(w).Encode(dcPullRequestPayload())
		case "/rest/build-status/1.0/commits/source-sha":
			_ = json.NewEncoder(w).Encode(map[string]any{"values": statuses, "isLastPage": true})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	client, err := bbdc.New(bbdc.Options{BaseURL: server.URL, Token: "token", AuthMethod: "bearer", Retry: httpx.RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatal(err)
	}

	checks, hasMore, err := (&dcBackend{client: client}).getPullRequestChecks(
		context.Background(), RepositoryRef{Scope: "PROJ", Slug: "api"}, 7,
	)
	if err != nil {
		t.Fatalf("getPullRequestChecks: %v", err)
	}
	if len(checks) != MaxListLimit || !hasMore {
		t.Fatalf("checks=%d hasMore=%v, want terminal cap reported as truncated", len(checks), hasMore)
	}
}

func TestBoundedTextSinkCountsFullStreamAndPreservesUTF8(t *testing.T) {
	sink := newBoundedTextSink(5)
	first := []byte{'a', 'b', 'c', 'd', 0xe2}
	second := []byte{0x82, 0xac, 't', 'a', 'i', 'l'}
	if n, err := sink.Write(first); err != nil || n != len(first) {
		t.Fatalf("first write = %d, %v", n, err)
	}
	if n, err := sink.Write(second); err != nil || n != len(second) {
		t.Fatalf("second write = %d, %v", n, err)
	}

	got := sink.boundedText()
	if got.Text != "abcd" || !utf8.ValidString(got.Text) || !got.Truncated {
		t.Fatalf("bounded text = %+v", got)
	}
	if got.OriginalSize == nil || *got.OriginalSize != len(first)+len(second) {
		t.Fatalf("original size = %v, want %d", got.OriginalSize, len(first)+len(second))
	}
}

func TestBoundedTextSinkUsesRawSizeWhenSanitizationCrossesLimit(t *testing.T) {
	sink := newBoundedTextSink(5)
	raw := []byte{'a', 'b', 'c', 'd', 0xff}
	if n, err := sink.Write(raw); err != nil || n != len(raw) {
		t.Fatalf("write = %d, %v", n, err)
	}

	got := sink.boundedText()
	if got.Text != "abcd" || !utf8.ValidString(got.Text) || !got.Truncated {
		t.Fatalf("bounded text = %+v", got)
	}
	if got.OriginalSize == nil || *got.OriginalSize != len(raw) {
		t.Fatalf("original size = %v, want raw size %d", got.OriginalSize, len(raw))
	}
}

func TestCloudBackendCommentsAndChecksUseBoundedPages(t *testing.T) {
	var requests []string
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/repositories/team/api/pullrequests/7":
			payload := cloudPullRequestPayload()
			payload["source"].(map[string]any)["repository"] = map[string]any{
				"slug": "fork", "full_name": "contributor/fork",
			}
			_ = json.NewEncoder(w).Encode(payload)
		case "/repositories/team/api/pullrequests/7/comments":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{{"id": 1, "content": map[string]any{"raw": "note"}}},
				"next":   server.URL + "/repositories/team/api/pullrequests/7/comments?pagelen=2&page=2",
			})
		case "/repositories/contributor/fork/commit/source-sha/statuses":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{{"key": "ci", "state": "SUCCESSFUL"}},
				"next":   server.URL + "/repositories/contributor/fork/commit/source-sha/statuses?pagelen=100&page=2",
			})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Token: "token", Retry: httpx.RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatal(err)
	}
	backend := &cloudBackend{client: client}
	locator := RepositoryRef{Scope: "team", Slug: "api"}

	comments, commentsMore, err := backend.listPullRequestComments(context.Background(), locator, 7, 2)
	if err != nil {
		t.Fatalf("listPullRequestComments: %v", err)
	}
	if len(comments) != 1 || comments[0].Body.Text != "note" || !commentsMore {
		t.Fatalf("comments=%+v hasMore=%v", comments, commentsMore)
	}

	checks, checksMore, err := backend.getPullRequestChecks(context.Background(), locator, 7)
	if err != nil {
		t.Fatalf("getPullRequestChecks: %v", err)
	}
	if len(checks) != 1 || checks[0].State != CheckSuccessful || !checksMore {
		t.Fatalf("checks=%+v hasMore=%v", checks, checksMore)
	}
	if len(requests) != 3 || !strings.Contains(requests[0], "pagelen=2") || !strings.Contains(requests[2], "pagelen=100") {
		t.Fatalf("requests = %v", requests)
	}
}

func TestCloudBackendChecksRejectIncompleteSourceRepositoryBeforeStatusHTTP(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/repositories/team/api/pullrequests/7" {
			t.Fatalf("unexpected request %q", r.URL.RequestURI())
		}
		payload := cloudPullRequestPayload()
		payload["source"].(map[string]any)["repository"] = map[string]any{"slug": "fork"}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	t.Cleanup(server.Close)
	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Token: "token", Retry: httpx.RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = (&cloudBackend{client: client}).getPullRequestChecks(
		context.Background(), RepositoryRef{Scope: "team", Slug: "api"}, 7,
	)
	if err == nil || !strings.Contains(err.Error(), "source repository identity") {
		t.Fatalf("error = %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want only pull request lookup", requests)
	}
}

func TestCloudBackendPullRequestDetailAndDiffCommits(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repositories/team/api/pullrequests/7":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(cloudPullRequestPayload())
		case "/repositories/team/api/pullrequests/7/diff":
			_, _ = w.Write([]byte("diff --git a/a b/a\n"))
		case "/repositories/team/api/commit/source-sha/statuses":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
		default:
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
	}))
	t.Cleanup(server.Close)
	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Token: "token", Retry: httpx.RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatal(err)
	}
	backend := &cloudBackend{client: client}
	locator := RepositoryRef{Scope: "team", Slug: "api"}

	pr, err := backend.getPullRequest(context.Background(), locator, 7)
	if err != nil {
		t.Fatalf("getPullRequest: %v", err)
	}
	if pr.Description == nil || pr.Description.Text != "full description" || pr.Repo != locator {
		t.Fatalf("pull request = %+v", pr)
	}
	diff, err := backend.getPullRequestDiff(context.Background(), locator, 7)
	if err != nil {
		t.Fatalf("getPullRequestDiff: %v", err)
	}
	if diff.SourceCommit != "source-sha" || diff.TargetCommit != "target-sha" || diff.Content.Text != "diff --git a/a b/a\n" {
		t.Fatalf("diff = %+v", diff)
	}
	checks, hasMore, err := backend.getPullRequestChecks(context.Background(), locator, 7)
	if err != nil {
		t.Fatalf("getPullRequestChecks: %v", err)
	}
	if checks == nil || len(checks) != 0 || hasMore {
		t.Fatalf("checks=%#v hasMore=%v", checks, hasMore)
	}
}

package bbcloud_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
)

func newTestClient(t *testing.T, handler http.Handler) *bbcloud.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := bbcloud.New(bbcloud.Options{
		BaseURL:  server.URL,
		Username: "user",
		Token:    "token",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	return client
}

func TestGetPullRequest(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    7,
			"title": "Test PR",
			"state": "OPEN",
		})
	}))

	pr, err := client.GetPullRequest(context.Background(), "myworkspace", "my-repo", 7)
	if err != nil {
		t.Fatalf("GetPullRequest: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/repositories/myworkspace/my-repo/pullrequests/7" {
		t.Errorf("path = %q, want /repositories/myworkspace/my-repo/pullrequests/7", gotPath)
	}
	if pr.ID != 7 {
		t.Errorf("pr.ID = %d, want 7", pr.ID)
	}
}

func TestGetPullRequestValidation(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		workspace string
		repo      string
	}{
		{"empty workspace", "", "repo"},
		{"empty repo", "ws", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetPullRequest(context.Background(), tt.workspace, tt.repo, 1)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestListPullRequestsPaginates(t *testing.T) {
	var hits int32
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		switch count {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{{"id": 1}, {"id": 2}},
				"next":   serverURL + "/repositories/ws/repo/pullrequests?pagelen=20&page=2",
			})
		case 2:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{{"id": 3}},
			})
		default:
			t.Fatalf("unexpected request %d", count)
		}
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Username: "u", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}

	prs, err := client.ListPullRequests(context.Background(), "ws", "repo", bbcloud.PullRequestListOptions{
		State: "OPEN",
		Limit: 0,
	})
	if err != nil {
		t.Fatalf("ListPullRequests: %v", err)
	}
	if len(prs) != 3 {
		t.Fatalf("expected 3 PRs, got %d", len(prs))
	}
	if hits != 2 {
		t.Fatalf("expected 2 requests, got %d", hits)
	}
}

func TestListPullRequestsRespectsLimit(t *testing.T) {
	var hits int32
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{{"id": 1}, {"id": 2}, {"id": 3}},
			"next":   serverURL + "/repositories/ws/repo/pullrequests?page=2",
		})
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Username: "u", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}

	prs, err := client.ListPullRequests(context.Background(), "ws", "repo", bbcloud.PullRequestListOptions{
		Limit: 2,
	})
	if err != nil {
		t.Fatalf("ListPullRequests: %v", err)
	}
	if len(prs) != 2 {
		t.Errorf("expected 2 PRs, got %d", len(prs))
	}
}

func TestListRepositoriesPaginates(t *testing.T) {
	var hits int32
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		switch count {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{{"slug": "repo1"}},
				"next":   serverURL + "/repositories/ws?pagelen=20&page=2",
			})
		case 2:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{{"slug": "repo2"}},
			})
		default:
			t.Fatalf("unexpected request %d", count)
		}
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Username: "u", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}

	repos, err := client.ListRepositories(context.Background(), "ws", 0)
	if err != nil {
		t.Fatalf("ListRepositories: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if hits != 2 {
		t.Fatalf("expected 2 requests, got %d", hits)
	}
}

func TestListRepositoriesRespectsLimit(t *testing.T) {
	var hits int32
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{{"slug": "r1"}, {"slug": "r2"}, {"slug": "r3"}},
			"next":   serverURL + "/repositories/ws?page=2",
		})
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Username: "u", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}

	repos, err := client.ListRepositories(context.Background(), "ws", 2)
	if err != nil {
		t.Fatalf("ListRepositories: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(repos))
	}
}

func TestDeclinePullRequest(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))

	if err := client.DeclinePullRequest(context.Background(), "myworkspace", "my-repo", 7); err != nil {
		t.Fatalf("DeclinePullRequest: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/repositories/myworkspace/my-repo/pullrequests/7/decline" {
		t.Errorf("path = %s, want .../7/decline", gotPath)
	}
}

func TestDeclinePullRequestValidation(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		workspace string
		repo      string
	}{
		{"empty workspace", "", "repo"},
		{"empty repo", "ws", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.DeclinePullRequest(context.Background(), tt.workspace, tt.repo, 1); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestReopenPullRequest(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))

	if err := client.ReopenPullRequest(context.Background(), "myworkspace", "my-repo", 7); err != nil {
		t.Fatalf("ReopenPullRequest: %v", err)
	}
	if gotMethod != "PUT" {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
	if gotPath != "/repositories/myworkspace/my-repo/pullrequests/7" {
		t.Errorf("path = %s, want .../pullrequests/7", gotPath)
	}
}

func TestReopenPullRequestValidation(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		workspace string
		repo      string
	}{
		{"empty workspace", "", "repo"},
		{"empty repo", "ws", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.ReopenPullRequest(context.Background(), tt.workspace, tt.repo, 1); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestCommentPullRequest(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))

	err := client.CommentPullRequest(context.Background(), "myworkspace", "my-repo", 7, bbcloud.CommentOptions{Text: "LGTM"})
	if err != nil {
		t.Fatalf("CommentPullRequest: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/repositories/myworkspace/my-repo/pullrequests/7/comments" {
		t.Errorf("path = %s, want .../comments", gotPath)
	}

	content, ok := gotBody["content"].(map[string]any)
	if !ok {
		t.Fatalf("request body missing content object")
	}
	if raw, ok := content["raw"].(string); !ok || raw != "LGTM" {
		t.Errorf("content.raw = %q, want LGTM", raw)
	}
}

func TestCommentPullRequestValidation(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		workspace string
		repo      string
		text      string
	}{
		{"empty workspace", "", "repo", "text"},
		{"empty repo", "ws", "", "text"},
		{"empty text", "ws", "repo", ""},
		{"blank text", "ws", "repo", "   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.CommentPullRequest(context.Background(), tt.workspace, tt.repo, 1, bbcloud.CommentOptions{Text: tt.text}); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestCommentPullRequestWithParent(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))

	err := client.CommentPullRequest(context.Background(), "myworkspace", "my-repo", 7, bbcloud.CommentOptions{Text: "reply", ParentID: 42})
	if err != nil {
		t.Fatalf("CommentPullRequest with parent: %v", err)
	}

	parent, ok := gotBody["parent"].(map[string]any)
	if !ok {
		t.Fatal("request body missing parent object")
	}
	if id, ok := parent["id"].(float64); !ok || int(id) != 42 {
		t.Errorf("parent.id = %v, want 42", parent["id"])
	}
}

func TestCommentPullRequestWithoutParent(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))

	err := client.CommentPullRequest(context.Background(), "myworkspace", "my-repo", 7, bbcloud.CommentOptions{Text: "top-level"})
	if err != nil {
		t.Fatalf("CommentPullRequest without parent: %v", err)
	}

	if _, ok := gotBody["parent"]; ok {
		t.Error("expected no parent field in body when parentID is 0")
	}
}

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

func TestCommentPullRequestPending(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))

	err := client.CommentPullRequest(context.Background(), "myworkspace", "my-repo", 7, bbcloud.CommentOptions{
		Text:    "draft feedback",
		Pending: true,
	})
	if err != nil {
		t.Fatalf("CommentPullRequest pending: %v", err)
	}

	pending, ok := gotBody["pending"].(bool)
	if !ok || !pending {
		t.Errorf("pending = %v, want true", gotBody["pending"])
	}
}

func TestCommentPullRequestNotPending(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))

	err := client.CommentPullRequest(context.Background(), "myworkspace", "my-repo", 7, bbcloud.CommentOptions{
		Text: "regular comment",
	})
	if err != nil {
		t.Fatalf("CommentPullRequest: %v", err)
	}

	if _, ok := gotBody["pending"]; ok {
		t.Error("expected no pending field when Pending is false")
	}
}

func TestPullRequestDiff(t *testing.T) {
	const wantDiff = "diff --git a/foo.go b/foo.go\n--- a/foo.go\n+++ b/foo.go\n"
	var gotMethod, gotPath, gotAccept string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(wantDiff))
	}))

	var buf strings.Builder
	err := client.PullRequestDiff(context.Background(), "myworkspace", "my-repo", 7, &buf)
	if err != nil {
		t.Fatalf("PullRequestDiff: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/repositories/myworkspace/my-repo/pullrequests/7/diff" {
		t.Errorf("path = %q, want /repositories/myworkspace/my-repo/pullrequests/7/diff", gotPath)
	}
	if gotAccept != "text/plain" {
		t.Errorf("Accept = %q, want text/plain", gotAccept)
	}
	if buf.String() != wantDiff {
		t.Errorf("diff body = %q, want %q", buf.String(), wantDiff)
	}
}

func TestPullRequestDiffValidation(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	tests := []struct {
		name      string
		workspace string
		repo      string
		writer    io.Writer
	}{
		{"empty workspace", "", "repo", &buf},
		{"empty repo", "ws", "", &buf},
		{"nil writer", "ws", "repo", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.PullRequestDiff(context.Background(), tt.workspace, tt.repo, 1, tt.writer); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestMergePullRequest(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))

	err := client.MergePullRequest(context.Background(), "myworkspace", "my-repo", 7, "squash commit", "squash", true)
	if err != nil {
		t.Fatalf("MergePullRequest: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/repositories/myworkspace/my-repo/pullrequests/7/merge" {
		t.Errorf("path = %s, want .../7/merge", gotPath)
	}
	if gotBody["message"] != "squash commit" {
		t.Errorf("body.message = %v, want %q", gotBody["message"], "squash commit")
	}
	if gotBody["merge_strategy"] != "squash" {
		t.Errorf("body.merge_strategy = %v, want %q", gotBody["merge_strategy"], "squash")
	}
	if gotBody["close_source_branch"] != true {
		t.Errorf("body.close_source_branch = %v, want true", gotBody["close_source_branch"])
	}
}

func TestMergePullRequestValidation(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		workspace string
		repo      string
	}{
		{"empty workspace", "", "repo"},
		{"empty repo", "ws", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.MergePullRequest(context.Background(), tt.workspace, tt.repo, 1, "", "", false); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestMergePullRequestInvalidStrategy(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Invalid strategy should return error
	invalidStrategies := []string{"squah", "rebase", "merge-commit", "SQUASH"}
	for _, s := range invalidStrategies {
		t.Run("invalid_"+s, func(t *testing.T) {
			if err := client.MergePullRequest(context.Background(), "ws", "repo", 1, "", s, false); err == nil {
				t.Errorf("expected error for strategy %q", s)
			}
		})
	}

	// Valid strategies should not fail validation (they'll fail on network, but not validation).
	// Empty string is valid (means "use default"). A network error is expected here.
	_ = client.MergePullRequest(context.Background(), "ws", "repo", 1, "", "", false)
}

func TestMergePullRequest202AsyncPolling(t *testing.T) {
	var pollCount int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/merge") {
			// Return 202 with task_id
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"task_id": "abc-123",
			})
			return
		}
		if strings.Contains(r.URL.Path, "/task-status/") {
			count := atomic.AddInt32(&pollCount, 1)
			w.Header().Set("Content-Type", "application/json")
			if count < 3 {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"task_status": "PENDING",
				})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"task_status": "SUCCESS",
				})
			}
			return
		}
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))

	err := client.MergePullRequest(context.Background(), "ws", "repo", 1, "", "squash", false)
	if err != nil {
		t.Fatalf("MergePullRequest with 202: %v", err)
	}
	if pollCount != 3 {
		t.Errorf("expected 3 poll attempts, got %d", pollCount)
	}
}

func TestMergePullRequest202AsyncFailure(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/merge") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"task_id": "abc-456",
			})
			return
		}
		if strings.Contains(r.URL.Path, "/task-status/") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"task_status": "FAILED",
			})
			return
		}
	}))

	err := client.MergePullRequest(context.Background(), "ws", "repo", 1, "", "", false)
	if err == nil {
		t.Fatal("expected error for failed merge task")
	}
	if !strings.Contains(err.Error(), "merge task failed") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreatePullRequestReviewerAutoDetect(t *testing.T) {
	tests := []struct {
		name       string
		reviewers  []string
		wantFields []string // expected key for each reviewer in request body
	}{
		{
			name:       "uuid with braces",
			reviewers:  []string{"{550e8400-e29b-41d4-a716-446655440000}"},
			wantFields: []string{"uuid"},
		},
		{
			name:       "uuid without braces",
			reviewers:  []string{"550e8400-e29b-41d4-a716-446655440000"},
			wantFields: []string{"uuid"},
		},
		{
			name:       "username",
			reviewers:  []string{"alice"},
			wantFields: []string{"username"},
		},
		{
			name:       "mixed",
			reviewers:  []string{"{550e8400-e29b-41d4-a716-446655440000}", "bob"},
			wantFields: []string{"uuid", "username"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewDecoder(r.Body).Decode(&gotBody)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
			}))

			_, _ = client.CreatePullRequest(context.Background(), "ws", "repo", bbcloud.CreatePullRequestInput{
				Title:       "Test PR",
				Source:      "feature",
				Destination: "main",
				Reviewers:   tt.reviewers,
			})

			reviewers, ok := gotBody["reviewers"].([]any)
			if !ok {
				t.Fatal("reviewers missing from request body")
			}
			if len(reviewers) != len(tt.wantFields) {
				t.Fatalf("expected %d reviewers, got %d", len(tt.wantFields), len(reviewers))
			}
			for i, field := range tt.wantFields {
				rev := reviewers[i].(map[string]any)
				if _, ok := rev[field]; !ok {
					t.Errorf("reviewer[%d]: expected %q field, got keys %v", i, field, rev)
				}
			}
		})
	}
}

func TestApprovePullRequest(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))

	if err := client.ApprovePullRequest(context.Background(), "myworkspace", "my-repo", 7); err != nil {
		t.Fatalf("ApprovePullRequest: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/repositories/myworkspace/my-repo/pullrequests/7/approve" {
		t.Errorf("path = %s, want .../7/approve", gotPath)
	}
}

func TestApprovePullRequestValidation(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		workspace string
		repo      string
	}{
		{"empty workspace", "", "repo"},
		{"empty repo", "ws", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.ApprovePullRequest(context.Background(), tt.workspace, tt.repo, 1); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestGetEffectiveDefaultReviewers(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{"user": map[string]any{"username": "alice", "display_name": "Alice A", "uuid": "{aaa}"}},
				{"user": map[string]any{"username": "bob", "display_name": "Bob B", "uuid": "{bbb}"}},
			},
		})
	}))

	users, err := client.GetEffectiveDefaultReviewers(context.Background(), "myworkspace", "my-repo")
	if err != nil {
		t.Fatalf("GetEffectiveDefaultReviewers: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/repositories/myworkspace/my-repo/effective-default-reviewers" {
		t.Errorf("path = %q, want /repositories/myworkspace/my-repo/effective-default-reviewers", gotPath)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].Username != "alice" {
		t.Errorf("users[0].Username = %q, want alice", users[0].Username)
	}
	if users[0].Display != "Alice A" {
		t.Errorf("users[0].Display = %q, want Alice A", users[0].Display)
	}
	if users[1].Username != "bob" {
		t.Errorf("users[1].Username = %q, want bob", users[1].Username)
	}
}

func TestGetEffectiveDefaultReviewersPagination(t *testing.T) {
	calls := 0
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if calls == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"user": map[string]any{"username": "alice"}},
				},
				"next": "http://" + r.Host + "/repositories/ws/repo/effective-default-reviewers?pagelen=100&page=2",
			})
		} else {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"user": map[string]any{"username": "bob"}},
				},
			})
		}
	}))

	users, err := client.GetEffectiveDefaultReviewers(context.Background(), "ws", "repo")
	if err != nil {
		t.Fatalf("GetEffectiveDefaultReviewers: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users across pages, got %d", len(users))
	}
	if calls != 2 {
		t.Errorf("expected 2 API calls for pagination, got %d", calls)
	}
}

func TestGetEffectiveDefaultReviewersValidation(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		workspace string
		repo      string
	}{
		{"empty workspace", "", "repo"},
		{"empty repo", "ws", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetEffectiveDefaultReviewers(context.Background(), tt.workspace, tt.repo)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestListPullRequestComments(t *testing.T) {
	var gotMethod, gotPath string
	resolution := map[string]any{
		"user":       map[string]any{"display_name": "Charlie"},
		"created_on": "2024-01-15T10:00:00.000000+00:00",
	}
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{
					"id":      1,
					"content": map[string]string{"raw": "Looks good"},
					"user": map[string]any{
						"display_name": "Alice",
						"nickname":     "alice",
					},
					"created_on": "2024-01-10T10:00:00.000000+00:00",
					"updated_on": "2024-01-10T10:00:00.000000+00:00",
					"resolution": nil,
				},
				{
					"id":      2,
					"content": map[string]string{"raw": "Fixed"},
					"user": map[string]any{
						"display_name": "Bob",
						"nickname":     "bob",
					},
					"created_on": "2024-01-11T10:00:00.000000+00:00",
					"updated_on": "2024-01-11T10:00:00.000000+00:00",
					"resolution": resolution,
				},
			},
		})
	}))

	comments, err := client.ListPullRequestComments(context.Background(), "myworkspace", "my-repo", 42, 0)
	if err != nil {
		t.Fatalf("ListPullRequestComments: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/repositories/myworkspace/my-repo/pullrequests/42/comments" {
		t.Errorf("path = %q, want /repositories/myworkspace/my-repo/pullrequests/42/comments", gotPath)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
	if comments[0].ID != 1 {
		t.Errorf("comments[0].ID = %d, want 1", comments[0].ID)
	}
	if comments[0].Content.Raw != "Looks good" {
		t.Errorf("comments[0].Content.Raw = %q, want %q", comments[0].Content.Raw, "Looks good")
	}
	if comments[0].User == nil {
		t.Fatal("comments[0].User is nil")
	}
	if comments[0].User.DisplayName != "Alice" {
		t.Errorf("comments[0].User.DisplayName = %q, want %q", comments[0].User.DisplayName, "Alice")
	}
	if comments[0].Resolution != nil {
		t.Error("comments[0].Resolution should be nil")
	}
	if comments[1].Resolution == nil {
		t.Fatal("comments[1].Resolution should not be nil")
	}
}

func TestListPullRequestCommentsPaginates(t *testing.T) {
	var hits int32
	var serverURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		switch count {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{{"id": 1, "content": map[string]string{"raw": "first"}}},
				"next":   serverURL + "/repositories/ws/repo/pullrequests/1/comments?pagelen=100&page=2",
			})
		case 2:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{{"id": 2, "content": map[string]string{"raw": "second"}}},
			})
		default:
			t.Fatalf("unexpected request %d", count)
		}
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Username: "u", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}

	comments, err := client.ListPullRequestComments(context.Background(), "ws", "repo", 1, 0)
	if err != nil {
		t.Fatalf("ListPullRequestComments: %v", err)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
	if hits != 2 {
		t.Fatalf("expected 2 requests, got %d", hits)
	}
}

func TestListPullRequestCommentsValidation(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		workspace string
		repo      string
	}{
		{"empty workspace", "", "repo"},
		{"empty repo", "ws", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.ListPullRequestComments(context.Background(), tt.workspace, tt.repo, 1, 0)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

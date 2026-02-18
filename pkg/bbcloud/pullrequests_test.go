package bbcloud_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	err := client.CommentPullRequest(context.Background(), "myworkspace", "my-repo", 7, "LGTM")
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
			if err := client.CommentPullRequest(context.Background(), tt.workspace, tt.repo, 1, tt.text); err == nil {
				t.Error("expected error")
			}
		})
	}
}

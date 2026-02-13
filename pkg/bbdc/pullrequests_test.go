package bbdc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
)

func newTestClient(t *testing.T, handler http.Handler) *bbdc.Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := bbdc.New(bbdc.Options{
		BaseURL:  server.URL,
		Username: "user",
		Token:    "token",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	return client
}

func TestGetPullRequestPathEscaping(t *testing.T) {
	var gotPath string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":    1,
			"title": "Test PR",
			"state": "OPEN",
		})
	}))

	_, err := client.GetPullRequest(context.Background(), "MY-PROJ", "my-repo", 99)
	if err != nil {
		t.Fatalf("GetPullRequest: %v", err)
	}
	want := "/rest/api/1.0/projects/MY-PROJ/repos/my-repo/pull-requests/99"
	if gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}

func TestGetPullRequestValidation(t *testing.T) {
	client, err := bbdc.New(bbdc.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		project string
		repo    string
	}{
		{"empty project", "", "repo"},
		{"empty repo", "PROJ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetPullRequest(context.Background(), tt.project, tt.repo, 1)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestListPullRequestsPaginates(t *testing.T) {
	var hits int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		switch count {
		case 1:
			json.NewEncoder(w).Encode(map[string]any{
				"values":        []map[string]any{{"id": 1, "title": "PR 1"}, {"id": 2, "title": "PR 2"}},
				"isLastPage":    false,
				"nextPageStart": 2,
			})
		case 2:
			json.NewEncoder(w).Encode(map[string]any{
				"values":     []map[string]any{{"id": 3, "title": "PR 3"}},
				"isLastPage": true,
			})
		default:
			t.Fatalf("unexpected request %d", count)
		}
	}))

	prs, err := client.ListPullRequests(context.Background(), "PROJ", "repo", "OPEN", 0)
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
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"values":        []map[string]any{{"id": 1}, {"id": 2}, {"id": 3}},
			"isLastPage":    false,
			"nextPageStart": 3,
		})
	}))

	prs, err := client.ListPullRequests(context.Background(), "PROJ", "repo", "OPEN", 2)
	if err != nil {
		t.Fatalf("ListPullRequests: %v", err)
	}
	if len(prs) != 2 {
		t.Errorf("expected 2 PRs, got %d", len(prs))
	}
}

func TestListPullRequestsPassesStateParam(t *testing.T) {
	var gotQuery string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"values":     []map[string]any{},
			"isLastPage": true,
		})
	}))

	_, err := client.ListPullRequests(context.Background(), "PROJ", "repo", "DECLINED", 10)
	if err != nil {
		t.Fatalf("ListPullRequests: %v", err)
	}
	if gotQuery == "" || !containsParam(gotQuery, "state=DECLINED") {
		t.Errorf("expected state=DECLINED in query, got %q", gotQuery)
	}
}

func TestListPullRequestsValidation(t *testing.T) {
	client, err := bbdc.New(bbdc.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		project string
		repo    string
	}{
		{"empty project", "", "repo"},
		{"empty repo", "PROJ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.ListPullRequests(context.Background(), tt.project, tt.repo, "OPEN", 10)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestListRepositoriesPaginates(t *testing.T) {
	var hits int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		default:
			t.Fatalf("unexpected request %d", count)
		}
	}))

	repos, err := client.ListRepositories(context.Background(), "PROJ", 0)
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
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"values":        []map[string]any{{"slug": "repo1"}, {"slug": "repo2"}, {"slug": "repo3"}},
			"isLastPage":    false,
			"nextPageStart": 3,
		})
	}))

	repos, err := client.ListRepositories(context.Background(), "PROJ", 2)
	if err != nil {
		t.Fatalf("ListRepositories: %v", err)
	}
	if len(repos) != 2 {
		t.Errorf("expected 2 repos, got %d", len(repos))
	}
}

func TestDeclinePullRequest(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))

	if err := client.DeclinePullRequest(context.Background(), "PROJ", "my-repo", 42, 3); err != nil {
		t.Fatalf("DeclinePullRequest: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests/42/decline" {
		t.Errorf("path = %s, want .../42/decline", gotPath)
	}
	if v, ok := gotBody["version"].(float64); !ok || int(v) != 3 {
		t.Errorf("version = %v, want 3", gotBody["version"])
	}
}

func TestDeclinePullRequestValidation(t *testing.T) {
	client, err := bbdc.New(bbdc.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		project string
		repo    string
	}{
		{"empty project", "", "repo"},
		{"empty repo", "PROJ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.DeclinePullRequest(context.Background(), tt.project, tt.repo, 1, 0); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestReopenPullRequest(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))

	if err := client.ReopenPullRequest(context.Background(), "PROJ", "my-repo", 42, 5); err != nil {
		t.Fatalf("ReopenPullRequest: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests/42/reopen" {
		t.Errorf("path = %s, want .../42/reopen", gotPath)
	}
	if v, ok := gotBody["version"].(float64); !ok || int(v) != 5 {
		t.Errorf("version = %v, want 5", gotBody["version"])
	}
}

func TestReopenPullRequestValidation(t *testing.T) {
	client, err := bbdc.New(bbdc.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		project string
		repo    string
	}{
		{"empty project", "", "repo"},
		{"empty repo", "PROJ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.ReopenPullRequest(context.Background(), tt.project, tt.repo, 1, 0); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func containsParam(query, param string) bool {
	for _, p := range splitQuery(query) {
		if p == param {
			return true
		}
	}
	return false
}

func splitQuery(query string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(query); i++ {
		if query[i] == '&' {
			parts = append(parts, query[start:i])
			start = i + 1
		}
	}
	parts = append(parts, query[start:])
	return parts
}

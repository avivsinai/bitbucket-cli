package bbdc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
		_ = json.NewEncoder(w).Encode(map[string]any{
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
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values":        []map[string]any{{"id": 1, "title": "PR 1"}, {"id": 2, "title": "PR 2"}},
				"isLastPage":    false,
				"nextPageStart": 2,
			})
		case 2:
			_ = json.NewEncoder(w).Encode(map[string]any{
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
		_ = json.NewEncoder(w).Encode(map[string]any{
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
		_ = json.NewEncoder(w).Encode(map[string]any{
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
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values":        []map[string]any{{"slug": "repo1"}},
				"isLastPage":    false,
				"nextPageStart": 1,
			})
		case 2:
			_ = json.NewEncoder(w).Encode(map[string]any{
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
		_ = json.NewEncoder(w).Encode(map[string]any{
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
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
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
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
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

func TestListPullRequestComments(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{
					"id":   10,
					"text": "Looks good to me",
					"author": map[string]any{
						"user": map[string]any{
							"name":        "alice",
							"displayName": "Alice A",
						},
					},
				},
				{
					"id":   11,
					"text": "Please fix the typo",
					"author": map[string]any{
						"user": map[string]any{
							"name":        "bob",
							"displayName": "Bob B",
						},
					},
				},
			},
			"isLastPage": true,
		})
	}))

	comments, err := client.ListPullRequestComments(context.Background(), "PROJ", "my-repo", 42)
	if err != nil {
		t.Fatalf("ListPullRequestComments: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests/42/comments" {
		t.Errorf("path = %q, want /rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests/42/comments", gotPath)
	}
	if len(comments) != 2 {
		t.Fatalf("expected 2 comments, got %d", len(comments))
	}
	if comments[0].ID != 10 {
		t.Errorf("comments[0].ID = %d, want 10", comments[0].ID)
	}
	if comments[0].Text != "Looks good to me" {
		t.Errorf("comments[0].Text = %q, want %q", comments[0].Text, "Looks good to me")
	}
	if comments[0].Author.User.Name != "alice" {
		t.Errorf("comments[0].Author.User.Name = %q, want %q", comments[0].Author.User.Name, "alice")
	}
}

func TestListPullRequestCommentsPaginates(t *testing.T) {
	var hits int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		switch count {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"id": 10, "text": "first comment", "author": map[string]any{"user": map[string]any{"name": "alice"}}},
				},
				"isLastPage":    false,
				"nextPageStart": 1,
			})
		case 2:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"id": 11, "text": "second comment", "author": map[string]any{"user": map[string]any{"name": "bob"}}},
				},
				"isLastPage": true,
			})
		default:
			t.Fatalf("unexpected request %d", count)
		}
	}))

	comments, err := client.ListPullRequestComments(context.Background(), "PROJ", "my-repo", 5)
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
			_, err := client.ListPullRequestComments(context.Background(), tt.project, tt.repo, 1)
			if err == nil {
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

	err := client.CommentPullRequest(context.Background(), "PROJ", "my-repo", 7, "LGTM", 0)
	if err != nil {
		t.Fatalf("CommentPullRequest: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests/7/comments" {
		t.Errorf("path = %s, want .../comments", gotPath)
	}
	if text, ok := gotBody["text"].(string); !ok || text != "LGTM" {
		t.Errorf("body.text = %v, want LGTM", gotBody["text"])
	}
}

func TestCommentPullRequestWithParent(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))

	err := client.CommentPullRequest(context.Background(), "PROJ", "my-repo", 7, "reply", 42)
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

	err := client.CommentPullRequest(context.Background(), "PROJ", "my-repo", 7, "top-level", 0)
	if err != nil {
		t.Fatalf("CommentPullRequest without parent: %v", err)
	}

	if _, ok := gotBody["parent"]; ok {
		t.Error("expected no parent field in body when parentID is 0")
	}
}

func TestCommentPullRequestValidation(t *testing.T) {
	client, err := bbdc.New(bbdc.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		text string
	}{
		{"empty text", ""},
		{"blank text", "   "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := client.CommentPullRequest(context.Background(), "PROJ", "repo", 1, tt.text, 0); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func containsParam(query, param string) bool {
	for _, p := range strings.Split(query, "&") {
		if p == param {
			return true
		}
	}
	return false
}

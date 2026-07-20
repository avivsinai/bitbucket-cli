package bbdc_test

import (
	"context"
	"encoding/json"
	"errors"
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

	if err := client.DeclinePullRequest(context.Background(), "PROJ", "my-repo", 42, 3, ""); err != nil {
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
	if _, ok := gotBody["comment"]; ok {
		t.Error("comment should be absent when empty string passed")
	}
}

func TestDeclinePullRequestWithComment(t *testing.T) {
	var gotBody map[string]any

	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusOK)
	}))

	if err := client.DeclinePullRequest(context.Background(), "PROJ", "my-repo", 42, 3, "needs more work"); err != nil {
		t.Fatalf("DeclinePullRequest: %v", err)
	}

	if gotBody["comment"] != "needs more work" {
		t.Errorf("comment = %v, want %q", gotBody["comment"], "needs more work")
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
			if err := client.DeclinePullRequest(context.Background(), tt.project, tt.repo, 1, 0, ""); err == nil {
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
					"action": "COMMENTED",
					"comment": map[string]any{
						"id":   10,
						"text": "Looks good to me",
						"author": map[string]any{
							"name":        "alice",
							"displayName": "Alice A",
						},
					},
				},
				{
					"action": "COMMENTED",
					"comment": map[string]any{
						"id":   11,
						"text": "Please fix the typo",
						"author": map[string]any{
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
	if gotPath != "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests/42/activities" {
		t.Errorf("path = %q, want /rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests/42/activities", gotPath)
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
	if comments[0].Author.Name != "alice" {
		t.Errorf("comments[0].Author.Name = %q, want %q", comments[0].Author.Name, "alice")
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
					{"action": "COMMENTED", "comment": map[string]any{"id": 10, "text": "first comment", "author": map[string]any{"name": "alice"}}},
				},
				"isLastPage":    false,
				"nextPageStart": 1,
			})
		case 2:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"action": "COMMENTED", "comment": map[string]any{"id": 11, "text": "second comment", "author": map[string]any{"name": "bob"}}},
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

func TestListPullRequestCommentsFlattensReplies(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{
					"action": "COMMENTED",
					"comment": map[string]any{
						"id": 1, "text": "parent", "author": map[string]any{"name": "alice"},
						"comments": []map[string]any{
							{
								"id": 2, "text": "reply", "author": map[string]any{"name": "bob"},
								"comments": []map[string]any{
									{"id": 3, "text": "nested reply", "author": map[string]any{"name": "alice"}},
								},
							},
						},
					},
				},
			},
			"isLastPage": true,
		})
	}))

	comments, err := client.ListPullRequestComments(context.Background(), "PROJ", "repo", 1)
	if err != nil {
		t.Fatalf("ListPullRequestComments: %v", err)
	}
	if len(comments) != 3 {
		t.Fatalf("expected 3 comments (flattened), got %d", len(comments))
	}
	wantIDs := []int{1, 2, 3}
	wantDepths := []int{0, 1, 2}
	for i, c := range comments {
		if c.ID != wantIDs[i] {
			t.Errorf("comments[%d].ID = %d, want %d", i, c.ID, wantIDs[i])
		}
		if c.Depth != wantDepths[i] {
			t.Errorf("comments[%d].Depth = %d, want %d", i, c.Depth, wantDepths[i])
		}
	}
}

func TestListPullRequestCommentsPagePreservesActivityPagination(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/projects/PROJ/repos/repo/pull-requests/42/activities" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "3" || r.URL.Query().Get("start") != "14" {
			t.Fatalf("query = %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{"action": "APPROVED"},
				{"action": "COMMENTED", "comment": map[string]any{"id": 9, "text": "note"}},
			},
			"isLastPage":    false,
			"nextPageStart": 17,
		})
	}))

	page, err := client.ListPullRequestCommentsPage(context.Background(), "PROJ", "repo", 42, 3, 14)
	if err != nil {
		t.Fatalf("ListPullRequestCommentsPage: %v", err)
	}
	if len(page.Values) != 1 || page.Values[0].ID != 9 || page.IsLast || page.NextStart != 17 {
		t.Fatalf("page = %+v", page)
	}
}

func TestSetPullRequestCommentThreadResolved(t *testing.T) {
	var gotPutPath string
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pull-requests/42/activities"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"isLastPage": true,
				"values": []map[string]any{
					{
						"action": "COMMENTED",
						"comment": map[string]any{
							"id":             9,
							"version":        3,
							"text":           "root",
							"severity":       "NORMAL",
							"state":          "OPEN",
							"properties":     map[string]any{"keep": "me"},
							"threadResolved": false,
							"author":         map[string]any{"displayName": "Reviewer"},
							"permittedOperations": map[string]any{
								"editable": true,
							},
							"anchor": map[string]any{
								"path": map[string]any{
									"components": []string{"src", "main.go"},
									"name":       "main.go",
									"parent":     "src",
								},
								"line":     10,
								"lineType": "ADDED",
								"fileType": "TO",
							},
							"comments": []map[string]any{
								{"id": 10, "version": 1, "text": "reply"},
							},
						},
					},
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/rest/api/1.0/projects/PROJ/repos/repo/pull-requests/42/comments/9":
			gotPutPath = r.URL.Path
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             9,
				"version":        4,
				"text":           "root",
				"threadResolved": true,
			})
		default:
			http.NotFound(w, r)
		}
	}))

	comment, err := client.SetPullRequestCommentThreadResolved(context.Background(), "PROJ", "repo", 42, 9, true)
	if err != nil {
		t.Fatalf("SetPullRequestCommentThreadResolved: %v", err)
	}
	if gotPutPath == "" {
		t.Fatal("expected PUT request")
	}
	if gotBody["version"] != float64(3) {
		t.Errorf("version = %v, want 3", gotBody["version"])
	}
	if gotBody["id"] != float64(9) {
		t.Errorf("id = %v, want 9", gotBody["id"])
	}
	if gotBody["text"] != "root" {
		t.Errorf("text = %v, want root", gotBody["text"])
	}
	if gotBody["severity"] != "NORMAL" {
		t.Errorf("severity = %v, want NORMAL", gotBody["severity"])
	}
	if gotBody["state"] != "OPEN" {
		t.Errorf("state = %v, want OPEN", gotBody["state"])
	}
	props, ok := gotBody["properties"].(map[string]any)
	if !ok || props["keep"] != "me" {
		t.Errorf("properties = %#v, want keep=me", gotBody["properties"])
	}
	anchor, ok := gotBody["anchor"].(map[string]any)
	path, _ := anchor["path"].(map[string]any)
	if !ok || path["parent"] != "src" || anchor["line"] != float64(10) {
		t.Errorf("anchor = %#v, want raw src/main.go anchor", gotBody["anchor"])
	}
	replies, ok := gotBody["comments"].([]any)
	if !ok || len(replies) != 1 {
		t.Errorf("comments = %#v, want preserved replies", gotBody["comments"])
	}
	if _, ok := gotBody["author"]; ok {
		t.Errorf("author should not be sent in update body: %#v", gotBody["author"])
	}
	if _, ok := gotBody["permittedOperations"]; ok {
		t.Errorf("permittedOperations should not be sent in update body: %#v", gotBody["permittedOperations"])
	}
	if gotBody["threadResolved"] != true {
		t.Errorf("threadResolved = %v, want true", gotBody["threadResolved"])
	}
	if !comment.ThreadResolved {
		t.Errorf("updated comment ThreadResolved = false, want true")
	}
}

func TestDeletePullRequestComment(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pull-requests/42/activities"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"isLastPage": true,
				"values": []map[string]any{
					{
						"action": "COMMENTED",
						"comment": map[string]any{
							"id":      9,
							"version": 3,
							"text":    "root",
						},
					},
				},
			})
		case r.Method == http.MethodDelete:
			gotMethod = r.Method
			gotPath = r.URL.Path
			gotQuery = r.URL.RawQuery
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))

	if err := client.DeletePullRequestComment(context.Background(), "PROJ", "repo", 42, 9); err != nil {
		t.Fatalf("DeletePullRequestComment: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", gotMethod)
	}
	if gotPath != "/rest/api/1.0/projects/PROJ/repos/repo/pull-requests/42/comments/9" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "version=3" {
		t.Errorf("query = %q, want version=3", gotQuery)
	}
}

func TestSetPullRequestCommentThreadResolvedAlreadyResolvedSkipsPUT(t *testing.T) {
	var putCalled bool
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			putCalled = true
			t.Fatalf("unexpected PUT to %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isLastPage": true,
			"values": []map[string]any{
				{
					"action": "COMMENTED",
					"comment": map[string]any{
						"id":             9,
						"version":        3,
						"text":           "root",
						"threadResolved": true,
					},
				},
			},
		})
	}))

	comment, err := client.SetPullRequestCommentThreadResolved(context.Background(), "PROJ", "repo", 42, 9, true)
	if err != nil {
		t.Fatalf("SetPullRequestCommentThreadResolved: %v", err)
	}
	if putCalled {
		t.Fatal("PUT should not be called")
	}
	if !comment.ThreadResolved {
		t.Fatal("comment should remain resolved")
	}
}

func TestSetPullRequestCommentThreadResolvedRejectsReply(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isLastPage": true,
			"values": []map[string]any{
				{
					"action": "COMMENTED",
					"comment": map[string]any{
						"id":      1,
						"version": 1,
						"text":    "root",
						"comments": []map[string]any{
							{"id": 2, "version": 1, "text": "reply"},
						},
					},
				},
			},
		})
	}))

	_, err := client.SetPullRequestCommentThreadResolved(context.Background(), "PROJ", "repo", 42, 2, true)
	if !errors.Is(err, bbdc.ErrPullRequestCommentNotTopLevel) {
		t.Fatalf("error = %v, want ErrPullRequestCommentNotTopLevel", err)
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

	err := client.CommentPullRequest(context.Background(), "PROJ", "my-repo", 7, bbdc.CommentOptions{Text: "LGTM"})
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

	err := client.CommentPullRequest(context.Background(), "PROJ", "my-repo", 7, bbdc.CommentOptions{Text: "reply", ParentID: 42})
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

	err := client.CommentPullRequest(context.Background(), "PROJ", "my-repo", 7, bbdc.CommentOptions{Text: "top-level"})
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
			if err := client.CommentPullRequest(context.Background(), "PROJ", "repo", 1, bbdc.CommentOptions{Text: tt.text}); err == nil {
				t.Error("expected error")
			}
		})
	}
}

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

func TestCommentPullRequestPending(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))

	err := client.CommentPullRequest(context.Background(), "PROJ", "my-repo", 7, bbdc.CommentOptions{
		Text:    "draft feedback",
		Pending: true,
	})
	if err != nil {
		t.Fatalf("CommentPullRequest pending: %v", err)
	}

	state, ok := gotBody["state"].(string)
	if !ok || state != "PENDING" {
		t.Errorf("state = %v, want PENDING", gotBody["state"])
	}
}

func TestCommentPullRequestNotPending(t *testing.T) {
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))

	err := client.CommentPullRequest(context.Background(), "PROJ", "my-repo", 7, bbdc.CommentOptions{
		Text: "regular comment",
	})
	if err != nil {
		t.Fatalf("CommentPullRequest: %v", err)
	}

	if _, ok := gotBody["state"]; ok {
		t.Error("expected no state field when Pending is false")
	}
}

func TestCreatePullRequestDraftFlag(t *testing.T) {
	tests := []struct {
		name      string
		draft     bool
		wantDraft bool
	}{
		{name: "draft true", draft: true, wantDraft: true},
		{name: "draft false", draft: false, wantDraft: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody map[string]any
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewDecoder(r.Body).Decode(&gotBody)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 1})
			}))

			_, _ = client.CreatePullRequest(context.Background(), "PRJ", "repo", bbdc.CreatePROptions{
				Title:        "Test PR",
				SourceBranch: "feature",
				TargetBranch: "main",
				Draft:        tt.draft,
			})

			got, ok := gotBody["draft"].(bool)
			if !ok {
				t.Fatal("draft field missing from request body")
			}
			if got != tt.wantDraft {
				t.Errorf("draft = %v, want %v", got, tt.wantDraft)
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

func TestListRepoPullRequestsPageEncodesParticipantFilters(t *testing.T) {
	var requests int32
	var gotQuery string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requests, 1)
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}, "isLastPage": true})
	}))

	page, err := client.ListRepoPullRequestsPage(context.Background(), "PROJ", "repo", bbdc.RepoPullRequestsOptions{
		State:    "open",
		Role:     "reviewer",
		Username: "alice",
		Limit:    50,
		Start:    30,
	})
	if err != nil {
		t.Fatalf("ListRepoPullRequestsPage: %v", err)
	}
	if !page.IsLast {
		t.Fatal("empty last page must report IsLast")
	}
	if got := atomic.LoadInt32(&requests); got != 1 {
		t.Fatalf("page fetch made %d requests, want exactly 1", got)
	}
	for _, want := range []string{"role.1=REVIEWER", "username.1=alice", "state=OPEN", "limit=50", "start=30"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing upstream filter %q", gotQuery, want)
		}
	}
}

func TestListRepoPullRequestsPageRoleValidation(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request expected on validation failure")
	}))

	if _, err := client.ListRepoPullRequestsPage(context.Background(), "PROJ", "repo", bbdc.RepoPullRequestsOptions{Role: "REVIEWER"}); err == nil || !strings.Contains(err.Error(), "requires a username") {
		t.Fatalf("role without username: err = %v, want username requirement", err)
	}
	if _, err := client.ListRepoPullRequestsPage(context.Background(), "PROJ", "repo", bbdc.RepoPullRequestsOptions{Role: "OWNER", Username: "a"}); err == nil || !strings.Contains(err.Error(), "unsupported participant role") {
		t.Fatalf("bad role: err = %v, want unsupported role", err)
	}
}

func TestPullRequestPagesPreserveEmptyNonFinalContinuation(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values":        []any{},
			"isLastPage":    false,
			"nextPageStart": 25,
		})
	}))

	repoPage, err := client.ListRepoPullRequestsPage(context.Background(), "PROJ", "repo", bbdc.RepoPullRequestsOptions{Limit: 25})
	if err != nil {
		t.Fatalf("ListRepoPullRequestsPage: %v", err)
	}
	if repoPage.IsLast || repoPage.NextStart != 25 {
		t.Fatalf("repo page = %+v, want empty non-final continuation", repoPage)
	}

	dashboardPage, err := client.ListDashboardPullRequestsPage(context.Background(), bbdc.DashboardPullRequestsOptions{Role: "AUTHOR", Limit: 25}, 0)
	if err != nil {
		t.Fatalf("ListDashboardPullRequestsPage: %v", err)
	}
	if dashboardPage.IsLast || dashboardPage.NextStart != 25 {
		t.Fatalf("dashboard page = %+v, want empty non-final continuation", dashboardPage)
	}
}

func TestListDashboardPullRequestsPageEncodesRole(t *testing.T) {
	var gotQuery string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}, "isLastPage": true})
	}))

	if _, err := client.ListDashboardPullRequestsPage(context.Background(), bbdc.DashboardPullRequestsOptions{
		State: "open",
		Role:  "reviewer",
		Limit: 25,
	}, 75); err != nil {
		t.Fatalf("ListDashboardPullRequestsPage: %v", err)
	}
	for _, want := range []string{"role=REVIEWER", "state=OPEN", "limit=25", "start=75"} {
		if !strings.Contains(gotQuery, want) {
			t.Errorf("query %q missing %q", gotQuery, want)
		}
	}
}

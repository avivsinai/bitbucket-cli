package bbdc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
)

func TestGetDefaultReviewersDataCenter75(t *testing.T) {
	var requests []string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		requests = append(requests, r.Method+" "+r.URL.Path)

		switch r.URL.Path {
		case "/rest/api/1.0/projects/PROJ/repos/my-repo":
			if r.Method != http.MethodGet {
				t.Fatalf("repository method = %s, want GET", r.Method)
			}
			_ = json.NewEncoder(w).Encode(bbdc.Repository{Slug: "my-repo", ID: 42})
		case "/rest/default-reviewers/1.0/projects/PROJ/repos/my-repo/reviewers":
			if r.Method != http.MethodGet {
				t.Fatalf("reviewers method = %s, want GET", r.Method)
			}
			query := r.URL.Query()
			wantQuery := map[string]string{
				"sourceRepoId": "42",
				"targetRepoId": "42",
				"sourceRefId":  "feature/x",
				"targetRefId":  "main",
			}
			for key, want := range wantQuery {
				if got := query.Get(key); got != want {
					t.Fatalf("%s = %q, want %q", key, got, want)
				}
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"name": "alice", "slug": "alice", "id": 10, "displayName": "Alice"},
				{"name": "bob", "slug": "bob", "id": 20, "displayName": "Bob"},
			})
		default:
			http.NotFound(w, r)
		}
	}))

	users, err := client.GetDefaultReviewers(context.Background(), "PROJ", "my-repo", "feature/x", "main")
	if err != nil {
		t.Fatalf("GetDefaultReviewers: %v", err)
	}
	wantRequests := []string{
		"GET /rest/api/1.0/projects/PROJ/repos/my-repo",
		"GET /rest/default-reviewers/1.0/projects/PROJ/repos/my-repo/reviewers",
	}
	if !reflect.DeepEqual(requests, wantRequests) {
		t.Fatalf("requests = %v, want %v", requests, wantRequests)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if got := []string{users[0].Name, users[1].Name}; !reflect.DeepEqual(got, []string{"alice", "bob"}) {
		t.Fatalf("users = %v, want [alice bob]", got)
	}
}

func TestGetDefaultReviewersDataCenter8BranchAndTagRefs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/api/1.0/projects/PROJ/repos/my-repo":
			_ = json.NewEncoder(w).Encode(bbdc.Repository{Slug: "my-repo", ID: 99})
		case "/rest/default-reviewers/1.0/projects/PROJ/repos/my-repo/reviewers":
			query := r.URL.Query()
			if got := query.Get("sourceRefId"); got != "feature/auth" {
				t.Fatalf("sourceRefId = %q, want feature/auth", got)
			}
			if got := query.Get("targetRefId"); got != "v1.2.3" {
				t.Fatalf("targetRefId = %q, want v1.2.3", got)
			}
			if got := query.Get("sourceRepoId"); got != "99" {
				t.Fatalf("sourceRepoId = %q, want 99", got)
			}
			if got := query.Get("targetRepoId"); got != "99" {
				t.Fatalf("targetRepoId = %q, want 99", got)
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"name":         "charlie",
					"slug":         "charlie",
					"id":           30,
					"displayName":  "Charlie",
					"emailAddress": "charlie@example.com",
					"active":       true,
					"type":         "NORMAL",
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))

	users, err := client.GetDefaultReviewers(context.Background(), "PROJ", "my-repo", "refs/heads/feature/auth", "refs/tags/v1.2.3")
	if err != nil {
		t.Fatalf("GetDefaultReviewers: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].Name != "charlie" || users[0].Email != "charlie@example.com" || !users[0].Active {
		t.Fatalf("unexpected user: %#v", users[0])
	}
}

func TestListProjectReviewerGroups(t *testing.T) {
	var requests []string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		requests = append(requests, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)

		if r.URL.Path != "/rest/api/1.0/projects/PROJ/settings/reviewer-groups" {
			http.NotFound(w, r)
			return
		}

		switch r.URL.Query().Get("start") {
		case "0":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"isLastPage":    false,
				"nextPageStart": 1,
				"values": []map[string]any{
					{
						"id":          7,
						"name":        "backend-team",
						"description": "Backend reviewers",
						"scope":       map[string]any{"type": "PROJECT", "resourceId": 3},
						"users": []map[string]any{
							{"name": "alice", "slug": "alice", "id": 10, "displayName": "Alice"},
						},
					},
				},
			})
		case "1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"isLastPage": true,
				"values": []map[string]any{
					{"id": 8, "name": "frontend-team", "users": []map[string]any{}},
				},
			})
		default:
			t.Errorf("unexpected start %q", r.URL.Query().Get("start"))
			http.Error(w, "unexpected start", http.StatusBadRequest)
		}
	}))

	groups, err := client.ListProjectReviewerGroups(context.Background(), "PROJ", 0)
	if err != nil {
		t.Fatalf("ListProjectReviewerGroups: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 requests, got %v", requests)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}
	if groups[0].Name != "backend-team" || groups[0].ID != 7 {
		t.Fatalf("unexpected first group: %#v", groups[0])
	}
	if groups[0].Scope.Type != "PROJECT" || groups[0].Scope.ResourceID != 3 {
		t.Fatalf("unexpected scope: %#v", groups[0].Scope)
	}
	if len(groups[0].Users) != 1 || groups[0].Users[0].Name != "alice" {
		t.Fatalf("unexpected users: %#v", groups[0].Users)
	}
	if groups[1].Name != "frontend-team" {
		t.Fatalf("unexpected second group: %#v", groups[1])
	}
}

func TestListProjectReviewerGroupsLimit(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if got := r.URL.Query().Get("limit"); got != "1" {
			t.Errorf("limit = %q, want 1", got)
		}
		// Over-return to prove the client truncates to the requested limit.
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isLastPage": true,
			"values": []map[string]any{
				{"id": 1, "name": "one"},
				{"id": 2, "name": "two"},
			},
		})
	}))

	groups, err := client.ListProjectReviewerGroups(context.Background(), "PROJ", 1)
	if err != nil {
		t.Fatalf("ListProjectReviewerGroups: %v", err)
	}
	if len(groups) != 1 || groups[0].Name != "one" {
		t.Fatalf("unexpected groups: %#v", groups)
	}
}

func TestListProjectReviewerGroupsStalledPagination(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isLastPage":    false,
			"nextPageStart": 0,
			"values": []map[string]any{
				{"id": 1, "name": "one"},
			},
		})
	}))

	_, err := client.ListProjectReviewerGroups(context.Background(), "PROJ", 0)
	if err == nil {
		t.Fatal("expected error for non-advancing pagination")
	}
	if !strings.Contains(err.Error(), "did not advance") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListProjectReviewerGroupsValidation(t *testing.T) {
	client, err := bbdc.New(bbdc.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ListProjectReviewerGroups(context.Background(), "", 0); err == nil {
		t.Error("expected error for empty project key")
	}
}

func TestGetDefaultReviewersValidation(t *testing.T) {
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
		source  string
		target  string
	}{
		{name: "empty project", project: "", repo: "repo", source: "feature/x", target: "main"},
		{name: "empty repo", project: "PROJ", repo: "", source: "feature/x", target: "main"},
		{name: "empty source", project: "PROJ", repo: "repo", source: "", target: "main"},
		{name: "empty target", project: "PROJ", repo: "repo", source: "feature/x", target: ""},
		{name: "blank source", project: "PROJ", repo: "repo", source: " ", target: "main"},
		{name: "blank target", project: "PROJ", repo: "repo", source: "feature/x", target: " "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetDefaultReviewers(context.Background(), tt.project, tt.repo, tt.source, tt.target)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

package bbdc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
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

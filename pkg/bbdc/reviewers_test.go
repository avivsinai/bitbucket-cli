package bbdc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
)

func TestGetDefaultReviewers(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id": 1,
				"sourceRefMatcher": map[string]any{
					"id":        "refs/heads/feature/x",
					"displayId": "feature/x",
				},
				"targetRefMatcher": map[string]any{
					"id":        "refs/heads/main",
					"displayId": "main",
				},
				"requiredApprovals": 1,
				"reviewers": []map[string]any{
					{
						"id":   101,
						"name": "backend-team",
						"users": []map[string]any{
							{"name": "alice", "slug": "alice", "id": 10, "displayName": "Alice"},
							{"name": "bob", "slug": "bob", "id": 20, "displayName": "Bob"},
						},
					},
				},
			},
		})
	}))

	users, err := client.GetDefaultReviewers(context.Background(), "PROJ", "my-repo", "feature/x", "main")
	if err != nil {
		t.Fatalf("GetDefaultReviewers: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/rest/default-reviewers/1.0/projects/PROJ/repos/my-repo/reviewers" {
		t.Errorf("path = %q, want /rest/default-reviewers/1.0/projects/PROJ/repos/my-repo/reviewers", gotPath)
	}
	if gotQuery != "sourceRefId=refs%2Fheads%2Ffeature%2Fx&targetRefId=refs%2Fheads%2Fmain" {
		t.Errorf("query = %q, want sourceRefId=refs%%2Fheads%%2Ffeature%%2Fx&targetRefId=refs%%2Fheads%%2Fmain", gotQuery)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].Name != "alice" {
		t.Errorf("users[0].Name = %q, want alice", users[0].Name)
	}
	if users[1].Name != "bob" {
		t.Errorf("users[1].Name = %q, want bob", users[1].Name)
	}
}

func TestGetDefaultReviewersDedup(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id": 1,
				"reviewers": []map[string]any{
					{
						"id":   101,
						"name": "backend-team",
						"users": []map[string]any{
							{"name": "alice", "slug": "alice", "id": 10},
							{"name": "bob", "slug": "bob", "id": 20},
						},
					},
				},
			},
			{
				"id": 2,
				"reviewers": []map[string]any{
					{
						"id":   102,
						"name": "frontend-team",
						"users": []map[string]any{
							{"name": "alice", "slug": "alice", "id": 10},
							{"name": "charlie", "slug": "charlie", "id": 30},
						},
					},
				},
			},
		})
	}))

	users, err := client.GetDefaultReviewers(context.Background(), "PROJ", "my-repo", "feature/x", "main")
	if err != nil {
		t.Fatalf("GetDefaultReviewers: %v", err)
	}
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	}
	if users[0].Name != "alice" || users[1].Name != "bob" || users[2].Name != "charlie" {
		t.Fatalf("unexpected user order/content: %#v", users)
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
	}{
		{name: "empty project", project: "", repo: "repo"},
		{name: "empty repo", project: "PROJ", repo: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetDefaultReviewers(context.Background(), tt.project, tt.repo, "feature/x", "main")
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

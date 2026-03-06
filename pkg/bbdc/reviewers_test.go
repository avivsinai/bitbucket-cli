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
			{"name": "alice", "id": 10},
			{"name": "bob", "id": 20},
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
	if gotQuery != "sourceRefId=feature%2Fx&targetRefId=main" {
		t.Errorf("query = %q, want sourceRefId=feature%%2Fx&targetRefId=main", gotQuery)
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
		{"empty project", "", "repo"},
		{"empty repo", "PROJ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.GetDefaultReviewers(context.Background(), tt.project, tt.repo, "src", "main")
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

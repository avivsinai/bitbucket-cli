package bbdc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
)

func TestGetDefaultReviewers(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{
			{
				"id": 1,
				"reviewers": []map[string]any{
					{"name": "alice", "id": 10},
					{"name": "bob", "id": 20},
				},
			},
			{
				"id": 2,
				"reviewers": []map[string]any{
					{"name": "bob", "id": 20},
					{"name": "charlie", "id": 30},
				},
			},
		})
	}))

	users, err := client.GetDefaultReviewers(context.Background(), "PROJ", "my-repo")
	if err != nil {
		t.Fatalf("GetDefaultReviewers: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %s, want GET", gotMethod)
	}
	if gotPath != "/rest/default-reviewers/1.0/projects/PROJ/repos/my-repo/conditions" {
		t.Errorf("path = %q, want /rest/default-reviewers/1.0/projects/PROJ/repos/my-repo/conditions", gotPath)
	}
	// bob appears in both conditions but should be deduplicated
	if len(users) != 3 {
		t.Fatalf("expected 3 unique users, got %d", len(users))
	}
	names := make(map[string]bool)
	for _, u := range users {
		names[u.Name] = true
	}
	for _, want := range []string{"alice", "bob", "charlie"} {
		if !names[want] {
			t.Errorf("expected user %q in results", want)
		}
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
			_, err := client.GetDefaultReviewers(context.Background(), tt.project, tt.repo)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

package bbdc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
)

func TestCreateRepositorySetsDefaultBranch(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/rest/api/1.0/projects/~USER/repos":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			if got := body["name"]; got != "example" {
				t.Errorf("create name = %v, want example", got)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"slug": "example",
				"name": "example",
				"project": map[string]any{
					"key": "~USER",
				},
			})

		case r.Method == http.MethodPut && r.URL.Path == "/rest/api/1.0/projects/~USER/repos/example/default-branch":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode default branch body: %v", err)
			}
			if got := body["id"]; got != "refs/heads/main" {
				t.Errorf("default branch id = %v, want refs/heads/main", got)
			}
			w.WriteHeader(http.StatusNoContent)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	client, err := bbdc.New(bbdc.Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	repo, err := client.CreateRepository(context.Background(), "~USER", bbdc.CreateRepositoryInput{
		Name:          "example",
		Description:   "Example repository",
		DefaultBranch: "main",
	})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	if repo.DefaultBranch != "main" {
		t.Errorf("default branch = %q, want main", repo.DefaultBranch)
	}
}

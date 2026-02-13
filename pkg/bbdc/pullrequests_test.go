package bbdc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
)

func TestDeclinePullRequest(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, err := bbdc.New(bbdc.Options{
		BaseURL:  srv.URL,
		Username: "user",
		Token:    "token",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	if err := client.DeclinePullRequest(context.Background(), "PROJ", "my-repo", 42, 3); err != nil {
		t.Fatalf("DeclinePullRequest: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests/42/decline" {
		t.Errorf("unexpected path: %s", gotPath)
	}
	if v, ok := gotBody["version"].(float64); !ok || int(v) != 3 {
		t.Errorf("expected version=3 in body, got %v", gotBody["version"])
	}
}

func TestDeclinePullRequestValidation(t *testing.T) {
	client, err := bbdc.New(bbdc.Options{
		BaseURL:  "http://localhost",
		Username: "user",
		Token:    "token",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	if err := client.DeclinePullRequest(context.Background(), "", "repo", 1, 0); err == nil {
		t.Error("expected error for empty project key")
	}
	if err := client.DeclinePullRequest(context.Background(), "PROJ", "", 1, 0); err == nil {
		t.Error("expected error for empty repo slug")
	}
}

func TestReopenPullRequest(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, err := bbdc.New(bbdc.Options{
		BaseURL:  srv.URL,
		Username: "user",
		Token:    "token",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	if err := client.ReopenPullRequest(context.Background(), "PROJ", "my-repo", 42, 5); err != nil {
		t.Fatalf("ReopenPullRequest: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests/42/reopen" {
		t.Errorf("unexpected path: %s", gotPath)
	}
	if v, ok := gotBody["version"].(float64); !ok || int(v) != 5 {
		t.Errorf("expected version=5 in body, got %v", gotBody["version"])
	}
}

func TestReopenPullRequestValidation(t *testing.T) {
	client, err := bbdc.New(bbdc.Options{
		BaseURL:  "http://localhost",
		Username: "user",
		Token:    "token",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	if err := client.ReopenPullRequest(context.Background(), "", "repo", 1, 0); err == nil {
		t.Error("expected error for empty project key")
	}
	if err := client.ReopenPullRequest(context.Background(), "PROJ", "", 1, 0); err == nil {
		t.Error("expected error for empty repo slug")
	}
}

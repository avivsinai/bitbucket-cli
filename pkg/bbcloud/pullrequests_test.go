package bbcloud_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
)

func TestDeclinePullRequest(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, err := bbcloud.New(bbcloud.Options{
		BaseURL:  srv.URL,
		Username: "user",
		Token:    "token",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	if err := client.DeclinePullRequest(context.Background(), "myworkspace", "my-repo", 7); err != nil {
		t.Fatalf("DeclinePullRequest: %v", err)
	}

	if gotMethod != "POST" {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/repositories/myworkspace/my-repo/pullrequests/7/decline" {
		t.Errorf("unexpected path: %s", gotPath)
	}
}

func TestDeclinePullRequestValidation(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{
		BaseURL:  "http://localhost",
		Username: "user",
		Token:    "token",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	if err := client.DeclinePullRequest(context.Background(), "", "repo", 1); err == nil {
		t.Error("expected error for empty workspace")
	}
	if err := client.DeclinePullRequest(context.Background(), "ws", "", 1); err == nil {
		t.Error("expected error for empty repo slug")
	}
}

func TestReopenPullRequest(t *testing.T) {
	var gotMethod, gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, err := bbcloud.New(bbcloud.Options{
		BaseURL:  srv.URL,
		Username: "user",
		Token:    "token",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	if err := client.ReopenPullRequest(context.Background(), "myworkspace", "my-repo", 7); err != nil {
		t.Fatalf("ReopenPullRequest: %v", err)
	}

	if gotMethod != "PUT" {
		t.Errorf("expected PUT, got %s", gotMethod)
	}
	if gotPath != "/repositories/myworkspace/my-repo/pullrequests/7" {
		t.Errorf("unexpected path: %s", gotPath)
	}
}

func TestReopenPullRequestValidation(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{
		BaseURL:  "http://localhost",
		Username: "user",
		Token:    "token",
	})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}

	if err := client.ReopenPullRequest(context.Background(), "", "repo", 1); err == nil {
		t.Error("expected error for empty workspace")
	}
	if err := client.ReopenPullRequest(context.Background(), "ws", "", 1); err == nil {
		t.Error("expected error for empty repo slug")
	}
}

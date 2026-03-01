package bbcloud_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
)

func TestCommitDiffPath(t *testing.T) {
	tests := []struct {
		name            string
		spec            string
		wantEscapedPath string
	}{
		{
			name:            "commit SHAs",
			spec:            "abc123..def456",
			wantEscapedPath: "/repositories/myworkspace/my-repo/diff/abc123..def456",
		},
		{
			name:            "branch with slash",
			spec:            "main..feature/my-branch",
			wantEscapedPath: "/repositories/myworkspace/my-repo/diff/main..feature%2Fmy-branch",
		},
		{
			name:            "tag refs",
			spec:            "v1.0.0..v2.0.0",
			wantEscapedPath: "/repositories/myworkspace/my-repo/diff/v1.0.0..v2.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotEscapedPath, gotAccept string
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotEscapedPath = r.URL.EscapedPath()
				gotAccept = r.Header.Get("Accept")
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte("diff content"))
			}))

			var buf strings.Builder
			err := client.CommitDiff(context.Background(), "myworkspace", "my-repo", tt.spec, &buf)
			if err != nil {
				t.Fatalf("CommitDiff: %v", err)
			}
			if gotMethod != "GET" {
				t.Errorf("method = %s, want GET", gotMethod)
			}
			if gotEscapedPath != tt.wantEscapedPath {
				t.Errorf("escaped path = %q, want %q", gotEscapedPath, tt.wantEscapedPath)
			}
			if gotAccept != "text/plain" {
				t.Errorf("Accept = %q, want text/plain", gotAccept)
			}
		})
	}
}

func TestCommitDiffHandlesHTTPError(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Repository not found"))
	}))
	var buf strings.Builder
	err := client.CommitDiff(context.Background(), "ws", "nonexistent", "a..b", &buf)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestCommitDiffValidation(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{
		BaseURL: "http://localhost", Username: "u", Token: "t",
	})
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	tests := []struct {
		name      string
		workspace string
		repo      string
		spec      string
		writer    io.Writer
	}{
		{"empty workspace", "", "repo", "a..b", &buf},
		{"empty repo", "ws", "", "a..b", &buf},
		{"empty spec", "ws", "repo", "", &buf},
		{"nil writer", "ws", "repo", "a..b", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.CommitDiff(context.Background(), tt.workspace, tt.repo, tt.spec, tt.writer)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

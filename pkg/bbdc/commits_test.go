package bbdc

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCommitDiffStreamsContent(t *testing.T) {
	expectedDiff := `diff --git a/file.txt b/file.txt
index abc123..def456 100644
--- a/file.txt
+++ b/file.txt
@@ -1,3 +1,4 @@
 line 1
 line 2
+added line
 line 3`

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/projects/PROJ/repos/my-repo/compare/diff") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("from") != "main" {
			t.Errorf("expected from=main, got %q", r.URL.Query().Get("from"))
		}
		if r.URL.Query().Get("to") != "feature-branch" {
			t.Errorf("expected to=feature-branch, got %q", r.URL.Query().Get("to"))
		}
		if r.Header.Get("Accept") != "text/plain" {
			t.Errorf("expected Accept: text/plain, got %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte(expectedDiff))
	})

	client := newTestClient(t, handler)
	var buf bytes.Buffer
	err := client.CommitDiff(context.Background(), "PROJ", "my-repo", "main", "feature-branch", &buf)
	if err != nil {
		t.Fatalf("CommitDiff: %v", err)
	}
	if buf.String() != expectedDiff {
		t.Errorf("diff mismatch:\ngot:\n%s\nwant:\n%s", buf.String(), expectedDiff)
	}
}

func TestCommitDiffValidation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	client := newTestClient(t, handler)

	tests := []struct {
		name    string
		project string
		repo    string
		from    string
		to      string
		writer  io.Writer
	}{
		{"empty project", "", "repo", "main", "develop", &bytes.Buffer{}},
		{"empty repo", "PROJ", "", "main", "develop", &bytes.Buffer{}},
		{"empty from", "PROJ", "repo", "", "develop", &bytes.Buffer{}},
		{"empty to", "PROJ", "repo", "main", "", &bytes.Buffer{}},
		{"nil writer", "PROJ", "repo", "main", "develop", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.CommitDiff(context.Background(), tt.project, tt.repo, tt.from, tt.to, tt.writer)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestCommitDiffHandlesHTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Repository not found"))
	})

	client := newTestClient(t, handler)
	var buf bytes.Buffer
	err := client.CommitDiff(context.Background(), "PROJ", "nonexistent", "main", "develop", &buf)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestCommitDiffContextCancellation(t *testing.T) {
	serverStarted := make(chan struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(serverStarted)
		// Block until context is cancelled
		<-r.Context().Done()
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	client, err := New(Options{
		BaseURL: server.URL,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		var buf bytes.Buffer
		errChan <- client.CommitDiff(ctx, "PROJ", "repo", "main", "develop", &buf)
	}()

	<-serverStarted
	cancel()

	err = <-errChan
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("expected context canceled error, got: %v", err)
	}
}

func TestCommitDiffSetsAcceptHeader(t *testing.T) {
	var gotAccept string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAccept = r.Header.Get("Accept")
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("diff"))
	})

	client := newTestClient(t, handler)
	var buf bytes.Buffer
	err := client.CommitDiff(context.Background(), "PROJ", "repo", "main", "develop", &buf)
	if err != nil {
		t.Fatalf("CommitDiff: %v", err)
	}
	if gotAccept != "text/plain" {
		t.Errorf("Accept header = %q, want text/plain", gotAccept)
	}
}

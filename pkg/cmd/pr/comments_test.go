package pr_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCommentsCommandValidation(t *testing.T) {
	cfg := dcConfig("http://localhost")

	t.Run("missing PR ID arg", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "comments")
		if err == nil {
			t.Fatal("expected error when no PR ID provided")
		}
	})

	t.Run("invalid PR ID", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "comments", "abc")
		if err == nil {
			t.Fatal("expected error for invalid PR ID")
		}
		if !strings.Contains(err.Error(), "invalid pull request id") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid state flag", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "comments", "42", "--state", "bogus")
		if err == nil {
			t.Fatal("expected error for invalid --state value")
		}
		if !strings.Contains(err.Error(), "invalid --state value") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("state flag rejected on DC", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "comments", "42", "--state", "resolved")
		if err == nil {
			t.Fatal("expected error for --state on DC")
		}
		if !strings.Contains(err.Error(), "only supported on Cloud") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestCommentsDC(t *testing.T) {
	t.Run("lists comments", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "/pull-requests/42/activities") {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"size":       2,
				"limit":      100,
				"isLastPage": true,
				"start":      0,
				"values": []map[string]any{
					{
						"action": "COMMENTED",
						"comment": map[string]any{
							"id":   1,
							"text": "Looks good to me",
							"author": map[string]any{
								"name":        "jdoe",
								"displayName": "Jane Doe",
							},
						},
					},
					{
						"action": "APPROVED",
					},
					{
						"action": "COMMENTED",
						"comment": map[string]any{
							"id":   2,
							"text": "Please fix the typo on line 10",
							"author": map[string]any{
								"name":        "bob",
								"displayName": "Bob Smith",
							},
						},
					},
				},
			})
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "comments", "42")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "jdoe") {
			t.Errorf("expected author 'jdoe' in output, got: %s", stdout)
		}
		if !strings.Contains(stdout, "Looks good to me") {
			t.Errorf("expected comment text in output, got: %s", stdout)
		}
		if !strings.Contains(stdout, "bob") {
			t.Errorf("expected author 'bob' in output, got: %s", stdout)
		}
		if !strings.Contains(stdout, "Please fix the typo on line 10") {
			t.Errorf("expected comment text in output, got: %s", stdout)
		}
	})
}

func TestCommentsCloud(t *testing.T) {
	makeServer := func(comments []map[string]any) *httptest.Server {
		return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "/pullrequests/42/comments") {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": comments,
			})
		}))
	}

	allComments := []map[string]any{
		{
			"id":      1,
			"content": map[string]string{"raw": "Unresolved comment"},
			"user":    map[string]string{"display_name": "Alice", "nickname": "alice"},
		},
		{
			"id":      2,
			"content": map[string]string{"raw": "Resolved comment"},
			"user":    map[string]string{"display_name": "Bob", "nickname": "bob"},
			"resolution": map[string]any{
				"user":       map[string]string{"display_name": "Bob"},
				"created_on": "2025-01-01T00:00:00+00:00",
			},
		},
	}

	t.Run("state all", func(t *testing.T) {
		srv := makeServer(allComments)
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "pr", "comments", "42", "--state", "all")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "Unresolved comment") {
			t.Errorf("expected unresolved comment in output, got: %s", stdout)
		}
		if !strings.Contains(stdout, "Resolved comment") {
			t.Errorf("expected resolved comment in output, got: %s", stdout)
		}
	})

	t.Run("state resolved", func(t *testing.T) {
		srv := makeServer(allComments)
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "pr", "comments", "42", "--state", "resolved")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "Resolved comment") {
			t.Errorf("expected resolved comment in output, got: %s", stdout)
		}
		if strings.Contains(stdout, "Unresolved comment") {
			t.Errorf("should not contain unresolved comment, got: %s", stdout)
		}
	})

	t.Run("state unresolved", func(t *testing.T) {
		srv := makeServer(allComments)
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "pr", "comments", "42", "--state", "unresolved")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "Unresolved comment") {
			t.Errorf("expected unresolved comment in output, got: %s", stdout)
		}
		if strings.Contains(stdout, "Resolved comment") {
			t.Errorf("should not contain resolved comment, got: %s", stdout)
		}
	})

	t.Run("default state is all", func(t *testing.T) {
		srv := makeServer(allComments)
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "pr", "comments", "42")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "Unresolved comment") || !strings.Contains(stdout, "Resolved comment") {
			t.Errorf("expected both comments in output (default state=all), got: %s", stdout)
		}
	})
}

func TestCommentsEmpty(t *testing.T) {
	t.Run("DC empty", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"size":       0,
				"limit":      100,
				"isLastPage": true,
				"start":      0,
				"values":     []any{},
			})
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "comments", "1")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "No comments") {
			t.Errorf("expected 'No comments' message, got: %s", stdout)
		}
	})

	t.Run("Cloud empty", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []any{},
			})
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "pr", "comments", "1")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "No comments") {
			t.Errorf("expected 'No comments' message, got: %s", stdout)
		}
	})
}

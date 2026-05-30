package pr_test

import (
	"encoding/json"
	"fmt"
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

	t.Run("invalid resolve comment ID", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "comments", "resolve", "42", "abc")
		if err == nil {
			t.Fatal("expected error for invalid comment ID")
		}
		if !strings.Contains(err.Error(), "invalid comment id") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid delete comment ID", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", "comments", "delete", "42", "abc")
		if err == nil {
			t.Fatal("expected error for invalid comment ID")
		}
		if !strings.Contains(err.Error(), "invalid comment id") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("help includes thread action examples", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfg, "pr", "comments", "--help")
		if err != nil {
			t.Fatalf("help: %v", err)
		}
		if !strings.Contains(stdout, "bkt pr comments delete 42 1001") ||
			!strings.Contains(stdout, "bkt pr comments resolve 42 1001") ||
			!strings.Contains(stdout, "bkt pr comments reopen 42 1001") {
			t.Errorf("help missing thread action examples:\n%s", stdout)
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
		{
			"id":      3,
			"deleted": true,
			"content": map[string]string{"raw": ""},
			"user":    map[string]string{"display_name": "Deleted User", "nickname": "deleted"},
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
		if !strings.Contains(stdout, "3\tDeleted User\t[deleted]") {
			t.Errorf("expected deleted marker in output, got: %s", stdout)
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
		if strings.Contains(stdout, "[deleted]") {
			t.Errorf("should not contain deleted comment, got: %s", stdout)
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
		if strings.Contains(stdout, "[deleted]") {
			t.Errorf("should not contain deleted comment, got: %s", stdout)
		}
	})

	t.Run("state deleted", func(t *testing.T) {
		srv := makeServer(allComments)
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "pr", "comments", "42", "--state", "deleted")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "3\tDeleted User\t[deleted]") {
			t.Errorf("expected deleted comment in output, got: %s", stdout)
		}
		if strings.Contains(stdout, "Unresolved comment") || strings.Contains(stdout, "Resolved comment") {
			t.Errorf("should only contain deleted comments, got: %s", stdout)
		}
	})

	t.Run("deleted json output", func(t *testing.T) {
		srv := makeServer(allComments)
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "--json", "pr", "comments", "42", "--state", "deleted")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		var payload struct {
			Comments []struct {
				ID      int  `json:"id"`
				Deleted bool `json:"deleted"`
			} `json:"comments"`
		}
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, stdout)
		}
		if len(payload.Comments) != 1 || payload.Comments[0].ID != 3 || !payload.Comments[0].Deleted {
			t.Errorf("comments = %+v, want only deleted comment 3", payload.Comments)
		}
	})

	t.Run("default state is all", func(t *testing.T) {
		srv := makeServer(allComments)
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "pr", "comments", "42")
		if err != nil {
			t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "Unresolved comment") || !strings.Contains(stdout, "Resolved comment") || !strings.Contains(stdout, "[deleted]") {
			t.Errorf("expected all comments in output (default state=all), got: %s", stdout)
		}
	})
}

func TestCommentsThreadResolveCloud(t *testing.T) {
	t.Run("resolve human output", func(t *testing.T) {
		var gotMethod, gotPath string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch {
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/comments/9"):
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":      9,
					"content": map[string]string{"raw": "Needs a tweak"},
				})
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comments/9/resolve"):
				gotMethod = r.Method
				gotPath = r.URL.Path
				_ = json.NewEncoder(w).Encode(map[string]any{
					"user":       map[string]string{"display_name": "Alice"},
					"created_on": "2026-01-01T00:00:00+00:00",
				})
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "pr", "comments", "resolve", "42", "9")
		if err != nil {
			t.Fatalf("resolve: %v (stderr=%s)", err, stderr)
		}
		if gotMethod != http.MethodPost {
			t.Errorf("method = %s, want POST", gotMethod)
		}
		if gotPath != "/repositories/myworkspace/my-repo/pullrequests/42/comments/9/resolve" {
			t.Errorf("path = %q", gotPath)
		}
		if !strings.Contains(stdout, "Resolved comment thread 9 on pull request #42") {
			t.Errorf("unexpected output: %s", stdout)
		}
	})

	t.Run("reopen json output", func(t *testing.T) {
		var gotMethod string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch {
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/comments/9"):
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":         9,
					"content":    map[string]string{"raw": "Needs a tweak"},
					"resolution": map[string]string{"created_on": "2026-01-01T00:00:00+00:00"},
				})
			case r.Method == http.MethodDelete && strings.HasSuffix(r.URL.Path, "/comments/9/resolve"):
				gotMethod = r.Method
				w.WriteHeader(http.StatusNoContent)
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "--json", "pr", "comments", "reopen", "42", "9")
		if err != nil {
			t.Fatalf("reopen --json: %v (stderr=%s)", err, stderr)
		}
		if gotMethod != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", gotMethod)
		}
		var payload struct {
			PullRequest int  `json:"pull_request"`
			CommentID   int  `json:"comment_id"`
			Resolved    bool `json:"resolved"`
		}
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, stdout)
		}
		if payload.PullRequest != 42 || payload.CommentID != 9 || payload.Resolved {
			t.Errorf("payload = %+v, want pr=42 comment=9 resolved=false", payload)
		}
	})

	t.Run("reply maps to top-level error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/comments/10") {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":      10,
					"parent":  map[string]int{"id": 9},
					"content": map[string]string{"raw": "A reply"},
				})
				return
			}
			http.NotFound(w, r)
		}))
		t.Cleanup(srv.Close)

		_, _, err := runCLI(t, cloudConfig(srv.URL), "pr", "comments", "resolve", "42", "10")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "only top-level pull request comment threads can be resolved") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("deleted comment maps to deleted error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/comments/10") {
				_ = json.NewEncoder(w).Encode(map[string]any{
					"id":      10,
					"deleted": true,
					"content": map[string]string{"raw": ""},
				})
				return
			}
			http.NotFound(w, r)
		}))
		t.Cleanup(srv.Close)

		_, _, err := runCLI(t, cloudConfig(srv.URL), "pr", "comments", "resolve", "42", "10")
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "pull request comment 10 has been deleted and cannot be resolved") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("conflict is idempotent success", func(t *testing.T) {
		getCount := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch {
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/comments/9"):
				getCount++
				comment := map[string]any{
					"id":      9,
					"content": map[string]string{"raw": "Needs a tweak"},
				}
				if getCount > 1 {
					comment["resolution"] = map[string]string{"created_on": "2026-01-01T00:00:00+00:00"}
				}
				_ = json.NewEncoder(w).Encode(comment)
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/comments/9/resolve"):
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{"message": "Already resolved"},
				})
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "pr", "comments", "resolve", "42", "9")
		if err != nil {
			t.Fatalf("resolve conflict: %v (stderr=%s)", err, stderr)
		}
		if !strings.Contains(stdout, "already resolved") {
			t.Errorf("unexpected output: %s", stdout)
		}
	})
}

func TestCommentsDeleteCloud(t *testing.T) {
	t.Run("delete human output", func(t *testing.T) {
		var gotMethod, gotPath string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			gotPath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "pr", "comments", "delete", "42", "9")
		if err != nil {
			t.Fatalf("delete: %v (stderr=%s)", err, stderr)
		}
		if gotMethod != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", gotMethod)
		}
		if gotPath != "/repositories/myworkspace/my-repo/pullrequests/42/comments/9" {
			t.Errorf("path = %q", gotPath)
		}
		if !strings.Contains(stdout, "Deleted comment 9 on pull request #42") {
			t.Errorf("unexpected output: %s", stdout)
		}
	})

	t.Run("delete json output", func(t *testing.T) {
		var gotMethod string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			w.WriteHeader(http.StatusNoContent)
		}))
		t.Cleanup(srv.Close)

		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "--json", "pr", "comments", "delete", "42", "9")
		if err != nil {
			t.Fatalf("delete --json: %v (stderr=%s)", err, stderr)
		}
		if gotMethod != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", gotMethod)
		}
		var payload struct {
			PullRequest int  `json:"pull_request"`
			CommentID   int  `json:"comment_id"`
			Deleted     bool `json:"deleted"`
		}
		if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
			t.Fatalf("output is not JSON: %v\n%s", err, stdout)
		}
		if payload.PullRequest != 42 || payload.CommentID != 9 || !payload.Deleted {
			t.Errorf("payload = %+v, want pr=42 comment=9 deleted=true", payload)
		}
	})
}

func TestCommentsThreadResolveDC(t *testing.T) {
	var gotBody map[string]any
	var gotPutPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pull-requests/42/activities"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"isLastPage": true,
				"values": []map[string]any{
					{
						"action": "COMMENTED",
						"comment": map[string]any{
							"id":             9,
							"version":        3,
							"text":           "root",
							"threadResolved": false,
						},
					},
				},
			})
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/pull-requests/42/comments/9"):
			gotPutPath = r.URL.Path
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":             9,
				"version":        4,
				"text":           "root",
				"threadResolved": true,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "comments", "resolve", "42", "9")
	if err != nil {
		t.Fatalf("dc resolve: %v (stderr=%s)", err, stderr)
	}
	if gotPutPath != "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests/42/comments/9" {
		t.Errorf("PUT path = %q", gotPutPath)
	}
	if gotBody["version"] != float64(3) || gotBody["threadResolved"] != true {
		t.Errorf("body = %#v, want version=3 threadResolved=true", gotBody)
	}
	if !strings.Contains(stdout, "Resolved comment thread 9 on pull request #42") {
		t.Errorf("unexpected output: %s", stdout)
	}
}

func TestCommentsDeleteDC(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "comments", "delete", "42", "9")
	if err != nil {
		t.Fatalf("delete: %v (stderr=%s)", err, stderr)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %s, want DELETE", gotMethod)
	}
	if gotPath != "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests/42/comments/9" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(stdout, "Deleted comment 9 on pull request #42") {
		t.Errorf("unexpected output: %s", stdout)
	}
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

// nestComment wraps a comment inside n levels of parent comments, producing
// a tree whose leaf sits at depth n when flattened.
func nestComment(inner map[string]any, extraDepth int) map[string]any {
	c := inner
	for i := 0; i < extraDepth; i++ {
		c = map[string]any{
			"id":       1000 + i,
			"text":     "parent",
			"author":   map[string]any{"name": "sys"},
			"comments": []map[string]any{c},
		}
	}
	return c
}

func TestDCCommentsDepthCap(t *testing.T) {
	newSrv := func(t *testing.T, body map[string]any) *httptest.Server {
		t.Helper()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "/activities") {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(body)
		}))
		t.Cleanup(srv.Close)
		return srv
	}

	// Root thread: depth 0 comment with a reply at depth 20 (at cap) and nested
	// replies at depths 21 and 25 (over cap). A separate top-level comment
	// ("top level comment B") that should appear after the cap.
	atCap := map[string]any{
		"id":     2,
		"text":   "at cap",
		"author": map[string]any{"name": "bob"},
		"comments": []map[string]any{
			nestComment(map[string]any{"id": 3, "text": "too deep", "author": map[string]any{"name": "carol"},
				"comments": []map[string]any{
					nestComment(map[string]any{"id": 4, "text": "very deep", "author": map[string]any{"name": "dave"}}, 3),
				},
			}, 0),
		},
	}
	toplevel := map[string]any{
		"id":       1,
		"text":     "top level comment A",
		"author":   map[string]any{"name": "alice"},
		"comments": []map[string]any{nestComment(atCap, 19)},
	}
	topLevel2 := map[string]any{
		"id":     5,
		"text":   "top level comment B",
		"author": map[string]any{"name": "eve"},
	}
	activities := map[string]any{
		"isLastPage": true,
		"values": []map[string]any{
			{"action": "COMMENTED", "comment": toplevel},
			{"action": "COMMENTED", "comment": topLevel2},
		},
	}

	t.Run("non-details: deep comments hidden, marker shown", func(t *testing.T) {
		srv := newSrv(t, activities)
		stdout, _, err := runCLI(t, dcConfig(srv.URL), "pr", "comments", "1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(stdout, "top level comment A") {
			t.Errorf("expected toplevel comment in output; got: %s", stdout)
		}
		if !strings.Contains(stdout, "at cap") {
			t.Errorf("expected depth-20 comment in output; got: %s", stdout)
		}
		if strings.Contains(stdout, "too deep") {
			t.Errorf("depth-21 comment should be hidden; got: %s", stdout)
		}
		if strings.Contains(stdout, "very deep") {
			t.Errorf("depth-25 comment should be hidden; got: %s", stdout)
		}
		if !strings.Contains(stdout, "[...]") {
			t.Errorf("expected [...] marker for skipped deep comments; got: %s", stdout)
		}
		if !strings.Contains(stdout, "top level comment B") {
			t.Errorf("expected top level comment B after marker; got: %s", stdout)
		}
	})

	t.Run("details: deep comments hidden, marker shown", func(t *testing.T) {
		srv := newSrv(t, activities)
		stdout, _, err := runCLI(t, dcConfig(srv.URL), "pr", "comments", "1", "--details")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(stdout, "top level comment A") {
			t.Errorf("expected top level comment A in output; got: %s", stdout)
		}
		if strings.Contains(stdout, "too deep") {
			t.Errorf("depth-21 comment should be hidden; got: %s", stdout)
		}
		if !strings.Contains(stdout, "[...]") {
			t.Errorf("expected [...] marker for skipped deep comments; got: %s", stdout)
		}
	})

	t.Run("marker appears only once for consecutive deep comments", func(t *testing.T) {
		srv := newSrv(t, activities)
		stdout, _, err := runCLI(t, dcConfig(srv.URL), "pr", "comments", "1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		count := strings.Count(stdout, "[...]")
		if count != 1 {
			t.Errorf("expected exactly 1 [...] marker, got %d; output: %s", count, stdout)
		}
	})

	t.Run("no marker when all comments within cap", func(t *testing.T) {
		shallow := map[string]any{
			"isLastPage": true,
			"values": []map[string]any{
				{"action": "COMMENTED", "comment": map[string]any{
					"id":     1,
					"text":   "hello",
					"author": map[string]any{"name": "alice"},
					"comments": []map[string]any{
						{"id": 2, "text": "reply", "author": map[string]any{"name": "bob"}},
					},
				}},
			},
		}
		srv := newSrv(t, shallow)
		stdout, _, err := runCLI(t, dcConfig(srv.URL), "pr", "comments", "1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(stdout, "[...]") {
			t.Errorf("unexpected [...] marker when no deep comments; got: %s", stdout)
		}
	})
}

func TestDCPRCommentsDetailsTask(t *testing.T) {
	tests := []struct {
		name           string
		state          string
		threadResolved bool
		wantComplete   string
		wantResolved   string
	}{
		{
			name:           "open unresolved task",
			state:          "OPEN",
			threadResolved: false,
			wantComplete:   "Complete: no",
			wantResolved:   "Resolved: no",
		},
		{
			name:           "resolved complete task",
			state:          "RESOLVED",
			threadResolved: true,
			wantComplete:   "Complete: yes",
			wantResolved:   "Resolved: yes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.Path, "/activities") {
					http.NotFound(w, r)
					return
				}
				resp := map[string]any{
					"isLastPage": true,
					"values": []map[string]any{
						{
							"action": "COMMENTED",
							"comment": map[string]any{
								"id":             1,
								"text":           "fix this",
								"severity":       "BLOCKER",
								"state":          tt.state,
								"threadResolved": tt.threadResolved,
								"author": map[string]any{
									"name": "alice",
								},
							},
						},
					},
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(resp)
			}))
			t.Cleanup(srv.Close)

			cfg := dcConfig(srv.URL)
			stdout, _, err := runCLI(t, cfg, "pr", "comments", "1", "--details")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(stdout, tt.wantComplete) {
				t.Errorf("stdout missing %q\ngot: %s", tt.wantComplete, stdout)
			}
			if !strings.Contains(stdout, tt.wantResolved) {
				t.Errorf("stdout missing %q\ngot: %s", tt.wantResolved, stdout)
			}
		})
	}
}

func TestCommentsDCDeepReplyTreeDefaultOutput(t *testing.T) {
	const deepestDepth = 40

	buildCommentTree := func(depth int) map[string]any {
		root := map[string]any{
			"id":   1,
			"text": "root comment",
			"author": map[string]any{
				"name": "root",
			},
		}

		current := root
		for i := 1; i <= depth; i++ {
			child := map[string]any{
				"id":   i + 1,
				"text": fmt.Sprintf("reply depth %d", i),
				"author": map[string]any{
					"name": "deep",
				},
			}
			current["comments"] = []map[string]any{child}
			current = child
		}

		return root
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/activities") {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isLastPage": true,
			"values": []map[string]any{
				{
					"action":  "COMMENTED",
					"comment": buildCommentTree(deepestDepth),
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "comments", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
	}

	// Replies beyond depth 20 are capped; the root and depth-20 reply render,
	// deeper ones are replaced by a single [...] marker.
	if !strings.Contains(stdout, "root comment") {
		t.Fatalf("expected root comment in output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "reply depth 20") {
		t.Fatalf("expected depth-20 reply in output, got: %s", stdout)
	}
	if strings.Contains(stdout, "reply depth 21") {
		t.Fatalf("depth-21 reply should be hidden by cap, got: %s", stdout)
	}
	if !strings.Contains(stdout, "[...]") {
		t.Fatalf("expected [...] marker for capped replies, got: %s", stdout)
	}
}

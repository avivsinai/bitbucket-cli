package pr_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublishCommandResolution(t *testing.T) {
	cfg := dcConfig("http://localhost")
	cmd := "publish"

	t.Run("name and alias", func(t *testing.T) {
		// "publish" is the canonical name
		_, _, err := runCLI(t, cfg, "pr", cmd, "--help")
		if err != nil {
			t.Fatalf("pr publish --help failed: %v", err)
		}

		// "ready" is an alias
		_, _, err = runCLI(t, cfg, "pr", "ready", "--help")
		if err != nil {
			t.Fatalf("pr ready --help failed: %v", err)
		}
	})

	t.Run("requires arg", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", cmd)
		if err == nil {
			t.Fatal("expected error when no PR ID provided")
		}
	})

	t.Run("invalid id", func(t *testing.T) {
		_, _, err := runCLI(t, cfg, "pr", cmd, "abc")
		if err == nil {
			t.Fatal("expected error for invalid PR ID")
		}
		if !strings.Contains(err.Error(), "invalid pull request id") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("has undo flag", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfg, "pr", cmd, "--help")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(stdout, "--undo") {
			t.Errorf("help output should mention --undo flag, got:\n%s", stdout)
		}
	})
}

func TestPublishDC(t *testing.T) {
	tests := []struct {
		name           string
		undo           bool
		prDraft        bool
		expectPUT      bool
		outputContains string
		warnContains   string
	}{
		{
			name:           "publish draft PR",
			undo:           false,
			prDraft:        true,
			expectPUT:      true,
			outputContains: "Published pull request #42",
		},
		{
			name:           "unpublish non-draft PR",
			undo:           true,
			prDraft:        false,
			expectPUT:      true,
			outputContains: "Unpublished pull request #42",
		},
		{
			name:         "already published warns",
			undo:         false,
			prDraft:      false,
			expectPUT:    false,
			warnContains: "already published",
		},
		{
			name:         "already draft warns",
			undo:         true,
			prDraft:      true,
			expectPUT:    false,
			warnContains: "already a draft",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var putCalled bool
			var putBody map[string]any

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				switch {
				case r.Method == "GET" && strings.HasSuffix(r.URL.Path, "/pull-requests/42"):
					_ = json.NewEncoder(w).Encode(map[string]any{
						"id":          42,
						"title":       "Test PR",
						"description": "desc",
						"state":       "OPEN",
						"version":     3,
						"draft":       tt.prDraft,
						"fromRef": map[string]any{
							"id":        "refs/heads/feature",
							"displayId": "feature",
							"repository": map[string]any{
								"slug":    "my-repo",
								"project": map[string]any{"key": "PROJ"},
							},
						},
						"toRef": map[string]any{
							"id":        "refs/heads/main",
							"displayId": "main",
							"repository": map[string]any{
								"slug":    "my-repo",
								"project": map[string]any{"key": "PROJ"},
							},
						},
						"reviewers": []any{},
					})
				case r.Method == "PUT" && strings.HasSuffix(r.URL.Path, "/pull-requests/42"):
					putCalled = true
					_ = json.NewDecoder(r.Body).Decode(&putBody)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"id":      42,
						"title":   "Test PR",
						"version": 4,
					})
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(srv.Close)

			args := []string{"pr", "publish", "42"}
			if tt.undo {
				args = []string{"pr", "publish", "--undo", "42"}
			}

			stdout, stderr, err := runCLI(t, dcConfig(srv.URL), args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectPUT {
				if !putCalled {
					t.Fatal("expected PUT to be called")
				}
				draft, ok := putBody["draft"].(bool)
				if !ok {
					t.Fatal("draft field missing from PUT body")
				}
				wantDraft := tt.undo
				if draft != wantDraft {
					t.Errorf("draft = %v, want %v", draft, wantDraft)
				}
				if !strings.Contains(stdout, tt.outputContains) {
					t.Errorf("stdout %q does not contain %q", stdout, tt.outputContains)
				}
			} else {
				if putCalled {
					t.Fatal("PUT should not have been called")
				}
				if !strings.Contains(stderr, tt.warnContains) {
					t.Errorf("stderr %q does not contain %q", stderr, tt.warnContains)
				}
			}
		})
	}
}

func TestPublishCloud(t *testing.T) {
	tests := []struct {
		name           string
		undo           bool
		prDraft        bool
		expectPUT      bool
		outputContains string
		warnContains   string
	}{
		{
			name:           "publish draft PR",
			undo:           false,
			prDraft:        true,
			expectPUT:      true,
			outputContains: "Published pull request #10",
		},
		{
			name:           "unpublish non-draft PR",
			undo:           true,
			prDraft:        false,
			expectPUT:      true,
			outputContains: "Unpublished pull request #10",
		},
		{
			name:         "already published warns",
			undo:         false,
			prDraft:      false,
			expectPUT:    false,
			warnContains: "already published",
		},
		{
			name:         "already draft warns",
			undo:         true,
			prDraft:      true,
			expectPUT:    false,
			warnContains: "already a draft",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var putCalled bool
			var putBody map[string]any

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")

				switch {
				case r.Method == "GET" && strings.Contains(r.URL.Path, "/pullrequests/10"):
					_ = json.NewEncoder(w).Encode(map[string]any{
						"id":    10,
						"title": "Cloud PR",
						"state": "OPEN",
						"draft": tt.prDraft,
					})
				case r.Method == "PUT" && strings.Contains(r.URL.Path, "/pullrequests/10"):
					putCalled = true
					_ = json.NewDecoder(r.Body).Decode(&putBody)
					_ = json.NewEncoder(w).Encode(map[string]any{
						"id":    10,
						"title": "Cloud PR",
					})
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(srv.Close)

			args := []string{"pr", "publish", "10"}
			if tt.undo {
				args = []string{"pr", "publish", "--undo", "10"}
			}

			stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), args...)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.expectPUT {
				if !putCalled {
					t.Fatal("expected PUT to be called")
				}
				draft, ok := putBody["draft"].(bool)
				if !ok {
					t.Fatal("draft field missing from PUT body")
				}
				wantDraft := tt.undo
				if draft != wantDraft {
					t.Errorf("draft = %v, want %v", draft, wantDraft)
				}
				// Cloud should only send draft, not title/description
				if _, ok := putBody["title"]; ok {
					t.Error("title should not be in PUT body for publish")
				}
				if !strings.Contains(stdout, tt.outputContains) {
					t.Errorf("stdout %q does not contain %q", stdout, tt.outputContains)
				}
			} else {
				if putCalled {
					t.Fatal("PUT should not have been called")
				}
				if !strings.Contains(stderr, tt.warnContains) {
					t.Errorf("stderr %q does not contain %q", stderr, tt.warnContains)
				}
			}
		})
	}
}

func TestPublishReadyAlias(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "GET":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      5,
				"title":   "PR",
				"state":   "OPEN",
				"version": 1,
				"draft":   true,
				"fromRef": map[string]any{
					"id":         "refs/heads/feature",
					"repository": map[string]any{"slug": "my-repo", "project": map[string]any{"key": "PROJ"}},
				},
				"toRef": map[string]any{
					"id":         "refs/heads/main",
					"repository": map[string]any{"slug": "my-repo", "project": map[string]any{"key": "PROJ"}},
				},
				"reviewers": []any{},
			})
		case r.Method == "PUT":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 5, "version": 2})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	stdout, _, err := runCLI(t, dcConfig(srv.URL), "pr", "ready", "5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Published pull request #5") {
		t.Errorf("unexpected output: %s", stdout)
	}
}

package pr_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCommentsDCActivityAnchor(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/pull-requests/42/activities") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isLastPage": true,
			"values": []map[string]any{{
				"action":  "COMMENTED",
				"comment": map[string]any{"id": 10, "text": "check this line", "author": map[string]string{"name": "alice"}},
				"commentAnchor": map[string]any{
					"path": "src/main.go", "line": 25, "lineType": "ADDED", "fileType": "TO", "orphaned": true,
				},
			}},
		})
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "comments", "42", "--details")
	if err != nil {
		t.Fatalf("details: %v (stderr=%s)", err, stderr)
	}
	if !strings.Contains(stdout, "File: src/main.go:25") {
		t.Fatalf("missing anchor in --details output: %s", stdout)
	}

	stdout, stderr, err = runCLI(t, dcConfig(srv.URL), "pr", "comments", "42", "--json")
	if err != nil {
		t.Fatalf("json: %v (stderr=%s)", err, stderr)
	}
	var payload struct {
		Comments []struct {
			Anchor struct {
				Path     string `json:"path"`
				Line     int    `json:"line"`
				LineType string `json:"lineType"`
				FileType string `json:"fileType"`
				Orphaned bool   `json:"orphaned"`
			} `json:"anchor"`
		} `json:"comments"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}
	if len(payload.Comments) != 1 {
		t.Fatalf("comments = %+v", payload.Comments)
	}
	a := payload.Comments[0].Anchor
	if a.Path != "src/main.go" || a.Line != 25 || a.LineType != "ADDED" || a.FileType != "TO" || !a.Orphaned {
		t.Fatalf("anchor = %+v", a)
	}
}

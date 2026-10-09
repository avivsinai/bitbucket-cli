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
	if !strings.Contains(stdout, "File: src/main.go:25 (orphaned)") {
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

func TestCommentsViewDCThread(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/pull-requests/42/activities") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isLastPage": true,
			"values": []map[string]any{
				{"action": "COMMENTED", "comment": map[string]any{"id": 5, "text": "general"}},
				{
					"action": "COMMENTED",
					"comment": map[string]any{
						"id": 10, "text": "check this line", "author": map[string]string{"name": "alice"},
						"comments": []map[string]any{{"id": 11, "text": "fixed", "author": map[string]string{"name": "bob"}}},
					},
					"commentAnchor": map[string]any{"path": "src/main.go", "line": 25, "lineType": "ADDED", "fileType": "TO"},
					"diff": map[string]any{"hunks": []map[string]any{{"segments": []map[string]any{
						{"type": "CONTEXT", "lines": []map[string]any{{"source": 24, "destination": 24, "line": "func main() {"}}},
						{"type": "ADDED", "lines": []map[string]any{{"source": 25, "destination": 25, "line": "\tpanic(err)", "commentIds": []int{10}}}},
					}}}},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "comments", "view", "42", "11")
	if err != nil {
		t.Fatalf("view: %v (stderr=%s)", err, stderr)
	}
	for _, want := range []string{
		"Thread #10 on pull request #42",
		">          25 +\tpanic(err)",
		"File: src/main.go:25",
		"check this line",
		"--- Comment #11 by bob --- (requested)",
		"fixed",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("view output missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "general") {
		t.Fatalf("view output includes another thread:\n%s", stdout)
	}

	stdout, stderr, err = runCLI(t, dcConfig(srv.URL), "pr", "comments", "view", "42", "11", "--json")
	if err != nil {
		t.Fatalf("view json: %v (stderr=%s)", err, stderr)
	}
	var payload struct {
		CommentID int `json:"comment_id"`
		Thread    struct {
			ID     int `json:"id"`
			Anchor struct {
				Path string `json:"path"`
				Line int    `json:"line"`
			} `json:"anchor"`
			Comments []struct {
				ID int `json:"id"`
			} `json:"comments"`
		} `json:"thread"`
		DiffContext []struct {
			Line     string `json:"line"`
			Anchored bool   `json:"anchored"`
		} `json:"diff_context"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("decode JSON: %v\n%s", err, stdout)
	}
	if payload.CommentID != 11 || payload.Thread.ID != 10 || payload.Thread.Anchor.Path != "src/main.go" || payload.Thread.Anchor.Line != 25 {
		t.Fatalf("payload = %+v", payload)
	}
	if len(payload.Thread.Comments) != 1 || payload.Thread.Comments[0].ID != 11 {
		t.Fatalf("replies = %+v", payload.Thread.Comments)
	}
	if len(payload.DiffContext) != 2 || !payload.DiffContext[1].Anchored {
		t.Fatalf("diff_context = %+v", payload.DiffContext)
	}
}

func TestCommentsViewCloudThread(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/pullrequests/42/comments") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{"id": 100, "content": map[string]string{"raw": "check this line"}, "user": map[string]string{"display_name": "Alice"},
					"inline": map[string]any{"path": "src/main.go", "to": 25}},
				{"id": 200, "content": map[string]string{"raw": "general"}},
				{"id": 101, "content": map[string]string{"raw": "fixed"}, "user": map[string]string{"display_name": "Bob"},
					"parent": map[string]int{"id": 100}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "pr", "comments", "view", "42", "101")
	if err != nil {
		t.Fatalf("view: %v (stderr=%s)", err, stderr)
	}
	for _, want := range []string{
		"Thread #100 on pull request #42",
		"File: src/main.go:25",
		"check this line",
		"  --- Comment #101 by Bob --- (requested)",
		"fixed",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("view output missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "general") {
		t.Fatalf("view output includes another thread:\n%s", stdout)
	}
}

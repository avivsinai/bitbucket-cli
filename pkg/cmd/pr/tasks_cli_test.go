package pr_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Cloud: list hits the tasks resource and normalizes UNRESOLVED -> OPEN.
func TestPRTaskListCloud(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{"id": 5, "state": "UNRESOLVED", "content": map[string]string{"raw": "do X"}},
				{"id": 6, "state": "RESOLVED", "content": map[string]string{"raw": "do Y"}},
			},
		})
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "pr", "task", "list", "42")
	if err != nil {
		t.Fatalf("pr task list: %v (stderr=%s)", err, stderr)
	}
	if gotPath != "/repositories/myworkspace/my-repo/pullrequests/42/tasks" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(stdout, "[OPEN] 5 do X") || !strings.Contains(stdout, "[RESOLVED] 6 do Y") {
		t.Errorf("unexpected output:\n%s", stdout)
	}
}

// DC auto: detects a 7.2+ version and creates via blocker-comments.
func TestPRTaskCreateDCAutoUsesBlockerComments(t *testing.T) {
	var versionProbed, createPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/application-properties"):
			versionProbed = r.URL.Path
			_ = json.NewEncoder(w).Encode(map[string]any{"version": "8.19.1"})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/blocker-comments"):
			createPath = r.URL.Path
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "text": "fix it", "state": "OPEN"})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "task", "create", "42", "--text", "fix it")
	if err != nil {
		t.Fatalf("pr task create: %v (stderr=%s)", err, stderr)
	}
	if versionProbed == "" {
		t.Error("auto mode did not probe application-properties")
	}
	if !strings.HasSuffix(createPath, "/pull-requests/42/blocker-comments") {
		t.Errorf("create path = %q, want blocker-comments", createPath)
	}
	if !strings.Contains(stdout, "Created task 77") {
		t.Errorf("unexpected output: %s", stdout)
	}
}

// Regression: the task command must not define its own PersistentPreRunE, which
// would shadow the root hook and skip global output-flag validation.
func TestPRTaskRespectsRootOutputValidation(t *testing.T) {
	_, _, err := runCLI(t, dcConfig("http://127.0.0.1:0"), "--json", "--yaml", "pr", "task", "list", "42")
	if err == nil {
		t.Fatal("expected --json --yaml to be rejected before reaching the task command")
	}
	if !strings.Contains(err.Error(), "json") || !strings.Contains(err.Error(), "yaml") {
		t.Errorf("error = %v, want a json/yaml conflict error", err)
	}
}

// DC legacy create without --comment-id fails fast with a targeted error and
// never reaches the network.
func TestPRTaskCreateLegacyRequiresCommentID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s %s", r.Method, r.URL.Path)
		http.Error(w, "should not be called", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	_, _, err := runCLI(t, dcConfig(srv.URL), "pr", "task", "create", "42", "--text", "x", "--task-api", "legacy")
	if err == nil {
		t.Fatal("expected error for legacy create without --comment-id")
	}
	if !strings.Contains(err.Error(), "comment-id") {
		t.Errorf("error = %v, want mention of --comment-id", err)
	}
}

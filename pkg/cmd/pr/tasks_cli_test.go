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

// DC: list goes straight to the blocker-comments resource (no version probe).
func TestPRTaskListDCUsesBlockerComments(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if strings.Contains(r.URL.Path, "application-properties") {
			t.Errorf("unexpected version probe: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isLastPage": true,
			"values":     []map[string]any{{"id": 3, "text": "fix it", "state": "OPEN"}},
		})
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "task", "list", "42")
	if err != nil {
		t.Fatalf("pr task list: %v (stderr=%s)", err, stderr)
	}
	if gotPath != "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests/42/blocker-comments" {
		t.Errorf("path = %q, want blocker-comments", gotPath)
	}
	if !strings.Contains(stdout, "[OPEN] 3 fix it") {
		t.Errorf("unexpected output:\n%s", stdout)
	}
}

// DC: create posts directly to blocker-comments with no version probe and no flags.
func TestPRTaskCreateDCUsesBlockerComments(t *testing.T) {
	var createPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "application-properties") {
			t.Errorf("unexpected version probe: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/blocker-comments") {
			createPath = r.URL.Path
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "text": "fix it", "state": "OPEN"})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "task", "create", "42", "--text", "fix it")
	if err != nil {
		t.Fatalf("pr task create: %v (stderr=%s)", err, stderr)
	}
	if !strings.HasSuffix(createPath, "/pull-requests/42/blocker-comments") {
		t.Errorf("create path = %q, want blocker-comments", createPath)
	}
	if !strings.Contains(stdout, "Created task 77") {
		t.Errorf("unexpected output: %s", stdout)
	}
}

// DC: complete fetches the comment version then PUTs state=RESOLVED.
func TestPRTaskCompleteDC(t *testing.T) {
	var methods []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 99, "version": 4, "state": "OPEN"})
		case http.MethodPut:
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if int(body["version"].(float64)) != 4 || body["state"] != "RESOLVED" {
				t.Errorf("PUT body = %#v, want version=4 state=RESOLVED", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 99, "version": 5, "state": "RESOLVED"})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "pr", "task", "complete", "42", "99")
	if err != nil {
		t.Fatalf("pr task complete: %v (stderr=%s)", err, stderr)
	}
	if len(methods) != 2 || methods[0] != http.MethodGet || methods[1] != http.MethodPut {
		t.Errorf("methods = %v, want [GET PUT]", methods)
	}
	if !strings.Contains(stdout, "Completed task 99") {
		t.Errorf("unexpected output: %s", stdout)
	}
}

// Cloud: complete PUTs state=RESOLVED to the task resource.
func TestPRTaskCompleteCloud(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 9, "state": "RESOLVED", "content": map[string]string{"raw": "x"}})
	}))
	t.Cleanup(srv.Close)

	stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "pr", "task", "complete", "42", "9")
	if err != nil {
		t.Fatalf("pr task complete: %v (stderr=%s)", err, stderr)
	}
	if gotMethod != http.MethodPut || gotPath != "/repositories/myworkspace/my-repo/pullrequests/42/tasks/9" {
		t.Errorf("method/path = %s %q", gotMethod, gotPath)
	}
	if gotBody["state"] != "RESOLVED" {
		t.Errorf("body = %#v, want state=RESOLVED", gotBody)
	}
	if !strings.Contains(stdout, "Completed task 9") {
		t.Errorf("unexpected output: %s", stdout)
	}
}

// --json is honored on create and complete, not just list (the whole command
// family must emit the structured shape for automation).
func TestPRTaskJSONOutput(t *testing.T) {
	t.Run("dc create", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 77, "text": "x", "state": "OPEN"})
		}))
		t.Cleanup(srv.Close)
		stdout, stderr, err := runCLI(t, dcConfig(srv.URL), "--json", "pr", "task", "create", "42", "--text", "x")
		if err != nil {
			t.Fatalf("create --json: %v (stderr=%s)", err, stderr)
		}
		assertTaskJSON(t, stdout, 77, "OPEN")
	})

	t.Run("cloud complete", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 9, "state": "RESOLVED", "content": map[string]string{"raw": "x"}})
		}))
		t.Cleanup(srv.Close)
		stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), "--json", "pr", "task", "complete", "42", "9")
		if err != nil {
			t.Fatalf("complete --json: %v (stderr=%s)", err, stderr)
		}
		assertTaskJSON(t, stdout, 9, "RESOLVED")
	})
}

func assertTaskJSON(t *testing.T, stdout string, wantID int, wantState string) {
	t.Helper()
	var payload struct {
		Task struct {
			ID    int    `json:"id"`
			State string `json:"state"`
		} `json:"task"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, stdout)
	}
	if payload.Task.ID != wantID || payload.Task.State != wantState {
		t.Errorf("task = %+v, want id=%d state=%s", payload.Task, wantID, wantState)
	}
}

// Non-positive PR and task IDs are rejected before any network call.
func TestPRTaskRejectsNonPositiveIDs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(srv.Close)

	if _, _, err := runCLI(t, dcConfig(srv.URL), "pr", "task", "list", "0"); err == nil {
		t.Error("expected error for PR id 0")
	}
	if _, _, err := runCLI(t, dcConfig(srv.URL), "pr", "task", "complete", "42", "0"); err == nil {
		t.Error("expected error for task id 0")
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

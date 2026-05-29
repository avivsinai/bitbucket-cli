package bbcloud_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
)

func TestListPullRequestTasksPagination(t *testing.T) {
	var paths []string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("page") {
		case "2":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"id": 3, "state": "RESOLVED", "content": map[string]string{"raw": "third"}},
				},
			})
		default:
			next := "http://" + r.Host + r.URL.Path + "?page=2&pagelen=100"
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{
					{"id": 1, "state": "UNRESOLVED", "content": map[string]string{"raw": "first"}},
					{"id": 2, "state": "UNRESOLVED", "content": map[string]string{"raw": "second"}},
				},
				"next": next,
			})
		}
	}))

	tasks, err := client.ListPullRequestTasks(context.Background(), "ws", "repo", 42, 0)
	if err != nil {
		t.Fatalf("ListPullRequestTasks: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("got %d tasks, want 3", len(tasks))
	}
	if tasks[0].Content.Raw != "first" || tasks[2].State != bbcloud.TaskStateResolved {
		t.Errorf("unexpected task contents: %+v", tasks)
	}
	wantFirst := "/repositories/ws/repo/pullrequests/42/tasks?pagelen=100"
	if paths[0] != wantFirst {
		t.Errorf("first path = %q, want %q", paths[0], wantFirst)
	}
	if len(paths) != 2 {
		t.Errorf("expected 2 requests (paginated), got %d: %v", len(paths), paths)
	}
}

func TestCreatePullRequestTask(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 9, "state": "UNRESOLVED", "content": map[string]string{"raw": "do the thing"},
		})
	}))

	task, err := client.CreatePullRequestTask(context.Background(), "ws", "repo", 42, "do the thing")
	if err != nil {
		t.Fatalf("CreatePullRequestTask: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/repositories/ws/repo/pullrequests/42/tasks" {
		t.Errorf("path = %q", gotPath)
	}
	content, _ := gotBody["content"].(map[string]any)
	if content["raw"] != "do the thing" {
		t.Errorf("body content.raw = %v, want %q", content["raw"], "do the thing")
	}
	if task.ID != 9 {
		t.Errorf("task.ID = %d, want 9", task.ID)
	}
}

func TestSetPullRequestTaskState(t *testing.T) {
	tests := []struct {
		name      string
		resolved  bool
		wantState string
	}{
		{"resolve", true, bbcloud.TaskStateResolved},
		{"reopen", false, bbcloud.TaskStateUnresolved},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath string
			var gotBody map[string]any
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				raw, _ := io.ReadAll(r.Body)
				_ = json.Unmarshal(raw, &gotBody)
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"id": 9, "state": tt.wantState})
			}))

			task, err := client.SetPullRequestTaskState(context.Background(), "ws", "repo", 42, 9, tt.resolved)
			if err != nil {
				t.Fatalf("SetPullRequestTaskState: %v", err)
			}
			if gotMethod != "PUT" {
				t.Errorf("method = %s, want PUT", gotMethod)
			}
			if gotPath != "/repositories/ws/repo/pullrequests/42/tasks/9" {
				t.Errorf("path = %q", gotPath)
			}
			if gotBody["state"] != tt.wantState {
				t.Errorf("body state = %v, want %s", gotBody["state"], tt.wantState)
			}
			if task.State != tt.wantState {
				t.Errorf("task.State = %s, want %s", task.State, tt.wantState)
			}
		})
	}
}

// A small requested limit is clamped up to Bitbucket Cloud's minimum pagelen of
// 10, while the returned slice is still truncated to the caller's limit.
func TestListPullRequestTasksClampsPagelen(t *testing.T) {
	var gotPagelen string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPagelen = r.URL.Query().Get("pagelen")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{"id": 1, "state": "UNRESOLVED", "content": map[string]string{"raw": "a"}},
				{"id": 2, "state": "UNRESOLVED", "content": map[string]string{"raw": "b"}},
			},
		})
	}))

	tasks, err := client.ListPullRequestTasks(context.Background(), "ws", "repo", 42, 1)
	if err != nil {
		t.Fatalf("ListPullRequestTasks: %v", err)
	}
	if gotPagelen != "10" {
		t.Errorf("pagelen = %q, want 10 (clamped minimum)", gotPagelen)
	}
	if len(tasks) != 1 {
		t.Errorf("len(tasks) = %d, want 1 (truncated to limit)", len(tasks))
	}
}

// Bitbucket Cloud returns an absolute next URL under /2.0; following it must not
// double the base path (e.g. /2.0/2.0/...).
func TestListPullRequestTasksFollowsAbsoluteNext(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "2" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []map[string]any{{"id": 2, "state": "RESOLVED", "content": map[string]string{"raw": "b"}}},
			})
			return
		}
		next := "http://" + r.Host + "/2.0/repositories/ws/repo/pullrequests/42/tasks?page=2&pagelen=100"
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{{"id": 1, "state": "UNRESOLVED", "content": map[string]string{"raw": "a"}}},
			"next":   next,
		})
	}))
	t.Cleanup(srv.Close)

	client, err := bbcloud.New(bbcloud.Options{BaseURL: srv.URL + "/2.0", Username: "u", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}

	tasks, err := client.ListPullRequestTasks(context.Background(), "ws", "repo", 42, 0)
	if err != nil {
		t.Fatalf("ListPullRequestTasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("len(tasks) = %d, want 2", len(tasks))
	}
	for _, p := range paths {
		if strings.Contains(p, "/2.0/2.0/") {
			t.Errorf("base path doubled: %q", p)
		}
	}
	if len(paths) != 2 || paths[1] != "/2.0/repositories/ws/repo/pullrequests/42/tasks" {
		t.Errorf("paths = %v, want second page under a single /2.0", paths)
	}
}

func TestCloudTaskValidation(t *testing.T) {
	client, err := bbcloud.New(bbcloud.Options{BaseURL: "http://localhost", Username: "u", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.CreatePullRequestTask(context.Background(), "", "repo", 1, "x"); err == nil {
		t.Error("expected error for empty workspace")
	}
	if _, err := client.CreatePullRequestTask(context.Background(), "ws", "repo", 1, "   "); err == nil {
		t.Error("expected error for blank task text")
	}
	if _, err := client.ListPullRequestTasks(context.Background(), "ws", "", 1, 0); err == nil {
		t.Error("expected error for empty repo")
	}
	if _, err := client.ListPullRequestTasks(context.Background(), "ws", "repo", 0, 0); err == nil {
		t.Error("expected error for non-positive pull request id")
	}
	if _, err := client.SetPullRequestTaskState(context.Background(), "ws", "repo", 42, 0, true); err == nil {
		t.Error("expected error for non-positive task id")
	}
}

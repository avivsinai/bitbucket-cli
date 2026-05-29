package bbdc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
)

func TestServerVersionUsesApplicationProperties(t *testing.T) {
	var gotPath string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version":     "7.6.0",
			"buildNumber": "760000",
			"displayName": "Bitbucket",
		})
	}))

	version, err := client.ServerVersion(context.Background())
	if err != nil {
		t.Fatalf("ServerVersion: %v", err)
	}
	if gotPath != "/rest/api/1.0/application-properties" {
		t.Fatalf("path = %q, want /rest/api/1.0/application-properties", gotPath)
	}
	if version != "7.6.0" {
		t.Fatalf("version = %q, want 7.6.0", version)
	}
}

func TestListPullRequestTasksKeepsLegacyEndpoint(t *testing.T) {
	var gotPath string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isLastPage": true,
			"values": []map[string]any{
				{"id": 11, "text": "fix this", "state": "OPEN"},
			},
		})
	}))

	tasks, err := client.ListPullRequestTasks(context.Background(), "PROJ", "repo", 42)
	if err != nil {
		t.Fatalf("ListPullRequestTasks: %v", err)
	}
	if gotPath != "/rest/api/1.0/projects/PROJ/repos/repo/pull-requests/42/tasks" {
		t.Fatalf("path = %q, want legacy tasks path", gotPath)
	}
	if len(tasks) != 1 || tasks[0].ID != 11 || tasks[0].Text != "fix this" {
		t.Fatalf("tasks = %+v", tasks)
	}
}

func TestListBlockerCommentsPaginates(t *testing.T) {
	var hits int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		switch count {
		case 1:
			if r.URL.Query().Get("start") != "0" {
				t.Fatalf("first start = %q, want 0", r.URL.Query().Get("start"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"isLastPage":    false,
				"nextPageStart": 25,
				"values":        []map[string]any{{"id": 1, "text": "first", "state": "OPEN"}},
			})
		case 2:
			if r.URL.Query().Get("start") != "25" {
				t.Fatalf("second start = %q, want 25", r.URL.Query().Get("start"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"isLastPage": true,
				"values":     []map[string]any{{"id": 2, "text": "second", "state": "RESOLVED"}},
			})
		default:
			t.Fatalf("unexpected request %d", count)
		}
	}))

	tasks, err := client.ListBlockerComments(context.Background(), "PROJ", "repo", 42)
	if err != nil {
		t.Fatalf("ListBlockerComments: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("len(tasks) = %d, want 2", len(tasks))
	}
	if hits != 2 {
		t.Fatalf("hits = %d, want 2", hits)
	}
}

func TestCreateBlockerComment(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/rest/api/1.0/projects/PROJ/repos/repo/pull-requests/42/blocker-comments" {
			t.Fatalf("path = %q, want blocker-comments path", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["text"] != "fix docs" {
			t.Fatalf("body = %#v, want text only", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 12, "version": 1, "text": "fix docs", "state": "OPEN"})
	}))

	task, err := client.CreateBlockerComment(context.Background(), "PROJ", "repo", 42, "fix docs")
	if err != nil {
		t.Fatalf("CreateBlockerComment: %v", err)
	}
	if task.ID != 12 || task.Text != "fix docs" {
		t.Fatalf("task = %+v", task)
	}
}

func TestSetBlockerCommentStateFetchesVersionBeforePUT(t *testing.T) {
	var hits int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		switch count {
		case 1:
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if r.URL.Path != "/rest/api/1.0/projects/PROJ/repos/repo/pull-requests/42/blocker-comments/99" {
				t.Fatalf("path = %q, want blocker-comments get path", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 99, "version": 7, "text": "fix docs", "state": "OPEN"})
		case 2:
			if r.Method != http.MethodPut {
				t.Fatalf("method = %s, want PUT", r.Method)
			}
			if r.URL.Path != "/rest/api/1.0/projects/PROJ/repos/repo/pull-requests/42/blocker-comments/99" {
				t.Fatalf("path = %q, want blocker-comments put path", r.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if int(body["version"].(float64)) != 7 || body["state"] != "RESOLVED" {
				t.Fatalf("body = %#v, want version=7 state=RESOLVED", body)
			}
			if _, ok := body["text"]; ok {
				t.Fatalf("body should not update text: %#v", body)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 99, "version": 8, "state": "RESOLVED"})
		default:
			t.Fatalf("unexpected request %d", count)
		}
	}))

	task, err := client.SetBlockerCommentState(context.Background(), "PROJ", "repo", 42, 99, true)
	if err != nil {
		t.Fatalf("SetBlockerCommentState: %v", err)
	}
	if task.ID != 99 || task.State != "RESOLVED" {
		t.Fatalf("task = %+v, want id=99 state=RESOLVED", task)
	}
	if hits != 2 {
		t.Fatalf("hits = %d, want 2", hits)
	}
}

func TestCreateLegacyTaskRequiresCommentID(t *testing.T) {
	client, err := bbdc.New(bbdc.Options{BaseURL: "http://localhost", Username: "u", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.CreateLegacyTask(context.Background(), "PROJ", "repo", 42, 0, "legacy task")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateLegacyTaskUsesTopLevelTaskAnchor(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/rest/api/1.0/tasks" {
			t.Fatalf("path = %q, want /rest/api/1.0/tasks", r.URL.Path)
		}
		var body struct {
			Anchor struct {
				ID   int    `json:"id"`
				Type string `json:"type"`
			} `json:"anchor"`
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Anchor.ID != 123 || body.Anchor.Type != "COMMENT" || body.Text != "legacy task" {
			t.Fatalf("body = %+v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 99, "text": "legacy task", "state": "OPEN"})
	}))

	task, err := client.CreateLegacyTask(context.Background(), "PROJ", "repo", 42, 123, "legacy task")
	if err != nil {
		t.Fatalf("CreateLegacyTask: %v", err)
	}
	if task.ID != 99 {
		t.Fatalf("task.ID = %d, want 99", task.ID)
	}
}

func TestSetLegacyTaskStateUsesTopLevelTaskPUT(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/rest/api/1.0/tasks/99" {
			t.Fatalf("path = %q, want /rest/api/1.0/tasks/99", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body["state"] != "RESOLVED" {
			t.Fatalf("body = %#v, want state=RESOLVED", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": 99, "state": "RESOLVED"})
	}))

	task, err := client.SetLegacyTaskState(context.Background(), 99, true)
	if err != nil {
		t.Fatalf("SetLegacyTaskState: %v", err)
	}
	if task.ID != 99 || task.State != "RESOLVED" {
		t.Fatalf("task = %+v, want id=99 state=RESOLVED", task)
	}
}

func TestDeleteLegacyTaskUsesTopLevelTaskDELETE(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/rest/api/1.0/tasks/99" {
			t.Fatalf("path = %q, want /rest/api/1.0/tasks/99", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	if err := client.DeleteLegacyTask(context.Background(), 99); err != nil {
		t.Fatalf("DeleteLegacyTask: %v", err)
	}
}

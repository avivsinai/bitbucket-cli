package bbdc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
)

func TestListPullRequestTasksPaginatesBlockerComments(t *testing.T) {
	var hits int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/rest/api/1.0/projects/PROJ/repos/repo/pull-requests/42/blocker-comments" {
			t.Fatalf("path = %q, want blocker-comments path", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "25" {
			t.Fatalf("limit = %q, want 25", r.URL.Query().Get("limit"))
		}
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

	tasks, err := client.ListPullRequestTasks(context.Background(), "PROJ", "repo", 42)
	if err != nil {
		t.Fatalf("ListPullRequestTasks: %v", err)
	}
	if len(tasks) != 2 {
		t.Fatalf("len(tasks) = %d, want 2", len(tasks))
	}
	if hits != 2 {
		t.Fatalf("hits = %d, want 2", hits)
	}
}

func TestCreatePullRequestTaskUsesBlockerComments(t *testing.T) {
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

	task, err := client.CreatePullRequestTask(context.Background(), "PROJ", "repo", 42, "fix docs")
	if err != nil {
		t.Fatalf("CreatePullRequestTask: %v", err)
	}
	if task.ID != 12 || task.Text != "fix docs" {
		t.Fatalf("task = %+v", task)
	}
}

func TestSetPullRequestTaskStateFetchesVersionBeforePUT(t *testing.T) {
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

	task, err := client.SetPullRequestTaskState(context.Background(), "PROJ", "repo", 42, 99, true)
	if err != nil {
		t.Fatalf("SetPullRequestTaskState: %v", err)
	}
	if task.ID != 99 || task.State != "RESOLVED" {
		t.Fatalf("task = %+v, want id=99 state=RESOLVED", task)
	}
	if hits != 2 {
		t.Fatalf("hits = %d, want 2", hits)
	}
}

func TestSetPullRequestTaskStateRequiresTaskID(t *testing.T) {
	client, err := bbdc.New(bbdc.Options{BaseURL: "http://localhost", Username: "u", Token: "t"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.SetPullRequestTaskState(context.Background(), "PROJ", "repo", 42, 0, true)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "task id must be positive") {
		t.Fatalf("error = %q, want task id validation", err)
	}
}

func TestPullRequestTaskHTTPFailuresIncludeDCHint(t *testing.T) {
	tests := []struct {
		name string
		run  func(*bbdc.Client) error
	}{
		{
			name: "list",
			run: func(client *bbdc.Client) error {
				_, err := client.ListPullRequestTasks(context.Background(), "PROJ", "repo", 42)
				return err
			},
		},
		{
			name: "create",
			run: func(client *bbdc.Client) error {
				_, err := client.CreatePullRequestTask(context.Background(), "PROJ", "repo", 42, "fix docs")
				return err
			},
		},
		{
			name: "set",
			run: func(client *bbdc.Client) error {
				_, err := client.SetPullRequestTaskState(context.Background(), "PROJ", "repo", 42, 99, true)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"errors": []map[string]any{{"message": "not found"}},
				})
			}))

			err := tt.run(client)
			if err == nil {
				t.Fatal("expected error")
			}
			got := err.Error()
			if !strings.Contains(got, "404 Not Found: not found") {
				t.Fatalf("error = %q, want original HTTP error", got)
			}
			if !strings.Contains(got, "DC tasks use the blocker-comments API introduced in Bitbucket Data Center 7.2+") {
				t.Fatalf("error = %q, want DC task API hint", got)
			}
		})
	}
}

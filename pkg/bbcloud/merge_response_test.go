package bbcloud_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
)

func TestMergeAcceptedResponse(t *testing.T) {
	const mergePath = "/2.0/repositories/ws/repo/pullrequests/1/merge"
	const taskPath = mergePath + "/task-status/task-123"
	tests := []struct {
		name, body, location string
		status               int
		wantPoll, wantError  bool
	}{
		{name: "completed object", status: 200, body: `{"state":"MERGED"}`},
		{name: "completed empty", status: 200},
		{name: "legacy task", status: 202, body: `{"task_id":"task-123"}`, wantPoll: true},
		{name: "absolute string", status: 202, body: `""`, location: "absolute", wantPoll: true},
		{name: "relative empty", status: 202, location: "merge/task-status/task-123", wantPoll: true},
		{name: "root relative null", status: 202, body: `null`, location: taskPath, wantPoll: true},
		{name: "root relative object", status: 202, body: `{}`, location: taskPath, wantPoll: true},
		{name: "header precedes body", status: 202, body: `{"task_id":"wrong"}`, location: taskPath, wantPoll: true},
		{name: "header ignores malformed body", status: 202, body: `unavailable`, location: taskPath, wantPoll: true},
		{name: "untracked empty", status: 202, wantError: true},
		{name: "untracked string", status: 202, body: `""`, wantError: true},
		{name: "untracked null", status: 202, body: `null`, wantError: true},
		{name: "untracked object", status: 202, body: `{}`, wantError: true},
		{name: "untracked malformed", status: 202, body: `{`, wantError: true},
		{name: "malformed location", status: 202, location: "%", wantError: true},
		{name: "wrong PR", status: 202, location: strings.Replace(taskPath, "/1/", "/2/", 1), wantError: true},
		{name: "wrong endpoint", status: 202, location: "/2.0/user", wantError: true},
		{name: "missing task", status: 202, location: mergePath + "/task-status/", wantError: true},
		{name: "dot task", status: 202, body: `{"task_id":".."}`, wantError: true},
		{name: "encoded slash task", status: 202, location: taskPath + "%2fextra", wantError: true},
		{name: "query", status: 202, location: taskPath + "?extra=1", wantError: true},
		{name: "fragment", status: 202, location: taskPath + "#extra", wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var posts, polls int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPost && r.URL.Path == mergePath:
					posts++
					location := tt.location
					if location == "absolute" {
						location = "http://" + r.Host + taskPath
					}
					if location != "" {
						w.Header().Set("Location", location)
					}
					w.WriteHeader(tt.status)
					_, _ = fmt.Fprint(w, tt.body)
				case r.Method == http.MethodGet && r.URL.Path == taskPath:
					polls++
					status := "PENDING"
					if polls == 2 {
						status = "SUCCESS"
					}
					_, _ = fmt.Fprintf(w, `{"task_status":%q}`, status)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(server.Close)
			client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL + "/2.0", MergePollInterval: time.Millisecond})
			if err != nil {
				t.Fatal(err)
			}
			err = client.MergePullRequest(context.Background(), "ws", "repo", 1, "", "", false)
			if tt.wantError {
				if err == nil || !strings.Contains(err.Error(), "accepted (HTTP 202)") || !strings.Contains(err.Error(), "completion is unknown") {
					t.Fatalf("error = %v, want accepted with unknown completion", err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			wantPolls := 0
			if tt.wantPoll {
				wantPolls = 2
			}
			if posts != 1 || polls != wantPolls {
				t.Fatalf("POSTs = %d, polls = %d; want 1, %d", posts, polls, wantPolls)
			}
		})
	}
}

func TestMergeLocationPendingCancellation(t *testing.T) {
	pollStarted := make(chan struct{})
	var polls atomic.Int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Location", r.URL.Path+"/task-status/pending")
			w.WriteHeader(http.StatusAccepted)
			return
		}
		if polls.Add(1) == 1 {
			close(pollStarted)
		}
		_, _ = fmt.Fprint(w, `{"task_status":"PENDING","queued_pullrequest":{"status":"QUEUED"}}`)
	}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- client.MergePullRequest(ctx, "ws", "repo", 1, "", "", false) }()
	select {
	case <-pollStarted:
	case <-time.After(time.Second):
		t.Fatal("polling did not start")
	}
	cancel()
	err := <-result
	if !errors.Is(err, context.Canceled) || !strings.Contains(err.Error(), "may still be running") {
		t.Fatalf("error = %v; want unknown completion wrapping cancellation", err)
	}
}

func TestMergeRejectsExternalPolling(t *testing.T) {
	var externalRequests atomic.Int32
	external := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		externalRequests.Add(1)
		t.Error("request reached another origin")
	}))
	t.Cleanup(external.Close)
	for _, redirect := range []bool{false, true} {
		t.Run(fmt.Sprintf("redirect=%t", redirect), func(t *testing.T) {
			client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodPost {
					location := external.URL + r.URL.Path + "/task-status/task-123"
					if redirect {
						location = r.URL.Path + "/task-status/task-123"
					}
					w.Header().Set("Location", location)
					w.WriteHeader(http.StatusAccepted)
					return
				}
				http.Redirect(w, r, external.URL+r.URL.Path, http.StatusFound)
			}))
			if err := client.MergePullRequest(context.Background(), "ws", "repo", 1, "", "", false); err == nil {
				t.Fatal("expected refusal to poll another origin")
			}
		})
	}
	if externalRequests.Load() != 0 {
		t.Fatal("external origin was contacted")
	}
}

func TestMergeRejectsPollingRedirectToAnotherPR(t *testing.T) {
	var wrongPRRequests atomic.Int32
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			w.Header().Set("Location", r.URL.Path+"/task-status/123")
			w.WriteHeader(http.StatusAccepted)
		case strings.Contains(r.URL.Path, "/pullrequests/1/"):
			http.Redirect(w, r, "/repositories/ws/repo/pullrequests/2/merge/task-status/123", http.StatusFound)
		default:
			wrongPRRequests.Add(1)
			_, _ = fmt.Fprint(w, `{"task_status":"SUCCESS"}`)
		}
	}))
	if err := client.MergePullRequest(context.Background(), "ws", "repo", 1, "", "", false); err == nil {
		t.Fatal("expected refusal of polling redirect")
	}
	if wrongPRRequests.Load() != 0 {
		t.Fatal("polling followed a redirect to another pull request")
	}
}

package bbcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestListPipelinesPaginates(t *testing.T) {
	var hits int32
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")

		switch count {
		case 1:
			if r.URL.Query().Get("pagelen") == "" {
				t.Fatalf("expected pagelen query in first request")
			}
			payload := PipelinePage{
				Values: []Pipeline{{UUID: "1"}, {UUID: "2"}},
				Next:   serverURL + "/repositories/work/repo/pipelines/?pagelen=20&page=2",
			}
			_ = json.NewEncoder(w).Encode(payload)
		case 2:
			payload := PipelinePage{
				Values: []Pipeline{{UUID: "3"}},
			}
			_ = json.NewEncoder(w).Encode(payload)
		default:
			t.Fatalf("unexpected extra request %d", count)
		}
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	pipelines, err := client.ListPipelines(ctx, "work", "repo", 0)
	if err != nil {
		t.Fatalf("ListPipelines: %v", err)
	}

	if len(pipelines) != 3 {
		t.Fatalf("expected 3 pipelines, got %d", len(pipelines))
	}
	if hits != 2 {
		t.Fatalf("expected 2 requests, got %d", hits)
	}
}

func TestListPipelinesRespectsLimit(t *testing.T) {
	var hits int32
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")

		if count == 1 {
			payload := PipelinePage{
				Values: []Pipeline{{UUID: "1"}, {UUID: "2"}},
				Next:   serverURL + "/repositories/work/repo/pipelines/?pagelen=20&page=2",
			}
			_ = json.NewEncoder(w).Encode(payload)
			return
		}

		t.Fatalf("unexpected second request when limit satisfied")
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	pipelines, err := client.ListPipelines(ctx, "work", "repo", 1)
	if err != nil {
		t.Fatalf("ListPipelines: %v", err)
	}

	if len(pipelines) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(pipelines))
	}
	if hits != 1 {
		t.Fatalf("expected 1 request, got %d", hits)
	}
}

func TestCommitStatuses(t *testing.T) {
	tests := []struct {
		name          string
		workspace     string
		repoSlug      string
		commit        string
		expectError   bool
		errorContains string
		mockResponses []struct {
			values []CommitStatus
			next   string
		}
		expectedCount int
	}{
		{
			name:      "single page of statuses",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			commit:    "abc123",
			mockResponses: []struct {
				values []CommitStatus
				next   string
			}{
				{
					values: []CommitStatus{
						{
							State: "SUCCESSFUL",
							Key:   "build-1",
							Name:  "Build 1",
							URL:   "https://example.com/build/1",
						},
						{
							State: "FAILED",
							Key:   "test-1",
							Name:  "Test 1",
							URL:   "https://example.com/test/1",
						},
					},
					next: "",
				},
			},
			expectedCount: 2,
		},
		{
			name:      "multiple pages of statuses",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			commit:    "def456",
			mockResponses: []struct {
				values []CommitStatus
				next   string
			}{
				{
					values: []CommitStatus{
						{State: "SUCCESSFUL", Key: "build-1"},
						{State: "INPROGRESS", Key: "build-2"},
					},
					next: "/page2",
				},
				{
					values: []CommitStatus{
						{State: "FAILED", Key: "build-3"},
					},
					next: "",
				},
			},
			expectedCount: 3,
		},
		{
			name:      "empty results",
			workspace: "myworkspace",
			repoSlug:  "myrepo",
			commit:    "nobuilds",
			mockResponses: []struct {
				values []CommitStatus
				next   string
			}{
				{
					values: []CommitStatus{},
					next:   "",
				},
			},
			expectedCount: 0,
		},
		{
			name:          "missing workspace",
			workspace:     "",
			repoSlug:      "myrepo",
			commit:        "abc123",
			expectError:   true,
			errorContains: "workspace and repository slug are required",
		},
		{
			name:          "missing repo slug",
			workspace:     "myworkspace",
			repoSlug:      "",
			commit:        "abc123",
			expectError:   true,
			errorContains: "workspace and repository slug are required",
		},
		{
			name:          "missing commit sha",
			workspace:     "myworkspace",
			repoSlug:      "myrepo",
			commit:        "",
			expectError:   true,
			errorContains: "commit SHA is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectError {
				client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
				if err != nil {
					t.Fatalf("New: %v", err)
				}

				ctx := context.Background()
				_, err = client.CommitStatuses(ctx, tt.workspace, tt.repoSlug, tt.commit)
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errorContains)
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			var hits int32
			var serverURL string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				count := atomic.AddInt32(&hits, 1)
				w.Header().Set("Content-Type", "application/json")

				if count > int32(len(tt.mockResponses)) {
					t.Fatalf("unexpected request %d, only %d responses configured", count, len(tt.mockResponses))
				}

				response := tt.mockResponses[count-1]
				nextURL := ""
				if response.next != "" {
					nextURL = serverURL + response.next
				}

				resp := struct {
					Values []CommitStatus `json:"values"`
					Next   string         `json:"next"`
				}{
					Values: response.values,
					Next:   nextURL,
				}
				_ = json.NewEncoder(w).Encode(resp)
			}))
			serverURL = server.URL
			t.Cleanup(server.Close)

			client, err := New(Options{BaseURL: server.URL})
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			ctx := context.Background()
			statuses, err := client.CommitStatuses(ctx, tt.workspace, tt.repoSlug, tt.commit)
			if err != nil {
				t.Fatalf("CommitStatuses: %v", err)
			}

			if len(statuses) != tt.expectedCount {
				t.Fatalf("expected %d statuses, got %d", tt.expectedCount, len(statuses))
			}

			if hits != int32(len(tt.mockResponses)) {
				t.Fatalf("expected %d requests, got %d", len(tt.mockResponses), hits)
			}
		})
	}
}

func TestCommitStatusesPathEncoding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		expectedPath := "/repositories/my-workspace/my-repo/commit/abc123def456/statuses"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}

		resp := struct {
			Values []CommitStatus `json:"values"`
			Next   string         `json:"next"`
		}{
			Values: []CommitStatus{
				{State: "SUCCESSFUL", Key: "test"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx := context.Background()
	_, err = client.CommitStatuses(ctx, "my-workspace", "my-repo", "abc123def456")
	if err != nil {
		t.Fatalf("CommitStatuses: %v", err)
	}
}

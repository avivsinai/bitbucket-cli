package bbcloud

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

func TestPipelineStepStatus(t *testing.T) {
	tests := []struct {
		name   string
		state  string
		result string
		want   string
	}{
		{"pending", "PENDING", "", "PENDING"},
		{"running", "RUNNING", "", "RUNNING"},
		{"completed successful", "COMPLETED", "SUCCESSFUL", "COMPLETED SUCCESSFUL"},
		{"completed failed", "COMPLETED", "FAILED", "COMPLETED FAILED"},
		{"completed error", "COMPLETED", "ERROR", "COMPLETED ERROR"},
		{"completed stopped", "COMPLETED", "STOPPED", "COMPLETED STOPPED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			step := PipelineStep{
				Name: "lint",
			}
			step.State.Name = tt.state
			step.Result.Name = tt.result
			if got := step.Status(); got != tt.want {
				t.Errorf("Status() = %q, want %q", got, tt.want)
			}
		})
	}
}

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
			if r.URL.Query().Get("sort") != "-created_on" {
				t.Fatalf("expected sort=-created_on query in first request")
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
			if r.URL.Query().Get("sort") != "-created_on" {
				t.Fatalf("expected sort=-created_on query in first request")
			}
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

func TestNormalizeUUID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"550e8400-e29b-41d4-a716-446655440000", "{550e8400-e29b-41d4-a716-446655440000}"},
		{"{550e8400-e29b-41d4-a716-446655440000}", "{550e8400-e29b-41d4-a716-446655440000}"},
		{" 550e8400-e29b-41d4-a716-446655440000 ", "{550e8400-e29b-41d4-a716-446655440000}"},
		{"550E8400-E29B-41D4-A716-446655440000", "{550E8400-E29B-41D4-A716-446655440000}"},
		{"abc-123", ""},
		{"{abc-123}", ""},
		{"abc-123}", ""},
		{"{abc-123", ""},
		{"{}", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeUUID(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeUUID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestLooksLikeUUID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"{550e8400-e29b-41d4-a716-446655440000}", true},
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{" 550e8400-e29b-41d4-a716-446655440000 ", true}, // trimmed
		{"{abc-123}", false},                             // not canonical UUID
		{"abc-123", false},                               // not canonical UUID
		{"{550e8400-e29b-41d4-a716-446655440000", false}, // half-brace (opening only)
		{"550e8400-e29b-41d4-a716-446655440000}", false}, // half-brace (closing only)
		{"cafe", false},                                  // hex-only username
		{"dead", false},                                  // hex-only username
		{"alice", false},
		{"bob_smith", false},
		{"user.name", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := LooksLikeUUID(tt.input)
			if got != tt.want {
				t.Errorf("LooksLikeUUID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestLooksLikeAccountID(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"557058:12345678-1234-1234-1234-123456789abc", true},
		{" 557058:12345678-1234-1234-1234-123456789abc ", true}, // trimmed
		{"712020:abcdef01-2345-6789-abcd-ef0123456789", true},
		{"alice", false},
		{"{550e8400-e29b-41d4-a716-446655440000}", false}, // UUID, not account ID
		{"550e8400-e29b-41d4-a716-446655440000", false},   // bare UUID
		{"bob_smith", false},
		{"user.name", false},
		{"", false},
		{":", false},
		{"abc:12345678-1234-1234-1234-123456789abc", false}, // non-numeric prefix
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := LooksLikeAccountID(tt.input)
			if got != tt.want {
				t.Errorf("LooksLikeAccountID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := New(Options{
		BaseURL: server.URL,
		Retry:   httpx.RetryPolicy{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func newTestClientWithBasePath(t *testing.T, basePath string, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := New(Options{
		BaseURL: server.URL + basePath,
		Retry:   httpx.RetryPolicy{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func TestNewDefaultsBaseURL(t *testing.T) {
	client, err := New(Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client == nil {
		t.Fatal("expected client to be created")
	}
}

func TestListRepositoriesPaginates(t *testing.T) {
	var hits int32
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")

		switch count {
		case 1:
			_ = json.NewEncoder(w).Encode(repositoryListPage{
				Values: []Repository{{Slug: "repo1"}, {Slug: "repo2"}},
				Next:   serverURL + "/repositories/ws?pagelen=20&page=2",
			})
		case 2:
			_ = json.NewEncoder(w).Encode(repositoryListPage{
				Values: []Repository{{Slug: "repo3"}},
			})
		default:
			t.Fatalf("unexpected request %d", count)
		}
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	repos, err := client.ListRepositories(context.Background(), "ws", 0)
	if err != nil {
		t.Fatalf("ListRepositories: %v", err)
	}
	if len(repos) != 3 {
		t.Fatalf("expected 3 repos, got %d", len(repos))
	}
	if hits != 2 {
		t.Fatalf("expected 2 requests, got %d", hits)
	}
}

func TestListRepositoriesRespectsLimit(t *testing.T) {
	var hits int32
	var serverURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(repositoryListPage{
			Values: []Repository{{Slug: "repo1"}, {Slug: "repo2"}, {Slug: "repo3"}},
			Next:   serverURL + "/repositories/ws?pagelen=20&page=2",
		})
	}))
	serverURL = server.URL
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	repos, err := client.ListRepositories(context.Background(), "ws", 2)
	if err != nil {
		t.Fatalf("ListRepositories: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if hits != 1 {
		t.Fatalf("expected 1 request, got %d", hits)
	}
}

func TestListRepositoriesRequiresWorkspace(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	_, err := client.ListRepositories(context.Background(), "", 10)
	if err == nil {
		t.Fatal("expected error for empty workspace")
	}
}

func TestGetRepository(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/repositories/ws/my-repo") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Repository{Slug: "my-repo", Name: "My Repo"})
	})

	client := newTestClient(t, handler)
	repo, err := client.GetRepository(context.Background(), "ws", "my-repo")
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}
	if repo.Slug != "my-repo" {
		t.Fatalf("expected my-repo, got %q", repo.Slug)
	}
}

func TestGetRepositoryRequiresParams(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	_, err := client.GetRepository(context.Background(), "", "repo")
	if err == nil {
		t.Fatal("expected error for empty workspace")
	}
	_, err = client.GetRepository(context.Background(), "ws", "")
	if err == nil {
		t.Fatal("expected error for empty repo slug")
	}
}

func TestCreateRepository(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/repositories/ws/new-repo") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["scm"] != "git" {
			t.Errorf("expected scm=git, got %v", body["scm"])
		}
		if body["is_private"] != true {
			t.Errorf("expected is_private=true, got %v", body["is_private"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Repository{Slug: "new-repo", Name: "New Repo"})
	})

	client := newTestClient(t, handler)
	repo, err := client.CreateRepository(context.Background(), "ws", CreateRepositoryInput{
		Slug:      "new-repo",
		IsPrivate: true,
	})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	if repo.Slug != "new-repo" {
		t.Fatalf("expected new-repo, got %q", repo.Slug)
	}
}

func TestCreateRepositoryValidation(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	_, err := client.CreateRepository(context.Background(), "", CreateRepositoryInput{Slug: "repo"})
	if err == nil {
		t.Fatal("expected error for empty workspace")
	}
	_, err = client.CreateRepository(context.Background(), "ws", CreateRepositoryInput{})
	if err == nil {
		t.Fatal("expected error for empty slug")
	}
}

func TestTriggerPipeline(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		target := body["target"].(map[string]any)
		if target["ref_name"] != "main" {
			t.Errorf("expected ref_name=main, got %v", target["ref_name"])
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Pipeline{UUID: "{abc-123}"})
	})

	client := newTestClient(t, handler)
	pipeline, err := client.TriggerPipeline(context.Background(), "ws", "repo", TriggerPipelineInput{
		Ref: "main",
	})
	if err != nil {
		t.Fatalf("TriggerPipeline: %v", err)
	}
	if pipeline.UUID != "{abc-123}" {
		t.Fatalf("expected UUID {abc-123}, got %q", pipeline.UUID)
	}
}

func TestTriggerPipelineWithVariables(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)

		vars, ok := body["variables"].([]any)
		if !ok || len(vars) == 0 {
			t.Fatal("expected variables in body")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Pipeline{UUID: "{abc}"})
	})

	client := newTestClient(t, handler)
	_, err := client.TriggerPipeline(context.Background(), "ws", "repo", TriggerPipelineInput{
		Ref:       "main",
		Variables: map[string]string{"ENV": "prod"},
	})
	if err != nil {
		t.Fatalf("TriggerPipeline: %v", err)
	}
}

func TestTriggerPipelineValidation(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	_, err := client.TriggerPipeline(context.Background(), "", "repo", TriggerPipelineInput{Ref: "main"})
	if err == nil {
		t.Fatal("expected error for empty workspace")
	}
	_, err = client.TriggerPipeline(context.Background(), "ws", "repo", TriggerPipelineInput{})
	if err == nil {
		t.Fatal("expected error for empty ref")
	}
}

func TestGetPipelineNormalizesUUID(t *testing.T) {
	const pipelineUUID = "{550e8400-e29b-41d4-a716-446655440000}"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, pipelineUUID) {
			t.Errorf("expected normalized UUID in path, got: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Pipeline{UUID: pipelineUUID})
	})

	client := newTestClient(t, handler)
	pipeline, err := client.GetPipeline(context.Background(), "ws", "repo", "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("GetPipeline: %v", err)
	}
	if pipeline.UUID != pipelineUUID {
		t.Fatalf("expected UUID preserved in response, got %q", pipeline.UUID)
	}
}

func TestListPipelineStepsNormalizesUUID(t *testing.T) {
	const pipelineUUID = "{550e8400-e29b-41d4-a716-446655440000}"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, pipelineUUID) {
			t.Errorf("expected normalized UUID in path, got: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{"uuid": "{123e4567-e89b-12d3-a456-426614174000}", "name": "Build"},
			},
		})
	})

	client := newTestClient(t, handler)
	steps, err := client.ListPipelineSteps(context.Background(), "ws", "repo", "550e8400-e29b-41d4-a716-446655440000")
	if err != nil {
		t.Fatalf("ListPipelineSteps: %v", err)
	}
	if len(steps) != 1 || steps[0].Name != "Build" {
		t.Fatalf("unexpected steps: %+v", steps)
	}
}

func TestGetPipelineLogsNormalizesUUIDs(t *testing.T) {
	const pipelineUUID = "{550e8400-e29b-41d4-a716-446655440000}"
	const stepUUID = "{123e4567-e89b-12d3-a456-426614174000}"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, pipelineUUID) {
			t.Errorf("expected normalized pipeline UUID in path, got: %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.Path, stepUUID) {
			t.Errorf("expected normalized step UUID in path, got: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("build output here"))
	})

	client := newTestClient(t, handler)
	logs, err := client.GetPipelineLogs(context.Background(), "ws", "repo", "550e8400-e29b-41d4-a716-446655440000", "123e4567-e89b-12d3-a456-426614174000")
	if err != nil {
		t.Fatalf("GetPipelineLogs: %v", err)
	}
	if string(logs) != "build output here" {
		t.Fatalf("expected log content, got %q", string(logs))
	}
}

func TestPipelineEndpointsRejectInvalidUUIDs(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected HTTP request to %s", r.URL.Path)
	}))

	tests := []struct {
		name string
		call func() error
		want string
	}{
		{
			name: "get pipeline",
			call: func() error {
				_, err := client.GetPipeline(context.Background(), "ws", "repo", "not-a-uuid")
				return err
			},
			want: "pipeline UUID must be a canonical UUID",
		},
		{
			name: "list pipeline steps",
			call: func() error {
				_, err := client.ListPipelineSteps(context.Background(), "ws", "repo", "not-a-uuid")
				return err
			},
			want: "pipeline UUID must be a canonical UUID",
		},
		{
			name: "get pipeline logs rejects pipeline UUID",
			call: func() error {
				_, err := client.GetPipelineLogs(context.Background(), "ws", "repo", "not-a-uuid", "123e4567-e89b-12d3-a456-426614174000")
				return err
			},
			want: "pipeline UUID must be a canonical UUID",
		},
		{
			name: "get pipeline logs rejects step UUID",
			call: func() error {
				_, err := client.GetPipelineLogs(context.Background(), "ws", "repo", "550e8400-e29b-41d4-a716-446655440000", "not-a-uuid")
				return err
			},
			want: "step UUID must be a canonical UUID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.call()
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %q, want substring %q", err, tt.want)
			}
		})
	}
}

func TestCurrentUser(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(User{Username: "admin", Display: "Admin User"})
	})

	client := newTestClient(t, handler)
	user, err := client.CurrentUser(context.Background())
	if err != nil {
		t.Fatalf("CurrentUser: %v", err)
	}
	if user.Username != "admin" {
		t.Fatalf("expected admin, got %q", user.Username)
	}
}

func TestCurrentUserPreservesVersionedBasePath(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2.0/user" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(User{Username: "admin", Display: "Admin User"})
	})

	client := newTestClientWithBasePath(t, "/2.0", handler)
	user, err := client.CurrentUser(context.Background())
	if err != nil {
		t.Fatalf("CurrentUser: %v", err)
	}
	if user.Username != "admin" {
		t.Fatalf("expected admin, got %q", user.Username)
	}
}

func TestPingPreservesVersionedBasePath(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/2.0/user" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	})

	client := newTestClientWithBasePath(t, "/2.0", handler)
	if err := client.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestNewWithExplicitAuthMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("Authorization = %q, want Bearer prefix", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(Options{
		BaseURL:    server.URL,
		Token:      "my-token",
		AuthMethod: "bearer",
		Retry:      httpx.RetryPolicy{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, _ := client.HTTP().NewRequest(context.Background(), "GET", "/test", nil)
	_ = client.HTTP().Do(req, nil)
}

func TestNewWithTokenRefresherImpliesBearer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("Authorization = %q, want Bearer prefix", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(Options{
		BaseURL: server.URL,
		Token:   "oauth-token",
		Retry:   httpx.RetryPolicy{MaxAttempts: 1},
		TokenRefresher: func(ctx context.Context) (string, error) {
			return "refreshed", nil
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, _ := client.HTTP().NewRequest(context.Background(), "GET", "/test", nil)
	_ = client.HTTP().Do(req, nil)
}

func TestNewExplicitAuthMethodOverridesRefresherDefault(t *testing.T) {
	// When AuthMethod is explicitly set, it should be used even if TokenRefresher is set.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("Authorization = %q, want Bearer prefix", auth)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(Options{
		BaseURL:    server.URL,
		Token:      "tok",
		AuthMethod: "bearer",
		Retry:      httpx.RetryPolicy{MaxAttempts: 1},
		TokenRefresher: func(ctx context.Context) (string, error) {
			return "refreshed", nil
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, _ := client.HTTP().NewRequest(context.Background(), "GET", "/test", nil)
	_ = client.HTTP().Do(req, nil)
}

func TestNewWithoutAuthMethodDefaultsToBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected basic auth")
		}
		if user != "user@example.com" || pass != "app-password" {
			t.Errorf("basic auth = (%q, %q)", user, pass)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := New(Options{
		BaseURL:  server.URL,
		Username: "user@example.com",
		Token:    "app-password",
		Retry:    httpx.RetryPolicy{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, _ := client.HTTP().NewRequest(context.Background(), "GET", "/test", nil)
	_ = client.HTTP().Do(req, nil)
}

func TestListPipelinesRequiresParams(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	_, err := client.ListPipelines(context.Background(), "", "repo", 0)
	if err == nil {
		t.Fatal("expected error for empty workspace")
	}
	_, err = client.ListPipelines(context.Background(), "ws", "", 0)
	if err == nil {
		t.Fatal("expected error for empty repo slug")
	}
}

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
			json.NewEncoder(w).Encode(repositoryListPage{
				Values: []Repository{{Slug: "repo1"}, {Slug: "repo2"}},
				Next:   serverURL + "/repositories/ws?pagelen=20&page=2",
			})
		case 2:
			json.NewEncoder(w).Encode(repositoryListPage{
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
		json.NewEncoder(w).Encode(repositoryListPage{
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
		json.NewEncoder(w).Encode(Repository{Slug: "my-repo", Name: "My Repo"})
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
		json.NewDecoder(r.Body).Decode(&body)
		if body["scm"] != "git" {
			t.Errorf("expected scm=git, got %v", body["scm"])
		}
		if body["is_private"] != true {
			t.Errorf("expected is_private=true, got %v", body["is_private"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Repository{Slug: "new-repo", Name: "New Repo"})
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
		json.NewDecoder(r.Body).Decode(&body)

		target := body["target"].(map[string]any)
		if target["ref_name"] != "main" {
			t.Errorf("expected ref_name=main, got %v", target["ref_name"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Pipeline{UUID: "{abc-123}"})
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
		json.NewDecoder(r.Body).Decode(&body)

		vars, ok := body["variables"].([]any)
		if !ok || len(vars) == 0 {
			t.Fatal("expected variables in body")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Pipeline{UUID: "{abc}"})
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

func TestGetPipelineTrimsBraces(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify braces were stripped from the UUID in the URL path
		if strings.Contains(r.URL.Path, "{") || strings.Contains(r.URL.Path, "}") {
			t.Errorf("expected braces to be trimmed, got path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Pipeline{UUID: "{abc-123}"})
	})

	client := newTestClient(t, handler)
	pipeline, err := client.GetPipeline(context.Background(), "ws", "repo", "{abc-123}")
	if err != nil {
		t.Fatalf("GetPipeline: %v", err)
	}
	if pipeline.UUID != "{abc-123}" {
		t.Fatalf("expected UUID preserved in response, got %q", pipeline.UUID)
	}
}

func TestListPipelineStepsTrimsBraces(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "{") || strings.Contains(r.URL.Path, "}") {
			t.Errorf("expected braces to be trimmed, got path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{"uuid": "{step-1}", "name": "Build"},
			},
		})
	})

	client := newTestClient(t, handler)
	steps, err := client.ListPipelineSteps(context.Background(), "ws", "repo", "{pipeline-uuid}")
	if err != nil {
		t.Fatalf("ListPipelineSteps: %v", err)
	}
	if len(steps) != 1 || steps[0].Name != "Build" {
		t.Fatalf("unexpected steps: %+v", steps)
	}
}

func TestGetPipelineLogsTrimsBraces(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "{") || strings.Contains(r.URL.Path, "}") {
			t.Errorf("expected braces to be trimmed, got path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("build output here"))
	})

	client := newTestClient(t, handler)
	logs, err := client.GetPipelineLogs(context.Background(), "ws", "repo", "{pipeline-uuid}", "{step-uuid}")
	if err != nil {
		t.Fatalf("GetPipelineLogs: %v", err)
	}
	if string(logs) != "build output here" {
		t.Fatalf("expected log content, got %q", string(logs))
	}
}

func TestCurrentUser(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(User{Username: "admin", Display: "Admin User"})
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

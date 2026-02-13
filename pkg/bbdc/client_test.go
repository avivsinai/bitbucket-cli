package bbdc

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

func TestNewRequiresBaseURL(t *testing.T) {
	_, err := New(Options{})
	if err == nil {
		t.Fatal("expected error for empty base URL")
	}
}

func TestListRepositoriesPaginates(t *testing.T) {
	var hits int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		if !strings.Contains(r.URL.Path, "/projects/PROJ/repos") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		switch count {
		case 1:
			json.NewEncoder(w).Encode(paged[Repository]{
				Values:        []Repository{{Slug: "repo1"}},
				IsLastPage:    false,
				NextPageStart: 1,
				Limit:         1,
			})
		case 2:
			json.NewEncoder(w).Encode(paged[Repository]{
				Values:     []Repository{{Slug: "repo2"}},
				IsLastPage: true,
				Limit:      1,
			})
		default:
			t.Errorf("unexpected request %d", count)
		}
	})

	client := newTestClient(t, handler)
	repos, err := client.ListRepositories(context.Background(), "PROJ", 0)
	if err != nil {
		t.Fatalf("ListRepositories: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].Slug != "repo1" || repos[1].Slug != "repo2" {
		t.Fatalf("unexpected slugs: %v, %v", repos[0].Slug, repos[1].Slug)
	}
	if hits != 2 {
		t.Fatalf("expected 2 requests, got %d", hits)
	}
}

func TestListRepositoriesRespectsLimit(t *testing.T) {
	var hits int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(paged[Repository]{
			Values:        []Repository{{Slug: "repo1"}, {Slug: "repo2"}, {Slug: "repo3"}},
			IsLastPage:    false,
			NextPageStart: 3,
		})
	})

	client := newTestClient(t, handler)
	repos, err := client.ListRepositories(context.Background(), "PROJ", 2)
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

func TestListRepositoriesRequiresProjectKey(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	_, err := client.ListRepositories(context.Background(), "", 10)
	if err == nil {
		t.Fatal("expected error for empty project key")
	}
}

func TestGetRepository(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/projects/PROJ/repos/my-repo") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Repository{Slug: "my-repo", Name: "My Repo", ID: 42})
	})

	client := newTestClient(t, handler)
	repo, err := client.GetRepository(context.Background(), "PROJ", "my-repo")
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}
	if repo.Slug != "my-repo" || repo.ID != 42 {
		t.Fatalf("unexpected repo: %+v", repo)
	}
}

func TestGetRepositoryRequiresParams(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	_, err := client.GetRepository(context.Background(), "", "repo")
	if err == nil {
		t.Fatal("expected error for empty project key")
	}
	_, err = client.GetRepository(context.Background(), "PROJ", "")
	if err == nil {
		t.Fatal("expected error for empty repo slug")
	}
}

func TestGetPullRequest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/pull-requests/42") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PullRequest{ID: 42, Title: "Fix bug", State: "OPEN"})
	})

	client := newTestClient(t, handler)
	pr, err := client.GetPullRequest(context.Background(), "PROJ", "repo", 42)
	if err != nil {
		t.Fatalf("GetPullRequest: %v", err)
	}
	if pr.ID != 42 || pr.Title != "Fix bug" || pr.State != "OPEN" {
		t.Fatalf("unexpected PR: %+v", pr)
	}
}

func TestListPullRequestsWithStateFilter(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != "MERGED" {
			t.Errorf("expected state=MERGED, got %q", r.URL.Query().Get("state"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(paged[PullRequest]{
			Values:     []PullRequest{{ID: 1, State: "MERGED"}, {ID: 2, State: "MERGED"}},
			IsLastPage: true,
		})
	})

	client := newTestClient(t, handler)
	prs, err := client.ListPullRequests(context.Background(), "PROJ", "repo", "merged", 0)
	if err != nil {
		t.Fatalf("ListPullRequests: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}
}

func TestListPullRequestsPaginates(t *testing.T) {
	var hits int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		switch count {
		case 1:
			json.NewEncoder(w).Encode(paged[PullRequest]{
				Values:        []PullRequest{{ID: 1}},
				IsLastPage:    false,
				NextPageStart: 1,
			})
		case 2:
			json.NewEncoder(w).Encode(paged[PullRequest]{
				Values:     []PullRequest{{ID: 2}},
				IsLastPage: true,
			})
		}
	})

	client := newTestClient(t, handler)
	prs, err := client.ListPullRequests(context.Background(), "PROJ", "repo", "", 0)
	if err != nil {
		t.Fatalf("ListPullRequests: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}
	if hits != 2 {
		t.Fatalf("expected 2 requests, got %d", hits)
	}
}

func TestCommitStatuses(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/rest/build-status/1.0/commits/abc123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{"state": "SUCCESSFUL", "key": "build-1", "name": "CI"},
			},
		})
	})

	client := newTestClient(t, handler)
	statuses, err := client.CommitStatuses(context.Background(), "abc123")
	if err != nil {
		t.Fatalf("CommitStatuses: %v", err)
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].State != "SUCCESSFUL" {
		t.Fatalf("expected SUCCESSFUL, got %q", statuses[0].State)
	}
}

func TestCommitStatusesRequiresSHA(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	_, err := client.CommitStatuses(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty SHA")
	}
}

func TestCurrentUser(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/rest/api/1.0/users/admin") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(User{Name: "admin", Slug: "admin", ID: 1})
	})

	client := newTestClient(t, handler)
	user, err := client.CurrentUser(context.Background(), "admin")
	if err != nil {
		t.Fatalf("CurrentUser: %v", err)
	}
	if user.Name != "admin" || user.ID != 1 {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestClientSendsBasicAuth(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != "token123" {
			t.Errorf("expected basic auth admin:token123, got %s:%s (ok=%v)", user, pass, ok)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(User{Name: "admin"})
	})

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := New(Options{
		BaseURL:  server.URL,
		Username: "admin",
		Token:    "token123",
		Retry:    httpx.RetryPolicy{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = client.CurrentUser(context.Background(), "admin")
	if err != nil {
		t.Fatalf("CurrentUser: %v", err)
	}
}

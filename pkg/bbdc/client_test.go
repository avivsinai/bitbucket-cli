package bbdc

import (
	"context"
	"encoding/json"
	"errors"
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

func TestNewRequestAddsNoCheckHeaderToMutatingMethods(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req, err := client.HTTP().NewRequest(context.Background(), method, "/rest/api/1.0/projects", nil)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			if got := req.Header.Get("X-Atlassian-Token"); got != "no-check" {
				t.Fatalf("X-Atlassian-Token = %q, want no-check", got)
			}
		})
	}

	req, err := client.HTTP().NewRequest(context.Background(), http.MethodGet, "/rest/api/1.0/projects", nil)
	if err != nil {
		t.Fatalf("NewRequest GET: %v", err)
	}
	if got := req.Header.Get("X-Atlassian-Token"); got != "" {
		t.Fatalf("GET X-Atlassian-Token = %q, want empty", got)
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
			_ = json.NewEncoder(w).Encode(paged[Repository]{
				Values:        []Repository{{Slug: "repo1"}},
				IsLastPage:    false,
				NextPageStart: 1,
				Limit:         1,
			})
		case 2:
			_ = json.NewEncoder(w).Encode(paged[Repository]{
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

func TestListRepositoriesPageReturnsContinuation(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/projects/PROJ/repos" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "2" {
			t.Fatalf("limit = %q, want 2", got)
		}
		if got := r.URL.Query().Get("start"); got != "7" {
			t.Fatalf("start = %q, want 7", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(paged[Repository]{
			Values:        []Repository{{Slug: "repo1"}, {Slug: "repo2"}},
			IsLastPage:    false,
			NextPageStart: 9,
		})
	}))

	page, err := client.ListRepositoriesPage(context.Background(), "PROJ", 2, 7)
	if err != nil {
		t.Fatalf("ListRepositoriesPage: %v", err)
	}
	if len(page.Values) != 2 || page.Values[0].Slug != "repo1" {
		t.Fatalf("values = %+v", page.Values)
	}
	if page.IsLast || page.NextStart != 9 {
		t.Fatalf("continuation = isLast:%v next:%d, want false/9", page.IsLast, page.NextStart)
	}
}

func TestListRepositoriesPagePreservesEmptyNonFinalContinuation(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(paged[Repository]{
			Values:        []Repository{},
			IsLastPage:    false,
			NextPageStart: 9,
		})
	}))

	page, err := client.ListRepositoriesPage(context.Background(), "PROJ", 2, 7)
	if err != nil {
		t.Fatalf("ListRepositoriesPage: %v", err)
	}
	if page.IsLast || page.NextStart != 9 {
		t.Fatalf("empty continuation = isLast:%v next:%d, want false/9", page.IsLast, page.NextStart)
	}
}

func TestListRepositoriesRespectsLimit(t *testing.T) {
	var hits int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(paged[Repository]{
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
		_ = json.NewEncoder(w).Encode(Repository{Slug: "my-repo", Name: "My Repo", ID: 42})
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
		_ = json.NewEncoder(w).Encode(PullRequest{ID: 42, Title: "Fix bug", State: "OPEN"})
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
		_ = json.NewEncoder(w).Encode(paged[PullRequest]{
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
			_ = json.NewEncoder(w).Encode(paged[PullRequest]{
				Values:        []PullRequest{{ID: 1}},
				IsLastPage:    false,
				NextPageStart: 1,
			})
		case 2:
			_ = json.NewEncoder(w).Encode(paged[PullRequest]{
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
		_ = json.NewEncoder(w).Encode(map[string]any{
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

func TestCommitStatusesPagePreservesUpstreamPagination(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/build-status/1.0/commits/abc123" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "100" || r.URL.Query().Get("start") != "25" {
			t.Fatalf("query = %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values":        []map[string]any{{"state": "SUCCESSFUL", "key": "ci"}},
			"isLastPage":    false,
			"nextPageStart": 125,
		})
	}))

	page, err := client.CommitStatusesPage(context.Background(), "abc123", 100, 25)
	if err != nil {
		t.Fatalf("CommitStatusesPage: %v", err)
	}
	if len(page.Values) != 1 || page.Values[0].Key != "ci" || page.IsLast || page.NextStart != 125 {
		t.Fatalf("page = %+v", page)
	}
}

func TestCurrentUser(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/rest/api/1.0/users/admin") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(User{Name: "admin", Slug: "admin", ID: 1})
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
		_ = json.NewEncoder(w).Encode(User{Name: "admin"})
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

func TestResolveUserFallsBackToSearchForLoginName(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/1.0/users/jane@corp.example":
			http.NotFound(w, r)
		case "/rest/api/1.0/users":
			if got := r.URL.Query().Get("filter"); got != "jane@corp.example" {
				t.Errorf("filter = %q", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values": []User{
					{Name: "jane@corp.example.old", Slug: "jane_corp.example.old", ID: 1},
					{Name: "jane@corp.example", Slug: "jane_corp.example", ID: 2},
				},
				"isLastPage": true,
			})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	user, err := newTestClient(t, handler).ResolveUser(context.Background(), "jane@corp.example")
	if err != nil {
		t.Fatalf("ResolveUser: %v", err)
	}
	if user.ID != 2 || user.Slug != "jane_corp.example" {
		t.Fatalf("unexpected user: %+v", user)
	}
}

func TestResolveUserReturnsNotFoundWithoutExactMatch(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/api/1.0/users" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"values":     []User{{Name: "janet", Slug: "janet", ID: 3}},
				"isLastPage": true,
			})
			return
		}
		http.NotFound(w, r)
	})

	_, err := newTestClient(t, handler).ResolveUser(context.Background(), "jane")
	var httpErr *httpx.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusNotFound {
		t.Fatalf("err = %v, want 404 HTTPError", err)
	}
}

func TestResolveUserDoesNotSearchOnUnauthorized(t *testing.T) {
	var calls int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, err := newTestClient(t, handler).ResolveUser(context.Background(), "jane")
	var httpErr *httpx.HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("err = %v, want 401 HTTPError", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestUserMatches(t *testing.T) {
	user := User{Name: "jane@corp.example", Slug: "jane_corp.example", Email: "Jane.Doe@corp.example"}
	tests := []struct {
		identifier string
		want       bool
	}{
		{"jane@corp.example", true},
		{"JANE_CORP.EXAMPLE", true},
		{"jane.doe@corp.example", true},
		{" jane@corp.example ", true},
		{"jane", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := user.Matches(tt.identifier); got != tt.want {
			t.Errorf("Matches(%q) = %v, want %v", tt.identifier, got, tt.want)
		}
	}
}

func TestResolveUserSearchPagesAndIdentity(t *testing.T) {
	const identifier = "jane@corp.example"
	login := User{Name: identifier, Slug: "jane_corp.example", ID: 2}
	email := User{Name: "jane.doe", Slug: "jane.doe", Email: identifier, ID: 3}
	otherEmail := User{Name: "jane.old", Slug: "jane.old", Email: identifier, ID: 4}
	partial := User{Name: identifier + ".old", Slug: "jane_corp.example.old", ID: 5}
	tests := []struct {
		name      string
		pages     [][]User
		wantID    int
		wantError string
	}{
		{name: "login on later page", pages: [][]User{{partial}, {login}}, wantID: login.ID},
		{name: "email on later page", pages: [][]User{{partial}, {email}}, wantID: email.ID},
		{name: "login takes precedence over email", pages: [][]User{{email}, {login}}, wantID: login.ID},
		{name: "login takes precedence within page", pages: [][]User{{email, login}}, wantID: login.ID},
		{name: "unique email", pages: [][]User{{email}}, wantID: email.ID},
		{name: "ambiguous email", pages: [][]User{{email}, {otherEmail}}, wantError: "multiple users"},
		{name: "no exact match", pages: [][]User{{partial}, {{Name: "janet", ID: 6}}}, wantError: "404"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			searchCalls := 0
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/rest/api/1.0/users/"+identifier {
					http.NotFound(w, r)
					return
				}
				if r.URL.Path != "/rest/api/1.0/users" {
					t.Errorf("unexpected path: %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				query := r.URL.Query()
				if query.Get("filter") != identifier || query.Get("limit") != "100" {
					t.Errorf("query = %q", r.URL.RawQuery)
				}
				page := 0
				if query.Get("start") == "17" {
					page = 1
				} else if query.Get("start") != "" && query.Get("start") != "0" {
					t.Errorf("unexpected start: %q", query.Get("start"))
				}
				searchCalls++
				if page >= len(tt.pages) || searchCalls > len(tt.pages) {
					t.Error("unexpected extra search request")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(paged[User]{
					Values: tt.pages[page], Limit: 1,
					IsLastPage: page == len(tt.pages)-1, NextPageStart: 17,
				})
			})
			user, err := newTestClient(t, handler).ResolveUser(context.Background(), identifier)
			if tt.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantError) {
					t.Fatalf("user = %+v, err = %v, want error containing %q", user, err, tt.wantError)
				}
			} else if err != nil || user.ID != tt.wantID {
				t.Fatalf("user = %+v, err = %v, want ID %d", user, err, tt.wantID)
			}
			if searchCalls != len(tt.pages) {
				t.Errorf("search calls = %d, want %d", searchCalls, len(tt.pages))
			}
		})
	}
}

func TestResolveUserDirectLookup(t *testing.T) {
	var calls int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/rest/api/1.0/users/jane_corp.example" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(User{Name: "jane_corp.example", Slug: "jane_corp.example", ID: 2})
	})
	user, err := newTestClient(t, handler).ResolveUser(context.Background(), " jane_corp.example ")
	if err != nil || user.ID != 2 || calls != 1 {
		t.Fatalf("user = %+v, err = %v, calls = %d", user, err, calls)
	}
}

func TestResolveUserRejectsLoginSlugCollision(t *testing.T) {
	const identifier = "jane_corp.example"
	slugUser := User{Name: "jane@corp.example", Slug: identifier, ID: 1}
	loginUser := User{Name: identifier, Slug: identifier + "-2", ID: 2}
	tests := []struct {
		name   string
		direct bool
		pages  [][]User
		wantID int
	}{
		{name: "direct slug and exact login", direct: true, pages: [][]User{{slugUser, loginUser}}},
		{name: "direct slug and login on later page", direct: true, pages: [][]User{{slugUser}, {loginUser}}},
		{name: "search slug before login", pages: [][]User{{slugUser, loginUser}}},
		{name: "search slug before later login", pages: [][]User{{slugUser}, {loginUser}}},
		{name: "search login before slug", pages: [][]User{{loginUser, slugUser}}},
		{name: "search login before later slug", pages: [][]User{{loginUser}, {slugUser}}},
		{name: "direct slug without exact login", direct: true, pages: [][]User{{}}, wantID: 1},
		{name: "search slug without exact login", pages: [][]User{{slugUser}}, wantID: 1},
		{name: "login and slug identify same user", pages: [][]User{{{Name: identifier, Slug: identifier, ID: 2}}}, wantID: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var searches int
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/rest/api/1.0/users/" + identifier:
					if tt.direct {
						_ = json.NewEncoder(w).Encode(slugUser)
					} else {
						http.NotFound(w, r)
					}
				case "/rest/api/1.0/users":
					page := searches
					searches++
					if page >= len(tt.pages) {
						t.Error("unexpected extra search request")
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					query := r.URL.Query()
					if query.Get("filter") != identifier || (page == 0 && query.Get("start") != "0") || (page == 1 && query.Get("start") != "17") {
						t.Errorf("page %d query = %q", page, r.URL.RawQuery)
					}
					_ = json.NewEncoder(w).Encode(paged[User]{
						Values: tt.pages[page], IsLastPage: page == len(tt.pages)-1, NextPageStart: 17,
					})
				default:
					t.Errorf("unexpected path: %s", r.URL.Path)
					http.NotFound(w, r)
				}
			})
			user, err := newTestClient(t, handler).ResolveUser(context.Background(), identifier)
			if tt.wantID == 0 {
				if err == nil || !strings.Contains(err.Error(), "ambiguous") || user != nil {
					t.Fatalf("user = %+v, err = %v, want ambiguity error", user, err)
				}
			} else if err != nil || user.ID != tt.wantID {
				t.Fatalf("user = %+v, err = %v, want ID %d", user, err, tt.wantID)
			}
			if searches != len(tt.pages) {
				t.Errorf("searches = %d, want %d", searches, len(tt.pages))
			}
		})
	}
}

func TestResolveUserSearchDoesNotRepeatPage(t *testing.T) {
	var searches int
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/1.0/users" {
			http.NotFound(w, r)
			return
		}
		searches++
		if searches > 1 {
			t.Error("repeated the same search page")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(paged[User]{Values: []User{{Name: "janet", ID: 3}}})
	})
	_, err := newTestClient(t, handler).ResolveUser(context.Background(), "jane")
	if err == nil || !strings.Contains(err.Error(), "non-advancing") || searches != 1 {
		t.Fatalf("err = %v, searches = %d", err, searches)
	}
}

func TestResolveUserReturnsSearchError(t *testing.T) {
	for name, directSlug := range map[string]bool{"direct not found": false, "direct slug": true} {
		t.Run(name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/rest/api/1.0/users" {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				if directSlug {
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode(User{Name: "jane@corp.example", Slug: "jane", ID: 1})
					return
				}
				http.NotFound(w, r)
			})
			_, err := newTestClient(t, handler).ResolveUser(context.Background(), "jane")
			var httpErr *httpx.HTTPError
			if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusForbidden {
				t.Fatalf("err = %v, want search's 403 HTTPError", err)
			}
		})
	}
}

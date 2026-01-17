package smoke_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/root"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func TestRepoListDataCenterTextOutput(t *testing.T) {
	mock := newBitbucketMock(t, "admin", "admin123")
	defer mock.Close()

	mock.StubRepoList("BKT", 5, repoListResponse{
		Values: []repoPayload{{
			Slug: "sample",
			Name: "Sample Repo",
			ID:   42,
			Project: projectRef{
				Key: "BKT",
			},
			WebLinks: []string{mock.URL() + "/projects/BKT/repos/sample"},
			CloneLinks: []cloneLink{
				{Name: "ssh", Href: "ssh://git@bitbucket.example.com/BKT/sample.git"},
				{Name: "http", Href: "https://bitbucket.example.com/scm/bkt/sample.git"},
			},
		}},
	})

	cfg := configForMock(mock.URL(), "admin", "admin123", "smoke", "bkt", "")

	stdout, stderr, err := runCLI(t, cfg, "repo", "list", "--limit", "5")
	if err != nil {
		t.Fatalf("repo list returned error: %v (stderr=%s)", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr output, got %q", stderr)
	}

	got := stdout
	if !strings.Contains(got, "BKT/sample\tSample Repo") {
		t.Fatalf("expected repo summary in output, got:\n%s", got)
	}
	if !strings.Contains(got, "web:   "+mock.URL()+"/projects/BKT/repos/sample") {
		t.Fatalf("expected web link in output, got:\n%s", got)
	}
	if !strings.Contains(got, "clone: ssh://git@bitbucket.example.com/BKT/sample.git (ssh), https://bitbucket.example.com/scm/bkt/sample.git (http)") {
		t.Fatalf("expected clone links in output, got:\n%s", got)
	}
	if calls := mock.RepoListCalls(); calls != 1 {
		t.Fatalf("expected single repo list request, got %d", calls)
	}
}

func TestRepoListDataCenterJSONOutput(t *testing.T) {
	mock := newBitbucketMock(t, "svc", "token")
	defer mock.Close()

	mock.StubRepoList("BKT", 7, repoListResponse{
		Values: []repoPayload{{
			Slug: "sample",
			Name: "Sample Repo",
			ID:   7,
			Project: projectRef{
				Key: "BKT",
			},
			WebLinks: []string{mock.URL() + "/projects/BKT/repos/sample"},
			CloneLinks: []cloneLink{
				{Name: "ssh", Href: "ssh://git@bitbucket.example.com/BKT/sample.git"},
			},
		}},
	})

	cfg := configForMock(mock.URL(), "svc", "token", "json", "bkt", "")

	stdout, stderr, err := runCLI(t, cfg, "repo", "list", "--limit", "7", "--json")
	if err != nil {
		t.Fatalf("repo list --json error: %v (stderr=%s)", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr output, got %q", stderr)
	}

	var payload struct {
		Project string `json:"project"`
		Repos   []struct {
			Project string   `json:"project"`
			Slug    string   `json:"slug"`
			Name    string   `json:"name"`
			ID      int      `json:"id"`
			Clone   []string `json:"clone_urls"`
		} `json:"repositories"`
	}
	if err := json.Unmarshal([]byte(stdout), &payload); err != nil {
		t.Fatalf("failed to decode JSON output: %v\nstdout=%s", err, stdout)
	}
	if payload.Project != "BKT" {
		t.Fatalf("expected project BKT, got %q", payload.Project)
	}
	if len(payload.Repos) != 1 {
		t.Fatalf("expected one repo, got %d", len(payload.Repos))
	}
	repo := payload.Repos[0]
	if repo.Slug != "sample" || repo.Project != "BKT" || repo.Name != "Sample Repo" {
		t.Fatalf("unexpected repo payload: %+v", repo)
	}
	if len(repo.Clone) != 1 || repo.Clone[0] != "ssh://git@bitbucket.example.com/BKT/sample.git (ssh)" {
		t.Fatalf("unexpected clone URLs: %v", repo.Clone)
	}
	if calls := mock.RepoListCalls(); calls != 1 {
		t.Fatalf("expected single repo list request, got %d", calls)
	}
}

func TestProjectListDataCenter(t *testing.T) {
	mock := newBitbucketMock(t, "svc", "token")
	defer mock.Close()

	mock.StubProjectList(9, projectListResponse{
		Values: []projectPayload{{
			Key:         "bkt",
			Name:        "CLI Fixtures",
			ID:          11,
			Type:        "NORMAL",
			Description: "fixture project",
			Public:      true,
		}},
	})

	cfg := configForMock(mock.URL(), "svc", "token", "projects", "bkt", "")

	stdout, stderr, err := runCLI(t, cfg, "project", "list", "--limit", "9")
	if err != nil {
		t.Fatalf("project list error: %v (stderr=%s)", err, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr output, got %q", stderr)
	}

	if !strings.Contains(stdout, "Projects on "+mock.URL()) {
		t.Fatalf("expected heading with base URL, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "BKT\tCLI Fixtures") {
		t.Fatalf("expected uppercased key in output, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "link: "+mock.URL()+"/projects/BKT") {
		t.Fatalf("expected project link in output, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "visibility: public") {
		t.Fatalf("expected visibility indicator, got:\n%s", stdout)
	}
	if calls := mock.ProjectListCalls(); calls != 1 {
		t.Fatalf("expected single project list request, got %d", calls)
	}
}

func runCLI(t *testing.T, cfg *config.Config, args ...string) (string, string, error) {
	t.Helper()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ios := &iostreams.IOStreams{
		In:     io.NopCloser(bytes.NewReader(nil)),
		Out:    stdout,
		ErrOut: stderr,
	}

	factory := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      ios,
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	rootCmd, err := root.NewCmdRoot(factory)
	if err != nil {
		t.Fatalf("NewCmdRoot: %v", err)
	}
	rootCmd.SetArgs(args)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	t.Setenv("BKT_NO_UPDATE_CHECK", "1")
	t.Setenv("NO_COLOR", "1")

	err = rootCmd.ExecuteContext(context.Background())
	return stdout.String(), stderr.String(), err
}

func configForMock(baseURL, username, token, contextName, projectKey, workspace string) *config.Config {
	return &config.Config{
		ActiveContext: contextName,
		Contexts: map[string]*config.Context{
			contextName: {
				Host:       "mock",
				ProjectKey: projectKey,
				Workspace:  workspace,
			},
		},
		Hosts: map[string]*config.Host{
			"mock": {
				Kind:     "dc",
				BaseURL:  baseURL,
				Username: username,
				Token:    token,
			},
		},
	}
}

type bitbucketMock struct {
	t            *testing.T
	server       *httptest.Server
	expectedAuth string
	repoStub     repoStub
	projectStub  projectStub
	repoCalls    atomic.Int32
	projectCalls atomic.Int32
}

type repoStub struct {
	enabled       bool
	projectKey    string
	expectedLimit int
	response      repoListResponse
}

type projectStub struct {
	enabled       bool
	expectedLimit int
	response      projectListResponse
}

func newBitbucketMock(t *testing.T, username, password string) *bitbucketMock {
	t.Helper()
	m := &bitbucketMock{
		t:            t,
		expectedAuth: "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password)),
	}
	m.server = httptest.NewServer(http.HandlerFunc(m.dispatch))
	return m
}

func (m *bitbucketMock) Close() {
	m.server.Close()
}

func (m *bitbucketMock) URL() string {
	return m.server.URL
}

func (m *bitbucketMock) StubRepoList(projectKey string, limit int, resp repoListResponse) {
	m.repoStub = repoStub{
		enabled:       true,
		projectKey:    projectKey,
		expectedLimit: limit,
		response:      resp,
	}
}

func (m *bitbucketMock) StubProjectList(limit int, resp projectListResponse) {
	m.projectStub = projectStub{
		enabled:       true,
		expectedLimit: limit,
		response:      resp,
	}
}

func (m *bitbucketMock) RepoListCalls() int {
	return int(m.repoCalls.Load())
}

func (m *bitbucketMock) ProjectListCalls() int {
	return int(m.projectCalls.Load())
}

func (m *bitbucketMock) dispatch(w http.ResponseWriter, r *http.Request) {
	switch {
	case strings.HasPrefix(r.URL.Path, "/rest/api/1.0/projects/") && strings.HasSuffix(r.URL.Path, "/repos"):
		m.handleRepoList(w, r)
	case r.URL.Path == "/rest/api/1.0/projects":
		m.handleProjectList(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (m *bitbucketMock) handleRepoList(w http.ResponseWriter, r *http.Request) {
	if !m.repoStub.enabled {
		http.NotFound(w, r)
		return
	}
	m.repoCalls.Add(1)

	if err := m.assertAuth(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
		return
	}

	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	var projectKey string
	for i := 0; i < len(segments); i++ {
		if segments[i] == "projects" && i+1 < len(segments) {
			projectKey = segments[i+1]
			break
		}
	}
	if projectKey == "" {
		http.Error(w, "unexpected path", http.StatusBadRequest)
		return
	}
	if projectKey != m.repoStub.projectKey {
		http.Error(w, fmt.Sprintf("expected project %s, got %s", m.repoStub.projectKey, projectKey), http.StatusBadRequest)
		return
	}

	query := r.URL.Query()
	if limit := query.Get("limit"); limit != fmt.Sprint(m.repoStub.expectedLimit) {
		http.Error(w, fmt.Sprintf("expected limit=%d, got %s", m.repoStub.expectedLimit, limit), http.StatusBadRequest)
		return
	}

	m.writeJSON(w, m.repoStub.response.page(m.repoStub.expectedLimit))
}

func (m *bitbucketMock) handleProjectList(w http.ResponseWriter, r *http.Request) {
	if !m.projectStub.enabled {
		http.NotFound(w, r)
		return
	}
	m.projectCalls.Add(1)

	if err := m.assertAuth(r); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "unsupported method", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query()
	if limit := query.Get("limit"); limit != fmt.Sprint(m.projectStub.expectedLimit) {
		http.Error(w, fmt.Sprintf("expected limit=%d, got %s", m.projectStub.expectedLimit, limit), http.StatusBadRequest)
		return
	}

	m.writeJSON(w, m.projectStub.response.page(m.projectStub.expectedLimit))
}

func (m *bitbucketMock) assertAuth(r *http.Request) error {
	if got := r.Header.Get("Authorization"); got != m.expectedAuth {
		return fmt.Errorf("unexpected auth header: got %q", got)
	}
	return nil
}

func (m *bitbucketMock) writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		m.t.Fatalf("encode mock response: %v", err)
	}
}

type cloneLink struct {
	Name string `json:"name"`
	Href string `json:"href"`
}

type projectRef struct {
	Key string `json:"key"`
}

type repoPayload struct {
	Slug       string      `json:"slug"`
	Name       string      `json:"name"`
	ID         int         `json:"id"`
	Project    projectRef  `json:"project"`
	WebLinks   []string    `json:"web_links"`
	CloneLinks []cloneLink `json:"clone_links"`
}

type repoListResponse struct {
	Values []repoPayload
}

func (r repoListResponse) page(limit int) map[string]any {
	var values []map[string]any
	for _, repo := range r.Values {
		links := map[string]any{
			"web":   toWebLinks(repo.WebLinks),
			"clone": repo.CloneLinks,
		}
		values = append(values, map[string]any{
			"slug":    repo.Slug,
			"name":    repo.Name,
			"id":      repo.ID,
			"project": map[string]any{"key": repo.Project.Key},
			"links":   links,
		})
	}

	return map[string]any{
		"size":          len(values),
		"limit":         limit,
		"isLastPage":    true,
		"start":         0,
		"nextPageStart": 0,
		"values":        values,
	}
}

func toWebLinks(urls []string) []map[string]string {
	var out []map[string]string
	for _, u := range urls {
		out = append(out, map[string]string{"href": u})
	}
	return out
}

type projectPayload struct {
	Key         string `json:"key"`
	Name        string `json:"name"`
	ID          int    `json:"id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Public      bool   `json:"public"`
}

type projectListResponse struct {
	Values []projectPayload
}

func (p projectListResponse) page(limit int) map[string]any {
	var values []map[string]any
	for _, proj := range p.Values {
		values = append(values, map[string]any{
			"key":         proj.Key,
			"name":        proj.Name,
			"id":          proj.ID,
			"type":        proj.Type,
			"description": proj.Description,
			"public":      proj.Public,
		})
	}

	return map[string]any{
		"size":          len(values),
		"limit":         limit,
		"isLastPage":    true,
		"start":         0,
		"nextPageStart": 0,
		"values":        values,
	}
}

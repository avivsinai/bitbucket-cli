package repo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func TestSelectCloneURLDCPrefersHTTPS(t *testing.T) {
	var r bbdc.Repository
	r.Links.Clone = []struct {
		Href string `json:"href"`
		Name string `json:"name"`
	}{
		{Href: "ssh://git@bitbucket.example.com:7999/PROJ/repo.git", Name: "ssh"},
		{Href: "https://bitbucket.example.com/scm/PROJ/repo.git", Name: "https"},
	}

	got, err := selectCloneURLDC(r, false)
	if err != nil {
		t.Fatalf("selectCloneURLDC returned error: %v", err)
	}
	if got != "https://bitbucket.example.com/scm/PROJ/repo.git" {
		t.Fatalf("got %q, want https link", got)
	}
}

func TestSelectCloneURLDCHttpAlias(t *testing.T) {
	var r bbdc.Repository
	r.Links.Clone = []struct {
		Href string `json:"href"`
		Name string `json:"name"`
	}{
		{Href: "http://bitbucket.example.com/scm/PROJ/repo.git", Name: "http"},
	}

	got, err := selectCloneURLDC(r, false)
	if err != nil {
		t.Fatalf("selectCloneURLDC returned error: %v", err)
	}
	if got != "http://bitbucket.example.com/scm/PROJ/repo.git" {
		t.Fatalf("got %q, want http link", got)
	}
}

func TestSelectCloneURLDCSsh(t *testing.T) {
	var r bbdc.Repository
	r.Links.Clone = []struct {
		Href string `json:"href"`
		Name string `json:"name"`
	}{
		{Href: "ssh://git@bitbucket.example.com:7999/PROJ/repo.git", Name: "ssh"},
		{Href: "https://bitbucket.example.com/scm/PROJ/repo.git", Name: "https"},
	}

	got, err := selectCloneURLDC(r, true)
	if err != nil {
		t.Fatalf("selectCloneURLDC returned error: %v", err)
	}
	if !strings.HasPrefix(got, "ssh://") {
		t.Fatalf("got %q, want ssh link", got)
	}
}

func TestSelectCloneURLDCMissing(t *testing.T) {
	var r bbdc.Repository
	r.Links.Clone = []struct {
		Href string `json:"href"`
		Name string `json:"name"`
	}{
		{Href: "https://bitbucket.example.com/scm/PROJ/repo.git", Name: "https"},
	}

	_, err := selectCloneURLDC(r, true)
	if err == nil {
		t.Fatalf("expected error when ssh clone missing")
	}
}

func TestBrowseWithoutRepoDefaults(t *testing.T) {
	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {
				Host:       "main",
				ProjectKey: "dev",
			},
		},
		Hosts: map[string]*config.Host{
			"main": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
				Token:   "test-token",
			},
		},
	}

	var stdout, stderr strings.Builder
	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams: &iostreams.IOStreams{
			Out:    &stdout,
			ErrOut: &stderr,
		},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	cmd := newBrowseCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error when repo not provided")
	}
	if !strings.Contains(err.Error(), "repository required") {
		t.Fatalf("expected error to mention repository requirement, got %q", err.Error())
	}
}

func TestDefaultReviewersListDataCenterRequiresRefs(t *testing.T) {
	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "repo"},
		},
		Hosts: map[string]*config.Host{
			"main": {Kind: "dc", BaseURL: "https://bitbucket.example.com", Token: "test-token"},
		},
	}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: &strings.Builder{}, ErrOut: &strings.Builder{}},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	cmd := newDefaultReviewersListCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when Data Center source/target refs are missing")
	}
	if !strings.Contains(err.Error(), "data center default reviewers require --source and --target refs") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDefaultReviewersListDataCenterUsesEffectiveReviewersEndpoint(t *testing.T) {
	var sawReviewersRequest bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/rest/api/1.0/projects/PROJ/repos/repo":
			_ = json.NewEncoder(w).Encode(bbdc.Repository{Slug: "repo", ID: 123})
		case "/rest/default-reviewers/1.0/projects/PROJ/repos/repo/reviewers":
			sawReviewersRequest = true
			query := r.URL.Query()
			if got := query.Get("sourceRepoId"); got != "123" {
				t.Fatalf("sourceRepoId = %q, want 123", got)
			}
			if got := query.Get("targetRepoId"); got != "123" {
				t.Fatalf("targetRepoId = %q, want 123", got)
			}
			if got := query.Get("sourceRefId"); got != "feature/auth" {
				t.Fatalf("sourceRefId = %q, want feature/auth", got)
			}
			if got := query.Get("targetRefId"); got != "main" {
				t.Fatalf("targetRefId = %q, want main", got)
			}
			_ = json.NewEncoder(w).Encode([]bbdc.User{
				{Name: "alice", FullName: "Alice", ID: 7},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "main", ProjectKey: "PROJ", DefaultRepo: "repo"},
		},
		Hosts: map[string]*config.Host{
			"main": {Kind: "dc", BaseURL: server.URL, Token: "test-token"},
		},
	}

	stdout := &strings.Builder{}
	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: &strings.Builder{}},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	cmd := newDefaultReviewersListCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--source", "refs/heads/feature/auth", "--target", "main"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sawReviewersRequest {
		t.Fatal("expected default reviewers request")
	}
	if !strings.Contains(stdout.String(), "Alice") || !strings.Contains(stdout.String(), "alice") {
		t.Fatalf("expected reviewer output, got %q", stdout.String())
	}
}

func TestDefaultReviewersListCloudRejectsRefFlags(t *testing.T) {
	cfg := &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "main", Workspace: "ws", DefaultRepo: "repo"},
		},
		Hosts: map[string]*config.Host{
			"main": {Kind: "cloud", BaseURL: "https://api.bitbucket.org/2.0", Token: "test-token"},
		},
	}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: &strings.Builder{}, ErrOut: &strings.Builder{}},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}

	cmd := newDefaultReviewersListCmd(f)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs([]string{"--source", "feature", "--target", "main"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when Cloud source/target refs are provided")
	}
	if !strings.Contains(err.Error(), "--source and --target are only supported for Data Center") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepoCreateRejectsHostNoOpFlags(t *testing.T) {
	tests := []struct {
		name          string
		hostKind      string
		args          []string
		errorContains string
	}{
		{
			name:          "cloud rejects project",
			hostKind:      "cloud",
			args:          []string{"service", "--project", "DATA"},
			errorContains: "--project is not supported for Cloud repo create",
		},
		{
			name:          "cloud rejects forkable",
			hostKind:      "cloud",
			args:          []string{"service", "--forkable"},
			errorContains: "--forkable is not supported for Cloud repo create",
		},
		{
			name:          "cloud rejects default branch",
			hostKind:      "cloud",
			args:          []string{"service", "--default-branch", "main"},
			errorContains: "--default-branch is not supported for Cloud repo create",
		},
		{
			name:          "cloud rejects explicit scm",
			hostKind:      "cloud",
			args:          []string{"service", "--scm", "git"},
			errorContains: "--scm is not supported for Cloud repo create",
		},
		{
			name:          "data center rejects workspace",
			hostKind:      "dc",
			args:          []string{"service", "--workspace", "team"},
			errorContains: "--workspace is not supported for Data Center repo create",
		},
		{
			name:          "data center rejects cloud project",
			hostKind:      "dc",
			args:          []string{"service", "--cloud-project", "WEB"},
			errorContains: "--cloud-project is not supported for Data Center repo create",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hits int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits++
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{
					"slug": "service",
					"name": "service",
					"project": map[string]any{
						"key": "PROJ",
					},
				})
			}))
			t.Cleanup(server.Close)

			cfg := &config.Config{
				ActiveContext: "default",
				Contexts: map[string]*config.Context{
					"default": {
						Host:        "main",
						ProjectKey:  "PROJ",
						Workspace:   "workspace",
						DefaultRepo: "repo",
					},
				},
				Hosts: map[string]*config.Host{
					"main": {
						Kind:    tt.hostKind,
						BaseURL: server.URL,
						Token:   "test-token",
					},
				},
			}

			var stdout, stderr strings.Builder
			f := &cmdutil.Factory{
				AppVersion:     "test",
				ExecutableName: "bkt",
				IOStreams: &iostreams.IOStreams{
					Out:    &stdout,
					ErrOut: &stderr,
				},
				Config: func() (*config.Config, error) {
					return cfg, nil
				},
			}

			cmd := newCreateCmd(f)
			cmd.SetContext(context.Background())
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errorContains)
			}
			if !strings.Contains(err.Error(), tt.errorContains) {
				t.Fatalf("error = %q, want substring %q", err, tt.errorContains)
			}
			if hits != 0 {
				t.Fatalf("expected validation to avoid HTTP requests, got %d", hits)
			}
		})
	}
}

func TestRepoListDataCenterMarksArchivedRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/rest/api/1.0/projects/PROJ/repos" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"isLastPage": true,
			"values": []map[string]any{
				{
					"slug":     "active",
					"name":     "Active Repo",
					"id":       1,
					"archived": false,
					"project":  map[string]any{"key": "PROJ"},
				},
				{
					"slug":     "legacy",
					"name":     "Legacy Repo",
					"id":       2,
					"archived": true,
					"project":  map[string]any{"key": "PROJ"},
				},
			},
		})
	}))
	t.Cleanup(server.Close)

	cfg := dcRepoConfig(server.URL)
	stdout := &strings.Builder{}
	cmd := newListCmd(repoTestFactory(cfg, stdout))
	cmd.SetContext(context.Background())
	cmd.PersistentFlags().Bool("json", false, "")
	cmd.SetArgs([]string{"--limit", "25"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("repo list: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "PROJ/active\tActive Repo\n") {
		t.Fatalf("expected active repo row, got %q", out)
	}
	if strings.Contains(out, "PROJ/active\tActive Repo (archived)") {
		t.Fatalf("did not expect archived marker on active repo, got %q", out)
	}
	if !strings.Contains(out, "PROJ/legacy\tLegacy Repo (archived)") {
		t.Fatalf("expected archived suffix, got %q", out)
	}

	stdout.Reset()
	cmd = newListCmd(repoTestFactory(cfg, stdout))
	cmd.SetContext(context.Background())
	cmd.PersistentFlags().Bool("json", false, "")
	cmd.SetArgs([]string{"--limit", "25", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("repo list --json: %v", err)
	}
	var payload struct {
		Repos []struct {
			Slug     string `json:"slug"`
			Archived bool   `json:"archived"`
		} `json:"repositories"`
	}
	if err := json.Unmarshal([]byte(stdout.String()), &payload); err != nil {
		t.Fatalf("decode list JSON: %v\n%s", err, stdout.String())
	}
	if len(payload.Repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(payload.Repos))
	}
	if payload.Repos[0].Slug != "active" || payload.Repos[0].Archived {
		t.Fatalf("unexpected first repo: %+v", payload.Repos[0])
	}
	if payload.Repos[1].Slug != "legacy" || !payload.Repos[1].Archived {
		t.Fatalf("unexpected second repo: %+v", payload.Repos[1])
	}
	if !strings.Contains(stdout.String(), `"archived": false`) {
		t.Fatalf("expected archived=false in JSON, got %s", stdout.String())
	}
}

func TestRepoViewDataCenterArchivedOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/rest/api/1.0/projects/PROJ/repos/sample" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"slug":     "sample",
			"name":     "Sample Repo",
			"id":       42,
			"archived": true,
			"project":  map[string]any{"key": "PROJ"},
		})
	}))
	t.Cleanup(server.Close)

	cfg := dcRepoConfig(server.URL)
	stdout := &strings.Builder{}
	cmd := newViewCmd(repoTestFactory(cfg, stdout))
	cmd.SetContext(context.Background())
	cmd.PersistentFlags().Bool("json", false, "")
	cmd.SetArgs([]string{"sample"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("repo view: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "PROJ/sample (42)") {
		t.Fatalf("expected heading, got %q", out)
	}
	if !strings.Contains(out, "Archived: true") {
		t.Fatalf("expected Archived line, got %q", out)
	}

	stdout.Reset()
	cmd = newViewCmd(repoTestFactory(cfg, stdout))
	cmd.SetContext(context.Background())
	cmd.PersistentFlags().Bool("json", false, "")
	cmd.SetArgs([]string{"sample", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("repo view --json: %v", err)
	}
	var payload struct {
		Slug     string `json:"slug"`
		Archived bool   `json:"archived"`
	}
	if err := json.Unmarshal([]byte(stdout.String()), &payload); err != nil {
		t.Fatalf("decode view JSON: %v\n%s", err, stdout.String())
	}
	if payload.Slug != "sample" || !payload.Archived {
		t.Fatalf("unexpected view payload: %+v", payload)
	}
	if !strings.Contains(stdout.String(), `"archived": true`) {
		t.Fatalf("expected archived=true in JSON, got %s", stdout.String())
	}
}

func dcRepoConfig(baseURL string) *config.Config {
	return &config.Config{
		ActiveContext: "default",
		Contexts: map[string]*config.Context{
			"default": {Host: "main", ProjectKey: "PROJ"},
		},
		Hosts: map[string]*config.Host{
			"main": {Kind: "dc", BaseURL: baseURL, Token: "test-token"},
		},
	}
}

func repoTestFactory(cfg *config.Config, stdout *strings.Builder) *cmdutil.Factory {
	return &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      &iostreams.IOStreams{Out: stdout, ErrOut: &strings.Builder{}},
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}
}

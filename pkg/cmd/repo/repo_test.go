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

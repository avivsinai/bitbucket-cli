package repo

import (
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

package repo

import (
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
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

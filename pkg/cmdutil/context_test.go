package cmdutil

import (
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
)

func newTestFactory(cfg *config.Config) *Factory {
	return &Factory{
		ExecutableName: "bkt",
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}
}

func TestResolveHostWithHostKey(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
				Token:   "test-token",
			},
		},
	}
	f := newTestFactory(cfg)

	key, host, err := ResolveHost(f, "", "bitbucket.example.com")
	if err != nil {
		t.Fatalf("ResolveHost returned error: %v", err)
	}
	if key != "bitbucket.example.com" {
		t.Fatalf("key = %q, want bitbucket.example.com", key)
	}
	if host == nil || host.BaseURL != "https://bitbucket.example.com" {
		t.Fatalf("unexpected host: %#v", host)
	}
}

func TestResolveHostWithHostURL(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
				Token:   "test-token",
			},
		},
	}
	f := newTestFactory(cfg)

	key, host, err := ResolveHost(f, "", "https://bitbucket.example.com")
	if err != nil {
		t.Fatalf("ResolveHost returned error: %v", err)
	}
	if key != "bitbucket.example.com" {
		t.Fatalf("key = %q, want bitbucket.example.com", key)
	}
	if host == nil || host.BaseURL != "https://bitbucket.example.com" {
		t.Fatalf("unexpected host: %#v", host)
	}
}

func TestResolveHostWithContext(t *testing.T) {
	cfg := &config.Config{
		ActiveContext: "dev",
		Contexts: map[string]*config.Context{
			"dev": {
				Host: "bitbucket.example.com",
			},
		},
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
				Token:   "test-token",
			},
		},
	}
	f := newTestFactory(cfg)

	key, host, err := ResolveHost(f, "", "")
	if err != nil {
		t.Fatalf("ResolveHost returned error: %v", err)
	}
	if key != "bitbucket.example.com" {
		t.Fatalf("key = %q, want bitbucket.example.com", key)
	}
	if host == nil || host.BaseURL != "https://bitbucket.example.com" {
		t.Fatalf("unexpected host: %#v", host)
	}
}

func TestResolveHostSingleHostFallback(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
				Token:   "test-token",
			},
		},
	}
	f := newTestFactory(cfg)

	key, host, err := ResolveHost(f, "", "")
	if err != nil {
		t.Fatalf("ResolveHost returned error: %v", err)
	}
	if key != "bitbucket.example.com" {
		t.Fatalf("key = %q, want bitbucket.example.com", key)
	}
	if host == nil || host.BaseURL != "https://bitbucket.example.com" {
		t.Fatalf("unexpected host: %#v", host)
	}
}

func TestResolveHostMultipleHostsError(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]*config.Host{
			"one.example.com": {
				Kind:    "dc",
				BaseURL: "https://one.example.com",
				Token:   "test-token",
			},
			"two.example.com": {
				Kind:    "dc",
				BaseURL: "https://two.example.com",
				Token:   "test-token",
			},
		},
	}
	f := newTestFactory(cfg)

	_, _, err := ResolveHost(f, "", "")
	if err == nil {
		t.Fatalf("expected error for multiple hosts")
	}
	if !strings.Contains(err.Error(), "multiple hosts") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveHostNoHostsError(t *testing.T) {
	cfg := &config.Config{
		Hosts: map[string]*config.Host{},
	}
	f := newTestFactory(cfg)

	_, _, err := ResolveHost(f, "", "")
	if err == nil {
		t.Fatalf("expected error when no hosts configured")
	}
	if !strings.Contains(err.Error(), "no hosts configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMarshalYAMLStripsToken(t *testing.T) {
	h := &Host{Kind: "dc", BaseURL: "https://example.com", Token: "secret-token", Username: "admin"}
	data, err := yaml.Marshal(h)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(data), "secret-token") {
		t.Fatalf("token was not stripped during marshaling: %s", data)
	}
	if !strings.Contains(string(data), "admin") {
		t.Fatalf("expected username to be preserved: %s", data)
	}
}

func TestMarshalYAMLNilHost(t *testing.T) {
	var h *Host
	got, err := h.MarshalYAML()
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %v", got)
	}
}

func TestLoadReturnsDefaultsWhenFileDoesNotExist(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BKT_CONFIG_DIR", dir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Version != currentVersion {
		t.Fatalf("expected version %d, got %d", currentVersion, cfg.Version)
	}
	if cfg.Contexts == nil {
		t.Fatalf("expected Contexts map to be initialized")
	}
	if cfg.Hosts == nil {
		t.Fatalf("expected Hosts map to be initialized")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BKT_CONFIG_DIR", dir)

	cfg := &Config{
		ActiveContext: "dev",
		Contexts: map[string]*Context{
			"dev": {Host: "main", ProjectKey: "PROJ"},
		},
		Hosts: map[string]*Host{
			"main": {Kind: "dc", BaseURL: "https://bitbucket.example.com", Username: "admin", Token: "secret"},
		},
		path: filepath.Join(dir, "config.yml"),
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify the file exists and has restrictive permissions.
	info, err := os.Stat(cfg.path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("expected 0600 permissions, got %o", perm)
	}

	// Verify token is NOT written to disk.
	data, err := os.ReadFile(cfg.path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), "secret") {
		t.Fatalf("token was written to disk: %s", data)
	}

	// Load back and verify structure.
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.ActiveContext != "dev" {
		t.Fatalf("active context = %q, want %q", loaded.ActiveContext, "dev")
	}
	ctx, err := loaded.Context("dev")
	if err != nil {
		t.Fatalf("Context: %v", err)
	}
	if ctx.ProjectKey != "PROJ" {
		t.Fatalf("project key = %q, want %q", ctx.ProjectKey, "PROJ")
	}
	h, err := loaded.Host("main")
	if err != nil {
		t.Fatalf("Host: %v", err)
	}
	if h.Kind != "dc" {
		t.Fatalf("kind = %q, want %q", h.Kind, "dc")
	}
	// Token should be empty because MarshalYAML strips it.
	if h.Token != "" {
		t.Fatalf("token should be empty after round-trip, got %q", h.Token)
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "dir")
	cfg := &Config{
		Hosts: map[string]*Host{},
		path:  filepath.Join(dir, "config.yml"),
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(cfg.path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}
}

func TestSaveSetsVersionWhenZero(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Version: 0,
		Hosts:   map[string]*Host{},
		path:    filepath.Join(dir, "config.yml"),
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(cfg.path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), "version: 1") {
		t.Fatalf("expected version to be set, got: %s", data)
	}
}

func TestContextCRUD(t *testing.T) {
	cfg := &Config{Contexts: make(map[string]*Context)}

	// Set
	cfg.SetContext("dev", &Context{Host: "main", ProjectKey: "PROJ"})
	ctx, err := cfg.Context("dev")
	if err != nil {
		t.Fatalf("Context: %v", err)
	}
	if ctx.Host != "main" {
		t.Fatalf("host = %q, want %q", ctx.Host, "main")
	}

	// Missing context
	_, err = cfg.Context("missing")
	if err != ErrContextNotFound {
		t.Fatalf("expected ErrContextNotFound, got %v", err)
	}

	// Delete
	cfg.ActiveContext = "dev"
	cfg.DeleteContext("dev")
	_, err = cfg.Context("dev")
	if err != ErrContextNotFound {
		t.Fatalf("expected ErrContextNotFound after delete, got %v", err)
	}
	if cfg.ActiveContext != "" {
		t.Fatalf("active context should be cleared after deleting active, got %q", cfg.ActiveContext)
	}
}

func TestSetContextNilMap(t *testing.T) {
	cfg := &Config{}
	cfg.SetContext("test", &Context{Host: "h"})
	ctx, err := cfg.Context("test")
	if err != nil {
		t.Fatalf("Context: %v", err)
	}
	if ctx.Host != "h" {
		t.Fatalf("host = %q, want %q", ctx.Host, "h")
	}
}

func TestSetActiveContext(t *testing.T) {
	cfg := &Config{
		Contexts: map[string]*Context{
			"dev": {Host: "main"},
		},
	}

	if err := cfg.SetActiveContext("dev"); err != nil {
		t.Fatalf("SetActiveContext: %v", err)
	}
	if cfg.ActiveContext != "dev" {
		t.Fatalf("active = %q, want %q", cfg.ActiveContext, "dev")
	}

	// Non-existent context
	if err := cfg.SetActiveContext("missing"); err != ErrContextNotFound {
		t.Fatalf("expected ErrContextNotFound, got %v", err)
	}

	// Empty clears
	if err := cfg.SetActiveContext(""); err != nil {
		t.Fatalf("SetActiveContext empty: %v", err)
	}
	if cfg.ActiveContext != "" {
		t.Fatalf("expected empty active context, got %q", cfg.ActiveContext)
	}
}

func TestHostCRUD(t *testing.T) {
	cfg := &Config{Hosts: make(map[string]*Host)}

	cfg.SetHost("bb", &Host{Kind: "dc", BaseURL: "https://bb.example.com"})
	h, err := cfg.Host("bb")
	if err != nil {
		t.Fatalf("Host: %v", err)
	}
	if h.BaseURL != "https://bb.example.com" {
		t.Fatalf("base_url = %q", h.BaseURL)
	}

	// Missing host
	_, err = cfg.Host("missing")
	if err != ErrHostNotFound {
		t.Fatalf("expected ErrHostNotFound, got %v", err)
	}

	// Delete
	cfg.DeleteHost("bb")
	_, err = cfg.Host("bb")
	if err != ErrHostNotFound {
		t.Fatalf("expected ErrHostNotFound after delete, got %v", err)
	}
}

func TestSetHostNilMap(t *testing.T) {
	cfg := &Config{}
	cfg.SetHost("test", &Host{Kind: "cloud"})
	h, err := cfg.Host("test")
	if err != nil {
		t.Fatalf("Host: %v", err)
	}
	if h.Kind != "cloud" {
		t.Fatalf("kind = %q, want %q", h.Kind, "cloud")
	}
}

func TestResolvePathUsesEnvVar(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BKT_CONFIG_DIR", dir)

	path, err := resolvePath()
	if err != nil {
		t.Fatalf("resolvePath: %v", err)
	}
	if path != filepath.Join(dir, "config.yml") {
		t.Fatalf("path = %q, want %q", path, filepath.Join(dir, "config.yml"))
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BKT_CONFIG_DIR", dir)

	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte("{{invalid yaml"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "decode config") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadInitializesNilMaps(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("BKT_CONFIG_DIR", dir)

	// Write a minimal config with no contexts or hosts keys.
	if err := os.WriteFile(filepath.Join(dir, "config.yml"), []byte("version: 1\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Contexts == nil {
		t.Fatal("expected Contexts to be initialized")
	}
	if cfg.Hosts == nil {
		t.Fatal("expected Hosts to be initialized")
	}
}

func TestDeleteContextDoesNotClearUnrelatedActiveContext(t *testing.T) {
	cfg := &Config{
		ActiveContext: "prod",
		Contexts: map[string]*Context{
			"dev":  {Host: "main"},
			"prod": {Host: "main"},
		},
	}

	cfg.DeleteContext("dev")
	if cfg.ActiveContext != "prod" {
		t.Fatalf("active context should remain %q, got %q", "prod", cfg.ActiveContext)
	}
}

func TestPath(t *testing.T) {
	cfg := &Config{path: "/tmp/test/config.yml"}
	if got := cfg.Path(); got != "/tmp/test/config.yml" {
		t.Fatalf("Path() = %q, want %q", got, "/tmp/test/config.yml")
	}
}

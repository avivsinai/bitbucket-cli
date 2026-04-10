package context

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// newTestFactory wires a Factory around the given config and returns
// buffers for stdout/stderr.
func newTestFactory(cfg *config.Config) (*cmdutil.Factory, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ios := &iostreams.IOStreams{
		In:     io.NopCloser(bytes.NewReader(nil)),
		Out:    stdout,
		ErrOut: stderr,
	}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      ios,
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}
	return f, stdout, stderr
}

// runContextCmd wires NewCmdContext beneath a fake root carrying a
// --context/--output persistent flag and executes the given args.
func runContextCmd(t *testing.T, f *cmdutil.Factory, args ...string) error {
	t.Helper()

	cmd := NewCmdContext(f)
	cmd.PersistentFlags().String("context", "", "Named context to use")
	cmd.PersistentFlags().String("output", "text", "Output format")
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	cmd.SetArgs(args)
	cmd.SetOut(f.IOStreams.Out)
	cmd.SetErr(f.IOStreams.ErrOut)

	return cmd.ExecuteContext(context.Background())
}

// setupTempConfigDir redirects BKT_CONFIG_DIR to a scratch dir so
// cfg.Save() writes to the sandbox rather than the host system.
func setupTempConfigDir(t *testing.T) {
	t.Helper()
	t.Setenv("BKT_CONFIG_DIR", t.TempDir())
}

func seedConfig() *config.Config {
	return &config.Config{
		Version: 1,
		Hosts: map[string]*config.Host{
			"bitbucket.example.com": {
				Kind:    "dc",
				BaseURL: "https://bitbucket.example.com",
			},
			"api.bitbucket.org": {
				Kind:    "cloud",
				BaseURL: "https://api.bitbucket.org/2.0",
			},
		},
		Contexts: map[string]*config.Context{},
	}
}

// ---------- create ----------

func TestContextCreateDC(t *testing.T) {
	setupTempConfigDir(t)
	cfg := seedConfig()

	f, stdout, stderr := newTestFactory(cfg)
	err := runContextCmd(t, f, "create", "work", "--host", "bitbucket.example.com", "--project", "team", "--repo", "api")
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr.String())
	}

	ctx, ok := cfg.Contexts["work"]
	if !ok {
		t.Fatal("expected context 'work' to be created")
	}
	if ctx.Host != "bitbucket.example.com" {
		t.Errorf("host = %q, want bitbucket.example.com", ctx.Host)
	}
	if ctx.ProjectKey != "TEAM" {
		t.Errorf("projectKey = %q, want TEAM (uppercased)", ctx.ProjectKey)
	}
	if ctx.DefaultRepo != "api" {
		t.Errorf("defaultRepo = %q, want api", ctx.DefaultRepo)
	}
	// First context created becomes active automatically.
	if cfg.ActiveContext != "work" {
		t.Errorf("expected work to auto-activate (first context), got %q", cfg.ActiveContext)
	}
	if !strings.Contains(stdout.String(), "Created context \"work\"") {
		t.Errorf("expected created message, got: %s", stdout.String())
	}
}

func TestContextCreateCloud(t *testing.T) {
	setupTempConfigDir(t)
	cfg := seedConfig()

	f, _, stderr := newTestFactory(cfg)
	err := runContextCmd(t, f, "create", "oss", "--host", "api.bitbucket.org", "--workspace", "my-team", "--repo", "cli")
	if err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr.String())
	}

	ctx := cfg.Contexts["oss"]
	if ctx == nil {
		t.Fatal("expected context 'oss' to be created")
	}
	if ctx.Workspace != "my-team" {
		t.Errorf("workspace = %q, want my-team", ctx.Workspace)
	}
	// Cloud contexts do NOT uppercase the workspace.
	if ctx.ProjectKey != "" {
		t.Errorf("projectKey should be empty for cloud, got %q", ctx.ProjectKey)
	}
}

func TestContextCreateRequiresHost(t *testing.T) {
	setupTempConfigDir(t)
	f, _, _ := newTestFactory(seedConfig())
	err := runContextCmd(t, f, "create", "work")
	if err == nil {
		t.Fatal("expected error when --host missing")
	}
	if !strings.Contains(err.Error(), "--host is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContextCreateRejectsUnknownHost(t *testing.T) {
	setupTempConfigDir(t)
	f, _, _ := newTestFactory(seedConfig())
	err := runContextCmd(t, f, "create", "work", "--host", "bitbucket.nonexistent.com", "--project", "TEAM")
	if err == nil {
		t.Fatal("expected error for unknown host")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContextCreateDCRequiresProject(t *testing.T) {
	setupTempConfigDir(t)
	f, _, _ := newTestFactory(seedConfig())
	err := runContextCmd(t, f, "create", "work", "--host", "bitbucket.example.com")
	if err == nil {
		t.Fatal("expected error when --project missing for DC")
	}
	if !strings.Contains(err.Error(), "--project is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContextCreateCloudRequiresWorkspace(t *testing.T) {
	setupTempConfigDir(t)
	f, _, _ := newTestFactory(seedConfig())
	err := runContextCmd(t, f, "create", "oss", "--host", "api.bitbucket.org")
	if err == nil {
		t.Fatal("expected error when --workspace missing for Cloud")
	}
	if !strings.Contains(err.Error(), "--workspace is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestContextCreateSetActive(t *testing.T) {
	setupTempConfigDir(t)
	cfg := seedConfig()
	cfg.Contexts["existing"] = &config.Context{Host: "bitbucket.example.com", ProjectKey: "OLD"}
	cfg.ActiveContext = "existing"

	f, stdout, _ := newTestFactory(cfg)
	err := runContextCmd(t, f, "create", "staging", "--host", "bitbucket.example.com", "--project", "stg", "--set-active")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ActiveContext != "staging" {
		t.Errorf("expected staging active, got %q", cfg.ActiveContext)
	}
	if !strings.Contains(stdout.String(), "staging\" is now active") {
		t.Errorf("expected active message, got: %s", stdout.String())
	}
}

func TestContextCreateSecondContextDoesNotAutoActivate(t *testing.T) {
	setupTempConfigDir(t)
	cfg := seedConfig()
	cfg.Contexts["existing"] = &config.Context{Host: "bitbucket.example.com", ProjectKey: "OLD"}
	cfg.ActiveContext = "existing"

	f, _, _ := newTestFactory(cfg)
	if err := runContextCmd(t, f, "create", "second", "--host", "bitbucket.example.com", "--project", "two"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Without --set-active, second context should NOT steal activation.
	if cfg.ActiveContext != "existing" {
		t.Errorf("expected existing to remain active, got %q", cfg.ActiveContext)
	}
}

// ---------- use ----------

func TestContextUse(t *testing.T) {
	setupTempConfigDir(t)
	cfg := seedConfig()
	cfg.Contexts["work"] = &config.Context{Host: "bitbucket.example.com", ProjectKey: "TEAM"}
	cfg.Contexts["personal"] = &config.Context{Host: "api.bitbucket.org", Workspace: "me"}
	cfg.ActiveContext = "work"

	f, stdout, _ := newTestFactory(cfg)
	if err := runContextCmd(t, f, "use", "personal"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ActiveContext != "personal" {
		t.Errorf("expected active personal, got %q", cfg.ActiveContext)
	}
	if !strings.Contains(stdout.String(), "Activated context \"personal\"") {
		t.Errorf("expected activation message, got: %s", stdout.String())
	}
}

func TestContextUseUnknown(t *testing.T) {
	setupTempConfigDir(t)
	cfg := seedConfig()
	cfg.Contexts["work"] = &config.Context{Host: "bitbucket.example.com", ProjectKey: "TEAM"}

	f, _, _ := newTestFactory(cfg)
	err := runContextCmd(t, f, "use", "ghost")
	if err == nil {
		t.Fatal("expected error for unknown context")
	}
}

// ---------- list ----------

func TestContextList(t *testing.T) {
	setupTempConfigDir(t)
	cfg := seedConfig()
	cfg.Contexts["work"] = &config.Context{
		Host:        "bitbucket.example.com",
		ProjectKey:  "TEAM",
		DefaultRepo: "api",
	}
	cfg.Contexts["personal"] = &config.Context{
		Host:      "api.bitbucket.org",
		Workspace: "me",
	}
	cfg.ActiveContext = "work"

	f, stdout, _ := newTestFactory(cfg)
	if err := runContextCmd(t, f, "list"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := stdout.String()
	// Active marker precedes the active context.
	if !strings.Contains(out, "* work (host: bitbucket.example.com)") {
		t.Errorf("expected active marker on work, got: %s", out)
	}
	if !strings.Contains(out, "  personal (host: api.bitbucket.org)") {
		t.Errorf("expected inactive row for personal, got: %s", out)
	}
	if !strings.Contains(out, "project: TEAM") {
		t.Errorf("expected project field, got: %s", out)
	}
	if !strings.Contains(out, "workspace: me") {
		t.Errorf("expected workspace field, got: %s", out)
	}
	if !strings.Contains(out, "repo: api") {
		t.Errorf("expected repo field, got: %s", out)
	}
}

func TestContextListEmpty(t *testing.T) {
	setupTempConfigDir(t)
	cfg := seedConfig()

	f, stdout, _ := newTestFactory(cfg)
	if err := runContextCmd(t, f, "list"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "No contexts configured") {
		t.Errorf("expected empty message, got: %s", stdout.String())
	}
}

// ---------- delete ----------

func TestContextDelete(t *testing.T) {
	setupTempConfigDir(t)
	cfg := seedConfig()
	cfg.Contexts["old"] = &config.Context{Host: "bitbucket.example.com", ProjectKey: "OLD"}
	cfg.Contexts["keep"] = &config.Context{Host: "bitbucket.example.com", ProjectKey: "KEEP"}
	cfg.ActiveContext = "keep"

	f, stdout, _ := newTestFactory(cfg)
	if err := runContextCmd(t, f, "delete", "old"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cfg.Contexts["old"]; ok {
		t.Error("expected 'old' to be deleted")
	}
	if _, ok := cfg.Contexts["keep"]; !ok {
		t.Error("expected 'keep' to remain")
	}
	if cfg.ActiveContext != "keep" {
		t.Errorf("active context should be unchanged, got %q", cfg.ActiveContext)
	}
	if !strings.Contains(stdout.String(), "Deleted context \"old\"") {
		t.Errorf("expected delete message, got: %s", stdout.String())
	}
}

func TestContextDeleteActiveClearsActive(t *testing.T) {
	setupTempConfigDir(t)
	cfg := seedConfig()
	cfg.Contexts["only"] = &config.Context{Host: "bitbucket.example.com", ProjectKey: "ONE"}
	cfg.ActiveContext = "only"

	f, _, _ := newTestFactory(cfg)
	if err := runContextCmd(t, f, "delete", "only"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ActiveContext != "" {
		t.Errorf("active context should be cleared after deleting active, got %q", cfg.ActiveContext)
	}
}

func TestContextDeleteUnknown(t *testing.T) {
	setupTempConfigDir(t)
	cfg := seedConfig()

	f, _, _ := newTestFactory(cfg)
	err := runContextCmd(t, f, "delete", "ghost")
	if err == nil {
		t.Fatal("expected error for unknown context")
	}
}

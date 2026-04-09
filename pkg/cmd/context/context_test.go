package context_test

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmd/root"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"

	gocontext "context"
)

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

	rootCmd.SilenceUsage = true

	err = rootCmd.ExecuteContext(gocontext.Background())
	return stdout.String(), stderr.String(), err
}

func TestContextCreateBootstrap(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		envToken     string
		cfg          *config.Config
		wantErr      string
		wantStdout   string
		wantStderr   string
		wantHostKind string
		wantHostKey  string
	}{
		{
			name:     "bootstrap dc with BKT_TOKEN",
			args:     []string{"context", "create", "work", "--host", "bitbucket.example.com", "--kind", "dc", "--project", "PROJ"},
			envToken: "test-token",
			cfg: &config.Config{
				Contexts: map[string]*config.Context{},
				Hosts:    map[string]*config.Host{},
			},
			wantStdout:   "Created context",
			wantStderr:   "Bootstrapped host",
			wantHostKind: "dc",
			wantHostKey:  "bitbucket.example.com",
		},
		{
			name:     "bootstrap cloud with BKT_TOKEN",
			args:     []string{"context", "create", "oss", "--host", "bitbucket.org", "--kind", "cloud", "--workspace", "myteam"},
			envToken: "test-token",
			cfg: &config.Config{
				Contexts: map[string]*config.Context{},
				Hosts:    map[string]*config.Host{},
			},
			wantStdout:   "Created context",
			wantStderr:   "Bootstrapped host",
			wantHostKind: "cloud",
			wantHostKey:  "bitbucket.org",
		},
		{
			name:     "bootstrap fails without --kind when BKT_TOKEN set",
			args:     []string{"context", "create", "work", "--host", "bitbucket.example.com", "--project", "PROJ"},
			envToken: "test-token",
			cfg: &config.Config{
				Contexts: map[string]*config.Context{},
				Hosts:    map[string]*config.Host{},
			},
			wantErr: "--kind is required when bootstrapping a new host via BKT_TOKEN",
		},
		{
			name:     "without BKT_TOKEN missing host returns auth login error",
			args:     []string{"context", "create", "work", "--host", "bitbucket.example.com", "--project", "PROJ"},
			envToken: "",
			cfg: &config.Config{
				Contexts: map[string]*config.Context{},
				Hosts:    map[string]*config.Host{},
			},
			wantErr: "run `bkt auth login` first",
		},
		{
			name:     "base-url overrides default URL",
			args:     []string{"context", "create", "work", "--host", "bitbucket.example.com", "--kind", "dc", "--project", "PROJ", "--base-url", "https://bb.internal:8443"},
			envToken: "test-token",
			cfg: &config.Config{
				Contexts: map[string]*config.Context{},
				Hosts:    map[string]*config.Host{},
			},
			wantStdout:   "Created context",
			wantStderr:   "Bootstrapped host",
			wantHostKind: "dc",
			wantHostKey:  "bb.internal:8443",
		},
		{
			name: "existing host works without bootstrap",
			args: []string{"context", "create", "work", "--host", "bitbucket.example.com", "--project", "PROJ"},
			cfg: &config.Config{
				Contexts: map[string]*config.Context{},
				Hosts: map[string]*config.Host{
					"bitbucket.example.com": {
						Kind:    "dc",
						BaseURL: "https://bitbucket.example.com",
					},
				},
			},
			wantStdout:   "Created context",
			wantHostKind: "dc",
			wantHostKey:  "bitbucket.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("BKT_TOKEN", tt.envToken)

			// Use a temp dir so Save() doesn't touch real config.
			cfgDir := t.TempDir()
			t.Setenv("BKT_CONFIG_DIR", cfgDir)

			stdout, stderr, err := runCLI(t, tt.cfg, tt.args...)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr)
			}

			if tt.wantStdout != "" && !strings.Contains(stdout, tt.wantStdout) {
				t.Errorf("stdout missing %q, got %q", tt.wantStdout, stdout)
			}

			if tt.wantStderr != "" && !strings.Contains(stderr, tt.wantStderr) {
				t.Errorf("stderr missing %q, got %q", tt.wantStderr, stderr)
			}

			if tt.wantHostKey != "" {
				host, ok := tt.cfg.Hosts[tt.wantHostKey]
				if !ok {
					t.Fatalf("expected host %q in config, hosts: %v", tt.wantHostKey, tt.cfg.Hosts)
				}
				if host.Kind != tt.wantHostKind {
					t.Errorf("expected host kind %q, got %q", tt.wantHostKind, host.Kind)
				}
			}
		})
	}
}

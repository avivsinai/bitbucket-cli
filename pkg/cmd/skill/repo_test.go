package skill

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// factoryWithHost returns a factory whose only configured context points at a
// host of the given kind, in an isolated config directory.
func factoryWithHost(t *testing.T, kind, baseURL, hostKey string) *cmdutil.Factory {
	t.Helper()
	f, _, _ := newTestFactory(t)

	cfg, err := f.ResolveConfig()
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	cfg.SetHost(hostKey, &config.Host{Kind: kind, BaseURL: baseURL, Username: "u", Token: "t"})
	ctxCfg := &config.Context{Host: hostKey}
	if kind == "cloud" {
		ctxCfg.Workspace = "myteam"
	} else {
		ctxCfg.ProjectKey = "PROJ"
	}
	cfg.SetContext("test", ctxCfg)
	if err := cfg.SetActiveContext("test"); err != nil {
		t.Fatalf("SetActiveContext: %v", err)
	}
	return f
}

// commandWithContextFlag mimics the root command's persistent --context flag,
// which openRepository reads.
func commandWithContextFlag() *cobra.Command {
	root := &cobra.Command{Use: "bkt"}
	root.PersistentFlags().StringP("context", "c", "", "")
	cmd := &cobra.Command{Use: "skill"}
	root.AddCommand(cmd)
	return cmd
}

func TestOpenRepositoryResolvesBothPlatforms(t *testing.T) {
	tests := []struct {
		name      string
		hostKind  string
		hostKey   string
		baseURL   string
		arg       string
		wantFull  string
		wantWeb   string
		wantClone string
	}{
		{
			name:      "cloud shorthand",
			hostKind:  "cloud",
			hostKey:   "bitbucket.org",
			baseURL:   "https://api.bitbucket.org/2.0",
			arg:       "myteam/agent-skills",
			wantFull:  "myteam/agent-skills",
			wantWeb:   "https://bitbucket.org/myteam/agent-skills",
			wantClone: "https://bitbucket.org/myteam/agent-skills.git",
		},
		{
			name:      "cloud url",
			hostKind:  "cloud",
			hostKey:   "bitbucket.org",
			baseURL:   "https://api.bitbucket.org/2.0",
			arg:       "https://bitbucket.org/myteam/agent-skills.git",
			wantFull:  "myteam/agent-skills",
			wantWeb:   "https://bitbucket.org/myteam/agent-skills",
			wantClone: "https://bitbucket.org/myteam/agent-skills.git",
		},
		{
			name:      "data center shorthand",
			hostKind:  "dc",
			hostKey:   "bitbucket.example.com",
			baseURL:   "https://bitbucket.example.com",
			arg:       "PROJ/agent-skills",
			wantFull:  "PROJ/agent-skills",
			wantWeb:   "https://bitbucket.example.com/projects/PROJ/repos/agent-skills",
			wantClone: "https://bitbucket.example.com/scm/proj/agent-skills.git",
		},
		{
			name:      "data center browse url",
			hostKind:  "dc",
			hostKey:   "bitbucket.example.com",
			baseURL:   "https://bitbucket.example.com",
			arg:       "https://bitbucket.example.com/projects/proj/repos/agent-skills/browse",
			wantFull:  "PROJ/agent-skills",
			wantWeb:   "https://bitbucket.example.com/projects/PROJ/repos/agent-skills",
			wantClone: "https://bitbucket.example.com/scm/proj/agent-skills.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := factoryWithHost(t, tt.hostKind, tt.baseURL, tt.hostKey)

			repo, err := openRepository(commandWithContextFlag(), f, tt.arg)
			if err != nil {
				t.Fatalf("openRepository(%q): %v", tt.arg, err)
			}
			if got := repo.FullName(); got != tt.wantFull {
				t.Errorf("FullName = %q, want %q", got, tt.wantFull)
			}
			if got := repo.WebURL(); got != tt.wantWeb {
				t.Errorf("WebURL = %q, want %q", got, tt.wantWeb)
			}
			if got := repo.CloneURL(); got != tt.wantClone {
				t.Errorf("CloneURL = %q, want %q", got, tt.wantClone)
			}
		})
	}
}

func TestOpenRepositoryErrors(t *testing.T) {
	tests := []struct {
		name     string
		hostKind string
		hostKey  string
		baseURL  string
		arg      string
		wantErr  string
	}{
		{
			name:     "malformed repository argument",
			hostKind: "cloud",
			hostKey:  "bitbucket.org",
			baseURL:  "https://api.bitbucket.org/2.0",
			arg:      "agent-skills",
			wantErr:  "expected WORKSPACE/REPO (Cloud), PROJECT/REPO (Data Center), or a Bitbucket URL",
		},
		{
			// A Cloud URL against a Data Center context is a mistake worth
			// naming rather than silently querying the wrong host.
			name:     "cloud url against a data center host",
			hostKind: "dc",
			hostKey:  "bitbucket.example.com",
			baseURL:  "https://bitbucket.example.com",
			arg:      "https://bitbucket.org/myteam/agent-skills",
			wantErr:  `is a Bitbucket Cloud URL, but bitbucket.org is not configured and the active context uses https://bitbucket.example.com (Bitbucket Data Center)`,
		},
		{
			name:     "unconfigured host of the same kind",
			hostKind: "dc",
			hostKey:  "bitbucket.example.com",
			baseURL:  "https://bitbucket.example.com",
			arg:      "https://bitbucket.other.com/projects/proj/repos/agent-skills",
			wantErr:  "names host bitbucket.other.com, which is not configured",
		},
		{
			name:     "unsupported host kind",
			hostKind: "server",
			hostKey:  "bitbucket.example.com",
			baseURL:  "https://bitbucket.example.com",
			arg:      "PROJ/agent-skills",
			wantErr:  `unsupported host kind "server"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := factoryWithHost(t, tt.hostKind, tt.baseURL, tt.hostKey)

			_, err := openRepository(commandWithContextFlag(), f, tt.arg)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestKindLabel(t *testing.T) {
	tests := []struct{ kind, want string }{
		{"cloud", "Cloud"},
		{"dc", "Data Center"},
		{"server", "server"},
	}
	for _, tt := range tests {
		if got := kindLabel(tt.kind); got != tt.want {
			t.Errorf("kindLabel(%q) = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

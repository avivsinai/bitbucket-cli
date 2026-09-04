package registry

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestFindByID(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		wantName string
		wantErr  string
	}{
		{name: "github-copilot", id: "github-copilot", wantName: "GitHub Copilot"},
		{name: "claude-code", id: "claude-code", wantName: "Claude Code"},
		{name: "universal default", id: DefaultAgentID, wantName: "Universal"},
		{name: "antigravity2.0 keeps dot in id", id: "antigravity2.0", wantName: "Antigravity 2.0"},
		{name: "unknown agent lists valid ids", id: "nonexistent", wantErr: `unknown agent "nonexistent", valid agents: github-copilot, claude-code`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, err := FindByID(tt.id)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("FindByID returned error: %v", err)
			}
			if host.Name != tt.wantName {
				t.Fatalf("Name = %q, want %q", host.Name, tt.wantName)
			}
		})
	}
}

func TestAgentsTableIsUniqueAndComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, h := range Agents {
		if h.ID == "" || h.Name == "" || h.ProjectDir == "" || h.UserDir == "" {
			t.Errorf("agent %+v has an empty field", h)
		}
		if seen[h.ID] {
			t.Errorf("duplicate agent id %q", h.ID)
		}
		seen[h.ID] = true
	}
	if len(Agents) != 48 {
		t.Fatalf("expected 48 agents (parity with gh 2.100), got %d", len(Agents))
	}
	if !strings.Contains(AgentHelpList(), "  - Claude Code (claude-code)") {
		t.Fatalf("AgentHelpList missing claude-code line:\n%s", AgentHelpList())
	}
}

func TestInstallDir(t *testing.T) {
	t.Setenv(claudeConfigDirEnv, "")
	t.Setenv(piCodingAgentDirEnv, "")

	gitRoot := filepath.Join("tmp", "monalisa-repo")
	home := filepath.Join("home", "monalisa")

	tests := []struct {
		name    string
		setup   func(*testing.T)
		hostID  string
		scope   Scope
		gitRoot string
		homeDir string
		wantDir string
		wantErr string
	}{
		{name: "copilot project scope uses shared dir", hostID: "github-copilot", scope: ScopeProject, gitRoot: gitRoot, homeDir: home, wantDir: filepath.Join(gitRoot, ".agents", "skills")},
		{name: "copilot user scope", hostID: "github-copilot", scope: ScopeUser, gitRoot: gitRoot, homeDir: home, wantDir: filepath.Join(home, ".copilot", "skills")},
		{name: "claude code project scope", hostID: "claude-code", scope: ScopeProject, gitRoot: gitRoot, homeDir: home, wantDir: filepath.Join(gitRoot, ".claude", "skills")},
		{name: "claude code user scope", hostID: "claude-code", scope: ScopeUser, gitRoot: gitRoot, homeDir: home, wantDir: filepath.Join(home, ".claude", "skills")},
		{
			name:    "claude code user scope respects CLAUDE_CONFIG_DIR",
			setup:   func(t *testing.T) { t.Setenv(claudeConfigDirEnv, filepath.Join("cfg", "claude")) },
			hostID:  "claude-code",
			scope:   ScopeUser,
			gitRoot: gitRoot,
			homeDir: home,
			wantDir: filepath.Join("cfg", "claude", "skills"),
		},
		{
			name:    "pi user scope respects PI_CODING_AGENT_DIR",
			setup:   func(t *testing.T) { t.Setenv(piCodingAgentDirEnv, filepath.Join("cfg", "pi")) },
			hostID:  "pi",
			scope:   ScopeUser,
			gitRoot: gitRoot,
			homeDir: home,
			wantDir: filepath.Join("cfg", "pi", "skills"),
		},
		{name: "universal user scope uses shared dir", hostID: "universal", scope: ScopeUser, gitRoot: gitRoot, homeDir: home, wantDir: filepath.Join(home, ".agents", "skills")},
		{name: "project scope without git root", hostID: "universal", scope: ScopeProject, gitRoot: "", homeDir: home, wantErr: "could not determine project root directory"},
		{name: "user scope without home dir", hostID: "universal", scope: ScopeUser, gitRoot: gitRoot, homeDir: "", wantErr: "could not determine home directory"},
		{name: "invalid scope", hostID: "universal", scope: "bogus", gitRoot: gitRoot, homeDir: home, wantErr: `invalid scope "bogus"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}
			host, err := FindByID(tt.hostID)
			if err != nil {
				t.Fatalf("FindByID: %v", err)
			}
			dir, err := host.InstallDir(tt.scope, tt.gitRoot, tt.homeDir)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("InstallDir returned error: %v", err)
			}
			if dir != tt.wantDir {
				t.Fatalf("InstallDir = %q, want %q", dir, tt.wantDir)
			}
		})
	}
}

func TestSharedProjectDirGroupsAgents(t *testing.T) {
	// Agents that share a project directory install one copy between them, so
	// the shared directory must be reused rather than duplicated per agent.
	byDir := map[string][]string{}
	for _, h := range Agents {
		byDir[h.ProjectDir] = append(byDir[h.ProjectDir], h.ID)
	}
	shared := byDir[".agents/skills"]
	if len(shared) < 10 {
		t.Fatalf("expected the shared .agents/skills dir to be used by many agents, got %v", shared)
	}
	for _, id := range []string{"github-copilot", "codex", "cursor", "universal"} {
		if !slices.Contains(shared, id) {
			t.Errorf("agent %q should install into the shared .agents/skills directory", id)
		}
	}
	if got := byDir[".claude/skills"]; len(got) != 1 || got[0] != "claude-code" {
		t.Errorf(".claude/skills should belong to claude-code alone, got %v", got)
	}
}

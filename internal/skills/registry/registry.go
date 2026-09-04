// Package registry lists the AI coding agents that consume skills and where
// each one looks for them.
package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AgentHost represents an AI agent that can use skills.
type AgentHost struct {
	ID         string // canonical identifier, used for --agent
	Name       string // human-readable display name
	ProjectDir string // skills directory relative to a project root
	UserDir    string // skills directory relative to the user's home directory
}

// Scope determines where skills are installed.
type Scope string

const (
	ScopeProject Scope = "project"
	ScopeUser    Scope = "user"

	// DefaultAgentID is used when --agent is not given. The shared
	// .agents/skills directory is read by most agents, so it is the neutral choice.
	DefaultAgentID = "universal"

	claudeConfigDirEnv  = "CLAUDE_CONFIG_DIR"
	piCodingAgentDirEnv = "PI_CODING_AGENT_DIR"

	sharedProjectSkillsDir = ".agents/skills"
)

// Agents contains all known agent hosts, in the same order as GitHub CLI so
// that help output and --agent values match across the two tools. The most
// widely used agents come first, followed by the rest in alphabetical order.
//
// Agents sharing a ProjectDir (such as the shared .agents/skills directory)
// install skills to the same project-scope location, so selecting multiple
// such agents writes each skill only once.
var Agents = []AgentHost{
	{ID: "github-copilot", Name: "GitHub Copilot", ProjectDir: sharedProjectSkillsDir, UserDir: ".copilot/skills"},
	{ID: "claude-code", Name: "Claude Code", ProjectDir: ".claude/skills", UserDir: ".claude/skills"},
	{ID: "cursor", Name: "Cursor", ProjectDir: sharedProjectSkillsDir, UserDir: ".cursor/skills"},
	{ID: "codex", Name: "Codex", ProjectDir: sharedProjectSkillsDir, UserDir: sharedProjectSkillsDir},
	{ID: "gemini-cli", Name: "Gemini CLI", ProjectDir: sharedProjectSkillsDir, UserDir: ".gemini/skills"},
	{ID: "antigravity", Name: "Antigravity", ProjectDir: sharedProjectSkillsDir, UserDir: ".gemini/antigravity/skills"},
	{ID: "antigravity-cli", Name: "Antigravity CLI", ProjectDir: sharedProjectSkillsDir, UserDir: ".gemini/antigravity-cli/skills"},
	{ID: "antigravity2.0", Name: "Antigravity 2.0", ProjectDir: sharedProjectSkillsDir, UserDir: ".gemini/config/skills"},
	{ID: "adal", Name: "AdaL", ProjectDir: ".adal/skills", UserDir: ".adal/skills"},
	{ID: "amp", Name: "Amp", ProjectDir: sharedProjectSkillsDir, UserDir: ".config/agents/skills"},
	{ID: "augment", Name: "Augment", ProjectDir: ".augment/skills", UserDir: ".augment/skills"},
	{ID: "bob", Name: "IBM Bob", ProjectDir: ".bob/skills", UserDir: ".bob/skills"},
	{ID: "cline", Name: "Cline", ProjectDir: sharedProjectSkillsDir, UserDir: ".agents/skills"},
	{ID: "codebuddy", Name: "CodeBuddy", ProjectDir: ".codebuddy/skills", UserDir: ".codebuddy/skills"},
	{ID: "command-code", Name: "Command Code", ProjectDir: ".commandcode/skills", UserDir: ".commandcode/skills"},
	{ID: "continue", Name: "Continue", ProjectDir: ".continue/skills", UserDir: ".continue/skills"},
	{ID: "cortex", Name: "Cortex Code", ProjectDir: ".cortex/skills", UserDir: ".snowflake/cortex/skills"},
	{ID: "crush", Name: "Crush", ProjectDir: ".crush/skills", UserDir: ".config/crush/skills"},
	{ID: "deepagents", Name: "Deep Agents", ProjectDir: sharedProjectSkillsDir, UserDir: ".deepagents/agent/skills"},
	{ID: "devin", Name: "Devin", ProjectDir: ".devin/skills", UserDir: ".devin/skills"},
	{ID: "droid", Name: "Droid", ProjectDir: ".factory/skills", UserDir: ".factory/skills"},
	{ID: "firebender", Name: "Firebender", ProjectDir: sharedProjectSkillsDir, UserDir: ".firebender/skills"},
	{ID: "goose", Name: "Goose", ProjectDir: ".goose/skills", UserDir: ".config/goose/skills"},
	{ID: "grok", Name: "Grok", ProjectDir: ".grok/skills", UserDir: ".grok/skills"},
	{ID: "iflow-cli", Name: "iFlow CLI", ProjectDir: ".iflow/skills", UserDir: ".iflow/skills"},
	{ID: "junie", Name: "Junie", ProjectDir: ".junie/skills", UserDir: ".junie/skills"},
	{ID: "kilo", Name: "Kilo Code", ProjectDir: ".kilocode/skills", UserDir: ".kilocode/skills"},
	{ID: "kimi-cli", Name: "Kimi Code CLI", ProjectDir: sharedProjectSkillsDir, UserDir: ".config/agents/skills"},
	{ID: "kiro-cli", Name: "Kiro CLI", ProjectDir: ".kiro/skills", UserDir: ".kiro/skills"},
	{ID: "kode", Name: "Kode", ProjectDir: ".kode/skills", UserDir: ".kode/skills"},
	{ID: "mcpjam", Name: "MCPJam", ProjectDir: ".mcpjam/skills", UserDir: ".mcpjam/skills"},
	{ID: "mistral-vibe", Name: "Mistral Vibe", ProjectDir: ".vibe/skills", UserDir: ".vibe/skills"},
	{ID: "mux", Name: "Mux", ProjectDir: ".mux/skills", UserDir: ".mux/skills"},
	{ID: "neovate", Name: "Neovate", ProjectDir: ".neovate/skills", UserDir: ".neovate/skills"},
	{ID: "openclaw", Name: "OpenClaw", ProjectDir: "skills", UserDir: ".openclaw/skills"},
	{ID: "opencode", Name: "OpenCode", ProjectDir: sharedProjectSkillsDir, UserDir: ".config/opencode/skills"},
	{ID: "openhands", Name: "OpenHands", ProjectDir: ".openhands/skills", UserDir: ".openhands/skills"},
	{ID: "pi", Name: "Pi", ProjectDir: ".pi/skills", UserDir: ".pi/agent/skills"},
	{ID: "pochi", Name: "Pochi", ProjectDir: ".pochi/skills", UserDir: ".pochi/skills"},
	{ID: "qoder", Name: "Qoder", ProjectDir: ".qoder/skills", UserDir: ".qoder/skills"},
	{ID: "qwen-code", Name: "Qwen Code", ProjectDir: ".qwen/skills", UserDir: ".qwen/skills"},
	{ID: "replit", Name: "Replit", ProjectDir: sharedProjectSkillsDir, UserDir: ".config/agents/skills"},
	{ID: "roo", Name: "Roo Code", ProjectDir: ".roo/skills", UserDir: ".roo/skills"},
	{ID: "trae", Name: "Trae", ProjectDir: ".trae/skills", UserDir: ".trae/skills"},
	{ID: "trae-cn", Name: "Trae CN", ProjectDir: ".trae/skills", UserDir: ".trae-cn/skills"},
	{ID: "universal", Name: "Universal", ProjectDir: sharedProjectSkillsDir, UserDir: sharedProjectSkillsDir},
	{ID: "warp", Name: "Warp", ProjectDir: sharedProjectSkillsDir, UserDir: ".agents/skills"},
	{ID: "zencoder", Name: "Zencoder", ProjectDir: ".zencoder/skills", UserDir: ".zencoder/skills"},
}

// FindByID returns the agent host with the given ID, or an error if not found.
func FindByID(id string) (*AgentHost, error) {
	for i := range Agents {
		if Agents[i].ID == id {
			return &Agents[i], nil
		}
	}
	return nil, fmt.Errorf("unknown agent %q, valid agents: %s", id, ValidAgentIDs())
}

// ValidAgentIDs returns a comma-separated list of valid agent IDs.
func ValidAgentIDs() string {
	return strings.Join(AgentIDs(), ", ")
}

// AgentIDs returns the IDs of all known agents.
func AgentIDs() []string {
	ids := make([]string, len(Agents))
	for i, h := range Agents {
		ids[i] = h.ID
	}
	return ids
}

// AgentHelpList returns a newline-separated bulleted list of agents for help text.
func AgentHelpList() string {
	lines := make([]string, len(Agents))
	for i, h := range Agents {
		lines[i] = fmt.Sprintf("  - %s (%s)", h.Name, h.ID)
	}
	return strings.Join(lines, "\n")
}

// InstallDir resolves the installation directory for an agent host and scope.
// Project scope is anchored at gitRoot so skills land at the top level
// regardless of the current subdirectory; user scope is anchored at homeDir.
func (h *AgentHost) InstallDir(scope Scope, gitRoot, homeDir string) (string, error) {
	switch scope {
	case ScopeProject:
		if gitRoot == "" {
			return "", fmt.Errorf("could not determine project root directory")
		}
		return filepath.Join(gitRoot, h.ProjectDir), nil
	case ScopeUser:
		var configDirEnv string
		switch h.ID {
		case "claude-code":
			configDirEnv = claudeConfigDirEnv
		case "pi":
			configDirEnv = piCodingAgentDirEnv
		}
		if configDirEnv != "" {
			if configDir := os.Getenv(configDirEnv); configDir != "" {
				return filepath.Join(configDir, "skills"), nil
			}
		}
		if homeDir == "" {
			return "", fmt.Errorf("could not determine home directory")
		}
		return filepath.Join(homeDir, h.UserDir), nil
	default:
		return "", fmt.Errorf("invalid scope %q", scope)
	}
}

// UniqueProjectDirs returns the deduplicated set of project-scope skill
// directories, in Agents order. Used to warn about installed skills that would
// be published alongside a repository's own.
func UniqueProjectDirs() []string {
	seen := map[string]bool{}
	var dirs []string
	for _, h := range Agents {
		if !seen[h.ProjectDir] {
			seen[h.ProjectDir] = true
			dirs = append(dirs, h.ProjectDir)
		}
	}
	return dirs
}

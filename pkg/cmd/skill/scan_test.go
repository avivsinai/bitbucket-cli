package skill

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/skills/registry"
)

// These tests cover the default (no --dir) scan, which is what "bkt skill list"
// and "bkt skill update" use in practice. Synthetic roots are passed in so no
// real home directory is touched.

func TestBuildScanTargetsCoversAgentsAndScopes(t *testing.T) {
	gitRoot := filepath.Join("repo", "root")
	homeDir := filepath.Join("home", "monalisa")

	targets, err := buildScanTargets(&listOptions{}, gitRoot, homeDir)
	if err != nil {
		t.Fatalf("buildScanTargets: %v", err)
	}

	// A (directory, scope) pair may appear more than once only with
	// complementary filters, so no skill is reported twice. That happens for
	// the repository's own skills/ folder: once as OpenClaw's project
	// directory (installed skills only) and once for authored skills.
	byDir := map[string][]scanTarget{}
	for _, tgt := range targets {
		byDir[tgt.dir] = append(byDir[tgt.dir], tgt)
	}
	for dir, group := range byDir {
		if len(group) == 1 {
			continue
		}
		filters := map[scanFilter]bool{}
		for _, tgt := range group {
			if filters[tgt.filter] {
				t.Errorf("directory %q scanned twice with the same filter %v", dir, tgt.filter)
			}
			filters[tgt.filter] = true
		}
		if !filters[scanInstalledOnly] || !filters[scanPublishedOnly] {
			t.Errorf("directory %q is scanned %d times without complementary filters: %+v", dir, len(group), group)
		}
	}

	first := func(dir string) (scanTarget, bool) {
		group := byDir[dir]
		if len(group) == 0 {
			return scanTarget{}, false
		}
		return group[0], true
	}

	// Agents sharing .agents/skills collapse into a single scan that names them all.
	shared, ok := first(filepath.Join(gitRoot, ".agents", "skills"))
	if !ok {
		t.Fatal("expected a scan target for the shared .agents/skills directory")
	}
	if len(shared.agentHostIDs) < 10 {
		t.Errorf("shared directory should list every agent that uses it, got %v", shared.agentHostIDs)
	}
	if slices.Contains(shared.agentHostIDs, "claude-code") {
		t.Error("claude-code has its own directory and must not appear under the shared one")
	}
	if _, ok := first(filepath.Join(gitRoot, ".claude", "skills")); !ok {
		t.Error("expected a scan target for .claude/skills")
	}

	// OpenClaw's project directory is the repository's own skills/ folder, so
	// that target only reports skills carrying install metadata.
	own := byDir[filepath.Join(gitRoot, "skills")]
	if len(own) != 2 {
		t.Fatalf("the repository's skills/ directory should be scanned for installed and authored skills, got %+v", own)
	}
	var sawPublished bool
	for _, tgt := range own {
		if slices.Contains(tgt.agentHostIDs, agentHostPublished) {
			sawPublished = true
			if tgt.filter != scanPublishedOnly {
				t.Errorf("published target filter = %v, want scanPublishedOnly", tgt.filter)
			}
		}
	}
	if !sawPublished {
		t.Error("expected authored skills to be listed as published")
	}

	var sawUser bool
	for _, tgt := range targets {
		if tgt.scope == string(registry.ScopeUser) {
			sawUser = true
			if !strings.HasPrefix(tgt.dir, homeDir) {
				t.Errorf("user-scope target %q is not under the home directory", tgt.dir)
			}
		}
	}
	if !sawUser {
		t.Error("expected user-scope targets")
	}
}

func TestBuildScanTargetsFilters(t *testing.T) {
	gitRoot := filepath.Join("repo", "root")
	homeDir := filepath.Join("home", "monalisa")

	t.Run("single agent", func(t *testing.T) {
		targets, err := buildScanTargets(&listOptions{Agent: "claude-code"}, gitRoot, homeDir)
		if err != nil {
			t.Fatal(err)
		}
		if len(targets) != 2 {
			t.Fatalf("targets = %+v, want project and user scope only", targets)
		}
		for _, tgt := range targets {
			if len(tgt.agentHostIDs) != 1 || tgt.agentHostIDs[0] != "claude-code" {
				t.Errorf("target %+v should belong to claude-code alone", tgt)
			}
		}
	})

	t.Run("single scope", func(t *testing.T) {
		targets, err := buildScanTargets(&listOptions{Scope: string(registry.ScopeUser)}, gitRoot, homeDir)
		if err != nil {
			t.Fatal(err)
		}
		if len(targets) == 0 {
			t.Fatal("expected user-scope targets")
		}
		for _, tgt := range targets {
			if tgt.scope != string(registry.ScopeUser) {
				t.Fatalf("target %+v is not user scope", tgt)
			}
		}
	})

	t.Run("project scope without a git root skips project directories", func(t *testing.T) {
		targets, err := buildScanTargets(&listOptions{Scope: string(registry.ScopeProject)}, "", homeDir)
		if err != nil {
			t.Fatal(err)
		}
		if len(targets) != 0 {
			t.Fatalf("targets = %+v, want none outside a repository", targets)
		}
	})

	t.Run("unknown agent", func(t *testing.T) {
		if _, err := buildScanTargets(&listOptions{Agent: "nope"}, gitRoot, homeDir); err == nil {
			t.Fatal("expected an error for an unknown agent")
		}
	})
}

func TestScanFilterForAgentHost(t *testing.T) {
	openclaw, err := registry.FindByID("openclaw")
	if err != nil {
		t.Fatal(err)
	}
	claude, err := registry.FindByID("claude-code")
	if err != nil {
		t.Fatal(err)
	}

	// OpenClaw reads the repository's own skills/ directory, which also holds
	// skills the repository authors, so only installed ones are listed there.
	if got := scanFilterForAgentHost(openclaw, registry.ScopeProject); got != scanInstalledOnly {
		t.Errorf("openclaw project filter = %v, want scanInstalledOnly", got)
	}
	if got := scanFilterForAgentHost(openclaw, registry.ScopeUser); got != scanAllSkills {
		t.Errorf("openclaw user filter = %v, want scanAllSkills", got)
	}
	if got := scanFilterForAgentHost(claude, registry.ScopeProject); got != scanAllSkills {
		t.Errorf("claude-code project filter = %v, want scanAllSkills", got)
	}
}

func TestScanAllAgentsForUpdateFindsSkillsAcrossHosts(t *testing.T) {
	gitRoot := t.TempDir()
	homeDir := t.TempDir()

	write := func(dir, name string) {
		t.Helper()
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(full, 0o755); err != nil {
			t.Fatal(err)
		}
		body := "---\nname: " + name + "\nmetadata:\n    bitbucket-repo: https://bitbucket.org/myteam/agent-skills\n    bitbucket-commit: c1\n---\n"
		if err := os.WriteFile(filepath.Join(full, "SKILL.md"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(gitRoot, ".agents", "skills"), "shared-skill")
	write(filepath.Join(gitRoot, ".claude", "skills"), "claude-skill")
	write(filepath.Join(homeDir, ".copilot", "skills"), "user-skill")

	found := scanAllAgentsForUpdate(gitRoot, homeDir)

	names := map[string]int{}
	for _, s := range found {
		names[s.name]++
	}
	for _, want := range []string{"shared-skill", "claude-skill", "user-skill"} {
		if names[want] == 0 {
			t.Errorf("missing %q; found %v", want, names)
		}
		if names[want] > 1 {
			t.Errorf("%q reported %d times; a shared directory must be scanned once", want, names[want])
		}
	}
	for _, s := range found {
		if s.repoURL != "https://bitbucket.org/myteam/agent-skills" {
			t.Errorf("%s repoURL = %q", s.name, s.repoURL)
		}
		if s.commit != "c1" {
			t.Errorf("%s commit = %q", s.name, s.commit)
		}
	}
}

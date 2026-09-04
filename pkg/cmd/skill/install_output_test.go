package skill

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalInstallJSONIsOneDocument(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	writeSkill(t, sourceDir, "skills/demo-skill", validSkill("demo-skill"))

	cmd := NewCmdSkill(f)
	root := structuredSkillRoot(cmd, stdout, stderr)
	root.SetArgs([]string{"skill", "install", sourceDir, "demo-skill", "--from-local", "--dir", targetDir, "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("install --json: %v (stderr=%s)", err, stderr)
	}

	var got installOutput
	if err := json.Unmarshal([]byte(stdout.String()), &got); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, stdout)
	}
	if len(got.Installed) != 1 {
		t.Fatalf("result = %+v", got)
	}
	item := got.Installed[0]
	if item.SkillName != "demo-skill" || item.SourceURL != sourceDir || item.Path != targetDir {
		t.Fatalf("installed item = %+v", item)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "demo-skill", "SKILL.md")); err != nil {
		t.Fatalf("installed file: %v", err)
	}
}

func TestRemoteInstallJSONIsOneDocument(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	targetDir := t.TempDir()
	repo := newSkillsRepo()
	stubRepository(t, repo)

	cmd := NewCmdSkill(f)
	root := structuredSkillRoot(cmd, stdout, stderr)
	root.SetArgs([]string{"skill", "install", "myteam/agent-skills", "alpha", "--dir", targetDir, "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("remote install --json: %v (stderr=%s)", err, stderr)
	}

	var got installOutput
	if err := json.Unmarshal([]byte(stdout.String()), &got); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, stdout)
	}
	if len(got.Installed) != 1 || got.Installed[0].SkillName != "alpha" || got.Installed[0].SourceURL != repo.FullName() || got.Installed[0].Version != "v1.0.0" {
		t.Fatalf("result = %+v", got)
	}
}

func TestInstallRejectsInvalidOutputFlagsBeforeWriting(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	writeSkill(t, sourceDir, "skills/demo-skill", validSkill("demo-skill"))

	cmd := NewCmdSkill(f)
	root := structuredSkillRoot(cmd, stdout, stderr)
	root.SetArgs([]string{"skill", "install", sourceDir, "demo-skill", "--from-local", "--dir", targetDir, "--json", "--yaml"})
	root.SetContext(context.Background())
	err := root.Execute()
	if err == nil || !strings.Contains(err.Error(), "cannot use --json and --yaml simultaneously") {
		t.Fatalf("error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "demo-skill")); !os.IsNotExist(err) {
		t.Fatalf("destination was written before output flags were rejected: %v", err)
	}
}

func TestLocalInstallDefaultOutputRemainsHumanReadable(t *testing.T) {
	f, stdout, stderr := newTestFactory(t)
	sourceDir := t.TempDir()
	targetDir := t.TempDir()
	writeSkill(t, sourceDir, "skills/demo-skill", validSkill("demo-skill"))

	if err := runSkill(t, f, stdout, stderr, "install", sourceDir, "demo-skill", "--from-local", "--dir", targetDir); err != nil {
		t.Fatalf("install: %v", err)
	}
	if !strings.Contains(stdout.String(), "Installed demo-skill (from "+sourceDir+")") {
		t.Fatalf("stdout = %q", stdout)
	}
}

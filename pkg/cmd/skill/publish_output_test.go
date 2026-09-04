package skill

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/skills/sourcetest"
)

func TestPublishTagJSONIsOneDocument(t *testing.T) {
	root, head := setupTagRepo(t)
	repo := sourcetest.New("myteam/agent-skills", nil)
	repo.Branches = map[string]string{"main": head}
	stubRepository(t, repo)

	f, stdout, stderr := newTestFactory(t)
	cmd := NewCmdSkill(f)
	rootCmd := structuredSkillRoot(cmd, stdout, stderr)
	rootCmd.SetArgs([]string{"skill", "publish", root, "--tag", "v1.0.0", "--json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("publish --json: %v (stderr=%s)", err, stderr)
	}

	var got publishResult
	if err := json.Unmarshal([]byte(stdout.String()), &got); err != nil {
		t.Fatalf("stdout is not one JSON document: %v\n%s", err, stdout)
	}
	if got.Tag != "v1.0.0" || got.Commit != head || got.Repository != "myteam/agent-skills" {
		t.Fatalf("result = %+v", got)
	}
	if len(got.Diagnostics) != 0 || !strings.Contains(got.PinCommand, "--pin v1.0.0") {
		t.Fatalf("result = %+v", got)
	}
}

func TestPublishTagJSONReportsValidationErrors(t *testing.T) {
	root, _ := setupTagRepo(t)
	writeSkill(t, root, "skills/code-review", "---\nname: wrong\ndescription: d\nlicense: MIT\n---\n")
	f, stdout, stderr := newTestFactory(t)
	cmd := NewCmdSkill(f)
	rootCmd := structuredSkillRoot(cmd, stdout, stderr)
	rootCmd.SetArgs([]string{"skill", "publish", root, "--tag", "v1.0.0", "--json"})
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "validation failed") {
		t.Fatalf("error = %v", err)
	}

	var got []diagnostic
	if err := json.Unmarshal([]byte(stdout.String()), &got); err != nil {
		t.Fatalf("validation output is not JSON: %v\n%s", err, stdout)
	}
	if len(got) == 0 || got[0].Severity != severityError {
		t.Fatalf("diagnostics = %+v", got)
	}
}

func TestPublishFixRejectsInvalidOutputFlagsBeforeWriting(t *testing.T) {
	root := t.TempDir()
	content := "---\nname: code-review\ndescription: d\nlicense: MIT\nmetadata:\n    bitbucket-repo: source\n---\n"
	writeSkill(t, root, "skills/code-review", content)
	skillFile := filepath.Join(root, "skills", "code-review", "SKILL.md")

	f, stdout, stderr := newTestFactory(t)
	cmd := NewCmdSkill(f)
	rootCmd := structuredSkillRoot(cmd, stdout, stderr)
	rootCmd.SetArgs([]string{"skill", "publish", root, "--fix", "--json", "--yaml"})
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "cannot use --json and --yaml simultaneously") {
		t.Fatalf("error = %v", err)
	}
	if got := readFile(t, skillFile); got != content {
		t.Fatalf("file changed before output flags were rejected:\n%s", got)
	}
}

func structuredSkillRoot(cmd *cobra.Command, stdout, stderr *strings.Builder) *cobra.Command {
	root := &cobra.Command{Use: "bkt"}
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("yaml", false, "")
	root.PersistentFlags().String("format", "", "")
	root.PersistentFlags().String("jq", "", "")
	root.PersistentFlags().String("template", "", "")
	root.AddCommand(cmd)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetContext(context.Background())
	return root
}

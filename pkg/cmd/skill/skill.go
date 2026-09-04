// Package skill implements "bkt skill", a mirror of GitHub CLI's "gh skill"
// for agent skills hosted in Bitbucket repositories.
package skill

import (
	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdSkill returns the agent skills command tree.
func NewCmdSkill(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "skill <command>",
		Aliases: []string{"skills"},
		Short:   "Install and manage agent skills",
		Long: `Install and manage agent skills from Bitbucket repositories.

Skills are directories containing a SKILL.md file that teach AI coding agents
(Claude Code, Codex, Cursor, GitHub Copilot, Gemini CLI, and many more) how to
perform a task. This command group mirrors "gh skill" so the same workflow works
for skills hosted on Bitbucket Cloud and Bitbucket Data Center. Repositories are
addressed as WORKSPACE/REPO (Cloud) or PROJECT/REPO (Data Center) and accessed
with the credentials of the active context.

See https://agentskills.io/specification for the skill format.`,
		Example: `  # Install a skill from a Bitbucket repository
  bkt skill install myteam/agent-skills code-review

  # List installed skills
  bkt skill list

  # Preview a skill before installing
  bkt skill preview myteam/agent-skills code-review

  # Update all installed skills
  bkt skill update --all`,
	}

	cmd.AddCommand(
		newInstallCmd(f),
		newListCmd(f),
		newPreviewCmd(f),
		newUpdateCmd(f),
		newPublishCmd(f),
		newSearchCmd(f),
	)

	return cmd
}

package branch

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type protectOptions struct {
	Project string
	Repo    string
	Branch  string
	Type    string
	Users   []string
	Groups  []string
	ID      int
}

func newProtectCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "protect",
		Short: "Manage branch protection rules (DC only)",
		Long: `Manage branch protection rules (restrictions) for a Bitbucket Data Center
repository. Supports listing existing restrictions, adding new ones, and
removing them by ID.

Restriction types include no-creates, no-deletes, fast-forward-only, and
require-approvals. Restrictions can target specific users or groups.

This command group currently supports Data Center contexts only.`,
		Example: `  # List all branch restrictions
  bkt branch protect list

  # Prevent direct pushes to main
  bkt branch protect add main --type fast-forward-only

  # Remove a restriction by ID
  bkt branch protect remove 42`,
	}

	cmd.AddCommand(newProtectListCmd(f))
	cmd.AddCommand(newProtectAddCmd(f))
	cmd.AddCommand(newProtectRemoveCmd(f))

	return cmd
}

func newProtectListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &protectOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List branch restrictions (DC only)",
		Long: `List all branch restrictions configured for a Bitbucket Data Center repository.

Each restriction is shown with its ID, type, and the branch matcher it applies
to. Use the restriction ID with "protect remove" to delete a rule.

This command currently supports Data Center contexts only.`,
		Example: `  # List all restrictions in the current context
  bkt branch protect list

  # List restrictions for a specific project and repo
  bkt branch protect list --project MYPROJ --repo backend`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProtectList(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func newProtectAddCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &protectOptions{}
	cmd := &cobra.Command{
		Use:   "add <branch>",
		Short: "Add a branch restriction (DC only)",
		Long: `Add a branch restriction to a Bitbucket Data Center repository.

Available restriction types:
  no-creates          Prevent creating branches matching the pattern
  no-deletes          Prevent deleting branches matching the pattern
  fast-forward-only   Only allow fast-forward merges (no force pushes)
  require-approvals   Require pull request approval before merging

Restrictions can be scoped to specific users or groups with the --user and
--group flags, which are repeatable.

This command currently supports Data Center contexts only.`,
		Example: `  # Prevent force pushes to main
  bkt branch protect add main --type fast-forward-only

  # Require PR approvals on release branches
  bkt branch protect add release/v2 --type require-approvals

  # Block branch deletion for specific users
  bkt branch protect add main --type no-deletes --user alice --user bob

  # Restrict a branch for a group
  bkt branch protect add develop --type no-creates --group developers`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Branch = args[0]
			return runProtectAdd(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Type, "type", "no-creates", "Restriction type (no-creates, no-deletes, fast-forward-only, require-approvals)")
	cmd.Flags().StringSliceVar(&opts.Users, "user", nil, "Usernames to apply the restriction to (repeatable)")
	cmd.Flags().StringSliceVar(&opts.Groups, "group", nil, "Group names to apply the restriction to (repeatable)")

	return cmd
}

func newProtectRemoveCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &protectOptions{}
	cmd := &cobra.Command{
		Use:   "remove <restriction-id>",
		Short: "Remove a branch restriction (DC only)",
		Long: `Remove a branch restriction from a Bitbucket Data Center repository by its
numeric ID. Use "bkt branch protect list" to find restriction IDs.

This command currently supports Data Center contexts only.`,
		Example: `  # Remove a restriction by ID
  bkt branch protect remove 42

  # Remove a restriction in a specific project and repo
  bkt branch protect remove 15 --project MYPROJ --repo backend`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid restriction id %q", args[0])
			}
			opts.ID = id
			return runProtectRemove(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func runProtectList(cmd *cobra.Command, f *cmdutil.Factory, opts *protectOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("branch protect list currently supports Data Center contexts only")
	}

	projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	restrictions, err := client.ListBranchRestrictions(ctx, projectKey, repoSlug)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"project":      projectKey,
		"repo":         repoSlug,
		"restrictions": restrictions,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(restrictions) == 0 {
			_, err := fmt.Fprintln(ios.Out, "No branch restrictions configured.")
			return err
		}
		for _, res := range restrictions {
			if _, err := fmt.Fprintf(ios.Out, "%d\t%s\t%s\n", res.ID, res.Type, res.Matcher.DisplayID); err != nil {
				return err
			}
		}
		return nil
	})
}

func runProtectAdd(cmd *cobra.Command, f *cmdutil.Factory, opts *protectOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("branch protect add currently supports Data Center contexts only")
	}

	projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	typeID := mapProtectType(opts.Type)
	if typeID == "" {
		return fmt.Errorf("unsupported restriction type %q", opts.Type)
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	restriction, err := client.CreateBranchRestriction(ctx, projectKey, repoSlug, bbdc.BranchRestrictionInput{
		Type:        typeID,
		MatcherID:   ensureBranchRef(opts.Branch),
		MatcherType: "BRANCH",
		Users:       opts.Users,
		Groups:      opts.Groups,
	})
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Added restriction %d (%s) on %s\n", restriction.ID, restriction.Type, restriction.Matcher.DisplayID); err != nil {
		return err
	}
	return nil
}

func runProtectRemove(cmd *cobra.Command, f *cmdutil.Factory, opts *protectOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("branch protect remove currently supports Data Center contexts only")
	}

	projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.DeleteBranchRestriction(ctx, projectKey, repoSlug, opts.ID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Removed restriction %d\n", opts.ID); err != nil {
		return err
	}
	return nil
}

func mapProtectType(t string) string {
	switch strings.ToLower(t) {
	case "no-creates":
		return "NO_CREATES"
	case "no-deletes":
		return "NO_DELETES"
	case "fast-forward-only":
		return "FAST_FORWARD_ONLY"
	case "require-approvals":
		return "PULL_REQUEST"
	default:
		return ""
	}
}

func ensureBranchRef(branch string) string {
	if branch == "" {
		return "refs/heads/*"
	}
	if strings.HasPrefix(branch, "refs/") {
		return branch
	}
	return "refs/heads/" + branch
}

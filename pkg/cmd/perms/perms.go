package perms

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCommand manages repository and project permissions.
func NewCommand(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "perms",
		Short: "Manage Bitbucket permissions",
		Long: `Manage user permissions at the project and repository level on Bitbucket Data Center.

Grant, revoke, and list permissions for individual users. Project-level
permissions apply to all repositories within that project, while repository-level
permissions override the project defaults for a specific repository.

This command group is available for Data Center contexts only.`,
		Example: `  # List who has access to a project
  bkt perms project list --project MYPROJ

  # Grant a user write access to a specific repository
  bkt perms repo grant --project MYPROJ --repo my-service --user jdoe --perm REPO_WRITE

  # Revoke a user's project-level permission
  bkt perms project revoke --project MYPROJ --user jdoe`,
	}

	cmd.AddCommand(newProjectCmd(f))
	cmd.AddCommand(newRepoCmd(f))

	return cmd
}

type projectListOptions struct {
	Project string
	Limit   int
}

type projectGrantOptions struct {
	Project    string
	Username   string
	Permission string
}

type projectRevokeOptions struct {
	Project  string
	Username string
}

func newProjectCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage project-level permissions",
		Long: `Manage project-level permissions on Bitbucket Data Center.

Project permissions control default access for all repositories within a project.
You can list current permission entries, grant a permission level to a user, or
revoke a user's project permission entirely. Valid permission levels are
PROJECT_READ, PROJECT_WRITE, and PROJECT_ADMIN.`,
		Example: `  # List all users with permissions on a project
  bkt perms project list --project MYPROJ

  # Grant admin access to a user
  bkt perms project grant --project MYPROJ --user jdoe --perm PROJECT_ADMIN

  # Revoke a user's project permission
  bkt perms project revoke --project MYPROJ --user jdoe`,
	}

	listOpts := &projectListOptions{Limit: 100}
	list := &cobra.Command{
		Use:   "list",
		Short: "List project permissions",
		Long: `List the permission entries for a Bitbucket Data Center project.

Displays each user who has been granted explicit access to the project along
with their permission level (PROJECT_READ, PROJECT_WRITE, or PROJECT_ADMIN).
Use --limit to control how many entries are returned; set it to 0 to fetch all.`,
		Example: `  # List permissions for a project
  bkt perms project list --project MYPROJ

  # List all permissions without a cap
  bkt perms project list --project MYPROJ --limit 0

  # Output as JSON
  bkt perms project list --project MYPROJ --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectList(cmd, f, listOpts)
		},
	}
	list.Flags().StringVar(&listOpts.Project, "project", "", "Bitbucket project key (required)")
	list.Flags().IntVar(&listOpts.Limit, "limit", listOpts.Limit, "Maximum entries to display (0 for all)")
	_ = list.MarkFlagRequired("project")

	grantOpts := &projectGrantOptions{}
	grant := &cobra.Command{
		Use:   "grant",
		Short: "Grant project permissions",
		Long: `Grant a permission level to a user on a Bitbucket Data Center project.

The user receives the specified permission for the project and inherits it
across all repositories within that project unless overridden at the repository
level. Valid values for --perm are PROJECT_READ, PROJECT_WRITE, and
PROJECT_ADMIN. If --perm is omitted it defaults to PROJECT_READ.`,
		Example: `  # Grant read access (default)
  bkt perms project grant --project MYPROJ --user jdoe

  # Grant write access
  bkt perms project grant --project MYPROJ --user jdoe --perm PROJECT_WRITE

  # Grant admin access
  bkt perms project grant --project MYPROJ --user jdoe --perm PROJECT_ADMIN`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectGrant(cmd, f, grantOpts)
		},
	}
	grant.Flags().StringVar(&grantOpts.Project, "project", "", "Bitbucket project key (required)")
	grant.Flags().StringVar(&grantOpts.Username, "user", "", "Username to grant (required)")
	grant.Flags().StringVar(&grantOpts.Permission, "perm", "PROJECT_READ", "Permission (PROJECT_READ, PROJECT_WRITE, PROJECT_ADMIN)")
	_ = grant.MarkFlagRequired("project")
	_ = grant.MarkFlagRequired("user")

	revokeOpts := &projectRevokeOptions{}
	revoke := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke project permissions",
		Long: `Revoke a user's permission on a Bitbucket Data Center project.

Removes the explicit project-level permission entry for the specified user.
After revocation the user loses access granted at the project level, though
they may still have access through repository-level or global permissions.`,
		Example: `  # Revoke a user's project permission
  bkt perms project revoke --project MYPROJ --user jdoe

  # Revoke using a different context
  bkt perms project revoke --project MYPROJ --user jdoe --context my-dc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProjectRevoke(cmd, f, revokeOpts)
		},
	}
	revoke.Flags().StringVar(&revokeOpts.Project, "project", "", "Bitbucket project key (required)")
	revoke.Flags().StringVar(&revokeOpts.Username, "user", "", "Username to revoke (required)")
	_ = revoke.MarkFlagRequired("project")
	_ = revoke.MarkFlagRequired("user")

	cmd.AddCommand(list, grant, revoke)
	return cmd
}

type repoListOptions struct {
	Project string
	Repo    string
	Limit   int
}

type repoGrantOptions struct {
	Project    string
	Repo       string
	Username   string
	Permission string
}

type repoRevokeOptions struct {
	Project  string
	Repo     string
	Username string
}

func newRepoCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Manage repository-level permissions",
		Long: `Manage repository-level permissions on Bitbucket Data Center.

Repository permissions override the project defaults for a specific repository.
You can list current permission entries, grant a permission level to a user, or
revoke a user's repository permission entirely. Valid permission levels are
REPO_READ, REPO_WRITE, and REPO_ADMIN.`,
		Example: `  # List permissions on a repository
  bkt perms repo list --project MYPROJ --repo my-service

  # Grant write access to a user
  bkt perms repo grant --project MYPROJ --repo my-service --user jdoe --perm REPO_WRITE

  # Revoke a user's repository permission
  bkt perms repo revoke --project MYPROJ --repo my-service --user jdoe`,
	}

	listOpts := &repoListOptions{Limit: 100}
	list := &cobra.Command{
		Use:   "list",
		Short: "List repository permissions",
		Long: `List the permission entries for a Bitbucket Data Center repository.

Displays each user who has been granted explicit access to the repository along
with their permission level (REPO_READ, REPO_WRITE, or REPO_ADMIN). Use --limit
to control how many entries are returned; set it to 0 to fetch all.`,
		Example: `  # List permissions for a repository
  bkt perms repo list --project MYPROJ --repo my-service

  # Fetch all permission entries
  bkt perms repo list --project MYPROJ --repo my-service --limit 0

  # Output as JSON
  bkt perms repo list --project MYPROJ --repo my-service --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRepoList(cmd, f, listOpts)
		},
	}
	list.Flags().StringVar(&listOpts.Project, "project", "", "Bitbucket project key (required)")
	list.Flags().StringVar(&listOpts.Repo, "repo", "", "Repository slug (required)")
	list.Flags().IntVar(&listOpts.Limit, "limit", listOpts.Limit, "Maximum entries to display (0 for all)")
	_ = list.MarkFlagRequired("project")
	_ = list.MarkFlagRequired("repo")

	grantOpts := &repoGrantOptions{}
	grant := &cobra.Command{
		Use:   "grant",
		Short: "Grant repository permissions",
		Long: `Grant a permission level to a user on a Bitbucket Data Center repository.

The user receives the specified permission for the repository, overriding any
project-level permission they may already have. Valid values for --perm are
REPO_READ, REPO_WRITE, and REPO_ADMIN. If --perm is omitted it defaults to
REPO_READ.`,
		Example: `  # Grant read access (default)
  bkt perms repo grant --project MYPROJ --repo my-service --user jdoe

  # Grant write access
  bkt perms repo grant --project MYPROJ --repo my-service --user jdoe --perm REPO_WRITE

  # Grant admin access
  bkt perms repo grant --project MYPROJ --repo my-service --user jdoe --perm REPO_ADMIN`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRepoGrant(cmd, f, grantOpts)
		},
	}
	grant.Flags().StringVar(&grantOpts.Project, "project", "", "Bitbucket project key (required)")
	grant.Flags().StringVar(&grantOpts.Repo, "repo", "", "Repository slug (required)")
	grant.Flags().StringVar(&grantOpts.Username, "user", "", "Username to grant (required)")
	grant.Flags().StringVar(&grantOpts.Permission, "perm", "REPO_READ", "Permission (REPO_READ, REPO_WRITE, REPO_ADMIN)")
	_ = grant.MarkFlagRequired("project")
	_ = grant.MarkFlagRequired("repo")
	_ = grant.MarkFlagRequired("user")

	revokeOpts := &repoRevokeOptions{}
	revoke := &cobra.Command{
		Use:   "revoke",
		Short: "Revoke repository permissions",
		Long: `Revoke a user's permission on a Bitbucket Data Center repository.

Removes the explicit repository-level permission entry for the specified user.
After revocation the user may still have access through project-level or global
permissions.`,
		Example: `  # Revoke a user's repository permission
  bkt perms repo revoke --project MYPROJ --repo my-service --user jdoe

  # Revoke using a different context
  bkt perms repo revoke --project MYPROJ --repo my-service --user jdoe --context my-dc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRepoRevoke(cmd, f, revokeOpts)
		},
	}
	revoke.Flags().StringVar(&revokeOpts.Project, "project", "", "Bitbucket project key (required)")
	revoke.Flags().StringVar(&revokeOpts.Repo, "repo", "", "Repository slug (required)")
	revoke.Flags().StringVar(&revokeOpts.Username, "user", "", "Username to revoke (required)")
	_ = revoke.MarkFlagRequired("project")
	_ = revoke.MarkFlagRequired("repo")
	_ = revoke.MarkFlagRequired("user")

	cmd.AddCommand(list, grant, revoke)
	return cmd
}

func runProjectList(cmd *cobra.Command, f *cmdutil.Factory, opts *projectListOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, _, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("perms project list currently supports Data Center contexts only")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	perms, err := client.ListProjectPermissions(ctx, opts.Project, opts.Limit)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"project":     opts.Project,
		"permissions": perms,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		for _, p := range perms {
			if _, err := fmt.Fprintf(ios.Out, "%s\t%s\n", cmdutil.FirstNonEmpty(p.User.FullName, p.User.Name), p.Permission); err != nil {
				return err
			}
		}
		if len(perms) == 0 {
			if _, err := fmt.Fprintln(ios.Out, "No permissions found."); err != nil {
				return err
			}
		}
		return nil
	})
}

func runProjectGrant(cmd *cobra.Command, f *cmdutil.Factory, opts *projectGrantOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, _, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("perms project grant currently supports Data Center contexts only")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.GrantProjectPermission(ctx, opts.Project, opts.Username, opts.Permission); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Granted %s on project %s to %s\n", strings.ToUpper(opts.Permission), opts.Project, opts.Username); err != nil {
		return err
	}
	return nil
}

func runProjectRevoke(cmd *cobra.Command, f *cmdutil.Factory, opts *projectRevokeOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, _, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("perms project revoke currently supports Data Center contexts only")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.RevokeProjectPermission(ctx, opts.Project, opts.Username); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Revoked project permission for %s on %s\n", opts.Username, opts.Project); err != nil {
		return err
	}
	return nil
}

func runRepoList(cmd *cobra.Command, f *cmdutil.Factory, opts *repoListOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, _, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("perms repo list currently supports Data Center contexts only")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	perms, err := client.ListRepoPermissions(ctx, opts.Project, opts.Repo, opts.Limit)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"project":     opts.Project,
		"repo":        opts.Repo,
		"permissions": perms,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		for _, p := range perms {
			if _, err := fmt.Fprintf(ios.Out, "%s\t%s\n", cmdutil.FirstNonEmpty(p.User.FullName, p.User.Name), p.Permission); err != nil {
				return err
			}
		}
		if len(perms) == 0 {
			if _, err := fmt.Fprintln(ios.Out, "No permissions found."); err != nil {
				return err
			}
		}
		return nil
	})
}

func runRepoGrant(cmd *cobra.Command, f *cmdutil.Factory, opts *repoGrantOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, _, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("perms repo grant currently supports Data Center contexts only")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.GrantRepoPermission(ctx, opts.Project, opts.Repo, opts.Username, opts.Permission); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Granted %s on %s/%s to %s\n", strings.ToUpper(opts.Permission), opts.Project, opts.Repo, opts.Username); err != nil {
		return err
	}
	return nil
}

func runRepoRevoke(cmd *cobra.Command, f *cmdutil.Factory, opts *repoRevokeOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, _, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("perms repo revoke currently supports Data Center contexts only")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if err := client.RevokeRepoPermission(ctx, opts.Project, opts.Repo, opts.Username); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Revoked repository permission for %s on %s/%s\n", opts.Username, opts.Project, opts.Repo); err != nil {
		return err
	}
	return nil
}

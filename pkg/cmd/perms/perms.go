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
	}

	listOpts := &projectListOptions{Limit: 100}
	list := &cobra.Command{
		Use:   "list",
		Short: "List project permissions",
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
	}

	listOpts := &repoListOptions{Limit: 100}
	list := &cobra.Command{
		Use:   "list",
		Short: "List repository permissions",
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

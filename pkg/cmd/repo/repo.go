package repo

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/format"
)

// NewCmdRepo wires repository subcommands.
func NewCmdRepo(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repo",
		Short: "Work with Bitbucket repositories",
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newViewCmd(f))
	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newCloneCmd(f))
	cmd.AddCommand(newBrowseCmd(f))

	return cmd
}

type listOptions struct {
	Project string
	Limit   int
}

type createOptions struct {
	Project       string
	Description   string
	Public        bool
	Forkable      bool
	DefaultBranch string
	SCM           string
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{
		Limit: 30,
	}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List repositories within the active scope",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().IntVar(&opts.Limit, "limit", opts.Limit, "Maximum repositories to display (0 for all)")
	return cmd
}

func runList(cmd *cobra.Command, f *cmdutil.Factory, opts *listOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	if host.Kind != "dc" {
		return fmt.Errorf("repo list currently supports Data Center contexts only")
	}

	projectKey := strings.TrimSpace(opts.Project)
	if projectKey == "" {
		projectKey = ctxCfg.ProjectKey
	}
	if projectKey == "" {
		return fmt.Errorf("project key required; set with --project or configure the context default")
	}
	projectKey = strings.ToUpper(projectKey)

	client, err := bbdc.New(bbdc.Options{
		BaseURL:  host.BaseURL,
		Username: host.Username,
		Token:    host.Token,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	repos, err := client.ListRepositories(ctx, projectKey, opts.Limit)
	if err != nil {
		return err
	}

	formatOpt, err := cmdutil.OutputFormat(cmd)
	if err != nil {
		return err
	}

	type repoSummary struct {
		Project string   `json:"project"`
		Slug    string   `json:"slug"`
		Name    string   `json:"name"`
		ID      int      `json:"id"`
		WebURL  string   `json:"web_url,omitempty"`
		Clone   []string `json:"clone_urls,omitempty"`
	}

	var summaries []repoSummary
	for _, repo := range repos {
		summaries = append(summaries, repoSummary{
			Project: repo.Project.Key,
			Slug:    repo.Slug,
			Name:    repo.Name,
			ID:      repo.ID,
			WebURL:  firstLink(repo, "web"),
			Clone:   cloneLinks(repo),
		})
	}

	if formatOpt != "" {
		payload := struct {
			Project string        `json:"project"`
			Repos   []repoSummary `json:"repositories"`
		}{
			Project: projectKey,
			Repos:   summaries,
		}
		return format.Write(ios.Out, formatOpt, payload, nil)
	}

	if len(summaries) == 0 {
		fmt.Fprintf(ios.Out, "No repositories found in project %s.\n", projectKey)
		return nil
	}

	for _, r := range summaries {
		fmt.Fprintf(ios.Out, "%s/%s\t%s\n", r.Project, r.Slug, r.Name)
		if r.WebURL != "" {
			fmt.Fprintf(ios.Out, "    web:   %s\n", r.WebURL)
		}
		if len(r.Clone) > 0 {
			fmt.Fprintf(ios.Out, "    clone: %s\n", strings.Join(r.Clone, ", "))
		}
	}
	return nil
}

type viewOptions struct {
	Project string
	Repo    string
}

type cloneOptions struct {
	Project string
	Repo    string
	UseSSH  bool
	Dest    string
}

func newViewCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &viewOptions{}
	cmd := &cobra.Command{
		Use:   "view [repository]",
		Short: "Display details for a repository",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Repo = args[0]
			}
			return runView(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func runView(cmd *cobra.Command, f *cmdutil.Factory, opts *viewOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	if host.Kind != "dc" {
		return fmt.Errorf("repo view currently supports Data Center contexts only")
	}

	projectKey := strings.TrimSpace(opts.Project)
	if projectKey == "" {
		projectKey = ctxCfg.ProjectKey
	}
	if projectKey == "" {
		return fmt.Errorf("project key required; set with --project or configure the context default")
	}
	projectKey = strings.ToUpper(projectKey)

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; pass --repo or set the context default")
	}

	client, err := bbdc.New(bbdc.Options{
		BaseURL:  host.BaseURL,
		Username: host.Username,
		Token:    host.Token,
	})
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	repo, err := client.GetRepository(ctx, projectKey, repoSlug)
	if err != nil {
		return err
	}

	formatOpt, err := cmdutil.OutputFormat(cmd)
	if err != nil {
		return err
	}

	type repoDetails struct {
		Project string   `json:"project"`
		Slug    string   `json:"slug"`
		Name    string   `json:"name"`
		ID      int      `json:"id"`
		WebURL  string   `json:"web_url,omitempty"`
		Clone   []string `json:"clone_urls,omitempty"`
	}

	details := repoDetails{
		Project: repo.Project.Key,
		Slug:    repo.Slug,
		Name:    repo.Name,
		ID:      repo.ID,
		WebURL:  firstLink(*repo, "web"),
		Clone:   cloneLinks(*repo),
	}

	if formatOpt != "" {
		return format.Write(ios.Out, formatOpt, details, nil)
	}

	fmt.Fprintf(ios.Out, "%s/%s (%d)\n", details.Project, details.Slug, details.ID)
	fmt.Fprintf(ios.Out, "Name: %s\n", details.Name)
	if details.WebURL != "" {
		fmt.Fprintf(ios.Out, "Web:  %s\n", details.WebURL)
	}
	if len(details.Clone) > 0 {
		for _, url := range details.Clone {
			fmt.Fprintf(ios.Out, "Clone: %s\n", url)
		}
	}
	return nil
}

func runClone(cmd *cobra.Command, f *cmdutil.Factory, opts *cloneOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("repo clone currently supports Data Center contexts only")
	}

	projectKey := strings.TrimSpace(opts.Project)
	if projectKey == "" {
		projectKey = ctxCfg.ProjectKey
	}
	if projectKey == "" {
		return fmt.Errorf("project key required; set with --project or configure the context default")
	}

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; pass argument or set the context default")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	repo, err := client.GetRepository(ctx, projectKey, repoSlug)
	if err != nil {
		return err
	}

	var cloneURL string
	desired := "http"
	if opts.UseSSH {
		desired = "ssh"
	}
	for _, link := range repo.Links.Clone {
		if strings.EqualFold(link.Name, desired) {
			cloneURL = link.Href
			break
		}
	}
	if cloneURL == "" {
		return fmt.Errorf("no %s clone URL available", desired)
	}

	args := []string{"clone", cloneURL}
	if opts.Dest != "" {
		args = append(args, opts.Dest)
	}

	cmdExec := exec.CommandContext(cmd.Context(), "git", args...)
	cmdExec.Stdout = ios.Out
	cmdExec.Stderr = ios.ErrOut
	cmdExec.Stdin = ios.In

	return cmdExec.Run()
}

func runBrowse(cmd *cobra.Command, f *cmdutil.Factory) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("repo browse currently supports Data Center contexts only")
	}

	projectKey := ctxCfg.ProjectKey
	repoSlug := ctxCfg.DefaultRepo
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("context must define project and default repo")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()
	repo, err := client.GetRepository(ctx, projectKey, repoSlug)
	if err != nil {
		return err
	}

	if link := firstLink(*repo, "web"); link != "" {
		fmt.Fprintln(ios.Out, link)
		return nil
	}

	return fmt.Errorf("repository does not expose a web URL")
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	var opts createOptions

	cmd := &cobra.Command{
		Use:   "create <repository>",
		Short: "Create a new repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoSlug := args[0]
			return runCreate(cmd, f, repoSlug, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Description, "description", "", "Repository description")
	cmd.Flags().BoolVar(&opts.Public, "public", false, "Create repository as public")
	cmd.Flags().BoolVar(&opts.Forkable, "forkable", false, "Allow forking of the repository")
	cmd.Flags().StringVar(&opts.DefaultBranch, "default-branch", "", "Default branch to set after creation")
	cmd.Flags().StringVar(&opts.SCM, "scm", "git", "SCM type (git)")

	return cmd
}

func runCreate(cmd *cobra.Command, f *cmdutil.Factory, slug string, opts createOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("repo create currently supports Data Center contexts only")
	}

	projectKey := strings.TrimSpace(opts.Project)
	if projectKey == "" {
		projectKey = ctxCfg.ProjectKey
	}
	if projectKey == "" {
		return fmt.Errorf("project key required; set with --project or configure the context default")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	input := bbdc.CreateRepositoryInput{
		Name:          slug,
		SCMID:         opts.SCM,
		Description:   opts.Description,
		Public:        opts.Public,
		Forkable:      opts.Forkable,
		DefaultBranch: opts.DefaultBranch,
	}

	repo, err := client.CreateRepository(ctx, projectKey, input)
	if err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Created %s/%s\n", repo.Project.Key, repo.Slug)
	if repo.DefaultBranch != "" {
		fmt.Fprintf(ios.Out, "  default branch: %s\n", repo.DefaultBranch)
	}
	for _, clone := range cloneLinks(*repo) {
		fmt.Fprintf(ios.Out, "  clone: %s\n", clone)
	}
	return nil
}

func newCloneCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &cloneOptions{}
	cmd := &cobra.Command{
		Use:   "clone <repository>",
		Short: "Clone a repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Repo = args[0]
			return runClone(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().BoolVar(&opts.UseSSH, "ssh", false, "Use SSH clone URL")
	cmd.Flags().StringVar(&opts.Dest, "dest", "", "Destination directory")

	return cmd
}

func newBrowseCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "browse",
		Short: "Print the repository web URL",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBrowse(cmd, f)
		},
	}
	return cmd
}

func firstLink(repo bbdc.Repository, kind string) string {
	switch kind {
	case "web":
		if len(repo.Links.Web) > 0 {
			return repo.Links.Web[0].Href
		}
		if len(repo.Links.Self) > 0 {
			return repo.Links.Self[0].Href
		}
	}
	return ""
}

func cloneLinks(repo bbdc.Repository) []string {
	var urls []string
	for _, link := range repo.Links.Clone {
		if strings.TrimSpace(link.Href) == "" {
			continue
		}
		urls = append(urls, fmt.Sprintf("%s (%s)", link.Href, link.Name))
	}
	return urls
}

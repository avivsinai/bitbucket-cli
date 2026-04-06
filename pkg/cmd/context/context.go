package context

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdContext returns the context management command tree.
func NewCmdContext(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Manage Bitbucket CLI contexts",
		Long: `Manage named contexts that store connection defaults for different Bitbucket
hosts, projects, workspaces, and repositories. Each context bundles a host with
its associated scope so you can switch between environments without repeating
flags on every command.`,
		Example: `  # Create a Data Center context and make it active
  bkt context create work --host bitbucket.mycompany.com --project TEAM --set-active

  # List all configured contexts
  bkt context list

  # Switch to a different context
  bkt context use personal`,
	}

	cmd.AddCommand(newCreateCmd(f))
	cmd.AddCommand(newUseCmd(f))
	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newDeleteCmd(f))

	return cmd
}

type createOptions struct {
	Host      string
	Project   string
	Workspace string
	Repo      string
	SetActive bool
}

func newCreateCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &createOptions{}
	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new CLI context",
		Long: `Create a named context that stores connection defaults for a Bitbucket host.
A context binds a host to a project (Data Center) or workspace (Cloud) and an
optional default repository, so subsequent commands inherit these values
without requiring flags.`,
		Example: `  # Create a Data Center context
  bkt context create work --host bitbucket.mycompany.com --project TEAM

  # Create a Cloud context with a default repository
  bkt context create oss --host bitbucket.org --workspace my-team --repo api-service

  # Create a context and immediately make it active
  bkt context create staging --host staging.bb.internal --project OPS --set-active`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCreate(cmd, f, args[0], opts)
		},
	}

	cmd.Flags().StringVar(&opts.Host, "host", "", "Host key or base URL (required)")
	cmd.Flags().StringVar(&opts.Project, "project", "", "Default Bitbucket project key (Data Center)")
	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Default Bitbucket workspace (Cloud)")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Default repository slug")
	cmd.Flags().BoolVar(&opts.SetActive, "set-active", false, "Set the new context as active")

	return cmd
}

func runCreate(cmd *cobra.Command, f *cmdutil.Factory, name string, opts *createOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	hostKey := strings.TrimSpace(opts.Host)
	if hostKey == "" {
		return fmt.Errorf("--host is required")
	}

	host, ok := cfg.Hosts[hostKey]
	if !ok {
		baseURL, err := cmdutil.NormalizeBaseURL(hostKey)
		if err != nil {
			return fmt.Errorf("host %q not found; run `%s auth login` first", hostKey, f.ExecutableName)
		}
		hostKey, err = cmdutil.HostKeyFromURL(baseURL)
		if err != nil {
			return err
		}
		host, ok = cfg.Hosts[hostKey]
		if !ok {
			return fmt.Errorf("host %q not found; run `%s auth login` first", opts.Host, f.ExecutableName)
		}
	}

	ctx := &config.Context{
		Host:        hostKey,
		DefaultRepo: strings.TrimSpace(opts.Repo),
	}

	switch host.Kind {
	case "dc":
		if opts.Project == "" {
			return fmt.Errorf("--project is required for Data Center contexts")
		}
		ctx.ProjectKey = strings.ToUpper(opts.Project)
	case "cloud":
		if opts.Workspace == "" {
			return fmt.Errorf("--workspace is required for Bitbucket Cloud contexts")
		}
		ctx.Workspace = opts.Workspace
	default:
		return fmt.Errorf("unknown host kind %q", host.Kind)
	}

	cfg.SetContext(name, ctx)

	if opts.SetActive || cfg.ActiveContext == "" {
		if err := cfg.SetActiveContext(name); err != nil {
			return err
		}
	}

	if err := cfg.Save(); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Created context %q (host: %s)\n", name, hostKey); err != nil {
		return err
	}
	if cfg.ActiveContext == name {
		if _, err := fmt.Fprintf(ios.Out, "✓ Context %q is now active\n", name); err != nil {
			return err
		}
	}
	return nil
}

func newUseCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use <name>",
		Short: "Activate an existing context",
		Long: `Set the named context as the active context. All subsequent commands will
resolve their host, project/workspace, and repository defaults from this
context unless overridden with explicit flags.`,
		Example: `  # Switch to the "work" context
  bkt context use work

  # Switch to a personal Cloud context
  bkt context use personal`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUse(cmd, f, args[0])
		},
	}
	return cmd
}

func runUse(cmd *cobra.Command, f *cmdutil.Factory, name string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	if err := cfg.SetActiveContext(name); err != nil {
		return err
	}

	if err := cfg.Save(); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Activated context %q\n", name); err != nil {
		return err
	}
	return nil
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available contexts",
		Long: `List all configured contexts along with their host, project or workspace,
and default repository. The currently active context is marked with an
asterisk (*).`,
		Example: `  # List all contexts
  bkt context list

  # List contexts using the short alias
  bkt context ls

  # List contexts as JSON
  bkt context list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, f)
		},
	}
	return cmd
}

func runList(cmd *cobra.Command, f *cmdutil.Factory) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	type summary struct {
		Name        string `json:"name"`
		Host        string `json:"host"`
		ProjectKey  string `json:"project_key,omitempty"`
		Workspace   string `json:"workspace,omitempty"`
		DefaultRepo string `json:"default_repo,omitempty"`
		Active      bool   `json:"active"`
	}

	var names []string
	for name := range cfg.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)

	var contexts []summary
	for _, name := range names {
		ctx := cfg.Contexts[name]
		contexts = append(contexts, summary{
			Name:        name,
			Host:        ctx.Host,
			ProjectKey:  ctx.ProjectKey,
			Workspace:   ctx.Workspace,
			DefaultRepo: ctx.DefaultRepo,
			Active:      cfg.ActiveContext == name,
		})
	}

	payload := struct {
		Active   string    `json:"active_context,omitempty"`
		Contexts []summary `json:"contexts"`
	}{
		Active:   cfg.ActiveContext,
		Contexts: contexts,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(contexts) == 0 {
			_, err := fmt.Fprintf(ios.Out, "No contexts configured. Use `%s context create` to add one.\n", f.ExecutableName)
			return err
		}

		for _, ctx := range contexts {
			marker := " "
			if ctx.Active {
				marker = "*"
			}
			if _, err := fmt.Fprintf(ios.Out, "%s %s (host: %s)\n", marker, ctx.Name, ctx.Host); err != nil {
				return err
			}
			if ctx.ProjectKey != "" {
				if _, err := fmt.Fprintf(ios.Out, "    project: %s\n", ctx.ProjectKey); err != nil {
					return err
				}
			}
			if ctx.Workspace != "" {
				if _, err := fmt.Fprintf(ios.Out, "    workspace: %s\n", ctx.Workspace); err != nil {
					return err
				}
			}
			if ctx.DefaultRepo != "" {
				if _, err := fmt.Fprintf(ios.Out, "    repo: %s\n", ctx.DefaultRepo); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete <name>",
		Aliases: []string{"rm"},
		Short:   "Delete a context",
		Long: `Remove a named context from the configuration. If the deleted context is
currently active, the active context is cleared and you will need to select
another one with "bkt context use".`,
		Example: `  # Delete a context by name
  bkt context delete old-server

  # Delete using the short alias
  bkt context rm staging`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDelete(cmd, f, args[0])
		},
	}
	return cmd
}

func runDelete(cmd *cobra.Command, f *cmdutil.Factory, name string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	if _, err := cfg.Context(name); err != nil {
		return err
	}

	cfg.DeleteContext(name)

	if err := cfg.Save(); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Deleted context %q\n", name); err != nil {
		return err
	}
	return nil
}

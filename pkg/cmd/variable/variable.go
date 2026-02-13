package variable

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCommand creates the variable command.
func NewCommand(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "variable",
		Short: "Manage pipeline variables",
		Long: `Create and manage pipeline variables in Bitbucket Cloud repositories.

Pipeline variables are available during pipeline execution. Variables can be
defined at three scopes:
  - Repository: Available to all pipelines in the repository
  - Workspace: Available to all pipelines in the workspace
  - Deployment: Available only during deployment to a specific environment

Note: Pipeline variables are only available for Bitbucket Cloud.`,
	}

	cmd.AddCommand(newListCmd(f))
	cmd.AddCommand(newGetCmd(f))
	cmd.AddCommand(newDeleteCmd(f))
	cmd.AddCommand(newSetCmd(f))

	return cmd
}

// Valid scope values
const (
	scopeRepository = "repository"
	scopeWorkspace  = "workspace"
	scopeDeployment = "deployment"
)

// resolveDeploymentEnvironment finds a deployment environment by name and returns its UUID.
func resolveDeploymentEnvironment(ctx context.Context, client *bbcloud.Client, workspace, repoSlug, envName string) (string, error) {
	environments, err := client.ListDeploymentEnvironments(ctx, workspace, repoSlug)
	if err != nil {
		return "", fmt.Errorf("failed to list deployment environments: %w", err)
	}

	for _, env := range environments {
		if strings.EqualFold(env.Name, envName) || strings.EqualFold(env.Slug, envName) {
			return env.UUID, nil
		}
	}

	var names []string
	for _, env := range environments {
		names = append(names, env.Name)
	}
	if len(names) == 0 {
		return "", fmt.Errorf("deployment environment %q not found; no environments configured", envName)
	}
	return "", fmt.Errorf("deployment environment %q not found; available: %s", envName, strings.Join(names, ", "))
}

// --- List Command ---

type listOptions struct {
	Workspace  string
	Repo       string
	Scope      string
	Deployment string
	Limit      int
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{
		Limit: 30,
		Scope: scopeRepository,
	}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List pipeline variables",
		Example: `  # List repository variables (default)
  bkt variable list

  # List workspace variables
  bkt variable list --scope workspace

  # List deployment environment variables
  bkt variable list --deployment production

  # List variables in JSON format
  bkt variable list --json

  # List variables for a specific repository
  bkt variable list --repo my-repo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If --deployment is set, override scope
			if opts.Deployment != "" {
				opts.Scope = scopeDeployment
			}
			return runList(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository slug")
	cmd.Flags().StringVar(&opts.Scope, "scope", opts.Scope, "Variable scope: repository, workspace, or deployment")
	cmd.Flags().StringVarP(&opts.Deployment, "deployment", "e", "", "Deployment environment name (implies --scope deployment)")
	cmd.Flags().IntVarP(&opts.Limit, "limit", "L", opts.Limit, "Maximum variables to display")

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

	if host.Kind != "cloud" {
		return fmt.Errorf("pipeline variables are only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	// Validate scope
	scope := strings.ToLower(strings.TrimSpace(opts.Scope))
	if scope != scopeRepository && scope != scopeWorkspace && scope != scopeDeployment {
		return fmt.Errorf("invalid scope %q; must be 'repository', 'workspace', or 'deployment'", opts.Scope)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	var repoSlug string
	if scope == scopeRepository || scope == scopeDeployment {
		repoSlug = strings.TrimSpace(opts.Repo)
		if repoSlug == "" {
			repoSlug = ctxCfg.DefaultRepo
		}
		if repoSlug == "" {
			return fmt.Errorf("repository slug required; set with --repo or configure the context default")
		}
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	var variables []bbcloud.PipelineVariable
	var envUUID string
	switch scope {
	case scopeWorkspace:
		variables, err = client.ListWorkspaceVariables(ctx, workspace, bbcloud.VariableListOptions{
			Limit: opts.Limit,
		})
	case scopeDeployment:
		if opts.Deployment == "" {
			return fmt.Errorf("deployment environment name is required; use --deployment")
		}
		envUUID, err = resolveDeploymentEnvironment(ctx, client, workspace, repoSlug, opts.Deployment)
		if err != nil {
			return err
		}
		variables, err = client.ListDeploymentVariables(ctx, workspace, repoSlug, envUUID, bbcloud.VariableListOptions{
			Limit: opts.Limit,
		})
	default:
		variables, err = client.ListRepositoryVariables(ctx, workspace, repoSlug, bbcloud.VariableListOptions{
			Limit: opts.Limit,
		})
	}
	if err != nil {
		return err
	}

	type variableSummary struct {
		UUID    string `json:"uuid"`
		Key     string `json:"key"`
		Value   string `json:"value,omitempty"`
		Secured bool   `json:"secured"`
	}

	var summaries []variableSummary
	for _, v := range variables {
		summaries = append(summaries, variableSummary{
			UUID:    v.UUID,
			Key:     v.Key,
			Value:   v.Value,
			Secured: v.Secured,
		})
	}

	payload := struct {
		Workspace  string            `json:"workspace"`
		Repository string            `json:"repository,omitempty"`
		Deployment string            `json:"deployment,omitempty"`
		Scope      string            `json:"scope"`
		Variables  []variableSummary `json:"variables"`
	}{
		Workspace:  workspace,
		Repository: repoSlug,
		Deployment: opts.Deployment,
		Scope:      scope,
		Variables:  summaries,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		location := workspace
		switch scope {
		case scopeRepository:
			location = workspace + "/" + repoSlug
		case scopeDeployment:
			location = workspace + "/" + repoSlug + " (deployment: " + opts.Deployment + ")"
		}
		if len(summaries) == 0 {
			_, err := fmt.Fprintf(ios.Out, "No %s variables found in %s.\n", scope, location)
			return err
		}

		for _, v := range summaries {
			value := v.Value
			if v.Secured {
				value = "********"
			}
			if _, err := fmt.Fprintf(ios.Out, "%s\t%s\t(secured=%v)\n", v.Key, value, v.Secured); err != nil {
				return err
			}
		}
		return nil
	})
}

// --- Get Command ---

type getOptions struct {
	Workspace  string
	Repo       string
	Scope      string
	Deployment string
}

func newGetCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &getOptions{
		Scope: scopeRepository,
	}
	cmd := &cobra.Command{
		Use:   "get <variable-name>",
		Short: "Get a pipeline variable",
		Example: `  # Get a repository variable by name
  bkt variable get MY_VAR

  # Get a workspace variable
  bkt variable get MY_VAR --scope workspace

  # Get a deployment environment variable
  bkt variable get MY_VAR --deployment production

  # Get a variable in JSON format
  bkt variable get MY_VAR --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Deployment != "" {
				opts.Scope = scopeDeployment
			}
			return runGet(cmd, f, opts, args[0])
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository slug")
	cmd.Flags().StringVar(&opts.Scope, "scope", opts.Scope, "Variable scope: repository, workspace, or deployment")
	cmd.Flags().StringVarP(&opts.Deployment, "deployment", "e", "", "Deployment environment name (implies --scope deployment)")

	return cmd
}

func runGet(cmd *cobra.Command, f *cmdutil.Factory, opts *getOptions, variableName string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	if host.Kind != "cloud" {
		return fmt.Errorf("pipeline variables are only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	// Validate scope
	scope := strings.ToLower(strings.TrimSpace(opts.Scope))
	if scope != scopeRepository && scope != scopeWorkspace && scope != scopeDeployment {
		return fmt.Errorf("invalid scope %q; must be 'repository', 'workspace', or 'deployment'", opts.Scope)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	var repoSlug string
	if scope == scopeRepository || scope == scopeDeployment {
		repoSlug = strings.TrimSpace(opts.Repo)
		if repoSlug == "" {
			repoSlug = ctxCfg.DefaultRepo
		}
		if repoSlug == "" {
			return fmt.Errorf("repository slug required; set with --repo or configure the context default")
		}
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	// List all variables and find the one with matching key
	var variables []bbcloud.PipelineVariable
	var envUUID string
	switch scope {
	case scopeWorkspace:
		variables, err = client.ListWorkspaceVariables(ctx, workspace, bbcloud.VariableListOptions{})
	case scopeDeployment:
		if opts.Deployment == "" {
			return fmt.Errorf("deployment environment name is required; use --deployment")
		}
		envUUID, err = resolveDeploymentEnvironment(ctx, client, workspace, repoSlug, opts.Deployment)
		if err != nil {
			return err
		}
		variables, err = client.ListDeploymentVariables(ctx, workspace, repoSlug, envUUID, bbcloud.VariableListOptions{})
	default:
		variables, err = client.ListRepositoryVariables(ctx, workspace, repoSlug, bbcloud.VariableListOptions{})
	}
	if err != nil {
		return err
	}

	var found *bbcloud.PipelineVariable
	for i := range variables {
		if variables[i].Key == variableName {
			found = &variables[i]
			break
		}
	}

	location := workspace
	switch scope {
	case scopeRepository:
		location = workspace + "/" + repoSlug
	case scopeDeployment:
		location = workspace + "/" + repoSlug + " (deployment: " + opts.Deployment + ")"
	}

	if found == nil {
		return fmt.Errorf("variable %q not found in %s", variableName, location)
	}

	payload := struct {
		UUID       string `json:"uuid"`
		Key        string `json:"key"`
		Value      string `json:"value,omitempty"`
		Secured    bool   `json:"secured"`
		Workspace  string `json:"workspace"`
		Repo       string `json:"repository,omitempty"`
		Deployment string `json:"deployment,omitempty"`
		Scope      string `json:"scope"`
	}{
		UUID:       found.UUID,
		Key:        found.Key,
		Value:      found.Value,
		Secured:    found.Secured,
		Workspace:  workspace,
		Repo:       repoSlug,
		Deployment: opts.Deployment,
		Scope:      scope,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		value := found.Value
		if found.Secured {
			value = "********"
			_, _ = fmt.Fprintf(ios.ErrOut, "Note: Variable %q is secured; value is not retrievable.\n", found.Key)
		}
		_, err := fmt.Fprintf(ios.Out, "%s=%s\n", found.Key, value)
		return err
	})
}

// --- Delete Command ---

type deleteOptions struct {
	Workspace  string
	Repo       string
	Scope      string
	Deployment string
	Yes        bool
}

func newDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &deleteOptions{
		Scope: scopeRepository,
	}
	cmd := &cobra.Command{
		Use:     "delete <variable-name>",
		Aliases: []string{"rm"},
		Short:   "Delete a pipeline variable",
		Example: `  # Delete a repository variable (will prompt for confirmation)
  bkt variable delete MY_VAR

  # Delete a workspace variable
  bkt variable delete MY_VAR --scope workspace

  # Delete a deployment environment variable
  bkt variable delete MY_VAR --deployment production

  # Delete without confirmation
  bkt variable delete MY_VAR --yes`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Deployment != "" {
				opts.Scope = scopeDeployment
			}
			return runDelete(cmd, f, opts, args[0])
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository slug")
	cmd.Flags().StringVar(&opts.Scope, "scope", opts.Scope, "Variable scope: repository, workspace, or deployment")
	cmd.Flags().StringVarP(&opts.Deployment, "deployment", "e", "", "Deployment environment name (implies --scope deployment)")
	cmd.Flags().BoolVarP(&opts.Yes, "yes", "y", false, "Skip confirmation prompt")

	return cmd
}

func runDelete(cmd *cobra.Command, f *cmdutil.Factory, opts *deleteOptions, variableName string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	if host.Kind != "cloud" {
		return fmt.Errorf("pipeline variables are only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	// Validate scope
	scope := strings.ToLower(strings.TrimSpace(opts.Scope))
	if scope != scopeRepository && scope != scopeWorkspace && scope != scopeDeployment {
		return fmt.Errorf("invalid scope %q; must be 'repository', 'workspace', or 'deployment'", opts.Scope)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	var repoSlug string
	if scope == scopeRepository || scope == scopeDeployment {
		repoSlug = strings.TrimSpace(opts.Repo)
		if repoSlug == "" {
			repoSlug = ctxCfg.DefaultRepo
		}
		if repoSlug == "" {
			return fmt.Errorf("repository slug required; set with --repo or configure the context default")
		}
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	// List all variables and find the one with matching key
	var variables []bbcloud.PipelineVariable
	var envUUID string
	switch scope {
	case scopeWorkspace:
		variables, err = client.ListWorkspaceVariables(ctx, workspace, bbcloud.VariableListOptions{})
	case scopeDeployment:
		if opts.Deployment == "" {
			return fmt.Errorf("deployment environment name is required; use --deployment")
		}
		envUUID, err = resolveDeploymentEnvironment(ctx, client, workspace, repoSlug, opts.Deployment)
		if err != nil {
			return err
		}
		variables, err = client.ListDeploymentVariables(ctx, workspace, repoSlug, envUUID, bbcloud.VariableListOptions{})
	default:
		variables, err = client.ListRepositoryVariables(ctx, workspace, repoSlug, bbcloud.VariableListOptions{})
	}
	if err != nil {
		return err
	}

	var found *bbcloud.PipelineVariable
	for i := range variables {
		if variables[i].Key == variableName {
			found = &variables[i]
			break
		}
	}

	location := workspace
	switch scope {
	case scopeRepository:
		location = workspace + "/" + repoSlug
	case scopeDeployment:
		location = workspace + "/" + repoSlug + " (deployment: " + opts.Deployment + ")"
	}

	if found == nil {
		return fmt.Errorf("variable %q not found in %s", variableName, location)
	}

	if !opts.Yes {
		prompter := f.Prompt()
		confirmed, err := prompter.Confirm(fmt.Sprintf("Delete variable %q from %s?", found.Key, location), false)
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintln(ios.Out, "Aborted.")
			return nil
		}
	}

	switch scope {
	case scopeWorkspace:
		err = client.DeleteWorkspaceVariable(ctx, workspace, found.UUID)
	case scopeDeployment:
		err = client.DeleteDeploymentVariable(ctx, workspace, repoSlug, envUUID, found.UUID)
	default:
		err = client.DeleteRepositoryVariable(ctx, workspace, repoSlug, found.UUID)
	}
	if err != nil {
		return err
	}

	payload := struct {
		Key        string `json:"key"`
		Deleted    bool   `json:"deleted"`
		Workspace  string `json:"workspace"`
		Repo       string `json:"repository,omitempty"`
		Deployment string `json:"deployment,omitempty"`
		Scope      string `json:"scope"`
	}{
		Key:        found.Key,
		Deleted:    true,
		Workspace:  workspace,
		Repo:       repoSlug,
		Deployment: opts.Deployment,
		Scope:      scope,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		_, err := fmt.Fprintf(ios.Out, "Deleted variable %q from %s.\n", found.Key, location)
		return err
	})
}

// --- Set Command ---

type setOptions struct {
	Workspace  string
	Repo       string
	Scope      string
	Deployment string
	Body       string
	Secured    bool
	EnvFile    string
}

func newSetCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &setOptions{
		Scope: scopeRepository,
	}
	cmd := &cobra.Command{
		Use:   "set [<variable-name>]",
		Short: "Create or update a pipeline variable",
		Long: `Create or update a pipeline variable.

If the variable already exists, it will be updated. Otherwise, a new variable
will be created.

The value can be provided via:
  - The --body flag
  - Standard input (if --body is not specified)
  - Interactive prompt (if running in a TTY)

You can also import multiple variables at once from an env file using the
--env-file flag. The file should contain KEY=VALUE pairs, one per line.
Lines starting with # are treated as comments.`,
		Example: `  # Set a repository variable with a value
  bkt variable set MY_VAR --body "my-value"

  # Set a workspace variable
  bkt variable set MY_VAR --body "my-value" --scope workspace

  # Set a deployment environment variable
  bkt variable set MY_VAR --body "my-value" --deployment production

  # Set a secured/secret variable
  bkt variable set MY_SECRET --body "secret-value" --secured

  # Set a variable from stdin
  echo "my-value" | bkt variable set MY_VAR

  # Import variables from an env file
  bkt variable set --env-file .env`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Deployment != "" {
				opts.Scope = scopeDeployment
			}
			if opts.Body != "" && opts.EnvFile != "" {
				return fmt.Errorf("cannot specify both --body and --env-file")
			}
			if opts.EnvFile != "" {
				if len(args) > 0 {
					return fmt.Errorf("cannot specify both variable name and --env-file")
				}
				return runSetFromEnvFile(cmd, f, opts)
			}
			if len(args) == 0 {
				return fmt.Errorf("variable name is required; use --env-file to import from a file")
			}
			return runSet(cmd, f, opts, args[0])
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVarP(&opts.Repo, "repo", "R", "", "Repository slug")
	cmd.Flags().StringVar(&opts.Scope, "scope", opts.Scope, "Variable scope: repository, workspace, or deployment")
	cmd.Flags().StringVarP(&opts.Deployment, "deployment", "e", "", "Deployment environment name (implies --scope deployment)")
	cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Variable value")
	cmd.Flags().BoolVarP(&opts.Secured, "secured", "s", false, "Mark variable as secured (secret)")
	cmd.Flags().StringVarP(&opts.EnvFile, "env-file", "f", "", "Path to env file containing KEY=VALUE pairs")

	return cmd
}

func runSet(cmd *cobra.Command, f *cmdutil.Factory, opts *setOptions, variableName string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	if host.Kind != "cloud" {
		return fmt.Errorf("pipeline variables are only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	// Validate scope
	scope := strings.ToLower(strings.TrimSpace(opts.Scope))
	if scope != scopeRepository && scope != scopeWorkspace && scope != scopeDeployment {
		return fmt.Errorf("invalid scope %q; must be 'repository', 'workspace', or 'deployment'", opts.Scope)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	var repoSlug string
	if scope == scopeRepository || scope == scopeDeployment {
		repoSlug = strings.TrimSpace(opts.Repo)
		if repoSlug == "" {
			repoSlug = ctxCfg.DefaultRepo
		}
		if repoSlug == "" {
			return fmt.Errorf("repository slug required; set with --repo or configure the context default")
		}
	}

	// Validate variable name
	if err := validateVariableKey(variableName); err != nil {
		return err
	}

	// Get value from --body flag, stdin, or interactive prompt
	value := opts.Body
	if value == "" {
		// Try to read from stdin if not a TTY
		if !ios.CanPrompt() {
			data, err := io.ReadAll(ios.In)
			if err != nil {
				return fmt.Errorf("failed to read from stdin: %w", err)
			}
			value = strings.TrimSuffix(string(data), "\n")
		} else {
			// Interactive prompt
			prompter := f.Prompt()
			promptStr := fmt.Sprintf("Value for %s", variableName)
			var err error
			if opts.Secured {
				// Use password prompt to avoid exposing secret in terminal
				value, err = prompter.Password(promptStr)
			} else {
				value, err = prompter.Input(promptStr, "")
			}
			if err != nil {
				return err
			}
		}
	}

	if value == "" {
		return fmt.Errorf("variable value is required; use --body to specify the value")
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	// Check if variable already exists
	var variables []bbcloud.PipelineVariable
	var envUUID string
	switch scope {
	case scopeWorkspace:
		variables, err = client.ListWorkspaceVariables(ctx, workspace, bbcloud.VariableListOptions{})
	case scopeDeployment:
		if opts.Deployment == "" {
			return fmt.Errorf("deployment environment name is required; use --deployment")
		}
		envUUID, err = resolveDeploymentEnvironment(ctx, client, workspace, repoSlug, opts.Deployment)
		if err != nil {
			return err
		}
		variables, err = client.ListDeploymentVariables(ctx, workspace, repoSlug, envUUID, bbcloud.VariableListOptions{})
	default:
		variables, err = client.ListRepositoryVariables(ctx, workspace, repoSlug, bbcloud.VariableListOptions{})
	}
	if err != nil {
		return err
	}

	var existing *bbcloud.PipelineVariable
	for i := range variables {
		if variables[i].Key == variableName {
			existing = &variables[i]
			break
		}
	}

	var result *bbcloud.PipelineVariable
	var action string

	if existing != nil {
		// Update existing variable
		// Preserve existing secured state unless --secured was explicitly set
		secured := existing.Secured
		if cmd.Flags().Changed("secured") {
			secured = opts.Secured
		}
		switch scope {
		case scopeWorkspace:
			result, err = client.UpdateWorkspaceVariable(ctx, workspace, existing.UUID, bbcloud.UpdateWorkspaceVariableInput{
				Key:     variableName,
				Value:   value,
				Secured: secured,
			})
		case scopeDeployment:
			result, err = client.UpdateDeploymentVariable(ctx, workspace, repoSlug, envUUID, existing.UUID, bbcloud.UpdateDeploymentVariableInput{
				Key:     variableName,
				Value:   value,
				Secured: secured,
			})
		default:
			result, err = client.UpdateRepositoryVariable(ctx, workspace, repoSlug, existing.UUID, bbcloud.UpdateRepositoryVariableInput{
				Key:     variableName,
				Value:   value,
				Secured: secured,
			})
		}
		if err != nil {
			return err
		}
		action = "Updated"
	} else {
		// Create new variable
		switch scope {
		case scopeWorkspace:
			result, err = client.CreateWorkspaceVariable(ctx, workspace, bbcloud.CreateWorkspaceVariableInput{
				Key:     variableName,
				Value:   value,
				Secured: opts.Secured,
			})
		case scopeDeployment:
			result, err = client.CreateDeploymentVariable(ctx, workspace, repoSlug, envUUID, bbcloud.CreateDeploymentVariableInput{
				Key:     variableName,
				Value:   value,
				Secured: opts.Secured,
			})
		default:
			result, err = client.CreateRepositoryVariable(ctx, workspace, repoSlug, bbcloud.CreateRepositoryVariableInput{
				Key:     variableName,
				Value:   value,
				Secured: opts.Secured,
			})
		}
		if err != nil {
			return err
		}
		action = "Created"
	}

	location := workspace
	switch scope {
	case scopeRepository:
		location = workspace + "/" + repoSlug
	case scopeDeployment:
		location = workspace + "/" + repoSlug + " (deployment: " + opts.Deployment + ")"
	}

	payload := struct {
		UUID       string `json:"uuid"`
		Key        string `json:"key"`
		Secured    bool   `json:"secured"`
		Action     string `json:"action"`
		Workspace  string `json:"workspace"`
		Repo       string `json:"repository,omitempty"`
		Deployment string `json:"deployment,omitempty"`
		Scope      string `json:"scope"`
	}{
		UUID:       result.UUID,
		Key:        result.Key,
		Secured:    result.Secured,
		Action:     strings.ToLower(action),
		Workspace:  workspace,
		Repo:       repoSlug,
		Deployment: opts.Deployment,
		Scope:      scope,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		_, err := fmt.Fprintf(ios.Out, "%s variable %q in %s.\n", action, result.Key, location)
		return err
	})
}

// validateVariableKey checks if a variable key is valid.
// Variable keys must start with a letter and contain only letters, numbers, and underscores.
func validateVariableKey(key string) error {
	if key == "" {
		return fmt.Errorf("variable key cannot be empty")
	}

	// Must start with a letter
	if (key[0] < 'a' || key[0] > 'z') && (key[0] < 'A' || key[0] > 'Z') {
		return fmt.Errorf("variable key must start with a letter: %q", key)
	}

	// Must contain only letters, numbers, and underscores
	for i, c := range key {
		isLetter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		isDigit := c >= '0' && c <= '9'
		if !isLetter && !isDigit && c != '_' {
			return fmt.Errorf("variable key contains invalid character at position %d: %q", i, key)
		}
	}

	return nil
}

// runSetFromEnvFile imports variables from an env file.
func runSetFromEnvFile(cmd *cobra.Command, f *cmdutil.Factory, opts *setOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	if host.Kind != "cloud" {
		return fmt.Errorf("pipeline variables are only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	// Validate scope
	scope := strings.ToLower(strings.TrimSpace(opts.Scope))
	if scope != scopeRepository && scope != scopeWorkspace && scope != scopeDeployment {
		return fmt.Errorf("invalid scope %q; must be 'repository', 'workspace', or 'deployment'", opts.Scope)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	var repoSlug string
	if scope == scopeRepository || scope == scopeDeployment {
		repoSlug = strings.TrimSpace(opts.Repo)
		if repoSlug == "" {
			repoSlug = ctxCfg.DefaultRepo
		}
		if repoSlug == "" {
			return fmt.Errorf("repository slug required; set with --repo or configure the context default")
		}
	}

	// Parse env file
	envVars, err := parseEnvFile(opts.EnvFile)
	if err != nil {
		return err
	}

	if len(envVars) == 0 {
		return fmt.Errorf("no variables found in %s", opts.EnvFile)
	}

	// Validate all keys first
	for key := range envVars {
		if err := validateVariableKey(key); err != nil {
			return fmt.Errorf("invalid variable in env file: %w", err)
		}
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
	defer cancel()

	// Get existing variables
	var existingVars []bbcloud.PipelineVariable
	var envUUID string
	switch scope {
	case scopeWorkspace:
		existingVars, err = client.ListWorkspaceVariables(ctx, workspace, bbcloud.VariableListOptions{})
	case scopeDeployment:
		if opts.Deployment == "" {
			return fmt.Errorf("deployment environment name is required; use --deployment")
		}
		envUUID, err = resolveDeploymentEnvironment(ctx, client, workspace, repoSlug, opts.Deployment)
		if err != nil {
			return err
		}
		existingVars, err = client.ListDeploymentVariables(ctx, workspace, repoSlug, envUUID, bbcloud.VariableListOptions{})
	default:
		existingVars, err = client.ListRepositoryVariables(ctx, workspace, repoSlug, bbcloud.VariableListOptions{})
	}
	if err != nil {
		return err
	}

	existingByKey := make(map[string]*bbcloud.PipelineVariable)
	for i := range existingVars {
		existingByKey[existingVars[i].Key] = &existingVars[i]
	}

	// Process each variable
	type varResult struct {
		Key     string `json:"key"`
		Action  string `json:"action"`
		Secured bool   `json:"secured"`
	}

	// Determine if --secured was explicitly set
	securedFlagChanged := cmd.Flags().Changed("secured")

	var results []varResult
	for key, value := range envVars {
		existing := existingByKey[key]
		var action string
		var secured bool

		if existing != nil {
			// Update existing variable
			// Preserve existing secured state unless --secured was explicitly set
			secured = existing.Secured
			if securedFlagChanged {
				secured = opts.Secured
			}
			switch scope {
			case scopeWorkspace:
				_, err = client.UpdateWorkspaceVariable(ctx, workspace, existing.UUID, bbcloud.UpdateWorkspaceVariableInput{
					Key:     key,
					Value:   value,
					Secured: secured,
				})
			case scopeDeployment:
				_, err = client.UpdateDeploymentVariable(ctx, workspace, repoSlug, envUUID, existing.UUID, bbcloud.UpdateDeploymentVariableInput{
					Key:     key,
					Value:   value,
					Secured: secured,
				})
			default:
				_, err = client.UpdateRepositoryVariable(ctx, workspace, repoSlug, existing.UUID, bbcloud.UpdateRepositoryVariableInput{
					Key:     key,
					Value:   value,
					Secured: secured,
				})
			}
			if err != nil {
				return fmt.Errorf("failed to update variable %q: %w", key, err)
			}
			action = "updated"
		} else {
			// Create new variable - use opts.Secured for new variables
			secured = opts.Secured
			switch scope {
			case scopeWorkspace:
				_, err = client.CreateWorkspaceVariable(ctx, workspace, bbcloud.CreateWorkspaceVariableInput{
					Key:     key,
					Value:   value,
					Secured: secured,
				})
			case scopeDeployment:
				_, err = client.CreateDeploymentVariable(ctx, workspace, repoSlug, envUUID, bbcloud.CreateDeploymentVariableInput{
					Key:     key,
					Value:   value,
					Secured: secured,
				})
			default:
				_, err = client.CreateRepositoryVariable(ctx, workspace, repoSlug, bbcloud.CreateRepositoryVariableInput{
					Key:     key,
					Value:   value,
					Secured: secured,
				})
			}
			if err != nil {
				return fmt.Errorf("failed to create variable %q: %w", key, err)
			}
			action = "created"
		}

		results = append(results, varResult{
			Key:     key,
			Action:  action,
			Secured: secured,
		})
	}

	location := workspace
	switch scope {
	case scopeRepository:
		location = workspace + "/" + repoSlug
	case scopeDeployment:
		location = workspace + "/" + repoSlug + " (deployment: " + opts.Deployment + ")"
	}

	payload := struct {
		Workspace  string      `json:"workspace"`
		Repository string      `json:"repository,omitempty"`
		Deployment string      `json:"deployment,omitempty"`
		Scope      string      `json:"scope"`
		Variables  []varResult `json:"variables"`
	}{
		Workspace:  workspace,
		Repository: repoSlug,
		Deployment: opts.Deployment,
		Scope:      scope,
		Variables:  results,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		created := 0
		updated := 0
		for _, r := range results {
			if r.Action == "created" {
				created++
			} else {
				updated++
			}
		}

		switch {
		case created > 0 && updated > 0:
			_, err := fmt.Fprintf(ios.Out, "Created %d and updated %d variables in %s.\n", created, updated, location)
			return err
		case created > 0:
			_, err := fmt.Fprintf(ios.Out, "Created %d variable(s) in %s.\n", created, location)
			return err
		default:
			_, err := fmt.Fprintf(ios.Out, "Updated %d variable(s) in %s.\n", updated, location)
			return err
		}
	})
}

// parseEnvFile reads a file and extracts KEY=VALUE pairs.
// Lines starting with # are treated as comments.
// Empty lines are skipped.
func parseEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open env file: %w", err)
	}
	defer func() { _ = file.Close() }()

	result := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		idx := strings.Index(line, "=")
		if idx == -1 {
			return nil, fmt.Errorf("line %d: invalid format, expected KEY=VALUE", lineNum)
		}

		key := strings.TrimSpace(line[:idx])
		value := line[idx+1:]

		// Remove surrounding quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		if key == "" {
			return nil, fmt.Errorf("line %d: empty key", lineNum)
		}

		result[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read env file: %w", err)
	}

	return result, nil
}

package admin

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdAdmin provides administrative operations for Bitbucket Data Center.
func NewCmdAdmin(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Administrative operations for Bitbucket",
		Long: `Perform administrative operations on a Bitbucket Data Center instance.

This command group provides access to server-level management tasks such as
secrets rotation and logging configuration. All subcommands require a Data
Center context; they are not available for Bitbucket Cloud.`,
		Example: `  # Rotate encryption keys on the configured DC instance
  bkt admin secrets rotate

  # Show the current logging configuration
  bkt admin logging get

  # Set the logging level to DEBUG
  bkt admin logging set --level DEBUG`,
	}

	cmd.AddCommand(newSecretsCmd(f))
	cmd.AddCommand(newLoggingCmd(f))

	return cmd
}

func newSecretsCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage secrets manager operations",
		Long: `Manage encryption keys and secrets through the Bitbucket Data Center
Secrets Manager plugin. Use the subcommands to rotate keys and perform
other secrets-related maintenance tasks on your DC instance.`,
		Example: `  # Rotate the encryption keys
  bkt admin secrets rotate

  # Rotate keys using a specific DC context
  bkt admin secrets rotate --context my-dc`,
	}

	cmd.AddCommand(newSecretsRotateCmd(f))
	return cmd
}

func newSecretsRotateCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotate",
		Short: "Rotate encryption keys via the Secrets Manager plugin",
		Long: `Trigger an encryption key rotation on a Bitbucket Data Center instance
through the Secrets Manager plugin. This is a server-side operation that
generates a new encryption key and re-encrypts stored secrets. The command
requires a Data Center context and will fail against Cloud instances.`,
		Example: `  # Rotate encryption keys on the default DC context
  bkt admin secrets rotate

  # Rotate keys on a named DC context
  bkt admin secrets rotate --context prod-dc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSecretsRotate(cmd, f)
		},
	}
	return cmd
}

func runSecretsRotate(cmd *cobra.Command, f *cmdutil.Factory) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, _, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("secrets rotation is only supported for Data Center contexts")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	spinner := f.ProgressSpinner()
	spinner.Start("rotating secrets")
	if err := client.RotateSecret(ctx); err != nil {
		spinner.Fail("secret rotation failed")
		return err
	}
	spinner.Stop("secret rotation complete")

	if _, err := fmt.Fprintf(ios.Out, "✓ Secrets rotated successfully\n"); err != nil {
		return err
	}
	return nil
}

func newLoggingCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logging",
		Short: "Inspect or update logging settings",
		Long: `Inspect or update the logging configuration of a Bitbucket Data Center
instance. You can view the current log level and async setting, or change
them at runtime without restarting the server. This command group is only
available for Data Center contexts.`,
		Example: `  # Show the current logging configuration
  bkt admin logging get

  # Set the log level to WARN
  bkt admin logging set --level WARN

  # Enable async logging at DEBUG level
  bkt admin logging set --level DEBUG --async`,
	}

	cmd.AddCommand(newLoggingGetCmd(f))
	cmd.AddCommand(newLoggingSetCmd(f))

	return cmd
}

func newLoggingGetCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show current logging configuration",
		Long: `Display the current logging configuration of a Bitbucket Data Center
instance, including the active log level and whether asynchronous logging
is enabled. Output defaults to human-readable text but supports JSON via
the --output flag. Requires a Data Center context.`,
		Example: `  # Show logging config in human-readable format
  bkt admin logging get

  # Show logging config as JSON
  bkt admin logging get --output json

  # Query a specific DC context
  bkt admin logging get --context prod-dc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLoggingGet(cmd, f)
		},
	}
	return cmd
}

func newLoggingSetCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &struct {
		Level string
		Async bool
	}{}

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Update logging configuration",
		Long: `Update the logging configuration of a Bitbucket Data Center instance at
runtime. You can change the log level (TRACE, DEBUG, INFO, WARN, ERROR)
and toggle asynchronous logging without restarting the server. This
command requires a Data Center context and will fail against Cloud
instances.`,
		Example: `  # Set the log level to INFO
  bkt admin logging set --level INFO

  # Enable async logging
  bkt admin logging set --async

  # Set DEBUG level with async on a named context
  bkt admin logging set --level DEBUG --async --context staging-dc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLoggingSet(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Level, "level", "", "Logging level: TRACE, DEBUG, INFO, WARN, ERROR")
	cmd.Flags().BoolVar(&opts.Async, "async", false, "Enable asynchronous logging")

	return cmd
}

func runLoggingGet(cmd *cobra.Command, f *cmdutil.Factory) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, _, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("logging inspection is only supported for Data Center contexts")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	cfg, err := client.GetLoggingConfig(ctx)
	if err != nil {
		return err
	}

	return cmdutil.WriteOutput(cmd, ios.Out, cfg, func() error {
		_, err := fmt.Fprintf(ios.Out, "Level: %s\nAsync: %t\n", cfg.Level, cfg.Async)
		return err
	})
}

func runLoggingSet(cmd *cobra.Command, f *cmdutil.Factory, opts *struct {
	Level string
	Async bool
}) error {
	_, _, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("logging configuration is only supported for Data Center contexts")
	}

	client, err := cmdutil.NewDCClient(host)
	if err != nil {
		return err
	}

	cfg := bbdc.LoggingConfig{}
	if opts.Level != "" {
		cfg.Level = strings.ToUpper(opts.Level)
	}
	cfg.Async = opts.Async

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	if err := client.UpdateLoggingConfig(ctx, cfg); err != nil {
		return err
	}

	ios, err := f.Streams()
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(ios.Out, "✓ Updated logging configuration\n"); err != nil {
		return err
	}
	return nil
}

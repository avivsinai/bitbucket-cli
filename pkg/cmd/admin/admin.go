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
	}

	cmd.AddCommand(newSecretsCmd(f))
	cmd.AddCommand(newLoggingCmd(f))

	return cmd
}

func newSecretsCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage secrets manager operations",
	}

	cmd.AddCommand(newSecretsRotateCmd(f))
	return cmd
}

func newSecretsRotateCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rotate",
		Short: "Rotate encryption keys via the Secrets Manager plugin",
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

	fmt.Fprintf(ios.Out, "✓ Secrets rotated successfully\n")
	return nil
}

func newLoggingCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logging",
		Short: "Inspect or update logging settings",
	}

	cmd.AddCommand(newLoggingGetCmd(f))
	cmd.AddCommand(newLoggingSetCmd(f))

	return cmd
}

func newLoggingGetCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show current logging configuration",
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
		fmt.Fprintf(ios.Out, "Level: %s\nAsync: %t\n", cfg.Level, cfg.Async)
		return nil
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
	fmt.Fprintf(ios.Out, "✓ Updated logging configuration\n")
	return nil
}

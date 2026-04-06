package status

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

func newRateLimitCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rate-limit",
		Short: "Show API rate limit telemetry for the active context",
		Long: `Query the Bitbucket API and display the current rate-limit counters for the
active context. The output includes the overall limit, remaining requests,
reset time, and the header source from which the values were read.

Works on both Data Center and Cloud contexts. A lightweight ping request is
sent to populate the rate-limit headers before reporting.`,
		Example: `  # Show rate limits for the active context
  bkt status rate-limit

  # Show rate limits as JSON
  bkt status rate-limit --output json

  # Check rate limits for a specific context
  bkt status rate-limit --context my-cloud-ctx`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRateLimit(cmd, f)
		},
	}
	return cmd
}

func runRateLimit(cmd *cobra.Command, f *cmdutil.Factory) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, _, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	switch host.Kind {
	case "dc":
		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}
		if err := client.Ping(ctx); err != nil {
			return err
		}
		rl := client.RateLimit()
		return renderRateLimit(cmd, ios.Out, rl)
	case "cloud":
		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}
		if err := client.Ping(ctx); err != nil {
			return err
		}
		rl := client.RateLimit()
		return renderRateLimit(cmd, ios.Out, rl)
	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

func renderRateLimit(cmd *cobra.Command, out io.Writer, rl httpx.RateLimit) error {
	payload := map[string]any{
		"limit":     rl.Limit,
		"remaining": rl.Remaining,
		"reset":     rl.Reset,
		"source":    rl.Source,
	}

	return cmdutil.WriteOutput(cmd, out, payload, func() error {
		if _, err := fmt.Fprintf(out, "Limit: %d\n", rl.Limit); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "Remaining: %d\n", rl.Remaining); err != nil {
			return err
		}
		if !rl.Reset.IsZero() {
			if _, err := fmt.Fprintf(out, "Resets At: %s\n", rl.Reset.Format(time.RFC3339)); err != nil {
				return err
			}
		}
		if rl.Source != "" {
			if _, err := fmt.Fprintf(out, "Source: %s\n", rl.Source); err != nil {
				return err
			}
		}
		return nil
	})
}

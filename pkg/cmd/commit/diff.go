package commit

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type commitDiffOptions struct {
	Workspace string
	Project   string
	Repo      string
	From      string
	To        string
}

func newDiffCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &commitDiffOptions{}
	cmd := &cobra.Command{
		Use:   "diff <from> <to>",
		Short: "Show the diff between two commits or refs",
		Long: `Display a unified diff between two commits, branches, or tags. The output is
streamed through the configured pager when available.

On Data Center the two refs are resolved independently by the server. On Cloud
the refs are joined with ".." and sent as a single diff spec; note that branch
names containing literal ".." characters may confuse the Cloud API.

Use --project/--repo (DC) or --workspace/--repo (Cloud) to override the values
from the current context.`,
		Example: `  # Diff two commits
  bkt commit diff abc1234 def5678

  # Diff the current branch against main
  bkt commit diff feature/signup main

  # Diff with explicit repo override
  bkt commit diff v1.0.0 v1.1.0 --repo my-service

  # Diff on a Cloud workspace
  bkt commit diff develop main --workspace myteam --repo backend`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.From = args[0]
			opts.To = args[1]
			return runCommitDiff(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket Cloud workspace override")
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")

	return cmd
}

func runCommitDiff(cmd *cobra.Command, f *cmdutil.Factory, opts *commitDiffOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	withPager := func(fn func(io.Writer) error) error {
		pager := f.PagerManager()
		if pager.Enabled() {
			if w, err := pager.Start(); err == nil {
				defer func() { _ = pager.Stop() }()
				return fn(w)
			}
		}
		return fn(ios.Out)
	}

	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return err
	}

	switch host.Kind {
	case "dc":
		projectKey := cmdutil.FirstNonEmpty(opts.Project, ctxCfg.ProjectKey)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if projectKey == "" || repoSlug == "" {
			return fmt.Errorf("context must supply project and repo; use --project/--repo if needed")
		}

		client, err := cmdutil.NewDCClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		return withPager(func(w io.Writer) error {
			cw := &countingWriter{w: w}
			if err := client.CommitDiff(ctx, projectKey, repoSlug, opts.From, opts.To, cw); err != nil {
				return err
			}
			if cw.n == 0 {
				fmt.Fprintln(ios.ErrOut, "(empty diff)")
			}
			return nil
		})

	case "cloud":
		workspace := cmdutil.FirstNonEmpty(opts.Workspace, ctxCfg.Workspace)
		repoSlug := cmdutil.FirstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
		if workspace == "" || repoSlug == "" {
			return fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
		}

		client, err := cmdutil.NewCloudClient(host)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		// Note: branch names containing ".." will confuse the Cloud API spec parsing.
		spec := opts.From + ".." + opts.To
		return withPager(func(w io.Writer) error {
			cw := &countingWriter{w: w}
			if err := client.CommitDiff(ctx, workspace, repoSlug, spec, cw); err != nil {
				return err
			}
			if cw.n == 0 {
				fmt.Fprintln(ios.ErrOut, "(empty diff)")
			}
			return nil
		})

	default:
		return fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

// countingWriter wraps an io.Writer and tracks the total number of bytes written.
type countingWriter struct {
	w io.Writer
	n int64
}

func (cw *countingWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.n += int64(n)
	return n, err
}

package pr

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type reactionOptions struct {
	Project string
	Repo    string
	ID      int
	Comment int
	Emoji   string
}

func newReactionCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reaction",
		Short: "Manage comment reactions",
		Long: `List, add, or remove emoji reactions on pull request comments.

Data Center only. Not yet supported on Cloud.`,
		Example: `  # List reactions on a comment
  bkt pr reaction list 42 1001

  # Add a thumbs-up reaction
  bkt pr reaction add 42 1001 --emoji :thumbsup:

  # Remove a reaction
  bkt pr reaction remove 42 1001 --emoji :thumbsup:`,
	}

	cmd.AddCommand(newReactionListCmd(f))
	cmd.AddCommand(newReactionAddCmd(f))
	cmd.AddCommand(newReactionRemoveCmd(f))

	return cmd
}

func newReactionListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &reactionOptions{}
	cmd := &cobra.Command{
		Use:   "list <id> <comment-id>",
		Short: "List comment reactions",
		Long:  `List all emoji reactions on a specific pull request comment. Shows each emoji and its count. Data Center only.`,
		Example: `  # List reactions on comment 1001 of PR #42
  bkt pr reaction list 42 1001`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			commentID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid comment id %q", args[1])
			}
			opts.ID = prID
			opts.Comment = commentID
			return runReactionList(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	return cmd
}

func newReactionAddCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &reactionOptions{}
	cmd := &cobra.Command{
		Use:   "add <id> <comment-id>",
		Short: "Add a reaction to a comment",
		Long:  `Add an emoji reaction to a pull request comment. Data Center only.`,
		Example: `  # Add a thumbs-up reaction to comment 1001
  bkt pr reaction add 42 1001 --emoji :thumbsup:`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			commentID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid comment id %q", args[1])
			}
			opts.ID = prID
			opts.Comment = commentID
			return runReactionAdd(cmd, f, opts)
		},
	}
	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Emoji, "emoji", "", "Emoji to add (e.g. :thumbsup:)")
	_ = cmd.MarkFlagRequired("emoji")
	return cmd
}

func newReactionRemoveCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &reactionOptions{}
	cmd := &cobra.Command{
		Use:   "remove <id> <comment-id>",
		Short: "Remove a reaction",
		Long:  `Remove an emoji reaction from a pull request comment. Data Center only.`,
		Example: `  # Remove a thumbs-up reaction from comment 1001
  bkt pr reaction remove 42 1001 --emoji :thumbsup:`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			commentID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid comment id %q", args[1])
			}
			opts.ID = prID
			opts.Comment = commentID
			return runReactionRemove(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().StringVar(&opts.Emoji, "emoji", "", "Emoji to remove")
	_ = cmd.MarkFlagRequired("emoji")
	return cmd
}

func runReactionList(cmd *cobra.Command, f *cmdutil.Factory, opts *reactionOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("reaction list currently supports Data Center contexts only")
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

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	reactions, err := client.ListCommentReactions(ctx, projectKey, repoSlug, opts.ID, opts.Comment)
	if err != nil {
		return err
	}

	payload := map[string]any{
		"project":   projectKey,
		"repo":      repoSlug,
		"reactions": reactions,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(reactions) == 0 {
			_, err := fmt.Fprintf(ios.Out, "No reactions for comment %d\n", opts.Comment)
			return err
		}
		for _, reaction := range reactions {
			if _, err := fmt.Fprintf(ios.Out, "%s x%d\n", reaction.Emoji, reaction.Count); err != nil {
				return err
			}
		}
		return nil
	})
}

func runReactionAdd(cmd *cobra.Command, f *cmdutil.Factory, opts *reactionOptions) error {
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("reaction add currently supports Data Center contexts only")
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

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	if err := client.AddCommentReaction(ctx, projectKey, repoSlug, opts.ID, opts.Comment, opts.Emoji); err != nil {
		return err
	}

	ios, err := f.Streams()
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(ios.Out, "✓ Added %s to comment %d\n", opts.Emoji, opts.Comment); err != nil {
		return err
	}
	return nil
}

func runReactionRemove(cmd *cobra.Command, f *cmdutil.Factory, opts *reactionOptions) error {
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("reaction remove currently supports Data Center contexts only")
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

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	if err := client.RemoveCommentReaction(ctx, projectKey, repoSlug, opts.ID, opts.Comment, opts.Emoji); err != nil {
		return err
	}

	ios, err := f.Streams()
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(ios.Out, "✓ Removed %s from comment %d\n", opts.Emoji, opts.Comment); err != nil {
		return err
	}
	return nil
}

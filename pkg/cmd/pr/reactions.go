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
		Args:  cobra.ExactArgs(2),
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
		Args:  cobra.ExactArgs(2),
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
		Args:  cobra.ExactArgs(2),
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

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
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

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
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

	projectKey := firstNonEmpty(opts.Project, ctxCfg.ProjectKey)
	repoSlug := firstNonEmpty(opts.Repo, ctxCfg.DefaultRepo)
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

package pr

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type suggestionOptions struct {
	Project      string
	Repo         string
	ID           int
	CommentID    int
	SuggestionID int
	Preview      bool
}

func newSuggestionCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &suggestionOptions{}
	cmd := &cobra.Command{
		Use:   "suggestion <id> <comment-id> <suggestion-id>",
		Short: "Apply or preview a code suggestion",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			prID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid pull request id %q", args[0])
			}
			commentID, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid comment id %q", args[1])
			}
			suggestionID, err := strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid suggestion id %q", args[2])
			}
			opts.ID = prID
			opts.CommentID = commentID
			opts.SuggestionID = suggestionID
			return runSuggestion(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Project, "project", "", "Bitbucket project key override")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug override")
	cmd.Flags().BoolVar(&opts.Preview, "preview", false, "Preview suggestion without applying")

	return cmd
}

func runSuggestion(cmd *cobra.Command, f *cmdutil.Factory, opts *suggestionOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, cmdutil.FlagValue(cmd, "context"))
	if err != nil {
		return err
	}
	if host.Kind != "dc" {
		return fmt.Errorf("suggestions currently support Data Center contexts only")
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

	ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
	defer cancel()

	if opts.Preview {
		suggestion, err := client.SuggestionPreview(ctx, projectKey, repoSlug, opts.ID, opts.CommentID, opts.SuggestionID)
		if err != nil {
			return err
		}

		payload := map[string]any{
			"project":    projectKey,
			"repo":       repoSlug,
			"suggestion": suggestion,
		}

		return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
			_, err := fmt.Fprintf(ios.Out, "%s\n", suggestion.Text)
			return err
		})
	}

	if err := client.ApplySuggestion(ctx, projectKey, repoSlug, opts.ID, opts.CommentID, opts.SuggestionID); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Applied suggestion %d\n", opts.SuggestionID); err != nil {
		return err
	}
	return nil
}

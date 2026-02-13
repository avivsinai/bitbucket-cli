package issue

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

func newAttachmentCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "attachment",
		Aliases: []string{"attach"},
		Short:   "Work with issue attachments",
		Long: `Manage file attachments on Bitbucket Cloud issues.

Attachments can be uploaded, downloaded, listed, and deleted from issues.`,
	}

	cmd.AddCommand(newAttachmentListCmd(f))
	cmd.AddCommand(newAttachmentUploadCmd(f))
	cmd.AddCommand(newAttachmentDownloadCmd(f))
	cmd.AddCommand(newAttachmentDeleteCmd(f))

	return cmd
}

// --- List Attachments ---

type attachmentListOptions struct {
	Workspace string
	Repo      string
}

func newAttachmentListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &attachmentListOptions{}
	cmd := &cobra.Command{
		Use:   "list <issue-id>",
		Short: "List attachments on an issue",
		Example: `  # List attachments on issue #42
  bkt issue attachment list 42

  # Output as JSON
  bkt issue attachment list 42 --json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue ID %q: must be a number", args[0])
			}
			return runAttachmentList(cmd, f, opts, issueID)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug")

	return cmd
}

func runAttachmentList(cmd *cobra.Command, f *cmdutil.Factory, opts *attachmentListOptions, issueID int) error {
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
		return fmt.Errorf("issue tracker is only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; set with --repo or configure the context default")
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	attachments, err := client.ListIssueAttachments(ctx, workspace, repoSlug, issueID)
	if err != nil {
		return err
	}

	type attachmentSummary struct {
		Name string `json:"name"`
		URL  string `json:"url,omitempty"`
	}

	var summaries []attachmentSummary
	for _, att := range attachments {
		summaries = append(summaries, attachmentSummary{
			Name: att.Name,
			URL:  att.Links.Self.Href,
		})
	}

	payload := struct {
		IssueID     int                 `json:"issue_id"`
		Attachments []attachmentSummary `json:"attachments"`
	}{
		IssueID:     issueID,
		Attachments: summaries,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(summaries) == 0 {
			_, err := fmt.Fprintf(ios.Out, "No attachments on issue #%d.\n", issueID)
			return err
		}

		for _, s := range summaries {
			if _, err := fmt.Fprintf(ios.Out, "%s\n", s.Name); err != nil {
				return err
			}
		}
		return nil
	})
}

// --- Upload Attachments ---

type attachmentUploadOptions struct {
	Workspace string
	Repo      string
}

func newAttachmentUploadCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &attachmentUploadOptions{}
	cmd := &cobra.Command{
		Use:   "upload <issue-id> <files>...",
		Short: "Upload file attachments to an issue",
		Example: `  # Upload a single file
  bkt issue attachment upload 42 screenshot.png

  # Upload multiple files
  bkt issue attachment upload 42 file1.txt file2.txt`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue ID %q: must be a number", args[0])
			}
			return runAttachmentUpload(cmd, f, opts, issueID, args[1:])
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug")

	return cmd
}

func runAttachmentUpload(cmd *cobra.Command, f *cmdutil.Factory, opts *attachmentUploadOptions, issueID int, files []string) error {
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
		return fmt.Errorf("issue tracker is only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; set with --repo or configure the context default")
	}

	// Validate all files exist and are not directories before uploading
	for _, filePath := range files {
		info, err := os.Stat(filePath)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("file not found: %s", filePath)
			}
			return fmt.Errorf("cannot access file %s: %w", filePath, err)
		}
		if info.IsDir() {
			return fmt.Errorf("cannot upload directory: %s", filePath)
		}
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
	defer cancel()

	for _, filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", filePath, err)
		}

		filename := filepath.Base(filePath)
		attachment, err := client.UploadIssueAttachment(ctx, workspace, repoSlug, issueID, filename, file)
		_ = file.Close()

		if err != nil {
			return fmt.Errorf("failed to upload %s: %w", filePath, err)
		}

		if _, err := fmt.Fprintf(ios.Out, "Uploaded: %s\n", attachment.Name); err != nil {
			return err
		}
	}

	return nil
}

// --- Download Attachments ---

type attachmentDownloadOptions struct {
	Workspace    string
	Repo         string
	Pattern      string
	Dir          string
	Output       string
	SkipExisting bool
	All          bool
}

func newAttachmentDownloadCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &attachmentDownloadOptions{
		Dir: ".",
	}
	cmd := &cobra.Command{
		Use:   "download <issue-id> [<filename>]",
		Short: "Download attachments from an issue",
		Long: `Download one or more attachments from an issue.

If a specific filename is provided, only that file is downloaded.
Use --all to download all attachments, or --pattern to filter by glob pattern.`,
		Example: `  # Download a specific attachment
  bkt issue attachment download 42 screenshot.png

  # Download all attachments
  bkt issue attachment download 42 --all

  # Download matching files to a directory
  bkt issue attachment download 42 --pattern "*.log" --dir ./logs/

  # Download to a specific filename (single file only)
  bkt issue attachment download 42 screenshot.png --output local-screenshot.png

  # Skip existing files
  bkt issue attachment download 42 --all --skip-existing`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue ID %q: must be a number", args[0])
			}

			var filename string
			if len(args) > 1 {
				filename = args[1]
			}

			return runAttachmentDownload(cmd, f, opts, issueID, filename)
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug")
	cmd.Flags().StringVar(&opts.Pattern, "pattern", "", "Filter by glob pattern (e.g., \"*.png\")")
	cmd.Flags().StringVar(&opts.Dir, "dir", opts.Dir, "Output directory")
	cmd.Flags().StringVar(&opts.Output, "output", "", "Output filename (single file only)")
	cmd.Flags().BoolVar(&opts.SkipExisting, "skip-existing", false, "Skip files that already exist locally")
	cmd.Flags().BoolVar(&opts.All, "all", false, "Download all attachments")

	return cmd
}

func runAttachmentDownload(cmd *cobra.Command, f *cmdutil.Factory, opts *attachmentDownloadOptions, issueID int, filename string) error {
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
		return fmt.Errorf("issue tracker is only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; set with --repo or configure the context default")
	}

	// Validate options
	if filename == "" && !opts.All && opts.Pattern == "" {
		return fmt.Errorf("specify a filename, --all, or --pattern")
	}
	if filename != "" && (opts.All || opts.Pattern != "") {
		return fmt.Errorf("cannot use --all or --pattern with a specific filename")
	}
	if opts.Output != "" && (opts.All || opts.Pattern != "") {
		return fmt.Errorf("--output can only be used when downloading a single file")
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
	defer cancel()

	// Collect files to download
	var filesToDownload []string

	if filename != "" {
		filesToDownload = append(filesToDownload, filename)
	} else {
		// List all attachments first
		attachments, err := client.ListIssueAttachments(ctx, workspace, repoSlug, issueID)
		if err != nil {
			return err
		}

		for _, att := range attachments {
			if opts.Pattern != "" {
				matched, err := filepath.Match(opts.Pattern, att.Name)
				if err != nil {
					return fmt.Errorf("invalid pattern %q: %w", opts.Pattern, err)
				}
				if !matched {
					continue
				}
			}
			filesToDownload = append(filesToDownload, att.Name)
		}
	}

	if len(filesToDownload) == 0 {
		_, err := fmt.Fprintln(ios.Out, "No attachments to download.")
		return err
	}

	// Create output directory if needed
	if opts.Dir != "." {
		if err := os.MkdirAll(opts.Dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", opts.Dir, err)
		}
	}

	// Download files
	for _, name := range filesToDownload {
		// Sanitize attachment name to prevent path traversal attacks
		safeName := filepath.Base(name)
		if safeName == "." || safeName == ".." || safeName == "" {
			return fmt.Errorf("invalid attachment name: %s", name)
		}

		outputPath := filepath.Join(opts.Dir, safeName)
		if opts.Output != "" {
			outputPath = opts.Output
		}

		// Check if file exists
		if opts.SkipExisting {
			if _, err := os.Stat(outputPath); err == nil {
				if _, err := fmt.Fprintf(ios.Out, "Skipped (exists): %s\n", outputPath); err != nil {
					return err
				}
				continue
			}
		}

		file, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", outputPath, err)
		}

		err = client.DownloadIssueAttachment(ctx, workspace, repoSlug, issueID, name, file)
		_ = file.Close()

		if err != nil {
			// Clean up partial file
			_ = os.Remove(outputPath)
			return fmt.Errorf("failed to download %s: %w", name, err)
		}

		if _, err := fmt.Fprintf(ios.Out, "Downloaded: %s\n", outputPath); err != nil {
			return err
		}
	}

	return nil
}

// --- Delete Attachment ---

type attachmentDeleteOptions struct {
	Workspace string
	Repo      string
	Confirm   bool
}

func newAttachmentDeleteCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &attachmentDeleteOptions{}
	cmd := &cobra.Command{
		Use:   "delete <issue-id> <filename>",
		Short: "Delete an attachment from an issue",
		Example: `  # Delete an attachment (will prompt for confirmation)
  bkt issue attachment delete 42 screenshot.png

  # Delete without confirmation
  bkt issue attachment delete 42 screenshot.png --confirm`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			issueID, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid issue ID %q: must be a number", args[0])
			}
			return runAttachmentDelete(cmd, f, opts, issueID, args[1])
		},
	}

	cmd.Flags().StringVar(&opts.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVar(&opts.Repo, "repo", "", "Repository slug")
	cmd.Flags().BoolVar(&opts.Confirm, "confirm", false, "Skip confirmation prompt")

	return cmd
}

func runAttachmentDelete(cmd *cobra.Command, f *cmdutil.Factory, opts *attachmentDeleteOptions, issueID int, filename string) error {
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
		return fmt.Errorf("issue tracker is only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return fmt.Errorf("repository slug required; set with --repo or configure the context default")
	}

	if !opts.Confirm {
		prompter := f.Prompt()
		confirmed, err := prompter.Confirm(fmt.Sprintf("Delete attachment %q from issue #%d?", filename, issueID), false)
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintln(ios.Out, "Aborted.")
			return nil
		}
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	if err := client.DeleteIssueAttachment(ctx, workspace, repoSlug, issueID, filename); err != nil {
		return err
	}

	type result struct {
		IssueID  int    `json:"issue_id"`
		Filename string `json:"filename"`
		Deleted  bool   `json:"deleted"`
	}

	r := result{
		IssueID:  issueID,
		Filename: filename,
		Deleted:  true,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, r, func() error {
		_, err := fmt.Fprintf(ios.Out, "Deleted attachment %q from issue #%d\n", filename, issueID)
		return err
	})
}

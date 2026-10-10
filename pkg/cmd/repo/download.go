package repo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

type downloadOptions struct {
	Workspace string
	Repo      string
}

func newDownloadCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "download",
		Short: "Work with repository downloads (Cloud only)",
		Long: `Manage files in a Bitbucket Cloud repository's Downloads section.

Downloads is the only place the public API accepts arbitrary file uploads
for a repository, which makes it the way to host images for pull request
descriptions and comments. Workspaces on the Free plan cannot upload or
download files; the API rejects those requests.`,
	}

	cmd.AddCommand(newDownloadListCmd(f))
	cmd.AddCommand(newDownloadUploadCmd(f))

	return cmd
}

func (o *downloadOptions) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&o.Workspace, "workspace", "", "Bitbucket workspace")
	cmd.Flags().StringVar(&o.Repo, "repo", "", "Repository slug")
}

// resolveCloudRepo resolves the Cloud workspace and repository from flags or the active context.
func resolveCloudRepo(cmd *cobra.Command, f *cmdutil.Factory, opts *downloadOptions) (*bbcloud.Client, string, string, error) {
	override := cmdutil.FlagValue(cmd, "context")
	_, ctxCfg, host, err := cmdutil.ResolveContext(f, cmd, override)
	if err != nil {
		return nil, "", "", err
	}

	if host.Kind != "cloud" {
		return nil, "", "", fmt.Errorf("repository downloads are only available for Bitbucket Cloud; current context uses %s", host.Kind)
	}

	workspace := strings.TrimSpace(opts.Workspace)
	if workspace == "" {
		workspace = ctxCfg.Workspace
	}
	if workspace == "" {
		return nil, "", "", fmt.Errorf("workspace required; set with --workspace or configure the context default")
	}

	repoSlug := strings.TrimSpace(opts.Repo)
	if repoSlug == "" {
		repoSlug = ctxCfg.DefaultRepo
	}
	if repoSlug == "" {
		return nil, "", "", fmt.Errorf("repository slug required; set with --repo or configure the context default")
	}

	client, err := cmdutil.NewCloudClient(host)
	if err != nil {
		return nil, "", "", err
	}

	return client, workspace, repoSlug, nil
}

type downloadSummary struct {
	Name string `json:"name"`
	Size int64  `json:"size,omitempty"`
	URL  string `json:"url"`
}

// --- List ---

func newDownloadListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &downloadOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List repository downloads (Cloud only)",
		Example: `  # List downloads of the active context repository
  bkt repo download list

  # List downloads of a specific repository as JSON
  bkt repo download list --workspace my-team --repo api-service --json`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDownloadList(cmd, f, opts)
		},
	}
	opts.addFlags(cmd)
	return cmd
}

func runDownloadList(cmd *cobra.Command, f *cmdutil.Factory, opts *downloadOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	client, workspace, repoSlug, err := resolveCloudRepo(cmd, f, opts)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
	defer cancel()

	downloads, err := client.ListDownloads(ctx, workspace, repoSlug)
	if err != nil {
		return err
	}

	summaries := make([]downloadSummary, 0, len(downloads))
	for _, d := range downloads {
		summaries = append(summaries, downloadSummary{
			Name: d.Name,
			Size: d.Size,
			URL:  bbcloud.DownloadURL(workspace, repoSlug, d.Name),
		})
	}

	payload := struct {
		Workspace string            `json:"workspace"`
		Repo      string            `json:"repo"`
		Downloads []downloadSummary `json:"downloads"`
	}{
		Workspace: workspace,
		Repo:      repoSlug,
		Downloads: summaries,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(summaries) == 0 {
			_, err := fmt.Fprintf(ios.Out, "No downloads in %s/%s.\n", workspace, repoSlug)
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

// --- Upload ---

func newDownloadUploadCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &downloadOptions{}
	cmd := &cobra.Command{
		Use:   "upload <files>...",
		Short: "Upload files to the repository downloads (Cloud only)",
		Long: `Upload one or more files to a Bitbucket Cloud repository's Downloads section
and print their URLs.

An existing download with the same file name is replaced, so give images
unique names. All files are validated before any upload begins. Directories
cannot be uploaded.

Downloads are visible to everyone with access to the repository. Do not
upload anything that should stay private to a subset of them.`,
		Example: `  # Upload a screenshot and print its URL
  bkt repo download upload screenshot.png

  # Upload several images and get Markdown image links for a PR description
  bkt repo download upload before.png after.png --json --jq '.downloads[] | "![" + .name + "](" + .url + ")"'`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDownloadUpload(cmd, f, opts, args)
		},
	}
	opts.addFlags(cmd)
	return cmd
}

func runDownloadUpload(cmd *cobra.Command, f *cmdutil.Factory, opts *downloadOptions, files []string) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	client, workspace, repoSlug, err := resolveCloudRepo(cmd, f, opts)
	if err != nil {
		return err
	}

	// Validate all files exist and are not directories before uploading anything
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

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Minute)
	defer cancel()

	uploaded := make([]downloadSummary, 0, len(files))
	for _, filePath := range files {
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open %s: %w", filePath, err)
		}

		filename := filepath.Base(filePath)
		err = client.UploadDownload(ctx, workspace, repoSlug, filename, file)
		_ = file.Close()
		if err != nil {
			return fmt.Errorf("failed to upload %s: %w", filePath, err)
		}

		uploaded = append(uploaded, downloadSummary{
			Name: filename,
			URL:  bbcloud.DownloadURL(workspace, repoSlug, filename),
		})
	}

	payload := struct {
		Workspace string            `json:"workspace"`
		Repo      string            `json:"repo"`
		Downloads []downloadSummary `json:"downloads"`
	}{
		Workspace: workspace,
		Repo:      repoSlug,
		Downloads: uploaded,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		for _, u := range uploaded {
			if _, err := fmt.Fprintf(ios.Out, "Uploaded: %s\n%s\n", u.Name, u.URL); err != nil {
				return err
			}
		}
		return nil
	})
}

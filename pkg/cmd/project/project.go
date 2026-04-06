package project

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// NewCmdProject wires project-focused subcommands.
func NewCmdProject(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Work with Bitbucket projects (DC only)",
		Long: `List and inspect Bitbucket projects. Projects are top-level containers that
group related repositories.

Note: Project commands are currently supported for Bitbucket Data Center only.
Cloud workspaces use a different organizational model and are managed through
other commands.`,
		Example: `  # List all visible projects
  bkt project list

  # List projects on a specific host
  bkt project list --host my-dc-server`,
	}

	cmd.AddCommand(newListCmd(f))

	return cmd
}

type listOptions struct {
	Host  string
	Limit int
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &listOptions{
		Limit: 30,
	}
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List Bitbucket Data Center projects (DC only)",
		Long: `List all projects visible to the authenticated user on a Bitbucket Data Center
instance. Each project is displayed with its key, name, description, web URL,
and visibility status. Use --limit to control the number of results returned.

This command is only available for Data Center hosts. Attempting to run it
against a Cloud context will return an error.`,
		Example: `  # List projects (default limit of 30)
  bkt project list

  # List all projects without a limit
  bkt project ls --limit 0

  # List projects on a specific Data Center host
  bkt project list --host my-dc-server

  # List projects in JSON format
  bkt project list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Host, "host", "", "Host key or base URL override")
	cmd.Flags().IntVar(&opts.Limit, "limit", opts.Limit, "Maximum projects to display (0 for all)")

	return cmd
}

func runList(cmd *cobra.Command, f *cmdutil.Factory, opts *listOptions) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	contextOverride := cmdutil.FlagValue(cmd, "context")
	hostKey, hostCfg, err := cmdutil.ResolveHost(f, contextOverride, opts.Host)
	if err != nil {
		return err
	}

	if hostCfg.Kind != "dc" {
		return fmt.Errorf("project listing is only supported for Bitbucket Data Center hosts")
	}

	client, err := cmdutil.NewDCClient(hostCfg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
	defer cancel()

	projects, err := client.ListProjects(ctx, opts.Limit)
	if err != nil {
		return err
	}

	type projectSummary struct {
		Key         string `json:"key"`
		Name        string `json:"name"`
		ID          int    `json:"id"`
		Type        string `json:"type"`
		Public      bool   `json:"public"`
		Description string `json:"description,omitempty"`
		WebURL      string `json:"web_url"`
	}

	baseURL := strings.TrimRight(hostCfg.BaseURL, "/")
	var summaries []projectSummary
	for _, p := range projects {
		key := strings.ToUpper(strings.TrimSpace(p.Key))
		webURL := fmt.Sprintf("%s/projects/%s", baseURL, url.PathEscape(key))

		summaries = append(summaries, projectSummary{
			Key:         key,
			Name:        p.Name,
			ID:          p.ID,
			Type:        p.Type,
			Public:      p.Public,
			Description: strings.TrimSpace(p.Description),
			WebURL:      webURL,
		})
	}

	payload := struct {
		HostKey  string           `json:"host_key"`
		BaseURL  string           `json:"base_url"`
		Projects []projectSummary `json:"projects"`
	}{
		HostKey:  hostKey,
		BaseURL:  baseURL,
		Projects: summaries,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(summaries) == 0 {
			_, err := fmt.Fprintf(ios.Out, "No projects visible on host %s.\n", baseURL)
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "Projects on %s:\n", baseURL); err != nil {
			return err
		}
		for _, p := range summaries {
			if _, err := fmt.Fprintf(ios.Out, "%s\t%s\n", p.Key, p.Name); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(ios.Out, "    link: %s\n", p.WebURL); err != nil {
				return err
			}
			if p.Description != "" {
				if _, err := fmt.Fprintf(ios.Out, "    desc: %s\n", p.Description); err != nil {
					return err
				}
			}
			if p.Public {
				if _, err := fmt.Fprintln(ios.Out, "    visibility: public"); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

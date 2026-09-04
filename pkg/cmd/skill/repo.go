package skill

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/internal/skills/source"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

// openRepositoryFunc builds the source.Repository for a repository argument.
// Tests replace it with an in-memory repository.
var openRepositoryFunc = openRepository

// openRepository binds "WORKSPACE/REPO", "PROJECT/REPO", or a Bitbucket URL to
// a configured host and returns the matching Cloud or Data Center adapter.
func openRepository(cmd *cobra.Command, f *cmdutil.Factory, arg string) (source.Repository, error) {
	ref, err := source.ParseRepoArg(arg)
	if err != nil {
		return nil, err
	}

	override := cmdutil.FlagValue(cmd, "context")
	var host *config.Host
	if ref.Host != "" {
		_, host, err = cmdutil.ResolveHost(f, override, ref.Host)
	} else {
		_, _, host, err = cmdutil.ResolveContext(f, cmd, override)
	}
	if err != nil {
		return nil, err
	}
	if ref.Kind != "" && ref.Kind != host.Kind {
		return nil, fmt.Errorf("repository %q is a Bitbucket %s URL but host %s is configured as %s", arg, kindLabel(ref.Kind), host.BaseURL, kindLabel(host.Kind))
	}

	switch host.Kind {
	case "cloud":
		client, err := f.CloudClient(host)
		if err != nil {
			return nil, err
		}
		return source.NewCloudRepository(client, ref.Owner, ref.Slug), nil
	case "dc":
		client, err := f.DCClient(host)
		if err != nil {
			return nil, err
		}
		return source.NewDCRepository(client, host.BaseURL, ref.Owner, ref.Slug), nil
	default:
		return nil, fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

func kindLabel(kind string) string {
	switch kind {
	case "cloud":
		return "Cloud"
	case "dc":
		return "Data Center"
	default:
		return kind
	}
}

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
	_, _, contextHost, contextErr := cmdutil.ResolveContext(f, cmd, override)

	host, err := resolveArgHost(f, override, arg, ref, contextHost, contextErr)
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

// resolveArgHost picks the host to talk to. A repository given as a URL is
// served by the host it names when that host is configured; otherwise the
// active context's host is used, and the error explains the mismatch.
func resolveArgHost(f *cmdutil.Factory, override, arg string, ref source.RepoRef, contextHost *config.Host, contextErr error) (*config.Host, error) {
	if ref.Host == "" {
		if contextErr != nil {
			return nil, contextErr
		}
		return contextHost, nil
	}

	if _, urlHost, err := cmdutil.ResolveHost(f, override, ref.Host); err == nil {
		return urlHost, nil
	} else if contextErr != nil {
		return nil, err
	}

	// The host in the URL is not configured. The usual cause is pasting a URL
	// from the other Bitbucket platform, so say that rather than "not found".
	if ref.Kind != "" && ref.Kind != contextHost.Kind {
		return nil, fmt.Errorf("repository %q is a Bitbucket %s URL, but %s is not configured and the active context uses %s (Bitbucket %s)",
			arg, kindLabel(ref.Kind), ref.Host, contextHost.BaseURL, kindLabel(contextHost.Kind))
	}
	return nil, fmt.Errorf("repository %q names host %s, which is not configured; run `bkt auth login https://%s` or use --context", arg, ref.Host, ref.Host)
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

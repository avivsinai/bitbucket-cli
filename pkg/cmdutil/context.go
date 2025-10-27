package cmdutil

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/internal/remote"
)

// ResolveContext fetches the context and host configuration given an optional
// override name (typically provided via --context). When the override is empty
// the active context from the config file is used.
func ResolveContext(f *Factory, cmd *cobra.Command, override string) (string, *config.Context, *config.Host, error) {
	cfg, err := f.ResolveConfig()
	if err != nil {
		return "", nil, nil, err
	}

	contextName := override
	if contextName == "" {
		contextName = cfg.ActiveContext
	}

	if contextName == "" {
		return "", nil, nil, fmt.Errorf("no active context; run `%s context use <name>`", f.ExecutableName)
	}

	ctx, err := cfg.Context(contextName)
	if err != nil {
		return "", nil, nil, err
	}

	if ctx.Host == "" {
		return "", nil, nil, fmt.Errorf("context %q has no host configured", contextName)
	}

	host, err := cfg.Host(ctx.Host)
	if err != nil {
		return "", nil, nil, err
	}

	applyRemoteDefaults(ctx, host)

	return contextName, ctx, host, nil
}

// ResolveHost locates a host configuration using optional context or host overrides.
// When neither override is provided it falls back to the active context, then to a
// single configured host. This enables commands to function prior to context setup.
func ResolveHost(f *Factory, contextOverride, hostOverride string) (string, *config.Host, error) {
	cfg, err := f.ResolveConfig()
	if err != nil {
		return "", nil, err
	}

	hostIdentifier := strings.TrimSpace(hostOverride)
	if hostIdentifier != "" {
		if host, ok := cfg.Hosts[hostIdentifier]; ok {
			return hostIdentifier, host, nil
		}

		baseURL, err := NormalizeBaseURL(hostIdentifier)
		if err == nil {
			if key, err := HostKeyFromURL(baseURL); err == nil {
				if host, ok := cfg.Hosts[key]; ok {
					return key, host, nil
				}
			}
		}

		return "", nil, fmt.Errorf("host %q not found; run `%s auth login` first", hostIdentifier, f.ExecutableName)
	}

	contextName := strings.TrimSpace(contextOverride)
	if contextName == "" {
		contextName = cfg.ActiveContext
	}
	if contextName != "" {
		ctx, err := cfg.Context(contextName)
		if err != nil {
			return "", nil, err
		}
		if ctx.Host == "" {
			return "", nil, fmt.Errorf("context %q has no host configured", contextName)
		}
		host, err := cfg.Host(ctx.Host)
		if err != nil {
			return "", nil, err
		}
		return ctx.Host, host, nil
	}

	switch len(cfg.Hosts) {
	case 0:
		return "", nil, fmt.Errorf("no hosts configured; run `%s auth login` first", f.ExecutableName)
	case 1:
		for key, host := range cfg.Hosts {
			return key, host, nil
		}
	default:
		var keys []string
		for key := range cfg.Hosts {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return "", nil, fmt.Errorf("multiple hosts configured (%s); specify --host or --context", strings.Join(keys, ", "))
	}

	return "", nil, fmt.Errorf("failed to resolve host configuration")
}

// FlagValue returns the value for the named flag if it exists.
func FlagValue(cmd *cobra.Command, name string) string {
	flag := cmd.Flags().Lookup(name)
	if flag == nil {
		return ""
	}
	return flag.Value.String()
}

func applyRemoteDefaults(ctx *config.Context, host *config.Host) {
	if ctx == nil || host == nil {
		return
	}

	needsWorkspace := host.Kind == "cloud" && ctx.Workspace == ""
	needsProject := host.Kind == "dc" && ctx.ProjectKey == ""
	needsRepo := ctx.DefaultRepo == ""
	if !needsWorkspace && !needsProject && !needsRepo {
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		return
	}

	loc, err := remote.Detect(wd)
	if err != nil {
		return
	}
	if !locatorMatchesHost(host, loc) {
		return
	}

	if needsRepo && loc.RepoSlug != "" {
		ctx.DefaultRepo = loc.RepoSlug
	}

	if needsWorkspace && loc.Workspace != "" {
		ctx.Workspace = loc.Workspace
	}

	if needsProject && loc.ProjectKey != "" {
		ctx.ProjectKey = loc.ProjectKey
	}
}

func locatorMatchesHost(host *config.Host, loc remote.Locator) bool {
	if host == nil {
		return false
	}

	switch host.Kind {
	case "cloud":
		return loc.Kind == "cloud" && strings.EqualFold(loc.Host, "bitbucket.org")
	case "dc":
		if loc.Kind != "dc" {
			return false
		}
		baseHost := hostHostname(host.BaseURL)
		return baseHost != "" && strings.EqualFold(baseHost, loc.Host)
	default:
		return false
	}
}

func hostHostname(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err == nil && u.Host != "" {
		raw = u.Host
	}
	raw = strings.Trim(raw, "[]")
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, ":") {
		if host, _, err := net.SplitHostPort(raw); err == nil {
			raw = host
		}
	}
	return strings.ToLower(raw)
}

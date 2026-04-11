package cmdutil

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/internal/remote"
	"github.com/avivsinai/bitbucket-cli/internal/secret"
	"github.com/avivsinai/bitbucket-cli/pkg/oauth"
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
		envKey, envHost, envErr := hostFromEnv(os.Getenv(secret.EnvHost))
		if envErr != nil {
			return "", nil, nil, fmt.Errorf("BKT_HOST: %w", envErr)
		}
		if envHost != nil {
			// Prefer the saved host entry when config already has one for this
			// key — it preserves persisted fields (username, auth_method, etc.)
			// that the synthesized host would silently drop.
			if saved, ok := cfg.Hosts[envKey]; ok {
				h, err := mergeEnvHost(f.ExecutableName, envKey, *saved)
				if err != nil {
					return "", nil, nil, err
				}
				envHost = h
			}
			ctx := contextFromEnv()
			ctx.Host = envKey
			applyRemoteDefaults(ctx, envHost)
			return "", ctx, envHost, nil
		}
		if secret.TokenFromEnv() != "" {
			return "", nil, nil, errTokenSetButNoHost
		}
		if os.Getenv(secret.EnvHost) != "" {
			return "", nil, nil, errHostSetButNoToken
		}
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

	if err := loadHostToken(f.ExecutableName, ctx.Host, host); err != nil {
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
			if err := loadHostToken(f.ExecutableName, hostIdentifier, host); err != nil {
				return "", nil, err
			}
			return hostIdentifier, host, nil
		}

		baseURL, err := NormalizeBaseURL(hostIdentifier)
		if err == nil {
			if key, err := HostKeyFromURL(baseURL); err == nil {
				if host, ok := cfg.Hosts[key]; ok {
					if err := loadHostToken(f.ExecutableName, key, host); err != nil {
						return "", nil, err
					}
					return key, host, nil
				}
			}
		}

		envKey, envHost, envErr := hostFromEnv(hostIdentifier)
		if envErr != nil {
			return "", nil, fmt.Errorf("--host %q: %w", hostIdentifier, envErr)
		}
		if envHost != nil {
			return envKey, envHost, nil
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
		if err := loadHostToken(f.ExecutableName, ctx.Host, host); err != nil {
			return "", nil, err
		}
		return ctx.Host, host, nil
	}

	switch len(cfg.Hosts) {
	case 0:
		envKey, envHost, envErr := hostFromEnv(os.Getenv(secret.EnvHost))
		if envErr != nil {
			return "", nil, fmt.Errorf("BKT_HOST: %w", envErr)
		}
		if envHost != nil {
			return envKey, envHost, nil
		}
		if secret.TokenFromEnv() != "" {
			return "", nil, errTokenSetButNoHost
		}
		if os.Getenv(secret.EnvHost) != "" {
			return "", nil, errHostSetButNoToken
		}
		return "", nil, fmt.Errorf("no hosts configured; run `%s auth login` first", f.ExecutableName)
	case 1:
		// BKT_HOST may target a different server than the single saved host;
		// honour it for consistency with the ResolveContext and default paths.
		envKey, envHost, envErr := hostFromEnv(os.Getenv(secret.EnvHost))
		if envErr != nil {
			return "", nil, fmt.Errorf("BKT_HOST: %w", envErr)
		}
		if envHost != nil {
			if saved, ok := cfg.Hosts[envKey]; ok {
				h, err := mergeEnvHost(f.ExecutableName, envKey, *saved)
				if err != nil {
					return "", nil, err
				}
				return envKey, h, nil
			}
			return envKey, envHost, nil
		}
		for key, host := range cfg.Hosts {
			if err := loadHostToken(f.ExecutableName, key, host); err != nil {
				return "", nil, err
			}
			return key, host, nil
		}
	default:
		// BKT_HOST disambiguates among multiple configured hosts.
		envKey, envHost, envErr := hostFromEnv(os.Getenv(secret.EnvHost))
		if envErr != nil {
			return "", nil, fmt.Errorf("BKT_HOST: %w", envErr)
		}
		if envHost != nil {
			if saved, ok := cfg.Hosts[envKey]; ok {
				h, err := mergeEnvHost(f.ExecutableName, envKey, *saved)
				if err != nil {
					return "", nil, err
				}
				return envKey, h, nil
			}
			return envKey, envHost, nil
		}
		if os.Getenv(secret.EnvHost) != "" {
			return "", nil, errHostSetButNoToken
		}
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

// mergeEnvHost returns a copy of base with BKT_TOKEN, BKT_USERNAME, and
// BKT_AUTH_METHOD applied. The caller's map entry is never mutated because
// base is passed by value; the returned pointer is a fresh allocation.
func mergeEnvHost(executable, key string, base config.Host) (*config.Host, error) {
	h := base
	if err := loadHostToken(executable, key, &h); err != nil {
		return nil, err
	}
	if u := strings.TrimSpace(os.Getenv(secret.EnvUsername)); u != "" {
		h.Username = u
	}
	if m := strings.TrimSpace(os.Getenv(secret.EnvAuthMethod)); m != "" {
		h.AuthMethod = m
	}
	return &h, nil
}

func loadHostToken(executable, hostKey string, host *config.Host) error {
	if host == nil {
		return fmt.Errorf("host %q not configured", hostKey)
	}

	// BKT_TOKEN applies to all hosts. For multi-host setups where each
	// host needs a different token, use the keyring instead.
	if envToken := secret.TokenFromEnv(); envToken != "" {
		host.Token = envToken
		// Apply BKT_USERNAME and BKT_AUTH_METHOD so that OAuth-saved hosts
		// (where Username is a Bitbucket ID, not an email) work correctly
		// when overridden with a Cloud API token.
		if u := strings.TrimSpace(os.Getenv(secret.EnvUsername)); u != "" {
			host.Username = u
		} else if host.AuthMethod == "oauth" && host.Kind == "cloud" {
			return fmt.Errorf("BKT_USERNAME is required when overriding an OAuth-authenticated Cloud host with BKT_TOKEN; set BKT_USERNAME to your Atlassian account email")
		}
		if m := strings.TrimSpace(os.Getenv(secret.EnvAuthMethod)); m != "" {
			host.AuthMethod = m
		}
		return nil
	}

	if host.Token != "" {
		return nil
	}

	opts := []secret.Option{}
	if host.AllowInsecureStore {
		opts = append(opts, secret.WithAllowFileFallback(true))
	}

	store, err := secret.Open(opts...)
	if err != nil {
		if secret.IsNoKeyringError(err) {
			return fmt.Errorf("no OS keychain backend available for host %q; rerun `%s auth login %s --allow-insecure-store` or set BKT_ALLOW_INSECURE_STORE=1: %w", hostKey, executable, hostKey, err)
		}
		return err
	}

	token, err := store.Get(secret.TokenKey(hostKey))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			target := host.BaseURL
			if target == "" {
				target = hostKey
			}
			return fmt.Errorf("credentials for host %q not found; run `%s auth login %s`", hostKey, executable, target)
		}
		return err
	}

	if oauth.IsTokenBlob(token) {
		tok, parseErr := oauth.Unmarshal(token)
		if parseErr != nil {
			return fmt.Errorf("parse OAuth token for host %q: %w", hostKey, parseErr)
		}
		host.Token = tok.AccessToken
		if host.AuthMethod == "" {
			host.AuthMethod = "oauth"
		}
		return nil
	}

	host.Token = token
	return nil
}

// errTokenSetButNoHost is returned when BKT_TOKEN is set but BKT_HOST is not.
var errTokenSetButNoHost = errors.New("BKT_TOKEN is set but BKT_HOST is not; set BKT_HOST to the Bitbucket server URL")

// errHostSetButNoToken is returned when BKT_HOST is set but BKT_TOKEN is not.
var errHostSetButNoToken = errors.New("BKT_HOST is set but BKT_TOKEN is not; did you forget to set BKT_TOKEN?")

// hostFromEnv synthesises an ephemeral *config.Host from environment variables.
// rawURL may be a full URL or bare hostname; it is normalised internally.
// Returns ("", nil, nil) when BKT_TOKEN or rawURL is empty — the caller should
// fall through to its original error path in that case.
func hostFromEnv(rawURL string) (string, *config.Host, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", nil, nil
	}
	token := secret.TokenFromEnv()
	if token == "" {
		return "", nil, nil
	}

	baseURL, err := NormalizeBaseURL(rawURL)
	if err != nil {
		return "", nil, err
	}

	// Canonicalise Bitbucket Cloud base URLs to the API origin.
	// Exact hostname matching avoids false positives for DC hosts whose names
	// happen to contain or end with "bitbucket.org" (e.g. bitbucket.org.corp.com).
	hostname := hostHostname(baseURL)
	isCloud := hostname == "bitbucket.org" || hostname == "api.bitbucket.org"
	if isCloud {
		// Covers: https://bitbucket.org, https://api.bitbucket.org (bare, no /2.0),
		// and https://api.bitbucket.org/2.0 (no-op rewrite).
		baseURL = "https://api.bitbucket.org/2.0"
	}

	key, err := HostKeyFromURL(baseURL)
	if err != nil {
		return "", nil, err
	}

	kind := "dc"
	if isCloud {
		kind = "cloud"
	}

	username := strings.TrimSpace(os.Getenv(secret.EnvUsername))
	authMethod := strings.TrimSpace(os.Getenv(secret.EnvAuthMethod))

	if isCloud {
		// Cloud only supports basic auth; a username (Atlassian account email)
		// is always required.
		if username == "" {
			return "", nil, fmt.Errorf("BKT_USERNAME is required for Bitbucket Cloud; set it to your Atlassian account email")
		}
		authMethod = "basic"
	} else {
		// DC: default to bearer when no username is available so that PAT-only
		// headless flows work without requiring BKT_AUTH_METHOD=bearer.
		if authMethod == "" && username == "" {
			authMethod = "bearer"
		}
		if authMethod == "basic" && username == "" {
			return "", nil, fmt.Errorf("BKT_AUTH_METHOD=basic requires BKT_USERNAME; set BKT_USERNAME or use BKT_AUTH_METHOD=bearer for token-only auth")
		}
	}

	return key, &config.Host{
		Kind:       kind,
		BaseURL:    baseURL,
		Username:   username,
		AuthMethod: authMethod,
		Token:      token,
	}, nil
}

// contextFromEnv builds an ephemeral *config.Context from environment variables.
// It is always non-nil; fields are empty strings when the vars are unset.
func contextFromEnv() *config.Context {
	return &config.Context{
		ProjectKey:  strings.TrimSpace(os.Getenv(secret.EnvProject)),
		Workspace:   strings.TrimSpace(os.Getenv(secret.EnvWorkspace)),
		DefaultRepo: strings.TrimSpace(os.Getenv(secret.EnvRepo)),
	}
}

func applyRemoteDefaults(ctx *config.Context, host *config.Host) {
	if ctx == nil || host == nil {
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
	if !LocatorMatchesHost(host, loc) {
		return
	}

	if loc.RepoSlug != "" {
		ctx.DefaultRepo = loc.RepoSlug
	}

	if host.Kind == "cloud" && loc.Workspace != "" {
		ctx.Workspace = loc.Workspace
	}

	if host.Kind == "dc" && loc.ProjectKey != "" {
		ctx.ProjectKey = loc.ProjectKey
	}
}

// LocatorMatchesHost reports whether a remote locator points at the same
// server as the supplied host configuration.
func LocatorMatchesHost(host *config.Host, loc remote.Locator) bool {
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

// FirstNonEmpty returns the first non-empty string value.
func FirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ResolveCloudRepo resolves workspace and repository for Cloud commands.
func ResolveCloudRepo(f *Factory, cmd *cobra.Command, workspaceOverride, repoOverride string) (string, string, *config.Host, error) {
	_, ctxCfg, host, err := ResolveContext(f, cmd, FlagValue(cmd, "context"))
	if err != nil {
		return "", "", nil, err
	}
	if host.Kind != "cloud" {
		return "", "", nil, fmt.Errorf("command supports Bitbucket Cloud contexts only")
	}
	workspace := FirstNonEmpty(workspaceOverride, ctxCfg.Workspace)
	repo := FirstNonEmpty(repoOverride, ctxCfg.DefaultRepo)
	if workspace == "" || repo == "" {
		return "", "", nil, fmt.Errorf("context must supply workspace and repo; use --workspace/--repo if needed")
	}
	return workspace, repo, host, nil
}

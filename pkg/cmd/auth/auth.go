package auth

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/internal/secret"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
	"github.com/avivsinai/bitbucket-cli/pkg/oauth"
)

// CloudTokenURL is Atlassian's token-management page. Atlassian does not
// currently expose a stable deep link into the Bitbucket scoped-token wizard.
const CloudTokenURL = "https://id.atlassian.com/manage-profile/security/api-tokens"

// CloudEmailPrompt is the prompt shown when asking for the Atlassian account email.
const CloudEmailPrompt = "Atlassian account email"

// CloudTokenPrompt is the prompt shown when asking for the API token.
const CloudTokenPrompt = "API token"

// NewCmdAuth returns the root auth command.
func NewCmdAuth(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Bitbucket authentication credentials",
		Long: `Manage authentication credentials for Bitbucket Data Center and Cloud hosts.

Tokens are stored in the OS keychain by default. For Data Center hosts, bkt
uses Personal Access Tokens (PATs). For Bitbucket Cloud, bkt uses Atlassian
API tokens with scopes.

Use "bkt auth login" to add a host, "bkt auth status" to inspect stored
credentials, and "bkt auth logout" to remove them.`,
	}

	cmd.AddCommand(newLoginCmd(f))
	cmd.AddCommand(newStatusCmd(f))
	cmd.AddCommand(newLogoutCmd(f))
	cmd.AddCommand(newDoctorCmd(f))

	return cmd
}

type loginOptions struct {
	Kind               string
	Host               string
	Username           string
	Token              string
	AuthMethod         string
	AllowInsecureStore bool
	AllowHTTP          bool
	Web                bool
	WebToken           bool
}

func newLoginCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &loginOptions{
		Kind: "dc",
	}

	cmd := &cobra.Command{
		Use:   "login [host]",
		Short: "Authenticate against a Bitbucket Data Center or Cloud host",
		Long: `Authenticate against a Bitbucket Data Center or Cloud host and store
credentials in the OS keychain.

For Data Center (--kind dc, the default), you authenticate with a Personal
Access Token (PAT). The token needs Repository Read/Write and Project Read
permissions. Use --web-token to open the PAT management page in your browser.

For Bitbucket Cloud (--kind cloud), the simplest method is --web-token, which
opens the Atlassian API token page and prompts for the token locally.
Browser-based OAuth via --web works out of the box in official release
binaries. For source and Nix builds, set BKT_OAUTH_CLIENT_ID and
BKT_OAUTH_CLIENT_SECRET before running --web. The CLI receives a short-lived
access token that is automatically refreshed.

Credentials are verified against the remote host before being stored. If no
OS keychain is available, pass --allow-insecure-store to use encrypted file
fallback. In non-interactive environments, provide --username and --token
on the command line or via stdin.`,
		Example: `  # Login to Bitbucket Cloud via OAuth
  bkt auth login https://bitbucket.org --kind cloud --web

  # Login to Bitbucket Cloud with an API token
  bkt auth login https://bitbucket.org --kind cloud --web-token

  # Interactive login to a Data Center instance
  bkt auth login https://bitbucket.example.com

  # Open browser to create a PAT, then prompt for credentials
  bkt auth login https://bitbucket.example.com --web-token

  # Non-interactive login with flags (CI pipelines)
  bkt auth login https://bitbucket.example.com --username admin --token "$PAT"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Host = args[0]
			}
			return runLogin(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Kind, "kind", opts.Kind, "Bitbucket deployment kind (dc or cloud)")
	cmd.Flags().StringVar(&opts.Username, "username", "", "Username (DC: PAT owner, Cloud: Atlassian email for API tokens)")
	cmd.Flags().StringVar(&opts.Token, "token", "", "Authentication token (DC: PAT, Cloud: API token). WARNING: visible in process list and shell history; prefer the interactive prompt")
	cmd.Flags().StringVar(&opts.AuthMethod, "auth-method", "basic", "Authentication method: basic (username+token) or bearer (token-only)")
	cmd.Flags().BoolVar(&opts.AllowInsecureStore, "allow-insecure-store", false, "Allow encrypted fallback secret storage when no OS keychain is available")
	cmd.Flags().BoolVar(&opts.AllowHTTP, "allow-http", false, "Allow http:// URLs for login even though credentials will be sent in plaintext")
	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Authenticate via OAuth in the browser (Cloud only)")
	cmd.Flags().BoolVar(&opts.WebToken, "web-token", false, "Open browser to create an API token, then prompt for credentials")

	return cmd
}

func runLogin(cmd *cobra.Command, f *cmdutil.Factory, opts *loginOptions) error {
	if secret.TokenFromEnv() != "" {
		return fmt.Errorf("%s environment variable is set; token is externally managed. Unset %s to use auth login", secret.EnvToken, secret.EnvToken)
	}

	ios, err := f.Streams()
	if err != nil {
		return err
	}

	reader := bufio.NewReader(ios.In)

	if opts.Host == "" {
		if !isTerminal(ios.In) {
			return fmt.Errorf("host is required when not running in a TTY")
		}
		opts.Host, err = promptString(reader, ios.Out, "Bitbucket base URL (e.g. https://bitbucket.example.com)")
		if err != nil {
			return err
		}
	}

	baseURL, err := cmdutil.NormalizeBaseURL(opts.Host)
	if err != nil {
		return err
	}
	if strings.HasPrefix(baseURL, "http://") {
		if !opts.AllowHTTP {
			return fmt.Errorf("http:// URLs are not allowed by default; rerun with --allow-http if you understand the credentials will be sent in plaintext")
		}
		if _, err := fmt.Fprintln(ios.ErrOut, "WARNING: using http:// will send credentials in plaintext"); err != nil {
			return err
		}
	}
	if opts.Token != "" {
		if _, err := fmt.Fprintln(ios.ErrOut, "WARNING: --token is visible in process listings and shell history; prefer the interactive prompt"); err != nil {
			return err
		}
	}

	kind := strings.ToLower(opts.Kind)
	if kind == "" {
		kind = "dc"
	}

	authMethod := strings.ToLower(strings.TrimSpace(opts.AuthMethod))
	if authMethod == "" {
		authMethod = "basic"
	}
	if authMethod != "basic" && authMethod != "bearer" {
		return fmt.Errorf("unsupported auth method %q; use \"basic\" or \"bearer\"", authMethod)
	}
	if kind == "cloud" && authMethod != "basic" && !opts.Web {
		return fmt.Errorf("--auth-method is only supported for Data Center hosts")
	}

	if opts.Web && opts.WebToken {
		return fmt.Errorf("--web and --web-token are mutually exclusive")
	}
	if opts.Web && kind == "dc" {
		return fmt.Errorf("--web OAuth login is only supported for Bitbucket Cloud; use --web-token to open the PAT page")
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	var hostKey string

	switch kind {
	case "dc":
		hostKey, err = cmdutil.HostKeyFromURL(baseURL)
		if err != nil {
			return err
		}

		if opts.WebToken && isTerminal(ios.In) {
			tokenURL := strings.TrimSuffix(baseURL, "/") + "/plugins/servlet/access-tokens/manage"
			if _, err := fmt.Fprintf(ios.Out, "Opening %s to create a Personal Access Token...\n", tokenURL); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(ios.Out, "\nRequired permissions: Repository Read, Repository Write, Project Read"); err != nil {
				return err
			}
			if err := f.BrowserOpener().Open(tokenURL); err != nil {
				if _, ferr := fmt.Fprintf(ios.Out, "Failed to open browser: %v\nPlease open the URL manually.\n", err); ferr != nil {
					return ferr
				}
			}
			if _, err := fmt.Fprintln(ios.Out, ""); err != nil {
				return err
			}
		}

		if authMethod != "bearer" {
			if opts.Username == "" {
				if !isTerminal(ios.In) {
					return fmt.Errorf("username is required when not running in a TTY")
				}
				opts.Username, err = promptString(reader, ios.Out, "Username (use x-token-auth for project/repo tokens)")
				if err != nil {
					return err
				}
			}
		}

		if opts.Token == "" {
			if !isTerminal(ios.In) {
				return fmt.Errorf("token is required when not running in a TTY")
			}
			opts.Token, err = promptSecret(ios, "Personal Access Token")
			if err != nil {
				return err
			}
		}

		client, err := bbdc.New(bbdc.Options{
			BaseURL:    baseURL,
			Username:   opts.Username,
			Token:      opts.Token,
			AuthMethod: authMethod,
		})
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		// Verify credentials by fetching user info from an authenticated endpoint.
		// /rest/api/1.0/users?limit=1 requires authentication, unlike /application-properties.
		var displayName string
		if authMethod == "bearer" && opts.Username == "" {
			var result struct {
				Values []struct {
					Name        string `json:"name"`
					DisplayName string `json:"displayName"`
				} `json:"values"`
			}
			req, reqErr := client.HTTP().NewRequest(ctx, "GET", "/rest/api/1.0/users?limit=1", nil)
			if reqErr != nil {
				return fmt.Errorf("verify credentials: %w", reqErr)
			}
			if doErr := client.HTTP().Do(req, &result); doErr != nil {
				return fmt.Errorf("verify credentials: %w", doErr)
			}
			displayName = "bearer token"
		} else {
			user, userErr := client.CurrentUser(ctx, opts.Username)
			if userErr != nil {
				return fmt.Errorf("verify credentials: %w", userErr)
			}
			displayName = cmdutil.FirstNonEmpty(user.FullName, user.Name, opts.Username)
		}

		if err := storeHostToken(hostKey, opts.Token, opts.AllowInsecureStore); err != nil {
			return fmt.Errorf("store token: %w", err)
		}

		cfg.SetHost(hostKey, &config.Host{
			Kind:               "dc",
			BaseURL:            baseURL,
			Username:           opts.Username,
			AuthMethod:         authMethod,
			AllowInsecureStore: opts.AllowInsecureStore,
		})

		if err := cfg.Save(); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "✓ Logged in to %s as %s\n", baseURL, displayName); err != nil {
			return err
		}
	case "cloud":
		apiURL := baseURL
		if strings.Contains(baseURL, "bitbucket.org") && !strings.Contains(baseURL, "api.bitbucket.org") {
			apiURL = "https://api.bitbucket.org/2.0"
		}

		hostKey, err = cmdutil.HostKeyFromURL(apiURL)
		if err != nil {
			return err
		}

		if opts.Web {
			// OAuth 2.0 browser-based flow.
			if oauth.CloudClientID() == "" || oauth.CloudClientSecret() == "" {
				return fmt.Errorf("cloud OAuth requires BKT_OAUTH_CLIENT_ID and BKT_OAUTH_CLIENT_SECRET in the environment; use --web-token for API token login")
			}
			if _, err := fmt.Fprintln(ios.Out, "Authenticating with Bitbucket Cloud via OAuth..."); err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 120*time.Second)
			defer cancel()

			result, flowErr := oauth.RunFlow(ctx, oauth.FlowOptions{
				ClientID:     oauth.CloudClientID(),
				ClientSecret: oauth.CloudClientSecret(),
				AuthorizeURL: oauth.CloudAuthorizeURL,
				TokenURL:     oauth.CloudTokenURL,
				Scopes:       oauth.CloudScopes(),
				Out:          ios.Out,
				OpenBrowser:  f.BrowserOpener().Open,
			})
			if flowErr != nil {
				return fmt.Errorf("OAuth login: %w", flowErr)
			}

			tokenBlob, marshalErr := result.Token.Marshal()
			if marshalErr != nil {
				return fmt.Errorf("encode token: %w", marshalErr)
			}

			if err := storeHostToken(hostKey, tokenBlob, opts.AllowInsecureStore); err != nil {
				return fmt.Errorf("store token: %w", err)
			}

			cfg.SetHost(hostKey, &config.Host{
				Kind:               "cloud",
				BaseURL:            apiURL,
				Username:           result.Username,
				AuthMethod:         "oauth",
				AllowInsecureStore: opts.AllowInsecureStore,
			})

			if err := cfg.Save(); err != nil {
				return err
			}

			displayLabel := result.Username
			if result.DisplayName != "" {
				displayLabel = fmt.Sprintf("%s (%s)", result.DisplayName, result.Username)
			}
			if _, err := fmt.Fprintf(ios.Out, "✓ Logged in to Bitbucket Cloud as %s via OAuth\n", displayLabel); err != nil {
				return err
			}
		} else {
			// API token flow (existing behavior).
			if opts.WebToken && isTerminal(ios.In) {
				tokenURL := CloudTokenURL
				if _, err := fmt.Fprintln(ios.Out, "Opening Atlassian to create a Bitbucket API token..."); err != nil {
					return err
				}
				if _, err := fmt.Fprintf(ios.Out, "This opens Atlassian's generic API token page: %s\n", tokenURL); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(ios.Out, "\nIMPORTANT: Click \"Create API token with scopes\" and select \"Bitbucket\" as the application."); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(ios.Out, "Existing token values cannot be viewed again; create a new token if you do not already have the token string."); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(ios.Out, "\nRequired scopes:"); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(ios.Out, "  - Account: Read / read:user:bitbucket (required for login)"); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(ios.Out, "  - Repositories: Read, Write"); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(ios.Out, "  - Pull requests: Read, Write"); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(ios.Out, "  - Issues: Read, Write (if using issue commands)"); err != nil {
					return err
				}
				if err := f.BrowserOpener().Open(tokenURL); err != nil {
					if _, ferr := fmt.Fprintf(ios.Out, "\nFailed to open browser: %v\nPlease open %s manually.\n", err, tokenURL); ferr != nil {
						return ferr
					}
				}
				if _, err := fmt.Fprintln(ios.Out, ""); err != nil {
					return err
				}
			}

			if opts.Username == "" {
				if !isTerminal(ios.In) {
					return fmt.Errorf("username is required when not running in a TTY")
				}
				opts.Username, err = promptString(reader, ios.Out, CloudEmailPrompt)
				if err != nil {
					return err
				}
			}

			if opts.Token == "" {
				if !isTerminal(ios.In) {
					return fmt.Errorf("token is required when not running in a TTY")
				}
				opts.Token, err = promptSecret(ios, CloudTokenPrompt)
				if err != nil {
					return err
				}
			}

			client, err := bbcloud.New(bbcloud.Options{
				BaseURL:     apiURL,
				Username:    opts.Username,
				Token:       opts.Token,
				EnableCache: true,
				Retry: httpx.RetryPolicy{
					MaxAttempts:    4,
					InitialBackoff: 200 * time.Millisecond,
					MaxBackoff:     2 * time.Second,
				},
			})
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()

			user, err := client.CurrentUser(ctx)
			if err != nil {
				return fmt.Errorf("verify credentials: %w", err)
			}

			if err := storeHostToken(hostKey, opts.Token, opts.AllowInsecureStore); err != nil {
				return fmt.Errorf("store token: %w", err)
			}

			cfg.SetHost(hostKey, &config.Host{
				Kind:               "cloud",
				BaseURL:            apiURL,
				Username:           opts.Username,
				AuthMethod:         "basic",
				AllowInsecureStore: opts.AllowInsecureStore,
			})

			if err := cfg.Save(); err != nil {
				return err
			}

			if _, err := fmt.Fprintf(ios.Out, "✓ Logged in to Bitbucket Cloud as %s (%s)\n", user.Display, user.Username); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported deployment kind %q", opts.Kind)
	}

	return nil
}

func newStatusCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show authentication status for configured hosts",
		Long: `Display the authentication status for all configured Bitbucket hosts and
contexts.

For each host, the output includes the base URL, deployment kind (dc or
cloud), the stored username, and the token source (OS keychain or the
BKT_TOKEN environment variable). Configured contexts are listed with their
associated host, project/workspace, and default repository.

Use --output json to get machine-readable output suitable for scripting.`,
		Example: `  # Show all configured hosts and contexts
  bkt auth status

  # Get status as JSON
  bkt auth status --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(cmd, f)
		},
	}
	return cmd
}

func runStatus(cmd *cobra.Command, f *cmdutil.Factory) error {
	ios, err := f.Streams()
	if err != nil {
		return err
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	type hostSummary struct {
		Key         string `json:"key"`
		Kind        string `json:"kind"`
		BaseURL     string `json:"base_url"`
		Username    string `json:"username,omitempty"`
		AuthMethod  string `json:"auth_method"`
		TokenSource string `json:"token_source"`
		Expires     string `json:"expires,omitempty"`
		Refresh     string `json:"refresh,omitempty"`
	}

	type contextSummary struct {
		Name        string `json:"name"`
		Host        string `json:"host"`
		ProjectKey  string `json:"project_key,omitempty"`
		Workspace   string `json:"workspace,omitempty"`
		DefaultRepo string `json:"default_repo,omitempty"`
		Active      bool   `json:"active"`
	}

	var hostKeys []string
	for key := range cfg.Hosts {
		hostKeys = append(hostKeys, key)
	}
	sort.Strings(hostKeys)

	tokenSource := resolvedTokenSource()

	var hosts []hostSummary
	for _, key := range hostKeys {
		h := cfg.Hosts[key]
		am := detectedAuthMethod(key, h, tokenSource)
		hs := hostSummary{
			Key:         key,
			Kind:        h.Kind,
			BaseURL:     h.BaseURL,
			Username:    h.Username,
			AuthMethod:  am,
			TokenSource: tokenSource,
		}
		if am == "oauth" && tokenSource != secret.EnvToken {
			hs.Expires = oauthExpiryLabel(key, h)
			hs.Refresh = oauthRefreshStatus(hs.Expires)
		}
		hosts = append(hosts, hs)
	}

	var contextNames []string
	for name := range cfg.Contexts {
		contextNames = append(contextNames, name)
	}
	sort.Strings(contextNames)

	var contexts []contextSummary
	for _, name := range contextNames {
		ctx := cfg.Contexts[name]
		contexts = append(contexts, contextSummary{
			Name:        name,
			Host:        ctx.Host,
			ProjectKey:  ctx.ProjectKey,
			Workspace:   ctx.Workspace,
			DefaultRepo: ctx.DefaultRepo,
			Active:      cfg.ActiveContext == name,
		})
	}

	payload := struct {
		ActiveContext string           `json:"active_context,omitempty"`
		Hosts         []hostSummary    `json:"hosts"`
		Contexts      []contextSummary `json:"contexts"`
	}{
		ActiveContext: cfg.ActiveContext,
		Hosts:         hosts,
		Contexts:      contexts,
	}

	return cmdutil.WriteOutput(cmd, ios.Out, payload, func() error {
		if len(hosts) == 0 {
			if _, err := fmt.Fprintln(ios.Out, "No hosts configured. Run `bkt auth login` to add one."); err != nil {
				return err
			}
			return nil
		}

		if _, err := fmt.Fprintln(ios.Out, "Hosts:"); err != nil {
			return err
		}
		for _, h := range hosts {
			if _, err := fmt.Fprintf(ios.Out, "  %s (%s)\n", h.BaseURL, h.Kind); err != nil {
				return err
			}
			if h.Username != "" {
				if _, err := fmt.Fprintf(ios.Out, "    user: %s\n", h.Username); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(ios.Out, "    auth: %s\n", h.AuthMethod); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(ios.Out, "    token source: %s\n", h.TokenSource); err != nil {
				return err
			}
			if h.Expires != "" {
				if _, err := fmt.Fprintf(ios.Out, "    expires: %s\n", h.Expires); err != nil {
					return err
				}
			}
			if h.Refresh != "" {
				if _, err := fmt.Fprintf(ios.Out, "    refresh: %s\n", h.Refresh); err != nil {
					return err
				}
			}
		}

		if len(contexts) == 0 {
			_, err := fmt.Fprintf(ios.Out, "\nNo contexts configured. Use `%s context create` to add one.\n", f.ExecutableName)
			return err
		}

		if _, err := fmt.Fprintln(ios.Out, "\nContexts:"); err != nil {
			return err
		}
		for _, ctx := range contexts {
			activeMarker := " "
			if ctx.Active {
				activeMarker = "*"
			}
			if _, err := fmt.Fprintf(ios.Out, "  %s %s (host: %s)\n", activeMarker, ctx.Name, ctx.Host); err != nil {
				return err
			}
			if ctx.ProjectKey != "" {
				if _, err := fmt.Fprintf(ios.Out, "    project: %s\n", ctx.ProjectKey); err != nil {
					return err
				}
			}
			if ctx.Workspace != "" {
				if _, err := fmt.Fprintf(ios.Out, "    workspace: %s\n", ctx.Workspace); err != nil {
					return err
				}
			}
			if ctx.DefaultRepo != "" {
				if _, err := fmt.Fprintf(ios.Out, "    repo: %s\n", ctx.DefaultRepo); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

type logoutOptions struct {
	Host string
}

func newLogoutCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &logoutOptions{}

	cmd := &cobra.Command{
		Use:   "logout [host]",
		Short: "Remove stored credentials for a host",
		Long: `Remove stored credentials for a Bitbucket host and delete the host entry
from the configuration file.

The host can be specified as a positional argument or with the --host flag,
using either the host key (e.g. "bitbucket.example.com") or the full base
URL. Any contexts associated with the removed host are also deleted.

This command does not work when the BKT_TOKEN environment variable is set,
because the token is externally managed in that case.`,
		Example: `  # Remove credentials by host key
  bkt auth logout bitbucket.example.com

  # Remove credentials by base URL
  bkt auth logout https://bitbucket.example.com

  # Remove Bitbucket Cloud credentials
  bkt auth logout api.bitbucket.org`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				opts.Host = args[0]
			}
			return runLogout(cmd, f, opts)
		},
	}

	cmd.Flags().StringVar(&opts.Host, "host", "", "Host key or base URL to remove")

	return cmd
}

func runLogout(cmd *cobra.Command, f *cmdutil.Factory, opts *logoutOptions) error {
	if secret.TokenFromEnv() != "" {
		return fmt.Errorf("%s environment variable is set; token is externally managed. Unset %s to use auth logout", secret.EnvToken, secret.EnvToken)
	}

	ios, err := f.Streams()
	if err != nil {
		return err
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	hostIdentifier := strings.TrimSpace(opts.Host)
	if hostIdentifier == "" {
		return fmt.Errorf("host is required")
	}

	key := hostIdentifier
	if _, ok := cfg.Hosts[key]; !ok {
		baseURL, err := cmdutil.NormalizeBaseURL(hostIdentifier)
		if err != nil {
			return fmt.Errorf("unknown host %q", hostIdentifier)
		}
		key, err = cmdutil.HostKeyFromURL(baseURL)
		if err != nil {
			return err
		}
		if _, ok := cfg.Hosts[key]; !ok {
			return fmt.Errorf("host %q not found in configuration", hostIdentifier)
		}
	}

	host := cfg.Hosts[key]
	if err := deleteHostToken(key, host); err != nil {
		return fmt.Errorf("delete credentials: %w", err)
	}

	cfg.DeleteHost(key)

	for name, ctx := range cfg.Contexts {
		if ctx.Host == key {
			cfg.DeleteContext(name)
		}
	}

	if err := cfg.Save(); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(ios.Out, "✓ Removed credentials for %s\n", key); err != nil {
		return err
	}
	return nil
}

func storeHostToken(hostKey, token string, allowInsecure bool) error {
	opts := []secret.Option{}
	if allowInsecure {
		opts = append(opts, secret.WithAllowFileFallback(true))
	}

	store, err := secret.Open(opts...)
	if err != nil {
		return err
	}

	key := secret.TokenKey(hostKey)

	// On darwin, an existing Keychain item's ACL is preserved by the update
	// path in 99designs/keyring (kcItem.SetAccess(nil) in updateItem). That
	// means a stale ACL entry from a previous bkt binary with a different
	// Designated Requirement — typically after a Homebrew upgrade — keeps
	// prompting forever. Delete first so Set() takes the create path with the
	// current binary's DR as the trusted app.
	if runtime.GOOS == "darwin" {
		if err := store.Delete(key); err != nil {
			return err
		}
	}

	return store.Set(key, token)
}

func deleteHostToken(hostKey string, host *config.Host) error {
	if host == nil {
		return fmt.Errorf("host %q not configured", hostKey)
	}

	opts := []secret.Option{}
	if host.AllowInsecureStore {
		opts = append(opts, secret.WithAllowFileFallback(true))
	}

	store, err := secret.Open(opts...)
	if err != nil {
		return err
	}

	if err := store.Delete(secret.TokenKey(hostKey)); err != nil {
		return err
	}
	host.Token = ""
	return nil
}

func promptString(reader *bufio.Reader, out io.Writer, label string) (string, error) {
	if _, err := fmt.Fprintf(out, "%s: ", label); err != nil {
		return "", err
	}
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func promptSecret(ios *iostreams.IOStreams, label string) (string, error) {
	file, ok := ios.In.(*os.File)
	if ok && term.IsTerminal(int(file.Fd())) {
		if _, err := fmt.Fprintf(ios.Out, "%s: ", label); err != nil {
			return "", err
		}
		bytes, err := term.ReadPassword(int(file.Fd()))
		if _, ferr := fmt.Fprintln(ios.Out); ferr != nil {
			return "", ferr
		}
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(bytes)), nil
	}

	reader := bufio.NewReader(ios.In)
	return promptString(reader, ios.Out, label)
}

func isTerminal(in io.Reader) bool {
	file, ok := in.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

// resolvedTokenSource returns the active token resolution strategy.
// When BKT_TOKEN is set it applies globally to all hosts; otherwise
// the strategy is keyring-based (actual token presence is not checked).
func resolvedTokenSource() string {
	if secret.TokenFromEnv() != "" {
		return secret.EnvToken
	}
	return "keyring"
}

func detectedAuthMethod(hostKey string, host *config.Host, tokenSource string) string {
	if tokenSource == secret.EnvToken {
		if m := strings.TrimSpace(os.Getenv(secret.EnvAuthMethod)); m != "" {
			return m
		}
		return "basic"
	}
	if host == nil {
		return "basic"
	}
	if host.AuthMethod != "" {
		return host.AuthMethod
	}
	if host.Kind != "cloud" {
		return "basic"
	}
	if storedTokenIsOAuthBlob(hostKey, host) {
		return "oauth"
	}
	return "basic"
}

func storedTokenIsOAuthBlob(hostKey string, host *config.Host) bool {
	opts := []secret.Option{}
	if host != nil && host.AllowInsecureStore {
		opts = append(opts, secret.WithAllowFileFallback(true))
	}
	store, err := secret.Open(opts...)
	if err != nil {
		return false
	}
	raw, err := store.Get(secret.TokenKey(hostKey))
	if err != nil {
		return false
	}
	return oauth.IsTokenBlob(raw)
}

// oauthExpiryLabel reads the OAuth JSON blob from the keyring and returns a
// human-readable expiry string. Returns "" on any error (best-effort).
func oauthExpiryLabel(hostKey string, host *config.Host) string {
	opts := []secret.Option{}
	if host.AllowInsecureStore {
		opts = append(opts, secret.WithAllowFileFallback(true))
	}
	store, err := secret.Open(opts...)
	if err != nil {
		return ""
	}
	raw, err := store.Get(secret.TokenKey(hostKey))
	if err != nil {
		return ""
	}
	if !oauth.IsTokenBlob(raw) {
		return ""
	}
	tok, err := oauth.Unmarshal(raw)
	if err != nil {
		return ""
	}
	if tok.IsExpired() {
		return "expired"
	}
	remaining := time.Until(tok.ExpiresAt).Truncate(time.Minute)
	return fmt.Sprintf("%s (in %s)", tok.ExpiresAt.Format(time.RFC3339), remaining)
}

func oauthRefreshStatus(expires string) string {
	if expires != "expired" {
		return ""
	}
	if oauth.CloudClientID() == "" || oauth.CloudClientSecret() == "" {
		return "unavailable; set BKT_OAUTH_CLIENT_ID and BKT_OAUTH_CLIENT_SECRET to refresh OAuth, or run `bkt auth login https://bitbucket.org --kind cloud --web-token` to replace it with a scoped API token"
	}
	return "available"
}

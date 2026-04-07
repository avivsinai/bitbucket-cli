package auth

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
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
)

// CloudTokenURL is the URL where users create Bitbucket Cloud API tokens.
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
	}

	cmd.AddCommand(newLoginCmd(f))
	cmd.AddCommand(newStatusCmd(f))
	cmd.AddCommand(newLogoutCmd(f))

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
}

func newLoginCmd(f *cmdutil.Factory) *cobra.Command {
	opts := &loginOptions{
		Kind: "dc",
	}

	cmd := &cobra.Command{
		Use:   "login [host]",
		Short: "Authenticate against a Bitbucket Data Center or Cloud host",
		Args:  cobra.MaximumNArgs(1),
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
	cmd.Flags().BoolVarP(&opts.Web, "web", "w", false, "Open browser to create token, then prompt for credentials")

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
	if kind == "cloud" && authMethod != "basic" {
		return fmt.Errorf("--auth-method is only supported for Data Center hosts")
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

		if opts.Web && isTerminal(ios.In) {
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
		if opts.Web && isTerminal(ios.In) {
			tokenURL := CloudTokenURL
			if _, err := fmt.Fprintln(ios.Out, "Opening Atlassian to create a Bitbucket API token..."); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(ios.Out, "\nIMPORTANT: Click \"Create API token with scopes\" and select \"Bitbucket\" as the application."); err != nil {
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

		apiURL := baseURL
		if strings.Contains(baseURL, "bitbucket.org") && !strings.Contains(baseURL, "api.bitbucket.org") {
			apiURL = "https://api.bitbucket.org/2.0"
		}

		hostKey, err = cmdutil.HostKeyFromURL(apiURL)
		if err != nil {
			return err
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
			AllowInsecureStore: opts.AllowInsecureStore,
		})

		if err := cfg.Save(); err != nil {
			return err
		}

		if _, err := fmt.Fprintf(ios.Out, "✓ Logged in to Bitbucket Cloud as %s (%s)\n", user.Display, user.Username); err != nil {
			return err
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
		am := h.AuthMethod
		if am == "" {
			am = "basic"
		}
		hosts = append(hosts, hostSummary{
			Key:         key,
			Kind:        h.Kind,
			BaseURL:     h.BaseURL,
			Username:    h.Username,
			AuthMethod:  am,
			TokenSource: tokenSource,
		})
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
		Args:  cobra.MaximumNArgs(1),
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

	return store.Set(secret.TokenKey(hostKey), token)
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

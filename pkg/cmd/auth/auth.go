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
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/format"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

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
	Kind     string
	Host     string
	Username string
	Token    string
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
	cmd.Flags().StringVar(&opts.Username, "username", "", "Username for authentication (PAT owner or x-token-auth for HTTP tokens)")
	cmd.Flags().StringVar(&opts.Token, "token", "", "Personal access token or HTTP access token")

	return cmd
}

func runLogin(cmd *cobra.Command, f *cmdutil.Factory, opts *loginOptions) error {
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
	hostKey, err := cmdutil.HostKeyFromURL(baseURL)
	if err != nil {
		return err
	}

	kind := strings.ToLower(opts.Kind)
	if kind == "" {
		kind = "dc"
	}

	cfg, err := f.ResolveConfig()
	if err != nil {
		return err
	}

	switch kind {
	case "dc":
		if opts.Username == "" {
			if !isTerminal(ios.In) {
				return fmt.Errorf("username is required when not running in a TTY")
			}
			opts.Username, err = promptString(reader, ios.Out, "Username (use x-token-auth for project/repo tokens)")
			if err != nil {
				return err
			}
		}

		if opts.Token == "" {
			if !isTerminal(ios.In) {
				return fmt.Errorf("token is required when not running in a TTY")
			}
			opts.Token, err = promptSecret(ios, "Token")
			if err != nil {
				return err
			}
		}

		client, err := bbdc.New(bbdc.Options{
			BaseURL:  baseURL,
			Username: opts.Username,
			Token:    opts.Token,
		})
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
		defer cancel()

		user, err := client.CurrentUser(ctx, opts.Username)
		if err != nil {
			return fmt.Errorf("verify credentials: %w", err)
		}

		cfg.SetHost(hostKey, &config.Host{
			Kind:     "dc",
			BaseURL:  baseURL,
			Username: opts.Username,
			Token:    opts.Token,
		})

		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Fprintf(ios.Out, "✓ Logged in to %s as %s (%s)\n", baseURL, user.FullName, user.Name)
	default:
		return fmt.Errorf("cloud authentication is not yet implemented")
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

	formatOpt, err := cmdutil.OutputFormat(cmd)
	if err != nil {
		return err
	}

	type hostSummary struct {
		Key      string `json:"key"`
		Kind     string `json:"kind"`
		BaseURL  string `json:"base_url"`
		Username string `json:"username,omitempty"`
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

	var hosts []hostSummary
	for _, key := range hostKeys {
		h := cfg.Hosts[key]
		hosts = append(hosts, hostSummary{
			Key:      key,
			Kind:     h.Kind,
			BaseURL:  h.BaseURL,
			Username: h.Username,
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

	if formatOpt != "" {
		payload := struct {
			ActiveContext string           `json:"active_context,omitempty"`
			Hosts         []hostSummary    `json:"hosts"`
			Contexts      []contextSummary `json:"contexts"`
		}{
			ActiveContext: cfg.ActiveContext,
			Hosts:         hosts,
			Contexts:      contexts,
		}
		return format.Write(ios.Out, formatOpt, payload, nil)
	}

	if len(hosts) == 0 {
		fmt.Fprintln(ios.Out, "No hosts configured. Run `bkt auth login` to add one.")
		return nil
	}

	fmt.Fprintln(ios.Out, "Hosts:")
	for _, h := range hosts {
		fmt.Fprintf(ios.Out, "  %s (%s)\n", h.BaseURL, h.Kind)
		if h.Username != "" {
			fmt.Fprintf(ios.Out, "    user: %s\n", h.Username)
		}
	}

	if len(contexts) == 0 {
		fmt.Fprintf(ios.Out, "\nNo contexts configured. Use `%s context create` to add one.\n", f.ExecutableName)
		return nil
	}

	fmt.Fprintln(ios.Out, "\nContexts:")
	for _, ctx := range contexts {
		activeMarker := " "
		if ctx.Active {
			activeMarker = "*"
		}
		fmt.Fprintf(ios.Out, "  %s %s (host: %s)\n", activeMarker, ctx.Name, ctx.Host)
		if ctx.ProjectKey != "" {
			fmt.Fprintf(ios.Out, "    project: %s\n", ctx.ProjectKey)
		}
		if ctx.Workspace != "" {
			fmt.Fprintf(ios.Out, "    workspace: %s\n", ctx.Workspace)
		}
		if ctx.DefaultRepo != "" {
			fmt.Fprintf(ios.Out, "    repo: %s\n", ctx.DefaultRepo)
		}
	}

	return nil
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

	cfg.DeleteHost(key)

	for name, ctx := range cfg.Contexts {
		if ctx.Host == key {
			cfg.DeleteContext(name)
		}
	}

	if err := cfg.Save(); err != nil {
		return err
	}

	fmt.Fprintf(ios.Out, "✓ Removed credentials for %s\n", key)
	return nil
}

func promptString(reader *bufio.Reader, out io.Writer, label string) (string, error) {
	fmt.Fprintf(out, "%s: ", label)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func promptSecret(ios *iostreams.IOStreams, label string) (string, error) {
	file, ok := ios.In.(*os.File)
	if ok && term.IsTerminal(int(file.Fd())) {
		fmt.Fprintf(ios.Out, "%s: ", label)
		bytes, err := term.ReadPassword(int(file.Fd()))
		fmt.Fprintln(ios.Out)
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

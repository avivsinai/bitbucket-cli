package cmdutil

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/internal/filelock"
	"github.com/avivsinai/bitbucket-cli/internal/secret"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
	"github.com/avivsinai/bitbucket-cli/pkg/oauth"
)

// NewDCClient constructs a Bitbucket Data Center client using the supplied host.
func NewDCClient(host *config.Host) (*bbdc.Client, error) {
	if host == nil {
		return nil, fmt.Errorf("missing host configuration")
	}
	if host.BaseURL == "" {
		return nil, fmt.Errorf("host %q has no base URL configured", host.Kind)
	}
	extra, err := ExtraHeadersFromEnv(host.BaseURL)
	if err != nil {
		return nil, err
	}
	opts := bbdc.Options{
		BaseURL:     host.BaseURL,
		Username:    host.Username,
		Token:       host.Token,
		AuthMethod:  host.AuthMethod,
		EnableCache: true,
		Retry: httpx.RetryPolicy{
			MaxAttempts:    4,
			InitialBackoff: 250 * time.Millisecond,
			MaxBackoff:     2 * time.Second,
		},
		ExtraHeaders: extra,
	}
	return bbdc.New(opts)
}

// NewCloudClient constructs a Bitbucket Cloud client using the supplied host.
// When the host uses OAuth authentication, a TokenRefresher is wired to
// transparently refresh expired tokens on 401.
func NewCloudClient(host *config.Host) (*bbcloud.Client, error) {
	return newCloudClient(host, true)
}

// RefreshExpiredCloudOAuth refreshes a saved OAuth token before a Cloud
// mutation that cannot be replayed on 401. Use the resolved host to construct
// the client after this call so the request uses the new credential.
func RefreshExpiredCloudOAuth(ctx context.Context, host *config.Host) error {
	if !usesKeyringOAuth(host) || host.OAuthExpiresAt.IsZero() {
		return nil
	}
	token := oauth.Token{ExpiresAt: host.OAuthExpiresAt}
	if !token.IsExpired() {
		return nil
	}
	hostKey, err := credentialKey(host)
	if err != nil {
		return fmt.Errorf("resolve host key: %w", err)
	}
	_, err = oauthTokenRefresher(hostKey, host)(ctx)
	return err
}

// NewFrozenCloudClient constructs a Bitbucket Cloud client that uses exactly
// the supplied credential for its lifetime. OAuth credentials use bearer auth,
// but the client never reopens the credential store or refreshes on 401.
func NewFrozenCloudClient(host *config.Host) (*bbcloud.Client, error) {
	if host == nil {
		return nil, fmt.Errorf("missing host configuration")
	}
	frozen := *host
	return newCloudClient(&frozen, false)
}

func newCloudClient(host *config.Host, enableOAuthRefresh bool) (*bbcloud.Client, error) {
	if host == nil {
		return nil, fmt.Errorf("missing host configuration")
	}
	if host.BaseURL == "" {
		host.BaseURL = "https://api.bitbucket.org/2.0"
	}
	extra, err := ExtraHeadersFromEnv(host.BaseURL)
	if err != nil {
		return nil, err
	}
	opts := bbcloud.Options{
		BaseURL:     host.BaseURL,
		Username:    host.Username,
		Token:       host.Token,
		EnableCache: true,
		Retry: httpx.RetryPolicy{
			MaxAttempts:    4,
			InitialBackoff: 250 * time.Millisecond,
			MaxBackoff:     2 * time.Second,
		},
		ExtraHeaders: extra,
	}

	if host.AuthMethod != "oauth" {
		opts.AuthMethod = host.AuthMethod
	}
	// Keyring-stored OAuth tokens use Bearer auth. When BKT_TOKEN overrides,
	// the caller controls the token type and the auth method defaults to
	// basic (matching API-token behavior), for frozen clients too.
	keyringOAuth := usesKeyringOAuth(host)
	if keyringOAuth {
		opts.AuthMethod = "bearer"
	}
	if enableOAuthRefresh && keyringOAuth {
		hostKey, err := credentialKey(host)
		if err != nil {
			return nil, fmt.Errorf("resolve host key: %w", err)
		}
		opts.TokenRefresher = oauthTokenRefresher(hostKey, host)
	}

	return bbcloud.New(opts)
}

func usesKeyringOAuth(host *config.Host) bool {
	return host != nil && host.AuthMethod == "oauth" && secret.TokenFromEnv() == ""
}

// credentialKey returns the key that a host's token is stored under: its own
// config key, which may be an alias (the base URL names the server, not the
// account), or the server's key for a host that did not come from the config.
func credentialKey(host *config.Host) (string, error) {
	if key := host.Key(); key != "" {
		return key, nil
	}
	return HostKeyFromURL(host.BaseURL)
}

// oauthTokenRefresher returns a function that refreshes the OAuth token and
// updates the keyring. Captured values are the host key and host config so the
// closure can persist the new token.
func oauthTokenRefresher(hostKey string, host *config.Host) func(ctx context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		lockPath, err := oauthRefreshLockPath(hostKey)
		if err != nil {
			return "", fmt.Errorf("resolve OAuth refresh lock: %w", err)
		}

		var accessToken string
		err = filelock.With(lockPath, func() error {
			store, err := secret.Open(secretOpts(host)...)
			if err != nil {
				return fmt.Errorf("open secret store: %w", err)
			}

			raw, err := store.Get(secret.TokenKey(hostKey))
			if err != nil {
				return fmt.Errorf("read token: %w", err)
			}

			tok, err := oauth.Unmarshal(raw)
			if err != nil {
				return fmt.Errorf("parse stored token: %w", err)
			}

			if !tok.IsExpired() && tok.AccessToken != host.Token {
				accessToken = tok.AccessToken
				host.OAuthExpiresAt = tok.ExpiresAt
				return nil
			}
			clients, err := oauth.CloudClientsFor(tok)
			if err != nil {
				return fmt.Errorf("refresh OAuth token for %s (re-login with `bkt auth login %s --kind cloud --web`): %w", hostKey, LoginTarget(hostKey, host), err)
			}

			var newTok *oauth.Token
			for _, client := range clients {
				newTok, err = oauth.RefreshToken(ctx,
					tok.RefreshToken,
					client.ID,
					client.Secret,
					oauth.CloudTokenURL,
				)
				if err == nil {
					break
				}
			}
			if err != nil {
				return fmt.Errorf("refresh failed (re-login with `bkt auth login %s --kind cloud --web`): %w", LoginTarget(hostKey, host), err)
			}

			blob, err := newTok.Marshal()
			if err != nil {
				return fmt.Errorf("encode refreshed token: %w", err)
			}
			if err := store.Set(secret.TokenKey(hostKey), blob); err != nil {
				return fmt.Errorf("store refreshed token: %w", err)
			}

			accessToken = newTok.AccessToken
			host.OAuthExpiresAt = newTok.ExpiresAt
			return nil
		})
		if err != nil {
			return "", err
		}

		host.Token = accessToken
		return accessToken, nil
	}
}

func oauthRefreshLockPath(hostKey string) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(dir, "bkt", "locks", "oauth-refresh-"+url.PathEscape(hostKey)+".lock"), nil
}

func secretOpts(host *config.Host) []secret.Option {
	if host != nil && host.AllowInsecureStore {
		return []secret.Option{secret.WithAllowFileFallback(true)}
	}
	return nil
}

// NewHTTPClient constructs a raw HTTP client for the configured host.
func NewHTTPClient(host *config.Host) (*httpx.Client, error) {
	if host == nil {
		return nil, fmt.Errorf("missing host configuration")
	}

	switch host.Kind {
	case "dc":
		client, err := NewDCClient(host)
		if err != nil {
			return nil, err
		}
		return client.HTTP(), nil
	case "cloud":
		client, err := NewCloudClient(host)
		if err != nil {
			return nil, err
		}
		return client.HTTP(), nil
	default:
		return nil, fmt.Errorf("unsupported host kind %q", host.Kind)
	}
}

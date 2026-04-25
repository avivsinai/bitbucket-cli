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
	}
	return bbdc.New(opts)
}

// NewCloudClient constructs a Bitbucket Cloud client using the supplied host.
// When the host uses OAuth authentication, a TokenRefresher is wired to
// transparently refresh expired tokens on 401.
func NewCloudClient(host *config.Host) (*bbcloud.Client, error) {
	if host == nil {
		return nil, fmt.Errorf("missing host configuration")
	}
	if host.BaseURL == "" {
		host.BaseURL = "https://api.bitbucket.org/2.0"
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
	}

	if host.AuthMethod == "oauth" && secret.TokenFromEnv() == "" {
		// Keyring-stored OAuth tokens use Bearer auth + auto-refresh.
		// When BKT_TOKEN overrides, the caller controls the token type
		// and auth method defaults to basic (matching API-token behavior).
		opts.AuthMethod = "bearer"
		hostKey, err := HostKeyFromURL(host.BaseURL)
		if err != nil {
			return nil, fmt.Errorf("resolve host key: %w", err)
		}
		if err := preflightExpiredOAuth(hostKey, host); err != nil {
			return nil, err
		}
		opts.TokenRefresher = oauthTokenRefresher(hostKey, host)
	}

	return bbcloud.New(opts)
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
			if oauth.CloudClientID() == "" || oauth.CloudClientSecret() == "" {
				return oauthMissingCredsError(hostKey, tok.ExpiresAt)
			}

			newTok, err := oauth.RefreshToken(ctx,
				tok.RefreshToken,
				oauth.CloudClientID(),
				oauth.CloudClientSecret(),
				oauth.CloudTokenURL,
			)
			if err != nil {
				return fmt.Errorf("refresh failed (re-login with `bkt auth login --web`): %w", err)
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

func preflightExpiredOAuth(hostKey string, host *config.Host) error {
	if host == nil || host.OAuthExpiresAt.IsZero() || time.Now().Before(host.OAuthExpiresAt) {
		return nil
	}
	if oauth.CloudClientID() != "" && oauth.CloudClientSecret() != "" {
		return nil
	}
	return oauthMissingCredsError(hostKey, host.OAuthExpiresAt)
}

func oauthMissingCredsError(hostKey string, expiresAt time.Time) error {
	if !expiresAt.IsZero() && time.Now().After(expiresAt) {
		return fmt.Errorf("oauth access token for %s expired at %s, and BKT_OAUTH_CLIENT_ID / BKT_OAUTH_CLIENT_SECRET are not set. Run `bkt auth login https://bitbucket.org --kind cloud --web-token` to replace it with a scoped API token, or restore the OAuth consumer env vars", hostKey, expiresAt.Format(time.RFC3339))
	}
	return fmt.Errorf("cloud OAuth consumer credentials are missing for %s; set BKT_OAUTH_CLIENT_ID and BKT_OAUTH_CLIENT_SECRET to refresh OAuth, or run `bkt auth login https://bitbucket.org --kind cloud --web-token` to replace the stored OAuth credential with a scoped API token", hostKey)
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

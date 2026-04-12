package cmdutil

import (
	"context"
	"fmt"
	"time"

	"github.com/avivsinai/bitbucket-cli/internal/config"
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
		opts.TokenRefresher = oauthTokenRefresher(hostKey, host)
	}

	return bbcloud.New(opts)
}

// oauthTokenRefresher returns a function that refreshes the OAuth token and
// updates the keyring. Captured values are the host key and host config so the
// closure can persist the new token.
func oauthTokenRefresher(hostKey string, host *config.Host) func(ctx context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		if oauth.CloudClientID() == "" || oauth.CloudClientSecret() == "" {
			return "", fmt.Errorf("token expired and OAuth client credentials are missing; rebuild with BKT_OAUTH_CLIENT_ID/BKT_OAUTH_CLIENT_SECRET or re-login with `bkt auth login --web-token`")
		}

		store, err := secret.Open(secretOpts(host)...)
		if err != nil {
			return "", fmt.Errorf("open secret store: %w", err)
		}

		raw, err := store.Get(secret.TokenKey(hostKey))
		if err != nil {
			return "", fmt.Errorf("read token: %w", err)
		}

		tok, err := oauth.Unmarshal(raw)
		if err != nil {
			return "", fmt.Errorf("parse stored token: %w", err)
		}

		newTok, err := oauth.RefreshToken(ctx,
			tok.RefreshToken,
			oauth.CloudClientID(),
			oauth.CloudClientSecret(),
			oauth.CloudTokenURL,
		)
		if err != nil {
			return "", fmt.Errorf("refresh failed (re-login with `bkt auth login --web`): %w", err)
		}

		blob, err := newTok.Marshal()
		if err != nil {
			return "", fmt.Errorf("encode refreshed token: %w", err)
		}
		if err := store.Set(secret.TokenKey(hostKey), blob); err != nil {
			return "", fmt.Errorf("store refreshed token: %w", err)
		}

		host.Token = newTok.AccessToken
		return newTok.AccessToken, nil
	}
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

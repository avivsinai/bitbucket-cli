package cmdutil

import (
	"fmt"
	"time"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
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
	return bbcloud.New(opts)
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

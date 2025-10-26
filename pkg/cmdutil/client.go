package cmdutil

import (
	"fmt"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/bbdc"
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
		BaseURL:  host.BaseURL,
		Username: host.Username,
		Token:    host.Token,
	}
	return bbdc.New(opts)
}

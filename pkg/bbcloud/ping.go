package bbcloud

import (
	"context"

	"github.com/example/bitbucket-cli/pkg/httpx"
)

// Ping performs a lightweight request to update rate limit telemetry.
func (c *Client) Ping(ctx context.Context) error {
	req, err := c.http.NewRequest(ctx, "GET", "/user", nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// RateLimit exposes the last recorded rate limit data.
func (c *Client) RateLimit() httpx.RateLimit {
	return c.http.RateLimitState()
}

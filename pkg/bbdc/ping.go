package bbdc

import (
	"context"

	"github.com/avivsinai/bitbucket-cli/pkg/httpx"
)

// Ping issues a lightweight request to populate telemetry such as rate limits.
func (c *Client) Ping(ctx context.Context) error {
	req, err := c.http.NewRequest(ctx, "GET", "/rest/api/1.0/application-properties", nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// RateLimit returns the last observed rate limit headers.
func (c *Client) RateLimit() httpx.RateLimit {
	return c.http.RateLimitState()
}

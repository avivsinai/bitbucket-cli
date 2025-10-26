package bbdc

import "context"

// RotateSecret triggers encryption key rotation via the secrets manager plugin.
func (c *Client) RotateSecret(ctx context.Context) error {
	req, err := c.http.NewRequest(ctx, "POST", "/rest/secrets-manager/1.0/keys/rotate", nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

// LoggingConfig captures logging control settings.
type LoggingConfig struct {
	Level string `json:"level"`
	Async bool   `json:"async"`
}

// GetLoggingConfig fetches the current logging configuration.
func (c *Client) GetLoggingConfig(ctx context.Context) (*LoggingConfig, error) {
	req, err := c.http.NewRequest(ctx, "GET", "/rest/api/1.0/admin/logs/settings", nil)
	if err != nil {
		return nil, err
	}

	var cfg LoggingConfig
	if err := c.http.Do(req, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// UpdateLoggingConfig updates logging level/async mode.
func (c *Client) UpdateLoggingConfig(ctx context.Context, cfg LoggingConfig) error {
	req, err := c.http.NewRequest(ctx, "PUT", "/rest/api/1.0/admin/logs/settings", cfg)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

package bbcloud

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Webhook models a Bitbucket Cloud repository webhook.
type Webhook struct {
	UUID        string   `json:"uuid"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	Active      bool     `json:"active"`
}

// WebhookInput configures webhook creation.
type WebhookInput struct {
	Description string
	URL         string
	Events      []string
	Active      bool
}

// ListWebhooks enumerates repository webhooks.
func (c *Client) ListWebhooks(ctx context.Context, workspace, repoSlug string) ([]Webhook, error) {
	path := fmt.Sprintf("/repositories/%s/%s/hooks",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)
	req, err := c.http.NewRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Values []Webhook `json:"values"`
	}
	if err := c.http.Do(req, &resp); err != nil {
		return nil, err
	}
	return resp.Values, nil
}

// CreateWebhook creates a new repository webhook.
func (c *Client) CreateWebhook(ctx context.Context, workspace, repoSlug string, input WebhookInput) (*Webhook, error) {
	if input.URL == "" {
		return nil, fmt.Errorf("webhook url is required")
	}
	if len(input.Events) == 0 {
		return nil, fmt.Errorf("at least one event is required")
	}

	body := map[string]any{
		"description": input.Description,
		"url":         input.URL,
		"events":      input.Events,
		"active":      input.Active,
	}

	path := fmt.Sprintf("/repositories/%s/%s/hooks",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
	)
	req, err := c.http.NewRequest(ctx, "POST", path, body)
	if err != nil {
		return nil, err
	}

	var hook Webhook
	if err := c.http.Do(req, &hook); err != nil {
		return nil, err
	}

	return &hook, nil
}

// DeleteWebhook removes a webhook by uuid.
func (c *Client) DeleteWebhook(ctx context.Context, workspace, repoSlug, uuid string) error {
	path := fmt.Sprintf("/repositories/%s/%s/hooks/%s",
		url.PathEscape(workspace),
		url.PathEscape(repoSlug),
		url.PathEscape(strings.Trim(uuid, "{}")),
	)
	req, err := c.http.NewRequest(ctx, "DELETE", path, nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

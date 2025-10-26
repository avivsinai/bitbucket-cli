package bbdc

import (
	"context"
	"fmt"
	"net/url"
)

// Webhook represents a Bitbucket webhook configuration.
type Webhook struct {
	ID            int            `json:"id"`
	Name          string         `json:"name"`
	URL           string         `json:"url"`
	Active        bool           `json:"active"`
	Events        []string       `json:"events"`
	Configuration map[string]any `json:"configuration,omitempty"`
}

// ListWebhooks retrieves repository webhooks.
func (c *Client) ListWebhooks(ctx context.Context, projectKey, repoSlug string) ([]Webhook, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}
	req, err := c.http.NewRequest(ctx, "GET", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/webhooks",
		url.PathEscape(projectKey), url.PathEscape(repoSlug)), nil)
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

// CreateWebhookInput describes webhook creation request.
type CreateWebhookInput struct {
	Name   string
	URL    string
	Events []string
	Active bool
}

// CreateWebhook registers a webhook for the repository.
func (c *Client) CreateWebhook(ctx context.Context, projectKey, repoSlug string, in CreateWebhookInput) (*Webhook, error) {
	if projectKey == "" || repoSlug == "" {
		return nil, fmt.Errorf("project key and repository slug are required")
	}
	if in.Name == "" || in.URL == "" || len(in.Events) == 0 {
		return nil, fmt.Errorf("name, url, and at least one event are required")
	}

	body := map[string]any{
		"name":   in.Name,
		"url":    in.URL,
		"events": in.Events,
		"active": in.Active,
	}

	req, err := c.http.NewRequest(ctx, "POST", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/webhooks",
		url.PathEscape(projectKey), url.PathEscape(repoSlug)), body)
	if err != nil {
		return nil, err
	}

	var hook Webhook
	if err := c.http.Do(req, &hook); err != nil {
		return nil, err
	}
	return &hook, nil
}

// DeleteWebhook removes a webhook by ID.
func (c *Client) DeleteWebhook(ctx context.Context, projectKey, repoSlug string, id int) error {
	if projectKey == "" || repoSlug == "" {
		return fmt.Errorf("project key and repository slug are required")
	}
	req, err := c.http.NewRequest(ctx, "DELETE", fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/webhooks/%d",
		url.PathEscape(projectKey), url.PathEscape(repoSlug), id), nil)
	if err != nil {
		return err
	}
	return c.http.Do(req, nil)
}

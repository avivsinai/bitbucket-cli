package bbdc

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type applicationProperties struct {
	Version string `json:"version"`
}

// ServerVersion returns the Bitbucket Data Center server version.
func (c *Client) ServerVersion(ctx context.Context) (string, error) {
	req, err := c.http.NewRequest(ctx, http.MethodGet, "/rest/api/1.0/application-properties", nil)
	if err != nil {
		return "", err
	}

	var props applicationProperties
	if err := c.http.Do(req, &props); err != nil {
		return "", err
	}

	version := strings.TrimSpace(props.Version)
	if version == "" {
		return "", fmt.Errorf("server version is missing from application properties")
	}

	return version, nil
}

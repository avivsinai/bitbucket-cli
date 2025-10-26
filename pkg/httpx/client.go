package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client wraps HTTP access with Bitbucket-aware defaults.
type Client struct {
	baseURL   *url.URL
	username  string
	password  string
	userAgent string

	httpClient *http.Client
}

// Options configures a Client.
type Options struct {
	BaseURL   string
	Username  string
	Password  string
	UserAgent string
	Timeout   time.Duration
}

// New constructs a Client from options.
func New(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	base, err := url.Parse(opts.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	if base.Scheme == "" {
		return nil, fmt.Errorf("base URL must include scheme (e.g. https)")
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL:  base,
		username: strings.TrimSpace(opts.Username),
		password: opts.Password,
		userAgent: func() string {
			if opts.UserAgent != "" {
				return opts.UserAgent
			}
			return "bkt-cli"
		}(),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// NewRequest builds an HTTP request relative to the base URL. Body values are
// JSON encoded when non-nil.
func (c *Client) NewRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	u := c.baseURL.ResolveReference(&url.URL{Path: path})

	var buf io.ReadWriter
	if body != nil {
		buf = &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		if err := enc.Encode(body); err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), buf)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)

	if c.username != "" || c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	return req, nil
}

// Do executes the HTTP request and decodes the response into v when provided.
func (c *Client) Do(req *http.Request, v any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeError(resp)
	}

	if v == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	switch dst := v.(type) {
	case io.Writer:
		_, err := io.Copy(dst, resp.Body)
		return err
	default:
		dec := json.NewDecoder(resp.Body)
		return dec.Decode(v)
	}
}

func decodeError(resp *http.Response) error {
	type apiErr struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	var payload apiErr
	data, err := io.ReadAll(resp.Body)
	if err == nil && len(data) > 0 {
		_ = json.Unmarshal(data, &payload)
	}

	if len(payload.Errors) > 0 {
		return fmt.Errorf("%s: %s", resp.Status, payload.Errors[0].Message)
	}

	if err == nil && len(data) > 0 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(data)))
	}

	return fmt.Errorf("%s", resp.Status)
}

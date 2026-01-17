package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client wraps HTTP access with Bitbucket-aware defaults.
type Client struct {
	baseURL   *url.URL
	username  string
	password  string
	userAgent string

	httpClient *http.Client

	enableCache bool
	cacheMu     sync.RWMutex
	cache       map[string]*cacheEntry

	rateMu sync.RWMutex
	rate   RateLimit

	retry RetryPolicy

	debug bool
}

// Options configures a Client.
type Options struct {
	BaseURL   string
	Username  string
	Password  string
	UserAgent string
	Timeout   time.Duration

	EnableCache bool
	Retry       RetryPolicy
	Debug       bool
}

// RetryPolicy defines exponential backoff characteristics for retries.
type RetryPolicy struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// RateLimit captures headers advertised by Bitbucket for throttling.
type RateLimit struct {
	Limit     int
	Remaining int
	Reset     time.Time
	Source    string
}

type cacheEntry struct {
	etag     string
	body     []byte
	storedAt time.Time
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

	client := &Client{
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
		enableCache: opts.EnableCache,
		cache:       make(map[string]*cacheEntry),
	}

	if opts.Debug || os.Getenv("BKT_HTTP_DEBUG") != "" {
		client.debug = true
	}

	policy := opts.Retry
	if policy.MaxAttempts == 0 {
		policy.MaxAttempts = 3
	}
	if policy.InitialBackoff == 0 {
		policy.InitialBackoff = 200 * time.Millisecond
	}
	if policy.MaxBackoff == 0 {
		policy.MaxBackoff = 2 * time.Second
	}
	client.retry = policy

	return client, nil
}

// NewRequest builds an HTTP request relative to the base URL. Body values are
// JSON encoded when non-nil.
func (c *Client) NewRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("path is required")
	}

	var rel *url.URL
	var err error

	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		rel, err = url.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("parse request URL: %w", err)
		}
	} else {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		rel, err = url.Parse(path)
		if err != nil {
			return nil, fmt.Errorf("parse request path: %w", err)
		}
	}

	if rel.Path == "" {
		rel.Path = "/"
	}

	u := c.baseURL.ResolveReference(rel)

	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
	}

	var reader io.Reader
	if payload != nil {
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.ContentLength = int64(len(payload))
		data := payload
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(data)), nil
		}
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
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	attempts := 0
	for {
		attemptReq, err := cloneRequest(req)
		if err != nil {
			return err
		}

		if c.enableCache && attemptReq.Method == http.MethodGet {
			if etag := c.cachedETag(attemptReq); etag != "" {
				attemptReq.Header.Set("If-None-Match", etag)
			}
		}

		if c.debug {
			fmt.Fprintf(os.Stderr, "--> %s %s\n", attemptReq.Method, attemptReq.URL.String())
		}

		resp, err := c.httpClient.Do(attemptReq)
		if err != nil {
			if !c.shouldRetry(attempts, 0) {
				if c.debug {
					fmt.Fprintf(os.Stderr, "<-- network error: %v\n", err)
				}
				return err
			}
			attempts++
			continueRetry, waitErr := c.backoff(req.Context(), attempts, resp)
			if waitErr != nil {
				return waitErr
			}
			if !continueRetry {
				if c.debug {
					fmt.Fprintf(os.Stderr, "<-- retry abort after error: %v\n", err)
				}
				return err
			}
			continue
		}

		c.updateRateLimit(resp)
		c.applyAdaptiveThrottle()

		if c.debug {
			fmt.Fprintf(os.Stderr, "<-- %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode))
		}

		if resp.StatusCode == http.StatusNotModified && c.enableCache && attemptReq.Method == http.MethodGet {
			_ = resp.Body.Close()
			if err := c.applyCachedResponse(attemptReq, v); err != nil {
				return err
			}
			return nil
		}

		if shouldRetryStatus(resp.StatusCode) {
			// Read body for retry logic; errors are intentionally ignored as we'll retry anyway
			bodyBytes, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if !c.shouldRetry(attempts, resp.StatusCode) {
				if len(bodyBytes) > 0 {
					resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				}
				return decodeError(resp)
			}
			attempts++
			continueRetry, waitErr := c.backoff(req.Context(), attempts, resp)
			if waitErr != nil {
				return waitErr
			}
			if !continueRetry {
				if len(bodyBytes) > 0 {
					resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
				}
				return decodeError(resp)
			}
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			defer func() {
				_ = resp.Body.Close()
			}()
			return decodeError(resp)
		}

		if v == nil {
			// Drain and discard response body when caller doesn't need it; errors are intentionally ignored
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if c.enableCache && attemptReq.Method == http.MethodGet {
				c.storeCache(attemptReq, nil, resp.Header.Get("ETag"))
			}
			return nil
		}

		if writer, ok := v.(io.Writer); ok {
			_, err := io.Copy(writer, resp.Body)
			_ = resp.Body.Close()
			return err
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return err
		}

		if c.enableCache && attemptReq.Method == http.MethodGet && resp.Header.Get("ETag") != "" {
			c.storeCache(attemptReq, bodyBytes, resp.Header.Get("ETag"))
		}

		if len(bodyBytes) == 0 {
			return nil
		}

		if err := json.Unmarshal(bodyBytes, v); err != nil {
			return err
		}
		return nil
	}
}

func decodeError(resp *http.Response) error {
	type apiErrEntry struct {
		Message       string `json:"message"`
		ExceptionName string `json:"exceptionName"`
	}
	type apiErr struct {
		Errors []apiErrEntry `json:"errors"`
	}

	var payload apiErr
	data, err := io.ReadAll(resp.Body)
	if err == nil && len(data) > 0 {
		// Attempt to parse structured error; intentionally ignore unmarshal errors and fall back to raw text
		_ = json.Unmarshal(data, &payload)
	}

	if len(payload.Errors) > 0 {
		// Prioritize user-actionable errors like CAPTCHA over generic ones
		bestErr := payload.Errors[0]
		for _, e := range payload.Errors {
			if isCaptchaException(e.ExceptionName) {
				bestErr = e
				break
			}
		}

		msg := bestErr.Message
		// Add hint for CAPTCHA-locked accounts
		if isCaptchaException(bestErr.ExceptionName) && !strings.Contains(strings.ToLower(msg), "captcha") {
			msg = "CAPTCHA verification required: " + msg
		}
		return fmt.Errorf("%s: %s", resp.Status, msg)
	}

	if err == nil && len(data) > 0 {
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(data)))
	}

	return fmt.Errorf("%s", resp.Status)
}

// isCaptchaException checks if the exception name indicates a CAPTCHA-locked account.
func isCaptchaException(exceptionName string) bool {
	return strings.Contains(strings.ToLower(exceptionName), "captcharequired")
}

func cloneRequest(req *http.Request) (*http.Request, error) {
	newReq := req.Clone(req.Context())
	newReq.Header = req.Header.Clone()
	if req.Body != nil {
		if req.GetBody == nil {
			return nil, fmt.Errorf("request body cannot be replayed")
		}
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		newReq.Body = body
	}
	return newReq, nil
}

func shouldRetryStatus(code int) bool {
	if code == http.StatusTooManyRequests {
		return true
	}
	return code >= 500 && code <= 599
}

func (c *Client) shouldRetry(attempts int, status int) bool {
	return attempts+1 < c.retry.MaxAttempts
}

func (c *Client) backoff(ctx context.Context, attempts int, resp *http.Response) (bool, error) {
	if attempts >= c.retry.MaxAttempts {
		return false, nil
	}

	delay := c.retry.InitialBackoff
	if attempts > 1 {
		delay = delay * time.Duration(1<<(attempts-1))
	}
	if delay > c.retry.MaxBackoff {
		delay = c.retry.MaxBackoff
	}

	if resp != nil {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if secs, err := strconv.Atoi(retryAfter); err == nil {
				delay = time.Duration(secs) * time.Second
			}
		}
	}

	if delay <= 0 {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
			return true, nil
		}
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-timer.C:
		return true, nil
	}
}

func (c *Client) cacheKey(req *http.Request) string {
	return req.Method + " " + req.URL.String()
}

func (c *Client) cachedETag(req *http.Request) string {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	if entry, ok := c.cache[c.cacheKey(req)]; ok {
		return entry.etag
	}
	return ""
}

func (c *Client) storeCache(req *http.Request, body []byte, etag string) {
	if etag == "" || len(body) == 0 {
		return
	}
	c.cacheMu.Lock()
	c.cache[c.cacheKey(req)] = &cacheEntry{etag: etag, body: append([]byte(nil), body...), storedAt: time.Now()}
	c.cacheMu.Unlock()
}

func (c *Client) applyCachedResponse(req *http.Request, v any) error {
	if v == nil {
		return nil
	}
	c.cacheMu.RLock()
	entry, ok := c.cache[c.cacheKey(req)]
	c.cacheMu.RUnlock()
	if !ok {
		return fmt.Errorf("cached response missing for %s", req.URL)
	}

	if writer, ok := v.(io.Writer); ok {
		_, err := writer.Write(entry.body)
		return err
	}
	if len(entry.body) == 0 {
		return nil
	}
	return json.Unmarshal(entry.body, v)
}

// RateLimitState returns the last observed rate limit headers.
func (c *Client) RateLimitState() RateLimit {
	c.rateMu.RLock()
	defer c.rateMu.RUnlock()
	return c.rate
}

func (c *Client) updateRateLimit(resp *http.Response) {
	headers := resp.Header

	readHeader := func(key string) int {
		val := headers.Get(key)
		if val == "" {
			return 0
		}
		n, err := strconv.Atoi(val)
		if err != nil {
			return 0
		}
		return n
	}

	limit := readHeader("X-RateLimit-Limit")
	remaining := readHeader("X-RateLimit-Remaining")
	resetHeader := headers.Get("X-RateLimit-Reset")

	var reset time.Time
	if resetHeader != "" {
		if epoch, err := strconv.ParseInt(resetHeader, 10, 64); err == nil {
			if epoch > 0 {
				reset = time.Unix(epoch, 0)
			}
		} else {
			if parsed, err := time.Parse(time.RFC1123, resetHeader); err == nil {
				reset = parsed
			}
		}
	}

	source := ""
	if limit != 0 || remaining != 0 {
		source = "bitbucket"
	}

	if limit == 0 && remaining == 0 {
		// Some endpoints expose Atlassian-RateLimit prefixed headers.
		limit = readHeader("X-Attempt-RateLimit-Limit")
		remaining = readHeader("X-Attempt-RateLimit-Remaining")
		if limit == 0 && remaining == 0 {
			limit = readHeader("X-RateLimit-Capacity")
			remaining = readHeader("X-RateLimit-Available")
		}
		if limit != 0 || remaining != 0 {
			source = "atlassian"
		}
	}

	if limit == 0 && remaining == 0 {
		return
	}

	c.rateMu.Lock()
	c.rate = RateLimit{Limit: limit, Remaining: remaining, Reset: reset, Source: source}
	c.rateMu.Unlock()
}

func (c *Client) applyAdaptiveThrottle() {
	c.rateMu.RLock()
	rl := c.rate
	c.rateMu.RUnlock()

	if rl.Remaining > 1 || rl.Reset.IsZero() {
		return
	}

	sleep := time.Until(rl.Reset)
	if sleep <= 0 {
		return
	}
	if sleep > 5*time.Second {
		sleep = 5 * time.Second
	}
	time.Sleep(sleep)
}

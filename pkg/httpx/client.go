package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
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
	userAgent string

	// credMu guards username, password, and authMethod: they are read on
	// every request and rewritten when a 401 triggers a token refresh, and
	// the client must stay safe for concurrent use.
	credMu     sync.RWMutex
	username   string
	password   string
	authMethod string

	httpClient *http.Client

	enableCache bool
	cacheMu     sync.RWMutex
	cache       map[string]*cacheEntry

	rateMu sync.RWMutex
	rate   RateLimit

	retry RetryPolicy

	// tokenRefresher is called on 401 Unauthorized responses. It should return
	// a new access token. The client retries the original request once with the
	// updated credentials. Concurrent 401s coalesce into a single refresher
	// call via refreshMu/refreshInFlight.
	tokenRefresher  func(ctx context.Context) (string, error)
	refreshMu       sync.Mutex
	refreshInFlight *refreshCall
	requestHook     func(*http.Request)

	debug bool
}

// refreshCall is a single in-flight token refresh that concurrent 401 paths
// wait on instead of issuing their own refresh.
type refreshCall struct {
	done chan struct{}
	err  error
}

// Options configures a Client.
type Options struct {
	BaseURL    string
	Username   string
	Password   string
	AuthMethod string // "basic" (default) or "bearer"
	UserAgent  string
	Timeout    time.Duration

	EnableCache bool
	Retry       RetryPolicy
	Debug       bool

	// TokenRefresher is an optional callback invoked on 401 Unauthorized
	// responses. It should return a new access token. The client updates its
	// credentials and retries the request once. If the retry also returns 401
	// the error is returned to the caller without a second refresh attempt.
	TokenRefresher func(ctx context.Context) (string, error)

	// RequestHook can apply host-specific defaults after standard headers and
	// auth are set.
	RequestHook func(*http.Request)
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

	authMethod := strings.ToLower(strings.TrimSpace(opts.AuthMethod))
	if authMethod == "" {
		authMethod = "basic"
	}
	if authMethod != "basic" && authMethod != "bearer" {
		return nil, fmt.Errorf("unsupported auth method %q; use \"basic\" or \"bearer\"", authMethod)
	}

	client := &Client{
		baseURL:    base,
		username:   strings.TrimSpace(opts.Username),
		password:   strings.TrimSpace(opts.Password),
		authMethod: authMethod,
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
	client.tokenRefresher = opts.TokenRefresher
	client.requestHook = opts.RequestHook

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

	// Join paths properly: for relative paths starting with "/", we want to
	// append to the base URL path, not replace it. Go's ResolveReference
	// treats "/foo" as an absolute path that replaces the base path.
	u := *c.baseURL
	basePath := c.baseURL.Path
	if strings.HasPrefix(path, "/") && basePath != "" {
		// Guard: if path already starts with base path, don't double it.
		// This handles cases where callers pass "/2.0/repositories" when base is
		// already "https://api.bitbucket.org/2.0" - we don't want "/2.0/2.0/repositories".
		if strings.HasPrefix(rel.Path, basePath) {
			u.Path = rel.Path
		} else {
			u.Path = strings.TrimSuffix(basePath, "/") + rel.Path
		}
	} else {
		resolved := c.baseURL.ResolveReference(rel)
		u = *resolved
	}
	u.RawQuery = rel.RawQuery

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

	c.applyAuth(req)
	c.applyRequestHook(req)

	return req, nil
}

// applyAuth sets the Authorization header based on the configured auth method.
func (c *Client) applyAuth(req *http.Request) {
	c.credMu.RLock()
	username, password, authMethod := c.username, c.password, c.authMethod
	c.credMu.RUnlock()

	switch authMethod {
	case "bearer":
		if password != "" {
			req.Header.Set("Authorization", "Bearer "+password)
		}
	default:
		if username != "" || password != "" {
			req.SetBasicAuth(username, password)
		}
	}
}

// authorizationHeader returns the Authorization header value the current
// credentials would produce, so a 401 path can detect that another goroutine
// already rotated them.
func (c *Client) authorizationHeader() string {
	probe, err := http.NewRequest(http.MethodGet, "http://probe.invalid/", nil)
	if err != nil {
		return ""
	}
	c.applyAuth(probe)
	return probe.Header.Get("Authorization")
}

// refreshCredentials handles a 401: if the credentials already changed since
// the failed request was built, it simply retries with them; otherwise it
// coalesces all concurrent callers into one tokenRefresher invocation and
// installs the resulting bearer token.
//
// The stale-credential check and leader election happen in one refreshMu
// critical section, so a goroutine that raced past a completed refresh cannot
// become a redundant second leader (rotating refresh tokens make redundant
// refreshes harmful, not just wasteful). Lock order is refreshMu -> credMu
// (read); credMu is never held while acquiring refreshMu.
func (c *Client) refreshCredentials(ctx context.Context, usedAuth string) error {
	for {
		c.refreshMu.Lock()
		if h := c.authorizationHeader(); h != "" && h != usedAuth {
			// Credentials rotated while this request was in flight; retry
			// with the current ones without spending another refresh.
			c.refreshMu.Unlock()
			return nil
		}
		if call := c.refreshInFlight; call != nil {
			c.refreshMu.Unlock()
			select {
			case <-call.done:
				// Inherit real refresh failures, but not another request's
				// context death: if the leader was cancelled while this
				// caller is still live, contend for leadership again.
				if isContextError(call.err) && ctx.Err() == nil {
					continue
				}
				return call.err
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		call := &refreshCall{done: make(chan struct{})}
		c.refreshInFlight = call
		c.refreshMu.Unlock()

		token, err := c.tokenRefresher(ctx)
		if err == nil {
			c.credMu.Lock()
			c.password = token
			// OAuth access tokens are always sent as Bearer regardless of
			// the auth method the client was originally constructed with.
			c.authMethod = "bearer"
			c.credMu.Unlock()
		}
		call.err = err

		c.refreshMu.Lock()
		c.refreshInFlight = nil
		c.refreshMu.Unlock()
		close(call.done)

		return err
	}
}

// isContextError reports whether err is a context cancellation or deadline.
func isContextError(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func (c *Client) applyRequestHook(req *http.Request) {
	if c.requestHook != nil {
		c.requestHook(req)
	}
}

// Do executes the HTTP request and decodes the response into v when provided.
func (c *Client) Do(req *http.Request, v any) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	attempts := 0
	tokenRefreshed := false
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
			if !c.shouldRetry(attempts, attemptReq.Method, 0) {
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
		if err := c.applyAdaptiveThrottle(req.Context()); err != nil {
			_ = resp.Body.Close()
			return err
		}

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
			if !c.shouldRetry(attempts, attemptReq.Method, resp.StatusCode) {
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

		if resp.StatusCode == http.StatusUnauthorized && c.tokenRefresher != nil && !tokenRefreshed {
			_ = resp.Body.Close()
			if refreshErr := c.refreshCredentials(req.Context(), attemptReq.Header.Get("Authorization")); refreshErr != nil {
				return fmt.Errorf("refresh token: %w", refreshErr)
			}
			c.applyAuth(req) // update auth header on original request for next clone
			tokenRefreshed = true
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
		Message       string   `json:"message"`
		ExceptionName string   `json:"exceptionName"`
		Details       []string `json:"details"`
	}
	type apiErr struct {
		Errors []apiErrEntry `json:"errors"`
	}
	type cloudAPIError struct {
		Type  string `json:"type"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	var payload apiErr
	var cloudPayload cloudAPIError
	data, err := io.ReadAll(resp.Body)
	if err == nil && len(data) > 0 {
		// Attempt to parse structured error; intentionally ignore unmarshal errors and fall back to raw text
		_ = json.Unmarshal(data, &payload)
		_ = json.Unmarshal(data, &cloudPayload)
	}

	if len(payload.Errors) > 0 {
		// Prioritize user-actionable errors like CAPTCHA over generic ones.
		// Track the index so the second loop below can skip the already-printed
		// best entry positionally; apiErrEntry is non-comparable now (it has a
		// slice field), so we cannot skip it by value.
		bestIdx := 0
		for i, e := range payload.Errors {
			if isCaptchaException(e.ExceptionName) {
				bestIdx = i
				break
			}
		}
		bestErr := payload.Errors[bestIdx]

		msg := bestErr.Message
		// Add hint for CAPTCHA-locked accounts
		if isCaptchaException(bestErr.ExceptionName) && !strings.Contains(strings.ToLower(msg), "captcha") {
			msg = "CAPTCHA verification required: " + msg
		}
		msg = withBitbucketCloudAuthHint(msg)

		// Preserve the historical single-line "<status>: <message>" prefix so
		// scripts that grep the first line keep working, then append any
		// actionable details and additional errors on indented continuation
		// lines. Bitbucket's details[] often carry the real remediation (e.g.
		// "create the pull request as draft instead"), which was previously
		// dropped.
		var b strings.Builder
		b.WriteString(msg)
		appendErrorDetails(&b, bestErr.Details)
		for j, e := range payload.Errors {
			if j == bestIdx {
				continue
			}
			if m := strings.TrimSpace(e.Message); m != "" {
				b.WriteString(errorContinuationIndent)
				b.WriteString(m)
			}
			appendErrorDetails(&b, e.Details)
		}
		return newHTTPError(resp, b.String())
	}

	if msg := strings.TrimSpace(cloudPayload.Error.Message); msg != "" {
		return newHTTPError(resp, withBitbucketCloudAuthHint(msg))
	}

	if err == nil && len(data) > 0 {
		return newHTTPError(resp, strings.TrimSpace(string(data)))
	}

	return newHTTPError(resp)
}

// Indentation for continuation lines rendered under the primary error message:
// a secondary error message sits at 2 spaces and its detail lines at 4, forming
// a small tree. Kept as constants so the spacing is defined in one place.
const (
	errorContinuationIndent = "\n  "
	errorDetailIndent       = "\n    "
)

// appendErrorDetails writes each non-empty line from a Bitbucket error entry's
// details[] to b, indented under the message. Individual detail strings may
// themselves contain embedded newlines (Bitbucket frequently formats multi-step
// guidance this way), so each embedded line is indented consistently and blank
// lines within an entry are preserved. Entirely empty/whitespace-only entries
// are skipped (they would otherwise emit a stray blank line), and CR is trimmed
// so CRLF-delimited details do not leave a trailing \r on the stable stderr output.
func appendErrorDetails(b *strings.Builder, details []string) {
	for _, d := range details {
		if strings.TrimSpace(d) == "" {
			continue
		}
		for _, line := range strings.Split(d, "\n") {
			line = strings.TrimRight(line, " \t\r")
			if line == "" {
				b.WriteString("\n")
				continue
			}
			b.WriteString(errorDetailIndent)
			b.WriteString(line)
		}
	}
}

// HTTPError preserves the response status code for callers that need stable
// classification while keeping the historical error text unchanged.
type HTTPError struct {
	StatusCode int
	text       string
}

func (e *HTTPError) Error() string {
	return e.text
}

func newHTTPError(resp *http.Response, message ...string) *HTTPError {
	text := resp.Status
	if len(message) > 0 {
		text += ": " + message[0]
	}
	return &HTTPError{StatusCode: resp.StatusCode, text: text}
}

// isCaptchaException checks if the exception name indicates a CAPTCHA-locked account.
func isCaptchaException(exceptionName string) bool {
	return strings.Contains(strings.ToLower(exceptionName), "captcharequired")
}

func withBitbucketCloudAuthHint(msg string) string {
	lower := strings.ToLower(msg)
	if !strings.Contains(lower, "token is invalid, expired, or not supported for this endpoint") {
		return msg
	}
	if strings.Contains(lower, "read:user:bitbucket") || strings.Contains(lower, "atlassian account email") {
		return msg
	}
	return msg + " Hint: for Bitbucket Cloud API tokens, set BKT_USERNAME to your Atlassian account email and include the read:user:bitbucket scope."
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

// shouldRetry reports whether another attempt should be made for the given
// method and status. A 429 (Too Many Requests) is safe to retry for any method
// because the request was rejected, not processed. 5xx responses and transport
// errors (status == 0) may have already applied a server-side side effect, so
// they are only retried for idempotent methods — this prevents a transient 500
// after a successful write from duplicating a non-idempotent POST (e.g. creating
// a pull request comment twice).
func (c *Client) shouldRetry(attempts int, method string, status int) bool {
	if attempts+1 >= c.retry.MaxAttempts {
		return false
	}
	if status == http.StatusTooManyRequests {
		return true
	}
	return isIdempotentMethod(method)
}

// isIdempotentMethod reports whether retrying the method is safe per HTTP
// semantics (RFC 9110 §9.2.2). POST and PATCH are excluded because they are not
// idempotent.
func isIdempotentMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions,
		http.MethodPut, http.MethodDelete, http.MethodTrace:
		return true
	default:
		return false
	}
}

func (c *Client) retryDelay(attempts int, resp *http.Response) time.Duration {
	delay := c.retry.InitialBackoff
	if attempts > 1 {
		delay *= time.Duration(1 << (attempts - 1))
	}
	if delay > c.retry.MaxBackoff {
		delay = c.retry.MaxBackoff
	}

	if resp != nil {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if secs, err := strconv.Atoi(retryAfter); err == nil {
				retryAfterDelay := time.Duration(secs) * time.Second
				if retryAfterDelay > 60*time.Second {
					retryAfterDelay = 60 * time.Second
				}
				if retryAfterDelay > 0 {
					delay = retryAfterDelay
				}
			}
		}
	}

	return delay
}

func (c *Client) backoff(ctx context.Context, attempts int, resp *http.Response) (bool, error) {
	if attempts >= c.retry.MaxAttempts {
		return false, nil
	}

	delay := c.retryDelay(attempts, resp)

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

func (c *Client) applyAdaptiveThrottle(ctx context.Context) error {
	c.rateMu.RLock()
	rl := c.rate
	c.rateMu.RUnlock()

	if rl.Remaining > 1 || rl.Reset.IsZero() {
		return nil
	}

	sleep := time.Until(rl.Reset)
	if sleep <= 0 {
		return nil
	}
	if sleep > 5*time.Second {
		sleep = 5 * time.Second
	}
	timer := time.NewTimer(sleep)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// MultipartFile represents a file for multipart/form-data upload.
type MultipartFile struct {
	FieldName string    // Form field name (e.g., "files")
	FileName  string    // Original filename
	Reader    io.Reader // File content
}

// NewMultipartRequest builds a multipart/form-data request for file uploads.
// The request body is buffered in memory to support retries.
func (c *Client) NewMultipartRequest(ctx context.Context, method, path string, files []MultipartFile) (*http.Request, error) {
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

	u := *c.baseURL
	basePath := c.baseURL.Path
	if strings.HasPrefix(path, "/") && basePath != "" {
		if strings.HasPrefix(rel.Path, basePath) {
			u.Path = rel.Path
		} else {
			u.Path = strings.TrimSuffix(basePath, "/") + rel.Path
		}
	} else {
		resolved := c.baseURL.ResolveReference(rel)
		u = *resolved
	}
	u.RawQuery = rel.RawQuery

	// Buffer the multipart content to support retries.
	// Note: This buffers the entire payload in memory, which is acceptable
	// for typical attachment sizes but may need review for very large files.
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	if len(files) == 0 {
		return nil, fmt.Errorf("at least one file is required")
	}

	for _, f := range files {
		if f.Reader == nil {
			return nil, fmt.Errorf("reader is nil for file %q", f.FileName)
		}
		part, err := mw.CreateFormFile(f.FieldName, f.FileName)
		if err != nil {
			return nil, fmt.Errorf("create form file: %w", err)
		}
		if _, err := io.Copy(part, f.Reader); err != nil {
			return nil, fmt.Errorf("copy file content: %w", err)
		}
	}

	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	payload := buf.Bytes()
	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	req.ContentLength = int64(len(payload))

	// Set GetBody for retry support
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(payload)), nil
	}

	c.applyAuth(req)
	c.applyRequestHook(req)

	return req, nil
}

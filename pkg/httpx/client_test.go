package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type payload struct {
	Message string `json:"message"`
}

func TestClientCachingWithETag(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", "etag-123")
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", "42")
		if r.Header.Get("If-None-Match") == "etag-123" {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		_ = json.NewEncoder(w).Encode(payload{Message: "hello"})
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL, EnableCache: true})
	if err != nil {
		t.Fatalf("New client: %v", err)
	}

	req1, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	var out payload
	if err := client.Do(req1, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if out.Message != "hello" {
		t.Fatalf("expected hello, got %q", out.Message)
	}

	req2, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	out = payload{}
	if err := client.Do(req2, &out); err != nil {
		t.Fatalf("Do cache: %v", err)
	}
	if out.Message != "hello" {
		t.Fatalf("expected cached hello, got %q", out.Message)
	}

	if hits != 2 {
		t.Fatalf("expected 2 hits (initial + 304), got %d", hits)
	}

	rate := client.RateLimitState()
	if rate.Remaining != 42 {
		t.Fatalf("expected remaining 42, got %d", rate.Remaining)
	}
}

func TestClientRetriesOnServerError(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		if count == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload{Message: "ok"})
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{
		BaseURL:     server.URL,
		EnableCache: false,
		Retry: RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     20 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	var out payload
	if err := client.Do(req, &out); err != nil {
		t.Fatalf("Do with retry: %v", err)
	}
	if out.Message != "ok" {
		t.Fatalf("expected ok, got %q", out.Message)
	}

	if hits != 2 {
		t.Fatalf("expected 2 attempts, got %d", hits)
	}
}

func TestClientNewRequestPreservesQuery(t *testing.T) {
	client, err := New(Options{BaseURL: "https://example.com/api"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/rest/projects?limit=25&start=0", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	// Paths starting with "/" should be joined to the base URL path, not replace it
	if got := req.URL.String(); got != "https://example.com/api/rest/projects?limit=25&start=0" {
		t.Fatalf("unexpected URL: %s", got)
	}
	if req.URL.RawQuery != "limit=25&start=0" {
		t.Fatalf("expected raw query preserved, got %q", req.URL.RawQuery)
	}
}

func TestClientNewRequestHandlesRelativeWithoutSlash(t *testing.T) {
	client, err := New(Options{BaseURL: "https://example.com/api"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "rest/repos", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	// Paths without leading "/" get one added, then joined to base path
	if got := req.URL.String(); got != "https://example.com/api/rest/repos" {
		t.Fatalf("unexpected URL: %s", got)
	}
}

func TestClientBackoffRespectsContextCancellation(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{
		BaseURL: server.URL,
		Retry: RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 500 * time.Millisecond,
			MaxBackoff:     time.Second,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	req, err := client.NewRequest(ctx, http.MethodGet, "/fail", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	var once sync.Once
	time.AfterFunc(50*time.Millisecond, func() {
		once.Do(cancel)
	})

	start := time.Now()
	err = client.Do(req, nil)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected error from cancelled context")
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context cancellation error, got %v", err)
	}
	if elapsed >= 400*time.Millisecond {
		t.Fatalf("expected cancellation to interrupt backoff, took %v", elapsed)
	}
	if hits != 1 {
		t.Fatalf("expected single request, got %d", hits)
	}
}

func TestAdaptiveThrottleRespectsContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "100")
		w.Header().Set("X-RateLimit-Remaining", "1")
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(2*time.Second).Unix(), 10))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)
	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, err := client.NewRequest(ctx, http.MethodGet, "/throttled", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	time.AfterFunc(25*time.Millisecond, cancel)
	start := time.Now()
	err = client.Do(req, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Do error = %T %v, want context.Canceled", err, err)
	}
	if elapsed := time.Since(start); elapsed >= 250*time.Millisecond {
		t.Fatalf("cancellation did not interrupt adaptive throttle: %v", elapsed)
	}
}

func TestClientNewRequestNoDoubledBasePath(t *testing.T) {
	client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Pass path that already includes /2.0 - should NOT become /2.0/2.0/repositories
	req, err := client.NewRequest(context.Background(), http.MethodGet, "/2.0/repositories", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	expected := "https://api.bitbucket.org/2.0/repositories"
	if got := req.URL.String(); got != expected {
		t.Fatalf("doubled base path: got %s, want %s", got, expected)
	}
}

func TestNewMultipartRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contentType := r.Header.Get("Content-Type")
		if contentType == "" {
			t.Error("missing Content-Type header")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Error("missing or incorrect Accept header")
		}
		if r.Header.Get("User-Agent") != "bkt-cli" {
			t.Error("missing or incorrect User-Agent header")
		}

		// Verify multipart content
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}

		file, header, err := r.FormFile("files")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer func() { _ = file.Close() }()

		if header.Filename != "test.txt" {
			t.Errorf("expected filename test.txt, got %s", header.Filename)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	files := []MultipartFile{
		{
			FieldName: "files",
			FileName:  "test.txt",
			Reader:    nil,
		},
	}
	// We need to provide actual content for the test
	files[0].Reader = http.NoBody

	req, err := client.NewMultipartRequest(context.Background(), http.MethodPost, "/upload", files)
	if err != nil {
		t.Fatalf("NewMultipartRequest: %v", err)
	}

	if err := client.Do(req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
}

func TestNewMultipartRequestContentType(t *testing.T) {
	client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	files := []MultipartFile{
		{
			FieldName: "files",
			FileName:  "test.txt",
			Reader:    http.NoBody,
		},
	}

	req, err := client.NewMultipartRequest(context.Background(), http.MethodPost, "/upload", files)
	if err != nil {
		t.Fatalf("NewMultipartRequest: %v", err)
	}

	contentType := req.Header.Get("Content-Type")
	if contentType == "" {
		t.Fatal("Content-Type header not set")
	}
	if len(contentType) < 30 {
		t.Fatalf("Content-Type should include boundary, got: %s", contentType)
	}
}

func TestNewMultipartRequestNilReader(t *testing.T) {
	client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	files := []MultipartFile{
		{
			FieldName: "files",
			FileName:  "test.txt",
			Reader:    nil,
		},
	}

	_, err = client.NewMultipartRequest(context.Background(), http.MethodPost, "/upload", files)
	if err == nil {
		t.Fatal("expected error for nil reader")
	}
	if err.Error() != `reader is nil for file "test.txt"` {
		t.Errorf("expected nil reader error, got %q", err.Error())
	}
}

func TestNewMultipartRequestEmptyFiles(t *testing.T) {
	client, err := New(Options{BaseURL: "https://api.bitbucket.org/2.0"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.NewMultipartRequest(context.Background(), http.MethodPost, "/upload", []MultipartFile{})
	if err == nil {
		t.Fatal("expected error for empty files slice")
	}
	if err.Error() != "at least one file is required" {
		t.Errorf("expected empty files error, got %q", err.Error())
	}
}

func TestDecodeErrorPrioritizesCaptchaException(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		status  int
		wantMsg string
	}{
		{
			name:    "captcha exception with clear message",
			status:  http.StatusForbidden,
			body:    `{"errors":[{"message":"CAPTCHA required. Your Bitbucket account has been locked.","exceptionName":"com.atlassian.bitbucket.auth.CaptchaRequiredAuthenticationException"}]}`,
			wantMsg: "403 Forbidden: CAPTCHA required. Your Bitbucket account has been locked.",
		},
		{
			name:    "captcha exception prioritized over generic error",
			status:  http.StatusForbidden,
			body:    `{"errors":[{"message":"XSRF check failed","exceptionName":""},{"message":"Account locked","exceptionName":"com.atlassian.bitbucket.auth.CaptchaRequiredAuthenticationException"}]}`,
			wantMsg: "403 Forbidden: CAPTCHA verification required: Account locked\n  XSRF check failed",
		},
		{
			name:    "normal error without captcha",
			status:  http.StatusNotFound,
			body:    `{"errors":[{"message":"Repository not found"}]}`,
			wantMsg: "404 Not Found: Repository not found",
		},
		{
			name:    "empty body",
			status:  http.StatusForbidden,
			body:    "",
			wantMsg: "403 Forbidden",
		},
		{
			name:    "whitespace body preserves historical separator",
			status:  http.StatusForbidden,
			body:    " \n",
			wantMsg: "403 Forbidden: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			t.Cleanup(server.Close)

			client, err := New(Options{
				BaseURL: server.URL,
				Retry:   RetryPolicy{MaxAttempts: 1},
			})
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			req, err := client.NewRequest(context.Background(), http.MethodPost, "/test", nil)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}

			err = client.Do(req, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if err.Error() != tt.wantMsg {
				t.Errorf("got %q, want %q", err.Error(), tt.wantMsg)
			}
		})
	}
}

func TestDecodeErrorStructuredMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{
				{"message": "Invalid project key"},
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	err = client.Do(req, nil)
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
	if !strings.Contains(err.Error(), "Invalid project key") {
		t.Fatalf("expected structured error message, got %v", err)
	}
}

func TestDecodeErrorRendersDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{
				{
					"context":       nil,
					"message":       "Pull request creation was canceled.",
					"exceptionName": "com.atlassian.bitbucket.pull.PullRequestOpenCanceledException",
					"details": []string{
						"Pull requests must open in draft state - non-draft creation is blocked.\n\n1. Create the pull request as draft instead",
					},
				},
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodPost, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	err = client.Do(req, nil)
	if err == nil {
		t.Fatal("expected error for 400 response")
	}

	got := err.Error()
	// First line preserves the historical "<status>: <message>" format so
	// scripts that grep the first line keep working.
	firstLine := strings.SplitN(got, "\n", 2)[0]
	if !strings.HasSuffix(firstLine, "Pull request creation was canceled.") {
		t.Fatalf("first line = %q, want it to end with the primary message", firstLine)
	}
	// details[] must now be surfaced (previously dropped).
	if !strings.Contains(got, "Pull requests must open in draft state") {
		t.Fatalf("expected details to be rendered, got %q", got)
	}
	if !strings.Contains(got, "Create the pull request as draft instead") {
		t.Fatalf("expected multi-line detail to be rendered, got %q", got)
	}
}

func TestDecodeErrorRendersAllErrorsAndDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{
				{"message": "First problem", "details": []string{"fix the first thing"}},
				{"message": "Second problem", "details": []string{"fix the second thing"}},
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	err = client.Do(req, nil)
	if err == nil {
		t.Fatal("expected error for 400 response")
	}

	got := err.Error()
	// Assert structure, not just presence: the prioritized (first) error is the
	// primary line; the second is a secondary continuation. Check ordering plus
	// the 2-space indent for secondary messages and 4-space indent for details.
	firstLine := strings.SplitN(got, "\n", 2)[0]
	if !strings.HasSuffix(firstLine, "First problem") {
		t.Fatalf("first line = %q, want it to end with the primary message", firstLine)
	}
	if !strings.Contains(got, "\n    fix the first thing") {
		t.Fatalf("expected primary detail indented 4 spaces, got %q", got)
	}
	if !strings.Contains(got, "\n  Second problem") {
		t.Fatalf("expected secondary message indented 2 spaces, got %q", got)
	}
	if !strings.Contains(got, "\n    fix the second thing") {
		t.Fatalf("expected secondary detail indented 4 spaces, got %q", got)
	}
	prev := -1
	for _, s := range []string{"First problem", "fix the first thing", "Second problem", "fix the second thing"} {
		idx := strings.Index(got, s)
		if idx <= prev {
			t.Fatalf("expected %q to appear in order after the previous item; got %q", s, got)
		}
		prev = idx
	}
}

func TestDecodeErrorTrimsCRLFAndSkipsEmptyDetails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]any{
				{
					"message": "Blocked",
					// A CRLF-delimited multi-line detail, plus a whitespace-only entry.
					"details": []string{"line one\r\nline two", "   "},
				},
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	req, err := client.NewRequest(context.Background(), http.MethodPost, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	got := client.Do(req, nil)
	if got == nil {
		t.Fatal("expected error for 400 response")
	}
	out := got.Error()
	if strings.Contains(out, "\r") {
		t.Fatalf("expected CR to be trimmed from CRLF details, got %q", out)
	}
	if !strings.Contains(out, "\n    line one") || !strings.Contains(out, "\n    line two") {
		t.Fatalf("expected both CRLF detail lines rendered indented, got %q", out)
	}
	// The whitespace-only detail entry must be skipped: no stray blank line.
	if strings.Contains(out, "\n    \n") || strings.HasSuffix(out, "\n") {
		t.Fatalf("expected no stray blank line from empty detail entry, got %q", out)
	}
}

func TestDecodeErrorTrimsBoundaryBlankLinesFromDetails(t *testing.T) {
	tests := []struct {
		name   string
		detail string
		want   string
	}{
		{
			name:   "trailing LF",
			detail: "step one\n",
			want:   "400 Bad Request: Blocked\n    step one",
		},
		{
			name:   "trailing CRLF",
			detail: "step one\r\n",
			want:   "400 Bad Request: Blocked\n    step one",
		},
		{
			name:   "boundary blank lines with internal blank line",
			detail: "\r\n \t\r\nstep one\r\n\r\nstep two\r\n \t\r\n",
			want:   "400 Bad Request: Blocked\n    step one\n\n    step two",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"errors": []map[string]any{
						{"message": "Blocked", "details": []string{tt.detail}},
					},
				})
			}))
			t.Cleanup(server.Close)

			client, err := New(Options{BaseURL: server.URL, Retry: RetryPolicy{MaxAttempts: 1}})
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			req, err := client.NewRequest(context.Background(), http.MethodPost, "/api", nil)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}

			err = client.Do(req, nil)
			if err == nil {
				t.Fatal("expected error for 400 response")
			}
			if got := err.Error(); got != tt.want {
				t.Fatalf("error = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDecodeErrorBitbucketCloudTokenHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "error",
			"error": map[string]any{
				"message": "Token is invalid, expired, or not supported for this endpoint.",
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	err = client.Do(req, nil)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if !strings.Contains(err.Error(), "read:user:bitbucket") {
		t.Fatalf("expected Bitbucket Cloud auth hint, got %v", err)
	}
	if !strings.Contains(err.Error(), "Atlassian account email") {
		t.Fatalf("expected username hint, got %v", err)
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("error = %T %v, want typed HTTPError status 401", err, err)
	}
}

func TestDecodeErrorBitbucketCloudAuthMechanismMessageHasNoTokenHint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "error",
			"error": map[string]any{
				"message": "This API is not accessible by this authentication mechanism",
			},
		})
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	err = client.Do(req, nil)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if strings.Contains(err.Error(), "read:user:bitbucket") {
		t.Fatalf("expected auth mechanism error without token hint, got %v", err)
	}
	if strings.Contains(err.Error(), "Atlassian account email") {
		t.Fatalf("expected auth mechanism error without username hint, got %v", err)
	}
}

func TestDecodeErrorPlainText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Access denied"))
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	err = client.Do(req, nil)
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
	if !strings.Contains(err.Error(), "Access denied") {
		t.Fatalf("expected plain text error, got %v", err)
	}
}

func TestDecodeErrorEmptyBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL, Retry: RetryPolicy{MaxAttempts: 1}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/missing", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	err = client.Do(req, nil)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Fatalf("expected status code in error, got %v", err)
	}
}

func TestNewRequestWithJSONBody(t *testing.T) {
	client, err := New(Options{BaseURL: "https://example.com"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	body := map[string]string{"name": "test-repo"}
	req, err := client.NewRequest(context.Background(), http.MethodPost, "/repos", body)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if req.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", req.Header.Get("Content-Type"))
	}

	// Verify body is JSON
	data, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	var parsed map[string]string
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if parsed["name"] != "test-repo" {
		t.Fatalf("unexpected body: %v", parsed)
	}

	// Verify GetBody works for retries
	if req.GetBody == nil {
		t.Fatal("expected GetBody to be set")
	}
	body2, err := req.GetBody()
	if err != nil {
		t.Fatalf("GetBody: %v", err)
	}
	data2, _ := io.ReadAll(body2)
	if !bytes.Equal(data, data2) {
		t.Fatalf("GetBody returned different content")
	}
}

func TestNewRequestSetsBasicAuth(t *testing.T) {
	client, err := New(Options{BaseURL: "https://example.com", Username: "admin", Password: "token"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	user, pass, ok := req.BasicAuth()
	if !ok {
		t.Fatal("expected basic auth to be set")
	}
	if user != "admin" || pass != "token" {
		t.Fatalf("basic auth = %s:%s, want admin:token", user, pass)
	}
}

func TestNewRequestSetsBearerAuth(t *testing.T) {
	client, err := New(Options{BaseURL: "https://example.com", Password: "my-pat-token", AuthMethod: "bearer"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	authHeader := req.Header.Get("Authorization")
	if authHeader != "Bearer my-pat-token" {
		t.Fatalf("Authorization = %q, want %q", authHeader, "Bearer my-pat-token")
	}

	// Basic auth should NOT be set
	_, _, ok := req.BasicAuth()
	if ok {
		t.Fatal("expected basic auth NOT to be set for bearer method")
	}
}

func TestNewRequestBearerAuthNoToken(t *testing.T) {
	client, err := New(Options{BaseURL: "https://example.com", AuthMethod: "bearer"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if got := req.Header.Get("Authorization"); got != "" {
		t.Fatalf("expected no Authorization header when token is empty, got %q", got)
	}
}

func TestNewRequestDefaultsToBasicAuth(t *testing.T) {
	client, err := New(Options{BaseURL: "https://example.com", Username: "admin", Password: "token"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	user, pass, ok := req.BasicAuth()
	if !ok {
		t.Fatal("expected basic auth when AuthMethod is empty")
	}
	if user != "admin" || pass != "token" {
		t.Fatalf("basic auth = %s:%s, want admin:token", user, pass)
	}
}

func TestMultipartRequestBearerAuth(t *testing.T) {
	client, err := New(Options{BaseURL: "https://example.com", Password: "my-pat-token", AuthMethod: "bearer"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	files := []MultipartFile{{FieldName: "files", FileName: "test.txt", Reader: strings.NewReader("content")}}
	req, err := client.NewMultipartRequest(context.Background(), http.MethodPost, "/upload", files)
	if err != nil {
		t.Fatalf("NewMultipartRequest: %v", err)
	}

	authHeader := req.Header.Get("Authorization")
	if authHeader != "Bearer my-pat-token" {
		t.Fatalf("Authorization = %q, want %q", authHeader, "Bearer my-pat-token")
	}
}

func TestNewRequestRejectsEmptyPath(t *testing.T) {
	client, err := New(Options{BaseURL: "https://example.com"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.NewRequest(context.Background(), http.MethodGet, "", nil)
	if err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestNewRequiresBaseURL(t *testing.T) {
	_, err := New(Options{})
	if err == nil {
		t.Fatal("expected error for empty base URL")
	}
}

func TestNewRequiresScheme(t *testing.T) {
	_, err := New(Options{BaseURL: "example.com"})
	if err == nil {
		t.Fatal("expected error for URL without scheme")
	}
}

func TestNewRejectsUnsupportedAuthMethod(t *testing.T) {
	_, err := New(Options{BaseURL: "https://example.com", AuthMethod: "oauth"})
	if err == nil {
		t.Fatal("expected error for unsupported auth method")
	}
	if !strings.Contains(err.Error(), "unsupported auth method") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoWithIOWriter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("raw response body"))
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/stream", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	var buf bytes.Buffer
	if err := client.Do(req, &buf); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if buf.String() != "raw response body" {
		t.Fatalf("expected 'raw response body', got %q", buf.String())
	}
}

func TestDoNilRequest(t *testing.T) {
	client, err := New(Options{BaseURL: "https://example.com"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := client.Do(nil, nil); err == nil {
		t.Fatal("expected error for nil request")
	}
}

func TestClientRetriesOn429(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		if count == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload{Message: "ok"})
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{
		BaseURL: server.URL,
		Retry: RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     20 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	var out payload
	if err := client.Do(req, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if out.Message != "ok" {
		t.Fatalf("expected ok, got %q", out.Message)
	}
	if hits != 2 {
		t.Fatalf("expected 2 attempts, got %d", hits)
	}
}

func TestClientDoesNotRetryPostOnServerError(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{
		BaseURL: server.URL,
		Retry: RetryPolicy{
			MaxAttempts:    4,
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     20 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodPost, "/comments", payload{Message: "hi"})
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if err := client.Do(req, nil); err == nil {
		t.Fatalf("expected error from 500, got nil")
	}

	// A non-idempotent POST must not be retried on 5xx: the server may have
	// already created the resource, so a retry would duplicate it.
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected exactly 1 POST attempt, got %d", got)
	}
}

func TestClientRetriesIdempotentPutOnServerError(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		if count == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload{Message: "ok"})
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{
		BaseURL: server.URL,
		Retry: RetryPolicy{
			MaxAttempts:    4,
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     20 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodPut, "/comments/1", payload{Message: "edit"})
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	var out payload
	if err := client.Do(req, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}
	// PUT is idempotent, so a 5xx is retried like GET.
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("expected 2 PUT attempts, got %d", got)
	}
}

func TestClientRetriesPostOn429(t *testing.T) {
	var hits int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		if count == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload{Message: "ok"})
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{
		BaseURL: server.URL,
		Retry: RetryPolicy{
			MaxAttempts:    4,
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     20 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodPost, "/comments", payload{Message: "hi"})
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	var out payload
	if err := client.Do(req, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}
	// 429 means the request was rejected, not processed, so retrying a POST is
	// safe and expected.
	if got := atomic.LoadInt32(&hits); got != 2 {
		t.Fatalf("expected 2 POST attempts on 429, got %d", got)
	}
}

func TestShouldRetry(t *testing.T) {
	client, err := New(Options{BaseURL: "https://example.com", Retry: RetryPolicy{MaxAttempts: 4}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	tests := []struct {
		name     string
		attempts int
		method   string
		status   int
		want     bool
	}{
		// status == 0 is a transport/network error: unsafe for non-idempotent methods.
		{"network error POST", 0, http.MethodPost, 0, false},
		{"network error PATCH", 0, http.MethodPatch, 0, false},
		{"network error GET", 0, http.MethodGet, 0, true},
		{"network error PUT", 0, http.MethodPut, 0, true},
		{"network error DELETE", 0, http.MethodDelete, 0, true},
		// 5xx: same idempotency rule as transport errors.
		{"500 POST", 0, http.MethodPost, http.StatusInternalServerError, false},
		{"503 PATCH", 0, http.MethodPatch, http.StatusServiceUnavailable, false},
		{"500 GET", 0, http.MethodGet, http.StatusInternalServerError, true},
		{"502 PUT", 0, http.MethodPut, http.StatusBadGateway, true},
		// 429: rejected, not processed — retryable for any method.
		{"429 POST", 0, http.MethodPost, http.StatusTooManyRequests, true},
		{"429 PATCH", 0, http.MethodPatch, http.StatusTooManyRequests, true},
		// attempt budget exhausted: no retry even for an idempotent GET.
		{"attempts exhausted GET 500", 3, http.MethodGet, http.StatusInternalServerError, false},
		{"attempts exhausted POST 429", 3, http.MethodPost, http.StatusTooManyRequests, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := client.shouldRetry(tt.attempts, tt.method, tt.status); got != tt.want {
				t.Errorf("shouldRetry(%d, %q, %d) = %v, want %v", tt.attempts, tt.method, tt.status, got, tt.want)
			}
		})
	}
}

func TestRetryDelayCapsRetryAfter(t *testing.T) {
	client, err := New(Options{
		BaseURL: "https://example.com",
		Retry: RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     20 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	resp := &http.Response{
		Header: http.Header{
			"Retry-After": []string{"3600"},
		},
	}

	if got := client.retryDelay(0, resp); got != 60*time.Second {
		t.Fatalf("retryDelay = %v, want %v", got, 60*time.Second)
	}
}

func TestRetryDelayIgnoresNonPositiveRetryAfter(t *testing.T) {
	client, err := New(Options{
		BaseURL: "https://example.com",
		Retry: RetryPolicy{
			MaxAttempts:    3,
			InitialBackoff: 10 * time.Millisecond,
			MaxBackoff:     20 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	resp := &http.Response{
		Header: http.Header{
			"Retry-After": []string{"0"},
		},
	}

	if got := client.retryDelay(0, resp); got != 10*time.Millisecond {
		t.Fatalf("retryDelay = %v, want %v", got, 10*time.Millisecond)
	}
}

func TestShouldRetryStatus(t *testing.T) {
	tests := []struct {
		code int
		want bool
	}{
		{200, false},
		{201, false},
		{400, false},
		{401, false},
		{403, false},
		{404, false},
		{429, true},
		{500, true},
		{502, true},
		{503, true},
		{599, true},
	}
	for _, tt := range tests {
		if got := shouldRetryStatus(tt.code); got != tt.want {
			t.Errorf("shouldRetryStatus(%d) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestUpdateRateLimitAtlassianHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Attempt-RateLimit-Limit", "200")
		w.Header().Set("X-Attempt-RateLimit-Remaining", "150")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload{Message: "ok"})
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	var out payload
	if err := client.Do(req, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}

	rate := client.RateLimitState()
	if rate.Limit != 200 || rate.Remaining != 150 {
		t.Fatalf("expected 200/150, got %d/%d", rate.Limit, rate.Remaining)
	}
	if rate.Source != "atlassian" {
		t.Fatalf("expected source 'atlassian', got %q", rate.Source)
	}
}

func TestDoDiscardsBodyWhenVNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"key":"value"}`))
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	// Should not error even though v is nil
	if err := client.Do(req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
}

func TestTokenRefresherCalledOn401(t *testing.T) {
	var hits int32
	var authHeaders [2]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&hits, 1)
		if int(count) <= len(authHeaders) {
			authHeaders[count-1] = r.Header.Get("Authorization")
		}
		if count == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload{Message: "ok"})
	}))
	t.Cleanup(server.Close)

	refreshed := false
	client, err := New(Options{
		BaseURL:  server.URL,
		Username: "user",
		Password: "old-token",
		TokenRefresher: func(ctx context.Context) (string, error) {
			refreshed = true
			return "new-token", nil
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	var out payload
	if err := client.Do(req, &out); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if out.Message != "ok" {
		t.Fatalf("expected ok, got %q", out.Message)
	}
	if !refreshed {
		t.Fatal("expected TokenRefresher to be called")
	}
	if hits != 2 {
		t.Fatalf("expected 2 requests (original + retry), got %d", hits)
	}
	if authHeaders[0] == authHeaders[1] {
		t.Errorf("expected Authorization header to change after token refresh, but both requests sent %q", authHeaders[0])
	}
	if authHeaders[1] != "Bearer new-token" {
		t.Errorf("retry Authorization = %q, want %q", authHeaders[1], "Bearer new-token")
	}
}

func TestTokenRefresherErrorPropagated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{
		BaseURL: server.URL,
		TokenRefresher: func(ctx context.Context) (string, error) {
			return "", errors.New("refresh failed: token revoked")
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	err = client.Do(req, nil)
	if err == nil {
		t.Fatal("expected error when refresh fails")
	}
	if !strings.Contains(err.Error(), "refresh failed") {
		t.Errorf("expected refresh error message, got %v", err)
	}
}

func TestTokenRefresherNotCalledTwice(t *testing.T) {
	// 401 → refresh → 401 again → error, no second refresh attempt.
	var refreshCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{
		BaseURL: server.URL,
		TokenRefresher: func(ctx context.Context) (string, error) {
			atomic.AddInt32(&refreshCalls, 1)
			return "new-token", nil
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	err = client.Do(req, nil)
	if err == nil {
		t.Fatal("expected error after two 401s")
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("error after refresh = %T %v, want typed HTTPError status 401", err, err)
	}
	if refreshCalls != 1 {
		t.Fatalf("expected TokenRefresher called once, got %d", refreshCalls)
	}
}

func TestWithoutTokenRefresher401ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	client, err := New(Options{
		BaseURL: server.URL,
		Retry:   RetryPolicy{MaxAttempts: 1},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	err = client.Do(req, nil)
	if err == nil {
		t.Fatal("expected error for 401 without refresher")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected 401 in error, got %v", err)
	}
}

func TestNewRequestAbsoluteURL(t *testing.T) {
	client, err := New(Options{BaseURL: "https://example.com"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	req, err := client.NewRequest(context.Background(), http.MethodGet, "https://other.com/api/test", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	if got := req.URL.String(); got != "https://other.com/api/test" {
		t.Fatalf("expected absolute URL to be preserved, got %s", got)
	}
}

func TestConcurrent401RefreshCoalesced(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer new-token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload{Message: "ok"})
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	var refreshCalls int32
	client, err := New(Options{
		BaseURL:  server.URL,
		Username: "user",
		Password: "old-token",
		TokenRefresher: func(ctx context.Context) (string, error) {
			atomic.AddInt32(&refreshCalls, 1)
			time.Sleep(100 * time.Millisecond) // widen the coalescing window
			return "new-token", nil
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	const workers = 8
	var wg sync.WaitGroup
	errs := make([]error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
			if err != nil {
				errs[i] = err
				return
			}
			var out payload
			errs[i] = client.Do(req, &out)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("worker %d: %v", i, err)
		}
	}
	if got := atomic.LoadInt32(&refreshCalls); got != 1 {
		t.Fatalf("TokenRefresher called %d times, want exactly 1 (coalesced)", got)
	}
}

func TestRefreshSkippedWhenCredentialsAlreadyRotated(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer new-token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload{Message: "ok"})
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	var refreshCalls int32
	client, err := New(Options{
		BaseURL:  server.URL,
		Username: "user",
		Password: "old-token",
		TokenRefresher: func(ctx context.Context) (string, error) {
			atomic.AddInt32(&refreshCalls, 1)
			return "new-token", nil
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Build a request while the old token is still installed, so its baked
	// Authorization header goes stale after the first refresh.
	staleReq, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}

	// First request triggers the one and only refresh.
	firstReq, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	var out payload
	if err := client.Do(firstReq, &out); err != nil {
		t.Fatalf("Do(first): %v", err)
	}

	// The stale request 401s, but the 401 path must notice the rotated
	// credentials and retry with them instead of refreshing again.
	if err := client.Do(staleReq, &out); err != nil {
		t.Fatalf("Do(stale): %v", err)
	}
	if got := atomic.LoadInt32(&refreshCalls); got != 1 {
		t.Fatalf("TokenRefresher called %d times, want exactly 1", got)
	}
}

func TestFollowerRetriesWhenLeaderContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer new-token" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(payload{Message: "ok"})
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(server.Close)

	var refreshCalls int32
	leaderInRefresh := make(chan struct{})
	client, err := New(Options{
		BaseURL:  server.URL,
		Username: "user",
		Password: "old-token",
		TokenRefresher: func(ctx context.Context) (string, error) {
			if atomic.AddInt32(&refreshCalls, 1) == 1 {
				close(leaderInRefresh)
				<-ctx.Done() // first leader hangs until its request is cancelled
				return "", ctx.Err()
			}
			return "new-token", nil
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	defer cancelLeader()

	var wg sync.WaitGroup
	var leaderErr, followerErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		req, err := client.NewRequest(leaderCtx, http.MethodGet, "/api", nil)
		if err != nil {
			leaderErr = err
			return
		}
		var out payload
		leaderErr = client.Do(req, &out)
	}()

	<-leaderInRefresh // leader is inside the refresher, holding the in-flight call

	wg.Add(1)
	go func() {
		defer wg.Done()
		req, err := client.NewRequest(context.Background(), http.MethodGet, "/api", nil)
		if err != nil {
			followerErr = err
			return
		}
		var out payload
		followerErr = client.Do(req, &out)
	}()

	// Give the follower a moment to join the in-flight call, then kill the
	// leader. Either interleaving (follower waiting on the call, or arriving
	// after it fails) must converge on the same outcome.
	time.Sleep(50 * time.Millisecond)
	cancelLeader()
	wg.Wait()

	if leaderErr == nil || !errors.Is(leaderErr, context.Canceled) {
		t.Fatalf("leader err = %v, want context.Canceled", leaderErr)
	}
	if followerErr != nil {
		t.Fatalf("follower err = %v, want success via replacement refresh", followerErr)
	}
	if got := atomic.LoadInt32(&refreshCalls); got != 2 {
		t.Fatalf("TokenRefresher called %d times, want 2 (failed leader + follower replacement)", got)
	}
}

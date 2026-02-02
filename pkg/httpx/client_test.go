package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
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
			wantMsg: "403 Forbidden: CAPTCHA verification required: Account locked",
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

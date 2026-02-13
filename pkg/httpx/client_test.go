package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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

	if got := req.URL.String(); got != "https://example.com/rest/projects?limit=25&start=0" {
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

	if got := req.URL.String(); got != "https://example.com/rest/repos" {
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

func TestDecodeErrorStructuredMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
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

func TestDecodeErrorPlainText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Access denied"))
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

func TestDoWithIOWriter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("raw response body"))
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
		json.NewEncoder(w).Encode(payload{Message: "ok"})
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
		json.NewEncoder(w).Encode(payload{Message: "ok"})
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
		w.Write([]byte(`{"key":"value"}`))
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

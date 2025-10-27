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

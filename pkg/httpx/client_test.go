package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

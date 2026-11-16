package httpx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoWithResponseRefreshAndMetadata(t *testing.T) {
	var requests, refreshes int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Header.Get("Authorization") == "Bearer old" {
			w.Header().Set("Location", "/wrong")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Location", "/task-status/123")
		w.WriteHeader(http.StatusAccepted)
		_, _ = fmt.Fprint(w, `""`)
	}))
	t.Cleanup(server.Close)
	client, err := New(Options{BaseURL: server.URL, Password: "old", AuthMethod: "bearer", TokenRefresher: func(context.Context) (string, error) {
		refreshes++
		return "new", nil
	}})
	if err != nil {
		t.Fatal(err)
	}
	req, err := client.NewRequest(context.Background(), http.MethodPost, "/merge", map[string]bool{"close_source_branch": false})
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	response, err := client.DoWithResponse(req, &body)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 202 || response.Header.Get("Location") != "/task-status/123" || body.String() != `""` || requests != 2 || refreshes != 1 {
		t.Fatalf("response = %+v, body = %q, requests = %d, refreshes = %d", response, body.String(), requests, refreshes)
	}
}

func TestDoWithResponseErrorMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = fmt.Fprint(w, `{"error":{"message":"merge conflict"}}`)
	}))
	t.Cleanup(server.Close)
	client, err := New(Options{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	req, err := client.NewRequest(context.Background(), http.MethodPost, "/merge", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := client.DoWithResponse(req, nil)
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != 409 || response == nil || response.StatusCode != 409 {
		t.Fatalf("response = %+v, error = %v", response, err)
	}
	if response, err := client.DoWithResponse(nil, nil); response != nil || err == nil {
		t.Fatalf("nil request: response = %+v, error = %v", response, err)
	}
}

package oauth

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestCallbackServerStartsAndReceivesCode(t *testing.T) {
	srv, err := NewCallbackServer()
	if err != nil {
		t.Fatalf("NewCallbackServer: %v", err)
	}
	t.Cleanup(srv.Close)

	if srv.Port() == 0 {
		t.Fatal("expected non-zero port")
	}
	if srv.RedirectURI() == "" {
		t.Fatal("expected non-empty redirect URI")
	}

	// Simulate the browser redirect.
	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?code=mycode&state=mystate", srv.Port())
	go func() {
		time.Sleep(10 * time.Millisecond)
		resp, err := http.Get(callbackURL) //nolint:noctx
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	code, state, err := srv.Wait(ctx)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if code != "mycode" {
		t.Errorf("code = %q, want %q", code, "mycode")
	}
	if state != "mystate" {
		t.Errorf("state = %q, want %q", state, "mystate")
	}
}

func TestCallbackServerHandlesOAuthError(t *testing.T) {
	srv, err := NewCallbackServer()
	if err != nil {
		t.Fatalf("NewCallbackServer: %v", err)
	}
	t.Cleanup(srv.Close)

	callbackURL := fmt.Sprintf(
		"http://127.0.0.1:%d/callback?error=access_denied&error_description=User+denied+access",
		srv.Port(),
	)
	go func() {
		time.Sleep(10 * time.Millisecond)
		resp, err := http.Get(callbackURL) //nolint:noctx
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, _, err = srv.Wait(ctx)
	if err == nil {
		t.Fatal("expected error for OAuth error callback")
	}
	if err.Error() != "User denied access" {
		t.Errorf("error = %q, want %q", err.Error(), "User denied access")
	}
}

func TestCallbackServerHandlesMissingCode(t *testing.T) {
	srv, err := NewCallbackServer()
	if err != nil {
		t.Fatalf("NewCallbackServer: %v", err)
	}
	t.Cleanup(srv.Close)

	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?state=only", srv.Port())
	go func() {
		time.Sleep(10 * time.Millisecond)
		resp, err := http.Get(callbackURL) //nolint:noctx
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, _, err = srv.Wait(ctx)
	if err == nil {
		t.Fatal("expected error when code is missing")
	}
}

func TestCallbackServerHandlesMissingState(t *testing.T) {
	srv, err := NewCallbackServer()
	if err != nil {
		t.Fatalf("NewCallbackServer: %v", err)
	}
	t.Cleanup(srv.Close)

	callbackURL := fmt.Sprintf("http://127.0.0.1:%d/callback?code=somecode", srv.Port())
	go func() {
		time.Sleep(10 * time.Millisecond)
		resp, err := http.Get(callbackURL) //nolint:noctx
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, _, err = srv.Wait(ctx)
	if err == nil {
		t.Fatal("expected error when state is missing")
	}
}

func TestCallbackServerContextCancellation(t *testing.T) {
	srv, err := NewCallbackServer()
	if err != nil {
		t.Fatalf("NewCallbackServer: %v", err)
	}
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, _, err = srv.Wait(ctx)
	if err == nil {
		t.Fatal("expected error on cancelled context")
	}
}

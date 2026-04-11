package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestBuildAuthorizeURL(t *testing.T) {
	opts := FlowOptions{
		ClientID:     "test-id",
		AuthorizeURL: "https://bitbucket.org/site/oauth2/authorize",
		Scopes:       []string{"account", "repository"},
	}
	got := buildAuthorizeURL(opts, "http://127.0.0.1:12345/callback", "test-state")
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if u.Host != "bitbucket.org" {
		t.Errorf("host = %q, want bitbucket.org", u.Host)
	}
	q := u.Query()
	if q.Get("client_id") != "test-id" {
		t.Errorf("client_id = %q", q.Get("client_id"))
	}
	if q.Get("response_type") != "code" {
		t.Errorf("response_type = %q", q.Get("response_type"))
	}
	if q.Get("redirect_uri") != "http://127.0.0.1:12345/callback" {
		t.Errorf("redirect_uri = %q", q.Get("redirect_uri"))
	}
	if q.Get("state") != "test-state" {
		t.Errorf("state = %q", q.Get("state"))
	}
	if q.Get("scope") != "account repository" {
		t.Errorf("scope = %q", q.Get("scope"))
	}
}

func TestExchangeCode(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("grant_type") != "authorization_code" {
			t.Errorf("grant_type = %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("code") != "test-code" {
			t.Errorf("code = %q", r.Form.Get("code"))
		}
		// redirect_uri must be present and match what was sent at authorize.
		if r.Form.Get("redirect_uri") == "" {
			t.Error("redirect_uri must be sent at token exchange")
		}
		// Verify Basic auth.
		user, pass, ok := r.BasicAuth()
		if !ok || user != "cid" || pass != "csecret" {
			t.Errorf("basic auth = (%q, %q, %v)", user, pass, ok)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "at-123",
			"refresh_token": "rt-456",
			"expires_in":    7200,
			"token_type":    "bearer",
			"scopes":        "account repository",
		})
	}))
	defer tokenSrv.Close()

	opts := FlowOptions{
		ClientID:     "cid",
		ClientSecret: "csecret",
		TokenURL:     tokenSrv.URL,
	}
	tok, err := exchangeCode(context.Background(), opts, "test-code", "http://127.0.0.1:12345/callback")
	if err != nil {
		t.Fatalf("exchangeCode: %v", err)
	}
	if tok.AccessToken != "at-123" {
		t.Errorf("access_token = %q", tok.AccessToken)
	}
	if tok.RefreshToken != "rt-456" {
		t.Errorf("refresh_token = %q", tok.RefreshToken)
	}
	if tok.IsExpired() {
		t.Error("token should not be expired")
	}
}

func TestExchangeCodeError(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error":             "invalid_grant",
			"error_description": "code already used",
		})
	}))
	defer tokenSrv.Close()

	opts := FlowOptions{
		ClientID:     "cid",
		ClientSecret: "csecret",
		TokenURL:     tokenSrv.URL,
	}
	_, err := exchangeCode(context.Background(), opts, "used-code", "http://127.0.0.1:12345/callback")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "code already used") {
		t.Errorf("error = %q, want to contain 'code already used'", err)
	}
}

func TestRefreshToken(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Errorf("grant_type = %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("refresh_token") != "rt-old" {
			t.Errorf("refresh_token = %q", r.Form.Get("refresh_token"))
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != "cid" || pass != "csecret" {
			t.Errorf("basic auth = (%q, %q, %v)", user, pass, ok)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "at-new",
			"refresh_token": "rt-new",
			"expires_in":    7200,
		})
	}))
	defer tokenSrv.Close()

	tok, err := RefreshToken(context.Background(), "rt-old", "cid", "csecret", tokenSrv.URL)
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}
	if tok.AccessToken != "at-new" {
		t.Errorf("access_token = %q", tok.AccessToken)
	}
	if tok.RefreshToken != "rt-new" {
		t.Errorf("refresh_token = %q", tok.RefreshToken)
	}
}

func TestRunFlowStateMismatch(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("token endpoint should not be called on state mismatch")
	}))
	defer tokenSrv.Close()

	opts := FlowOptions{
		ClientID:     "cid",
		ClientSecret: "csecret",
		AuthorizeURL: "https://bitbucket.org/site/oauth2/authorize",
		TokenURL:     tokenSrv.URL,
		OpenBrowser: func(authURL string) error {
			// Parse the state from the authorize URL.
			u, _ := url.Parse(authURL)
			// Simulate callback with wrong state.
			callbackURL := u.Query().Get("redirect_uri") + "?code=test-code&state=wrong-state"
			resp, err := http.Get(callbackURL)
			if err != nil {
				return err
			}
			_ = resp.Body.Close()
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := RunFlow(ctx, opts)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "state mismatch") {
		t.Errorf("error = %q, want state mismatch", err)
	}
}

func TestRunFlowSuccess(t *testing.T) {
	// Mock token endpoint.
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "at-flow",
			"refresh_token": "rt-flow",
			"expires_in":    7200,
		})
	}))
	defer tokenSrv.Close()

	// Mock user endpoint.
	userSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer at-flow" {
			t.Errorf("Authorization = %q, want Bearer at-flow", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"username":     "testuser",
			"display_name": "Test User",
		})
	}))
	defer userSrv.Close()

	var buf strings.Builder
	opts := FlowOptions{
		ClientID:     "cid",
		ClientSecret: "csecret",
		AuthorizeURL: "https://bitbucket.org/site/oauth2/authorize",
		TokenURL:     tokenSrv.URL,
		UserInfoURL:  userSrv.URL,
		Out:          &buf,
		OpenBrowser: func(authURL string) error {
			u, _ := url.Parse(authURL)
			state := u.Query().Get("state")
			redirectURI := u.Query().Get("redirect_uri")
			callbackURL := fmt.Sprintf("%s?code=auth-code&state=%s", redirectURI, state)
			resp, err := http.Get(callbackURL)
			if err != nil {
				return err
			}
			_ = resp.Body.Close()
			return nil
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := RunFlow(ctx, opts)
	if err != nil {
		t.Fatalf("RunFlow: %v", err)
	}
	if result.Token.AccessToken != "at-flow" {
		t.Errorf("access_token = %q", result.Token.AccessToken)
	}
	if result.Username != "testuser" {
		t.Errorf("username = %q", result.Username)
	}
	if result.DisplayName != "Test User" {
		t.Errorf("display_name = %q, want %q", result.DisplayName, "Test User")
	}
}

func TestFetchUsernameSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"username":     "erank",
			"account_id":   "123:abc",
			"display_name": "Eran K",
		})
	}))
	defer srv.Close()

	username, display, err := fetchUsername(context.Background(), "tok", srv.URL)
	if err != nil {
		t.Fatalf("fetchUsername: %v", err)
	}
	if username != "erank" {
		t.Errorf("username = %q, want erank", username)
	}
	if display != "Eran K" {
		t.Errorf("display = %q, want Eran K", display)
	}
}

func TestFetchUsernameFallsBackToAccountID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"account_id":   "123:abc",
			"display_name": "Eran K",
		})
	}))
	defer srv.Close()

	username, _, err := fetchUsername(context.Background(), "tok", srv.URL)
	if err != nil {
		t.Fatalf("fetchUsername: %v", err)
	}
	if username != "123:abc" {
		t.Errorf("username = %q, want 123:abc", username)
	}
}

func TestFetchUsernameMissingBothFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"display_name": "Eran K",
		})
	}))
	defer srv.Close()

	_, _, err := fetchUsername(context.Background(), "tok", srv.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "missing both username and account_id") {
		t.Errorf("error = %q", err)
	}
}

func TestFetchUsernameNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	_, _, err := fetchUsername(context.Background(), "tok", srv.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error = %q, want 403", err)
	}
}

func TestDoTokenRequestEmptyAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"refresh_token": "rt",
			"expires_in":    7200,
		})
	}))
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL, nil)
	_, err := doTokenRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "empty access_token") {
		t.Errorf("error = %q", err)
	}
}

func TestDoTokenRequestErrorNoDescription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid_grant",
		})
	}))
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL, nil)
	_, err := doTokenRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid_grant") {
		t.Errorf("error = %q, want invalid_grant", err)
	}
}

func TestDoTokenRequestErrorPlainStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	req, _ := http.NewRequest("POST", srv.URL, nil)
	_, err := doTokenRequest(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error = %q, want 500", err)
	}
}

func TestRunFlowNoBrowserPrintsURL(t *testing.T) {
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "at-nobrowser",
			"refresh_token": "rt-nobrowser",
			"expires_in":    7200,
		})
	}))
	defer tokenSrv.Close()

	userSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"username": "testuser",
		})
	}))
	defer userSrv.Close()

	var buf strings.Builder
	// OpenBrowser is nil → should print URL to Out.
	// But we need to simulate the callback, so we use OpenBrowser to do it.
	// Instead, test the browser-failure fallback path.
	opts := FlowOptions{
		ClientID:     "cid",
		ClientSecret: "csecret",
		AuthorizeURL: "https://bitbucket.org/site/oauth2/authorize",
		TokenURL:     tokenSrv.URL,
		UserInfoURL:  userSrv.URL,
		Out:          &buf,
		OpenBrowser: func(authURL string) error {
			// Simulate browser failure → fallback prints URL.
			u, _ := url.Parse(authURL)
			state := u.Query().Get("state")
			redirectURI := u.Query().Get("redirect_uri")
			// Still send the callback so the flow completes.
			callbackURL := fmt.Sprintf("%s?code=code&state=%s", redirectURI, state)
			resp, err := http.Get(callbackURL) //nolint:gosec
			if err != nil {
				return err
			}
			_ = resp.Body.Close()
			return fmt.Errorf("browser unavailable")
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := RunFlow(ctx, opts)
	if err != nil {
		t.Fatalf("RunFlow: %v", err)
	}
	if result.Token.AccessToken != "at-nobrowser" {
		t.Errorf("access_token = %q", result.Token.AccessToken)
	}
	output := buf.String()
	if !strings.Contains(output, "Failed to open browser") {
		t.Errorf("output missing browser failure message: %q", output)
	}
	if !strings.Contains(output, "Open this URL manually") {
		t.Errorf("output missing manual URL fallback: %q", output)
	}
}

func TestRandomState(t *testing.T) {
	s1, err := randomState()
	if err != nil {
		t.Fatalf("randomState: %v", err)
	}
	s2, _ := randomState()
	if s1 == s2 {
		t.Error("two calls returned same state")
	}
	if len(s1) != 32 {
		t.Errorf("state length = %d, want 32", len(s1))
	}
}

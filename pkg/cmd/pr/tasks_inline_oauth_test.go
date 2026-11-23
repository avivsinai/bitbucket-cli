package pr_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/internal/secret"
	"github.com/avivsinai/bitbucket-cli/pkg/oauth"
)

type taskOAuthTransport func(*http.Request) (*http.Response, error)

func (f taskOAuthTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestPRTaskCreateInlineCloudOAuth(t *testing.T) {
	const (
		baseURL        = "https://fixture.invalid"
		commentURL     = baseURL + "/repositories/myworkspace/my-repo/pullrequests/42/comments"
		taskURL        = baseURL + "/repositories/myworkspace/my-repo/pullrequests/42/tasks"
		clientID       = "fixture-consumer"
		clientSecret   = "fixture-secret"
		oldAccessToken = "fixture-old"
		newAccessToken = "fixture-new"
		refreshToken   = "fixture-refresh"
		commentReply   = `{"id":19}`
		taskReply      = `{"id":77,"content":{"raw":"fixture"},"state":"UNRESOLVED"}`
	)
	type request struct {
		url    string
		bearer string
		status int
		reply  string
	}
	refresh := request{oauth.CloudTokenURL, "", http.StatusOK, `{"access_token":"fixture-new","refresh_token":"fixture-rotated","expires_in":3600}`}
	comment := request{commentURL, newAccessToken, http.StatusCreated, commentReply}
	task := request{taskURL, newAccessToken, http.StatusCreated, taskReply}
	tests := []struct {
		name      string
		expiresIn int
		requests  []request
		wantError string
	}{
		{
			name:      "expired token refreshes before the comment",
			expiresIn: -60,
			requests:  []request{refresh, comment, task},
		},
		{
			name:      "refresh failure stops before any mutation",
			expiresIn: -60,
			requests: []request{
				{oauth.CloudTokenURL, "", http.StatusBadRequest, `{"error":"invalid_grant"}`},
			},
			wantError: "refresh OAuth token before creating inline task",
		},
		{
			name:      "task 401 preserves the comment without replay",
			expiresIn: -60,
			requests: []request{
				refresh, comment,
				{taskURL, newAccessToken, http.StatusUnauthorized, `{"error":{"message":"fixture unauthorized"}}`},
			},
			wantError: "created inline comment 19 on pull request #42. Could not confirm task creation",
		},
		{
			name:      "unexpired token skips refresh",
			expiresIn: 3600,
			requests: []request{
				{commentURL, oldAccessToken, http.StatusCreated, commentReply},
				{taskURL, oldAccessToken, http.StatusCreated, taskReply},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configDir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", configDir)
			t.Setenv("APPDATA", configDir)
			t.Setenv(secret.EnvHost, "")
			t.Setenv(secret.EnvToken, "")
			t.Setenv("KEYRING_BACKEND", "file")
			t.Setenv("KEYRING_FILE_DIR", t.TempDir())
			t.Setenv("BKT_KEYRING_PASSPHRASE", "test-pass")
			t.Setenv("BKT_OAUTH_CLIENT_ID", clientID)
			t.Setenv("BKT_OAUTH_CLIENT_SECRET", clientSecret)

			cfg := cloudConfig(baseURL)
			cfg.SetHost("mock", &config.Host{Kind: "cloud", BaseURL: baseURL, AuthMethod: "oauth", AllowInsecureStore: true})
			store, err := secret.Open(secret.WithAllowFileFallback(true))
			if err != nil {
				t.Fatal(err)
			}
			token := oauth.FromResponse(oldAccessToken, refreshToken, tt.expiresIn)
			token.ClientID = clientID
			blob, err := token.Marshal()
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Set(secret.TokenKey("mock"), blob); err != nil {
				t.Fatal(err)
			}

			// Both the OAuth exchange and PR writes are handled entirely in memory.
			var received int
			transport := http.DefaultTransport
			http.DefaultTransport = taskOAuthTransport(func(r *http.Request) (*http.Response, error) {
				if r.Body != nil {
					defer func() { _ = r.Body.Close() }()
				}
				index := received
				received++
				if index >= len(tt.requests) {
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
					return nil, fmt.Errorf("unexpected fixture request")
				}
				want := tt.requests[index]
				if r.Method != http.MethodPost || r.URL.String() != want.url {
					t.Errorf("request %d = %s %s, want POST %s", index, r.Method, r.URL, want.url)
					return nil, fmt.Errorf("unexpected fixture request")
				}
				if want.url == oauth.CloudTokenURL {
					id, password, ok := r.BasicAuth()
					if !ok || id != clientID || password != clientSecret {
						t.Error("the OAuth exchange did not use the fixture consumer")
					}
					if err := r.ParseForm(); err != nil {
						return nil, err
					}
					if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != refreshToken {
						t.Error("the OAuth exchange did not use the saved fixture refresh token")
					}
				} else if r.Header.Get("Authorization") != "Bearer "+want.bearer {
					t.Errorf("request %d did not use the expected fixture access token", index)
				}

				response := httptest.NewRecorder()
				response.Header().Set("Content-Type", "application/json")
				response.WriteHeader(want.status)
				_, _ = fmt.Fprint(response, want.reply)
				return response.Result(), nil
			})
			t.Cleanup(func() { http.DefaultTransport = transport })

			stdout, stderr, err := runCLI(t, cfg, "--json", "pr", "task", "create", "42", "--text", "fixture", "--file", "test.go", "--to-line", "11")
			if received != len(tt.requests) {
				t.Errorf("sent %d requests, want %d", received, len(tt.requests))
			}
			if tt.wantError != "" {
				if err == nil || stdout != "" || !strings.Contains(stderr+err.Error(), tt.wantError) {
					t.Fatalf("stdout = %q, stderr = %q, error = %v, want %q", stdout, stderr, err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("pr task create: %v (stderr=%s)", err, stderr)
			}
			assertTaskJSON(t, stdout, 77, "OPEN")
		})
	}
}

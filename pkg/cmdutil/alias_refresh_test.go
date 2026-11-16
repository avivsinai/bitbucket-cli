package cmdutil

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/internal/secret"
	"github.com/avivsinai/bitbucket-cli/pkg/oauth"
)

type aliasRoundTripper func(*http.Request) (*http.Response, error)

func (f aliasRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestAliasRefreshThroughResolvedContext(t *testing.T) {
	for _, scenario := range []string{"rotate concurrently", "reuse", "missing", "refresh fails"} {
		t.Run(scenario, func(t *testing.T) {
			t.Setenv(secret.EnvToken, "")
			t.Setenv(secret.EnvHost, "")
			t.Setenv("BKT_CONFIG_DIR", t.TempDir())
			t.Setenv("KEYRING_BACKEND", "file")
			t.Setenv("KEYRING_FILE_DIR", t.TempDir())
			t.Setenv("BKT_ALLOW_INSECURE_STORE", "1")
			t.Setenv("BKT_KEYRING_PASSPHRASE", "test-pass")
			t.Setenv("BKT_OAUTH_CLIENT_ID", "fixture-consumer")
			t.Setenv("BKT_OAUTH_CLIENT_SECRET", "fixture-secret") // gitleaks:allow
			store, err := secret.Open(secret.WithAllowFileFallback(true))
			if err != nil {
				t.Fatal(err)
			}
			put := func(key, access, refresh string, expires int) string {
				t.Helper()
				tok := oauth.FromResponse(access, refresh, expires)
				tok.ClientID = "fixture-consumer"
				blob, err := tok.Marshal()
				if err != nil {
					t.Fatal(err)
				}
				if err := store.Set(secret.TokenKey(key), blob); err != nil {
					t.Fatal(err)
				}
				return blob
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Header.Get("Authorization") {
				case "Bearer alias-old":
					w.WriteHeader(http.StatusUnauthorized)
				case "Bearer alias-new":
					_, _ = fmt.Fprint(w, `{"username":"alias-user"}`)
				default:
					t.Error("request used a credential outside the selected alias")
					w.WriteHeader(http.StatusForbidden)
				}
			}))
			t.Cleanup(server.Close)
			serverKey, err := HostKeyFromURL(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			otherBlob := put(serverKey, "other-access", "other-refresh", 7200)
			put("team-b", "alias-old", "alias-refresh", -60)
			cfg, err := config.Load()
			if err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{serverKey, "team-b"} {
				cfg.SetHost(key, &config.Host{Kind: "cloud", BaseURL: server.URL, AuthMethod: "oauth", AllowInsecureStore: true})
			}
			cfg.SetContext("team-b", &config.Context{Host: "team-b", Workspace: "fixture"})
			if err := cfg.SetActiveContext("team-b"); err != nil {
				t.Fatal(err)
			}
			if err := cfg.Save(); err != nil {
				t.Fatal(err)
			}
			loaded, err := config.Load()
			if err != nil {
				t.Fatal(err)
			}
			factory := &Factory{ExecutableName: "bkt", Config: func() (*config.Config, error) { return loaded, nil }}
			_, _, host, err := ResolveContextStatic(factory, "")
			if err != nil {
				t.Fatal(err)
			}
			if host.Token != "alias-old" || host.Key() != "team-b" {
				t.Fatal("context did not load its own identity and credential")
			}
			if scenario == "reuse" {
				put("team-b", "alias-new", "alias-rotated", 7200)
			}
			if scenario == "missing" {
				if err := store.Delete(secret.TokenKey("team-b")); err != nil {
					t.Fatal(err)
				}
			}

			// Every network destination is either this local fixture or a mocked
			// OAuth exchange. No request can reach a live Bitbucket service.
			var refreshes atomic.Int32
			transport := http.DefaultTransport
			http.DefaultTransport = aliasRoundTripper(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == oauth.CloudTokenURL {
					refreshes.Add(1)
					if err := r.ParseForm(); err != nil {
						return nil, err
					}
					if r.Form.Get("refresh_token") != "alias-refresh" {
						t.Error("exchange used a refresh token outside the alias")
					}
					status, body := http.StatusOK, `{"access_token":"alias-new","refresh_token":"alias-rotated","expires_in":7200}`
					if scenario == "refresh fails" {
						status, body = http.StatusBadRequest, `{"error":"invalid_grant"}`
					}
					return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
				}
				if r.URL.Scheme+"://"+r.URL.Host != server.URL {
					return nil, fmt.Errorf("unexpected outbound request")
				}
				return transport.RoundTrip(r)
			})
			t.Cleanup(func() { http.DefaultTransport = transport })
			count := 1
			if scenario == "rotate concurrently" {
				count = 8
			}
			results := make(chan error, count)
			for range count {
				copyHost := *host
				client, err := NewCloudClient(&copyHost)
				if err != nil {
					t.Fatal(err)
				}
				go func() {
					user, err := client.CurrentUser(context.Background())
					if err == nil && user.Username != "alias-user" {
						err = fmt.Errorf("unexpected fixture user")
					}
					results <- err
				}()
			}
			for range count {
				err := <-results
				if scenario == "missing" || scenario == "refresh fails" {
					if err == nil {
						t.Fatal("expected refresh failure")
					}
					if scenario == "refresh fails" && !strings.Contains(err.Error(), "--alias team-b --kind cloud --web") {
						t.Fatalf("missing identity in recovery hint: %v", err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
			}
			wantRefreshes := int32(0)
			if scenario == "rotate concurrently" || scenario == "refresh fails" {
				wantRefreshes = 1
			}
			if refreshes.Load() != wantRefreshes {
				t.Fatalf("exchanges = %d, want %d", refreshes.Load(), wantRefreshes)
			}
			if got, err := store.Get(secret.TokenKey(serverKey)); err != nil || got != otherBlob {
				t.Fatal("another account's stored credential changed")
			}
			if scenario == "rotate concurrently" {
				blob, err := store.Get(secret.TokenKey("team-b"))
				if err != nil {
					t.Fatal(err)
				}
				tok, err := oauth.Unmarshal(blob)
				if err != nil || tok.AccessToken != "alias-new" || tok.RefreshToken != "alias-rotated" {
					t.Fatal("alias credential was not rotated in its own entry")
				}
			}
			aliasLock, err := oauthRefreshLockPath("team-b")
			if err != nil {
				t.Fatal(err)
			}
			otherLock, err := oauthRefreshLockPath(serverKey)
			if err != nil || aliasLock == otherLock {
				t.Fatal("accounts share a refresh lock")
			}
		})
	}
}

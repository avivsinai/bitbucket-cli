package cmdutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
)

func TestParseExtraHeaders(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    http.Header
		wantErr string
	}{
		{
			name: "cookie value keeps its semicolons",
			raw:  "Cookie: session=abc; other=1",
			want: http.Header{"Cookie": {"session=abc; other=1"}},
		},
		{
			name: "several lines, blank lines and CRLF",
			raw:  "X-Proxy-Token: t1\r\n\r\nX-Trace:  on \n",
			want: http.Header{"X-Proxy-Token": {"t1"}, "X-Trace": {"on"}},
		},
		{
			name: "value may contain a colon",
			raw:  "X-Forwarded-Host: example.com:8443",
			want: http.Header{"X-Forwarded-Host": {"example.com:8443"}},
		},
		{name: "missing colon", raw: "Cookie session=abc", wantErr: "line 1"},
		{name: "empty name", raw: "X-A: 1\n: value", wantErr: "line 2"},
		{name: "name with space", raw: "Bad Name: 1", wantErr: "line 1"},
		{name: "authorization refused", raw: "authorization: Bearer x", wantErr: "Authorization"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseExtraHeaders(tt.raw)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to mention %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseExtraHeaders: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for name, values := range tt.want {
				if g := got.Values(name); strings.Join(g, "|") != strings.Join(values, "|") {
					t.Errorf("%s = %q, want %q", name, g, values)
				}
			}
		})
	}
}

func TestExtraHeadersFromEnvScope(t *testing.T) {
	t.Run("refused without BKT_HOST", func(t *testing.T) {
		t.Setenv("BKT_HOST", "")
		t.Setenv("BKT_EXTRA_HEADERS", "Cookie: s=1")
		if _, err := ExtraHeadersFromEnv("https://bitbucket.example.com"); err == nil || !strings.Contains(err.Error(), "BKT_HOST") {
			t.Fatalf("err = %v, want a BKT_HOST error", err)
		}
	})
	t.Run("not sent to another server", func(t *testing.T) {
		t.Setenv("BKT_HOST", "https://bitbucket.example.com")
		t.Setenv("BKT_EXTRA_HEADERS", "Cookie: s=1")
		got, err := ExtraHeadersFromEnv("https://other.example.com")
		if err != nil || got != nil {
			t.Fatalf("got %v, %v; want nil, nil", got, err)
		}
	})
	t.Run("not sent over http to an https BKT_HOST", func(t *testing.T) {
		t.Setenv("BKT_HOST", "https://bitbucket.example.com")
		t.Setenv("BKT_EXTRA_HEADERS", "Cookie: s=1")
		got, err := ExtraHeadersFromEnv("http://bitbucket.example.com")
		if err != nil || got != nil {
			t.Fatalf("got %v, %v; want nil, nil", got, err)
		}
	})
	t.Run("sent to the BKT_HOST server", func(t *testing.T) {
		t.Setenv("BKT_HOST", "https://bitbucket.example.com")
		t.Setenv("BKT_EXTRA_HEADERS", "Cookie: s=1")
		got, err := ExtraHeadersFromEnv("https://bitbucket.example.com/")
		if err != nil || got.Get("Cookie") != "s=1" {
			t.Fatalf("got %v, %v; want the cookie", got, err)
		}
	})
	t.Run("unset is a no-op", func(t *testing.T) {
		t.Setenv("BKT_HOST", "")
		t.Setenv("BKT_EXTRA_HEADERS", "")
		got, err := ExtraHeadersFromEnv("https://bitbucket.example.com")
		if err != nil || got != nil {
			t.Fatalf("got %v, %v; want nil, nil", got, err)
		}
	})
}

func TestNewDCClientSendsExtraHeadersToBKTHost(t *testing.T) {
	var gotCookie, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	t.Setenv("BKT_HOST", srv.URL)
	t.Setenv("BKT_EXTRA_HEADERS", "Cookie: proxy_session=abc")

	client, err := NewDCClient(&config.Host{Kind: "dc", BaseURL: srv.URL, AuthMethod: "bearer", Token: "pat"})
	if err != nil {
		t.Fatalf("NewDCClient: %v", err)
	}
	req, err := client.HTTP().NewRequest(context.Background(), http.MethodGet, "/rest/api/1.0/projects", nil)
	if err != nil {
		t.Fatalf("NewRequest: %v", err)
	}
	if err := client.HTTP().Do(req, nil); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if gotCookie != "proxy_session=abc" {
		t.Errorf("Cookie = %q, want proxy_session=abc", gotCookie)
	}
	if gotAuth != "Bearer pat" {
		t.Errorf("Authorization = %q, want the token", gotAuth)
	}
}

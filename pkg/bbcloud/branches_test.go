package bbcloud_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
)

func TestListBranchesEscapesBBQLFilter(t *testing.T) {
	var capturedURL string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"values": []map[string]any{}})
	}))
	t.Cleanup(server.Close)

	client, err := bbcloud.New(bbcloud.Options{BaseURL: server.URL, Username: "u", Token: "t"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.ListBranches(context.Background(), "workspace", "repo", bbcloud.BranchListOptions{
		Filter: `release "1"\draft`,
	})
	if err != nil {
		t.Fatalf("ListBranches: %v", err)
	}

	parsedURL, err := url.Parse(capturedURL)
	if err != nil {
		t.Fatalf("Parse URL: %v", err)
	}
	rawQuery, _ := url.QueryUnescape(parsedURL.RawQuery)
	if !strings.Contains(rawQuery, `q=name ~ "release \"1\"\\draft"`) {
		t.Fatalf("raw query = %q", rawQuery)
	}
}

package pr_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestJQOutputCLI(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "pr list implies json", args: []string{"pr", "list", "--jq", ".pull_requests[0].id"}},
		{name: "pr list explicit json", args: []string{"--json", "pr", "list", "--jq", ".pull_requests[0].id"}},
		{name: "api explicit json format", args: []string{"api", "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests", "--format", "json", "--jq", ".values[0].id"}},
		{name: "api implies json", args: []string{"api", "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests", "--jq", ".values[0].id"}},
		{name: "pr list rejects yaml", args: []string{"--yaml", "pr", "list", "--jq", ".pull_requests[0].id"}, wantErr: true},
		{name: "pr list rejects yaml format", args: []string{"pr", "list", "--format=yaml", "--jq", ".pull_requests[0].id"}, wantErr: true},
		{name: "pr list respects json false", args: []string{"pr", "list", "--json=false", "--jq", ".pull_requests[0].id"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests" {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, "{\"isLastPage\":true,\"values\":[{\"id\":42,\"title\":\"fixture\",\"state\":\"OPEN\"}]}")
			}))
			t.Cleanup(server.Close)

			stdout, stderr, err := runCLI(t, dcConfig(server.URL), tt.args...)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "--jq requires JSON output") {
					t.Fatalf("error = %v, want incompatible output flags", err)
				}
				if stdout != "" || requests.Load() != 0 {
					t.Fatalf("invalid flags produced stdout %q and %d requests", stdout, requests.Load())
				}
				return
			}
			if err != nil {
				t.Fatalf("command failed: %v (stderr=%s)", err, stderr)
			}
			if stdout != "42\n" || requests.Load() != 1 {
				t.Fatalf("stdout = %q and requests = %d, want one filtered response", stdout, requests.Load())
			}
		})
	}
}

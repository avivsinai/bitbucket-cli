package pr_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPRMergeabilityCloud(t *testing.T) {
	for _, format := range []string{"text", "json", "yaml", "jq", "failure"} {
		t.Run(format, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/repositories/other-workspace/other-repo/pullrequests/42/mergeability/checks" {
					t.Fatalf("unexpected request %s %s", r.Method, r.URL)
				}
				if format == "failure" {
					http.Error(w, "not allowed", http.StatusForbidden)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"values":[
				{"type":"pullrequest_state_check","status":"PASSED","blocking":false,"required":true},
				{"type":"current_user_permission_check","status":"PASSED","blocking":false,"required":true},
				{"type":"git_mergeability_check","status":"PASSED","reason":"clean","blocking":false,"required":true}]}`))
			}))
			defer srv.Close()
			args := []string{"pr", "mergeability", "42", "--workspace", "other-workspace", "--repo", "other-repo"}
			if format == "yaml" {
				args = append(args, "--yaml")
			} else if format != "text" {
				args = append(args, "--json")
			}
			if format == "jq" {
				args = append(args, "--jq", ".mergeable")
			}
			stdout, stderr, err := runCLI(t, cloudConfig(srv.URL), args...)
			if format == "failure" {
				if err == nil || !strings.Contains(stdout, `"mergeable": "UNKNOWN"`) {
					t.Fatalf("stdout=%s, error=%v; want UNKNOWN and failure", stdout, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("error=%v, stderr=%s", err, stderr)
			}
			switch format {
			case "text":
				if !strings.Contains(stdout, "Mergeable: MERGEABLE\nMerge state: CLEAN") {
					t.Fatal(stdout)
				}
			case "json":
				var result struct{ Mergeable, MergeStateStatus string }
				if err := json.Unmarshal([]byte(stdout), &result); err != nil || result.Mergeable != "MERGEABLE" || result.MergeStateStatus != "CLEAN" {
					t.Fatalf("invalid JSON summary %s: %v", stdout, err)
				}
			case "yaml":
				var result map[string]any
				if err := yaml.Unmarshal([]byte(stdout), &result); err != nil || result["mergeable"] != "MERGEABLE" || result["mergeStateStatus"] != "CLEAN" {
					t.Fatalf("invalid YAML summary %s: %v", stdout, err)
				}
				checks, ok := result["checks"].([]any)
				if !ok || len(checks) != 3 {
					t.Fatalf("invalid YAML checks: %s", stdout)
				}
				check, ok := checks[0].(map[string]any)
				if !ok || check["type"] != "pullrequest_state_check" {
					t.Fatalf("lost YAML check fields: %s", stdout)
				}
			case "jq":
				if strings.TrimSpace(stdout) != `"MERGEABLE"` {
					t.Fatal(stdout)
				}
			}
		})
	}
}

func TestPRMergeabilityRejectsDataCenterAndInvalidIDs(t *testing.T) {
	for _, args := range [][]string{{"pr", "mergeability", "42"}, {"pr", "mergeability", "0"}, {"pr", "mergeability", "nope"}} {
		_, _, err := runCLI(t, dcConfig("http://localhost"), args...)
		if err == nil {
			t.Fatalf("expected error for %v", args)
		}
	}
}

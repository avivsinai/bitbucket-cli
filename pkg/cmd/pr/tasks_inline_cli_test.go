package pr_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
)

func TestPRTaskCreateInlineValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(server.Close)

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"from without file", []string{"--from-line", "10"}, "--file is required"},
		{"to without file", []string{"--to-line", "10"}, "--file is required"},
		{"blank file", []string{"--file", "  ", "--to-line", "10"}, "--file value must not be blank"},
		{"file without line", []string{"--file", "test.go"}, "--file must be used with either"},
		{"both sides", []string{"--file", "test.go", "--from-line", "10", "--to-line", "11"}, "mutually exclusive"},
		{"zero from", []string{"--file", "test.go", "--from-line", "0"}, "--from-line must be a positive integer"},
		{"negative to", []string{"--file", "test.go", "--to-line", "-1"}, "--to-line must be a positive integer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append([]string{"pr", "task", "create", "42", "--text", "fixture"}, tt.args...)
			_, _, err := runCLI(t, dcConfig(server.URL), args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}

func TestPRTaskCreateInlineRequests(t *testing.T) {
	const (
		dcTasks       = "/rest/api/1.0/projects/PROJ/repos/my-repo/pull-requests/42/blocker-comments"
		cloudComments = "/repositories/myworkspace/my-repo/pullrequests/42/comments"
		cloudTasks    = "/repositories/myworkspace/my-repo/pullrequests/42/tasks"
		dcResponse    = `{"id":77,"text":"fixture","state":"OPEN"}`
		cloudResponse = `{"id":77,"content":{"raw":"fixture"},"state":"UNRESOLVED"}`
		commentID     = `{"id":19}`
		linkedTask    = `{"content":{"raw":"fixture"},"comment":{"id":19}}`
		inlineTo      = `{"content":{"raw":"fixture"},"inline":{"path":"test.go","to":11}}`
		inlineFrom    = `{"content":{"raw":"fixture"},"inline":{"path":"test.go","from":10}}`
	)
	type write struct {
		path   string
		body   string
		status int
		reply  string
	}
	tests := []struct {
		name      string
		config    func(string) *config.Config
		flags     []string
		writes    []write
		wantError string
	}{
		{
			name: "dc destination line", config: dcConfig,
			flags:  []string{"--file", " test.go ", "--to-line", "11"},
			writes: []write{{dcTasks, `{"text":"fixture","anchor":{"path":"test.go","line":11,"lineType":"ADDED","fileType":"TO"}}`, 201, dcResponse}},
		},
		{
			name: "dc source line", config: dcConfig,
			flags:  []string{"--file", "test.go", "--from-line", "10"},
			writes: []write{{dcTasks, `{"text":"fixture","anchor":{"path":"test.go","line":10,"lineType":"REMOVED","fileType":"FROM"}}`, 201, dcResponse}},
		},
		{
			name: "dc standalone", config: dcConfig,
			writes: []write{{dcTasks, `{"text":"fixture"}`, 201, dcResponse}},
		},
		{
			name: "cloud destination line", config: cloudConfig,
			flags:  []string{"--file", "test.go", "--to-line", "11"},
			writes: []write{{cloudComments, inlineTo, 201, commentID}, {cloudTasks, linkedTask, 201, cloudResponse}},
		},
		{
			name: "cloud source line", config: cloudConfig,
			flags:  []string{"--file", "test.go", "--from-line", "10"},
			writes: []write{{cloudComments, inlineFrom, 201, commentID}, {cloudTasks, linkedTask, 201, cloudResponse}},
		},
		{
			name: "cloud standalone", config: cloudConfig,
			writes: []write{{cloudTasks, `{"content":{"raw":"fixture"}}`, 201, cloudResponse}},
		},
		{
			name: "cloud preserves comment on task error", config: cloudConfig,
			flags:     []string{"--file", "test.go", "--to-line", "11"},
			writes:    []write{{cloudComments, inlineTo, 201, commentID}, {cloudTasks, linkedTask, 500, `{"error":{"message":"fixture task error"}}`}},
			wantError: "created inline comment 19 on pull request #42. Could not confirm task creation",
		},
		{
			name: "cloud does not retry a rate limited task", config: cloudConfig,
			flags:     []string{"--file", "test.go", "--to-line", "11"},
			writes:    []write{{cloudComments, inlineTo, 201, commentID}, {cloudTasks, linkedTask, 429, `{"error":{"message":"fixture rate limit"}}`}},
			wantError: "created inline comment 19 on pull request #42. Could not confirm task creation",
		},
		{
			name: "cloud does not send task without comment id", config: cloudConfig,
			flags:     []string{"--file", "test.go", "--to-line", "11"},
			writes:    []write{{cloudComments, inlineTo, 201, `{}`}},
			wantError: "without a comment ID in the response. No task request was sent",
		},
		{
			name: "cloud does not send task after comment error", config: cloudConfig,
			flags:     []string{"--file", "test.go", "--to-line", "11"},
			writes:    []write{{cloudComments, inlineTo, 403, `{"error":{"message":"fixture comment error"}}`}},
			wantError: "could not confirm inline comment creation. No task request was sent",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hits atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				index := int(hits.Add(1)) - 1
				if index >= len(tt.writes) {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				want := tt.writes[index]
				if r.Method != http.MethodPost || r.URL.Path != want.path {
					t.Errorf("request %d = %s %s, want POST %s", index, r.Method, r.URL.Path, want.path)
				}
				var gotBody, wantBody any
				if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
					t.Errorf("decode request: %v", err)
				}
				if err := json.Unmarshal([]byte(want.body), &wantBody); err != nil {
					t.Errorf("decode fixture: %v", err)
				}
				if !reflect.DeepEqual(gotBody, wantBody) {
					t.Errorf("request %d body = %#v, want %#v", index, gotBody, wantBody)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(want.status)
				_, _ = fmt.Fprint(w, want.reply)
			}))
			t.Cleanup(server.Close)

			args := append([]string{"--json", "pr", "task", "create", "42", "--text", "fixture"}, tt.flags...)
			stdout, stderr, err := runCLI(t, tt.config(server.URL), args...)
			if int(hits.Load()) != len(tt.writes) {
				t.Errorf("sent %d requests, want %d", hits.Load(), len(tt.writes))
			}
			if tt.wantError != "" {
				if err == nil || !strings.Contains(stderr+err.Error(), tt.wantError) || stdout != "" {
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

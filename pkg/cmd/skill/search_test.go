package skill

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/bbcloud"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
)

func TestSkillSearchUsesContextWorkspace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workspaces/myteam/search/code" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("search_query"); got != "(code review) path:SKILL.md" {
			t.Errorf("search_query = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"values":[{"content_match_count":2,"file":{"path":"skills/review/SKILL.md","links":{"self":{"href":"https://api.bitbucket.org/file"}},"commit":{"repository":{"full_name":"myteam/agent-skills"}}}}]}`))
	}))
	t.Cleanup(server.Close)
	f := factoryWithHost(t, "cloud", server.URL, "bitbucket.org")
	f.NewCloudClientFunc = func(*config.Host) (*bbcloud.Client, error) {
		return bbcloud.New(bbcloud.Options{BaseURL: server.URL})
	}
	stdout := f.IOStreams.Out.(*strings.Builder)
	stderr := f.IOStreams.ErrOut.(*strings.Builder)

	if err := executeSearch(t, f, "skill", "search", "code review"); err != nil {
		t.Fatalf("search: %v (stderr=%s)", err, stderr)
	}
	if got := stdout.String(); got != "myteam/agent-skills\tskills/review/SKILL.md\t2 match(es)\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestSkillSearchJSONHasNoHumanSuffix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"values":[{"content_match_count":1,"file":{"path":"SKILL.md","commit":{"repository":{"full_name":"team/root-skill"}}}}]}`))
	}))
	t.Cleanup(server.Close)
	f := factoryWithHost(t, "cloud", server.URL, "bitbucket.org")
	f.NewCloudClientFunc = func(*config.Host) (*bbcloud.Client, error) {
		return bbcloud.New(bbcloud.Options{BaseURL: server.URL})
	}
	stdout := f.IOStreams.Out.(*strings.Builder)

	if err := executeSearch(t, f, "skill", "search", "SKILL.md", "--workspace", "other", "--json"); err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Workspace string `json:"workspace"`
		Query     string `json:"query"`
		Results   []struct {
			Repository string `json:"repository"`
			Path       string `json:"path"`
			Matches    int    `json:"matches"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(stdout.String()), &payload); err != nil {
		t.Fatalf("JSON output contains non-JSON text: %v\n%s", err, stdout)
	}
	if payload.Workspace != "other" || payload.Query != "SKILL.md" || len(payload.Results) != 1 {
		t.Fatalf("payload = %+v", payload)
	}
	if got := payload.Results[0]; got.Repository != "team/root-skill" || got.Path != "SKILL.md" || got.Matches != 1 {
		t.Fatalf("result = %+v", got)
	}
}

func TestSkillSearchRejectsDataCenter(t *testing.T) {
	f := factoryWithHost(t, "dc", "https://bitbucket.example.com", "bitbucket.example.com")
	err := executeSearch(t, f, "skill", "search", "SKILL.md")
	if err == nil || !strings.Contains(err.Error(), "not supported for Bitbucket Data Center") {
		t.Fatalf("error = %v", err)
	}
}

func TestSkillSearchValidatesLimitBeforeRequest(t *testing.T) {
	f := factoryWithHost(t, "cloud", "https://api.bitbucket.org/2.0", "bitbucket.org")
	err := executeSearch(t, f, "skill", "search", "SKILL.md", "--limit", "0")
	if err == nil || !strings.Contains(err.Error(), "--limit must be at least 1") {
		t.Fatalf("error = %v", err)
	}
}

func TestSkillSearchRejectsExplicitBlankWorkspace(t *testing.T) {
	f := factoryWithHost(t, "cloud", "https://api.bitbucket.org/2.0", "bitbucket.org")
	err := executeSearch(t, f, "skill", "search", "review", "--workspace", "   ")
	if err == nil || !strings.Contains(err.Error(), "--workspace cannot be blank") {
		t.Fatalf("error = %v", err)
	}
}

func TestSkillSearchNoResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"values":[]}`))
	}))
	t.Cleanup(server.Close)
	f := factoryWithHost(t, "cloud", server.URL, "bitbucket.org")
	f.NewCloudClientFunc = func(*config.Host) (*bbcloud.Client, error) {
		return bbcloud.New(bbcloud.Options{BaseURL: server.URL})
	}
	stdout := f.IOStreams.Out.(*strings.Builder)
	if err := executeSearch(t, f, "skill", "search", "missing"); err != nil {
		t.Fatal(err)
	}
	if got := stdout.String(); got != "No skills found in workspace myteam.\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestSkillSearchReportsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	f := factoryWithHost(t, "cloud", server.URL, "bitbucket.org")
	f.NewCloudClientFunc = func(*config.Host) (*bbcloud.Client, error) {
		return bbcloud.New(bbcloud.Options{BaseURL: server.URL})
	}
	err := executeSearch(t, f, "skill", "search", "review")
	if err == nil || !strings.Contains(err.Error(), `search workspace "myteam"`) || !strings.Contains(err.Error(), "503") {
		t.Fatalf("error = %v", err)
	}
}

func TestSkillSearchSanitizesHumanOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{\"values\":[{\"content_match_count\":1,\"file\":{\"path\":\"skills/evil\\u001b[31m\\n/SKILL.md\",\"commit\":{\"repository\":{\"full_name\":\"team\\t/evil\\u0007\"}}}}]}"))
	}))
	t.Cleanup(server.Close)
	f := factoryWithHost(t, "cloud", server.URL, "bitbucket.org")
	f.NewCloudClientFunc = func(*config.Host) (*bbcloud.Client, error) {
		return bbcloud.New(bbcloud.Options{BaseURL: server.URL})
	}
	stdout := f.IOStreams.Out.(*strings.Builder)
	if err := executeSearch(t, f, "skill", "search", "evil"); err != nil {
		t.Fatal(err)
	}
	got := stdout.String()
	if strings.Contains(got, "\x1b") || strings.Contains(got, "\a") || strings.Count(got, "\n") != 1 {
		t.Fatalf("unsafe terminal output = %q", got)
	}
	if got != "team /evil\tskills/evil[31m /SKILL.md\t1 match(es)\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func executeSearch(t *testing.T, factory *cmdutil.Factory, args ...string) error {
	t.Helper()
	root := &cobra.Command{Use: "bkt", SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().StringP("context", "c", "", "")
	root.PersistentFlags().Bool("json", false, "")
	root.PersistentFlags().Bool("yaml", false, "")
	root.PersistentFlags().String("format", "", "")
	root.PersistentFlags().String("jq", "", "")
	root.PersistentFlags().String("template", "", "")
	root.AddCommand(NewCmdSkill(factory))
	root.SetArgs(args)
	root.SetContext(context.Background())
	return root.Execute()
}

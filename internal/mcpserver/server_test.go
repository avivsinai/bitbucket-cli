package mcpserver

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

func testFactory(cfg *config.Config) *cmdutil.Factory {
	return &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      iostreams.System(),
		Config:         func() (*config.Config, error) { return cfg, nil },
	}
}

func dcConfig() *config.Config {
	return &config.Config{
		ActiveContext: "work",
		Contexts: map[string]*config.Context{
			"work": {Host: "dc-host", ProjectKey: "PROJ"},
		},
		Hosts: map[string]*config.Host{
			"dc-host": {Kind: "dc", BaseURL: "https://bitbucket.example.com", Username: "u", Token: "t"},
		},
	}
}

// The server resolver must never derive defaults from the working
// directory's git remotes: spawn cwd of an MCP server is not a stable
// target. This runs inside a git repo whose origin points at the configured
// host, which the CLI resolver would use to fill the default repo.
func TestResolveSnapshotIgnoresWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q"},
		{"remote", "add", "origin", "https://bitbucket.example.com/scm/proj/sniffed-repo.git"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	t.Chdir(dir)

	snap, err := ResolveSnapshot(testFactory(dcConfig()), "")
	if err != nil {
		t.Fatalf("ResolveSnapshot: %v", err)
	}
	if snap.DefaultRepo != "" {
		t.Fatalf("DefaultRepo = %q — leaked from cwd git remote; want empty (config has none)", snap.DefaultRepo)
	}
	if snap.Platform != "dc" || snap.DefaultScope != "PROJ" || snap.HostLabel != "dc-host" {
		t.Fatalf("snapshot = %+v, want dc/PROJ/dc-host from config only", snap)
	}
	if snap.ContextName != "work" {
		t.Fatalf("ContextName = %q, want work", snap.ContextName)
	}
}

func TestResolveSnapshotCloudUsesWorkspaceScope(t *testing.T) {
	cfg := &config.Config{
		ActiveContext: "cl",
		Contexts: map[string]*config.Context{
			"cl": {Host: "cloud-host", Workspace: "myteam", DefaultRepo: "api"},
		},
		Hosts: map[string]*config.Host{
			"cloud-host": {Kind: "cloud", BaseURL: "https://api.bitbucket.org/2.0", Username: "u", Token: "t"},
		},
	}
	snap, err := ResolveSnapshot(testFactory(cfg), "")
	if err != nil {
		t.Fatalf("ResolveSnapshot: %v", err)
	}
	if snap.Platform != "cloud" || snap.DefaultScope != "myteam" || snap.DefaultRepo != "api" {
		t.Fatalf("snapshot = %+v, want cloud/myteam/api", snap)
	}
}

// connectPair wires the server to an in-memory client and returns the client
// session for making calls.
func connectPair(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	serverT, clientT := mcp.NewInMemoryTransports()

	if _, err := server.Connect(context.Background(), serverT, nil); err != nil {
		t.Fatalf("server.Connect: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "0"}, nil)
	session, err := client.Connect(context.Background(), clientT, nil)
	if err != nil {
		t.Fatalf("client.Connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestGetContextToolRoundTrip(t *testing.T) {
	snap := &Snapshot{
		ContextName:  "work",
		Platform:     "dc",
		HostLabel:    "dc-host",
		DefaultScope: "PROJ",
		DefaultRepo:  "api",
	}
	session := connectPair(t, New(snap, "test"))

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(tools.Tools) != 1 || tools.Tools[0].Name != "bkt_get_context" {
		t.Fatalf("tools = %+v, want exactly bkt_get_context", tools.Tools)
	}
	if tools.Tools[0].OutputSchema == nil {
		t.Fatal("bkt_get_context has no output schema; typed contract missing")
	}

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "bkt_get_context"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool errored: %+v", res.Content)
	}

	raw, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured content: %v", err)
	}
	var info ContextInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		t.Fatalf("structured content does not match ContextInfo: %v\n%s", err, raw)
	}
	if info.Platform != "dc" || info.HostLabel != "dc-host" || info.DefaultScope != "PROJ" || info.DefaultRepo != "api" {
		t.Fatalf("ContextInfo = %+v, want snapshot values", info)
	}
	if len(info.Capabilities) == 0 {
		t.Fatal("capabilities must not be empty")
	}
	for _, s := range []string{"token", "Token", "secret"} {
		if string(raw) != "" && json.Valid(raw) && containsInsensitive(raw, s) {
			t.Fatalf("structured content leaks credential-looking field %q: %s", s, raw)
		}
	}
}

func containsInsensitive(b []byte, sub string) bool {
	// cheap scan; both inputs are short
	haystack := string(b)
	for i := 0; i+len(sub) <= len(haystack); i++ {
		match := true
		for j := 0; j < len(sub); j++ {
			c1, c2 := haystack[i+j], sub[j]
			if c1|0x20 != c2|0x20 {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

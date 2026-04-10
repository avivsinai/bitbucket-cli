package branch

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/avivsinai/bitbucket-cli/internal/config"
	"github.com/avivsinai/bitbucket-cli/pkg/cmdutil"
	"github.com/avivsinai/bitbucket-cli/pkg/iostreams"
)

// newTestFactory returns a factory wired to the given config plus buffers for
// stdout/stderr. The returned context command is attached to a root command
// that declares a `--context` persistent flag, which cmdutil.ResolveContext
// looks up via FlagValue.
func newTestFactory(cfg *config.Config) (*cmdutil.Factory, *bytes.Buffer, *bytes.Buffer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	ios := &iostreams.IOStreams{
		In:     io.NopCloser(bytes.NewReader(nil)),
		Out:    stdout,
		ErrOut: stderr,
	}

	f := &cmdutil.Factory{
		AppVersion:     "test",
		ExecutableName: "bkt",
		IOStreams:      ios,
		Config: func() (*config.Config, error) {
			return cfg, nil
		},
	}
	return f, stdout, stderr
}

// runBranchCmd mounts NewCmdBranch under a root with a --context flag and
// executes it with the provided args. This mirrors how the real CLI wires
// the persistent flag that ResolveContext reads via FlagValue.
func runBranchCmd(t *testing.T, f *cmdutil.Factory, args ...string) error {
	t.Helper()

	cmd := NewCmdBranch(f)
	cmd.PersistentFlags().String("context", "", "Named context to use")
	cmd.PersistentFlags().String("output", "text", "Output format")
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true

	cmd.SetArgs(args)
	cmd.SetOut(f.IOStreams.Out)
	cmd.SetErr(f.IOStreams.ErrOut)

	return cmd.ExecuteContext(context.Background())
}

func dcConfig(baseURL string) *config.Config {
	return &config.Config{
		ActiveContext: "test",
		Contexts: map[string]*config.Context{
			"test": {
				Host:        "mock",
				ProjectKey:  "PROJ",
				DefaultRepo: "my-repo",
			},
		},
		Hosts: map[string]*config.Host{
			"mock": {
				Kind:     "dc",
				BaseURL:  baseURL,
				Username: "admin",
				Token:    "token",
			},
		},
	}
}

func cloudConfig(baseURL string) *config.Config {
	return &config.Config{
		ActiveContext: "test",
		Contexts: map[string]*config.Context{
			"test": {
				Host:        "mock",
				Workspace:   "myworkspace",
				DefaultRepo: "my-repo",
			},
		},
		Hosts: map[string]*config.Host{
			"mock": {
				Kind:     "cloud",
				BaseURL:  baseURL,
				Username: "admin",
				Token:    "token",
			},
		},
	}
}

// ---------- list ----------

func TestBranchListDC(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/projects/PROJ/repos/my-repo/branches") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"size":       2,
			"limit":      25,
			"isLastPage": true,
			"start":      0,
			"values": []map[string]any{
				{"id": "refs/heads/main", "displayId": "main", "latestCommit": "abc123", "isDefault": true},
				{"id": "refs/heads/feature/login", "displayId": "feature/login", "latestCommit": "def456", "isDefault": false},
			},
		})
	}))
	t.Cleanup(srv.Close)

	f, stdout, stderr := newTestFactory(dcConfig(srv.URL))
	if err := runBranchCmd(t, f, "list"); err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "main") || !strings.Contains(out, "abc123") {
		t.Errorf("expected main branch in output, got: %s", out)
	}
	if !strings.Contains(out, "feature/login") {
		t.Errorf("expected feature/login in output, got: %s", out)
	}
	// Default marker precedes the default branch.
	if !strings.Contains(out, "* main") {
		t.Errorf("expected '* main' marker, got: %s", out)
	}
}

func TestBranchListDCEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"size":       0,
			"limit":      25,
			"isLastPage": true,
			"start":      0,
			"values":     []any{},
		})
	}))
	t.Cleanup(srv.Close)

	f, stdout, stderr := newTestFactory(dcConfig(srv.URL))
	if err := runBranchCmd(t, f, "list"); err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "No branches found") {
		t.Errorf("expected empty message, got: %s", stdout.String())
	}
}

func TestBranchListDCMissingContextFields(t *testing.T) {
	cfg := dcConfig("http://localhost")
	cfg.Contexts["test"].ProjectKey = ""
	cfg.Contexts["test"].DefaultRepo = ""

	f, _, _ := newTestFactory(cfg)
	err := runBranchCmd(t, f, "list")
	if err == nil {
		t.Fatal("expected error when project/repo unset")
	}
	if !strings.Contains(err.Error(), "project and repo") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBranchListCloud(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/repositories/myworkspace/my-repo/refs/branches") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{
					"name":    "main",
					"default": true,
					"target":  map[string]any{"hash": "0123456789abcdef0000", "type": "commit"},
				},
				{
					"name":    "develop",
					"default": false,
					"target":  map[string]any{"hash": "fedcba9876543210", "type": "commit"},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	f, stdout, stderr := newTestFactory(cloudConfig(srv.URL))
	if err := runBranchCmd(t, f, "list"); err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "* main") {
		t.Errorf("expected '* main' marker, got: %s", out)
	}
	if !strings.Contains(out, "develop") {
		t.Errorf("expected develop branch, got: %s", out)
	}
	// Cloud formatter truncates hashes to 12 chars.
	if !strings.Contains(out, "0123456789ab") {
		t.Errorf("expected 12-char truncated hash, got: %s", out)
	}
	if strings.Contains(out, "0123456789abcdef0000") {
		t.Errorf("hash should be truncated to 12 chars, got: %s", out)
	}
}

func TestBranchListCloudMissingContextFields(t *testing.T) {
	cfg := cloudConfig("http://localhost")
	cfg.Contexts["test"].Workspace = ""
	cfg.Contexts["test"].DefaultRepo = ""

	f, _, _ := newTestFactory(cfg)
	err := runBranchCmd(t, f, "list")
	if err == nil {
		t.Fatal("expected error when workspace/repo unset")
	}
	if !strings.Contains(err.Error(), "workspace and repo") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------- create ----------

func TestBranchCreateDC(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.Contains(r.URL.Path, "/projects/PROJ/repos/my-repo/branches") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body["name"] != "refs/heads/feature/login" {
			t.Errorf("expected name to be ensureRef'd, got %v", body["name"])
		}
		if body["startPoint"] != "refs/heads/main" {
			t.Errorf("expected startPoint to be ensureRef'd, got %v", body["startPoint"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":           "refs/heads/feature/login",
			"displayId":    "feature/login",
			"latestCommit": "abc123",
		})
	}))
	t.Cleanup(srv.Close)

	f, stdout, stderr := newTestFactory(dcConfig(srv.URL))
	if err := runBranchCmd(t, f, "create", "feature/login", "--from", "main"); err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Created branch feature/login") {
		t.Errorf("expected success message, got: %s", stdout.String())
	}
}

func TestBranchCreateRejectsCloud(t *testing.T) {
	f, _, _ := newTestFactory(cloudConfig("http://localhost"))
	err := runBranchCmd(t, f, "create", "feature/x", "--from", "main")
	if err == nil {
		t.Fatal("expected error when creating branch on Cloud")
	}
	if !strings.Contains(err.Error(), "Data Center") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBranchCreateRequiresFromFlag(t *testing.T) {
	f, _, _ := newTestFactory(dcConfig("http://localhost"))
	err := runBranchCmd(t, f, "create", "feature/x")
	if err == nil {
		t.Fatal("expected error when --from is missing")
	}
	if !strings.Contains(err.Error(), "from") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------- delete ----------

func TestBranchDeleteDC(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || !strings.Contains(r.URL.Path, "/projects/PROJ/repos/my-repo/branches") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body["name"] != "refs/heads/feature/old" {
			t.Errorf("expected name ensureRef'd, got %v", body["name"])
		}
		if body["dryRun"] != false {
			t.Errorf("expected dryRun false, got %v", body["dryRun"])
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	f, stdout, stderr := newTestFactory(dcConfig(srv.URL))
	if err := runBranchCmd(t, f, "delete", "feature/old"); err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "Deleted branch feature/old") {
		t.Errorf("expected success message, got: %s", out)
	}
}

func TestBranchDeleteDCDryRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["dryRun"] != true {
			t.Errorf("expected dryRun true, got %v", body["dryRun"])
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	f, stdout, _ := newTestFactory(dcConfig(srv.URL))
	if err := runBranchCmd(t, f, "delete", "feature/old", "--dry-run"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Validated branch feature/old") {
		t.Errorf("expected 'Validated' dry-run message, got: %s", stdout.String())
	}
}

func TestBranchDeleteRejectsCloud(t *testing.T) {
	f, _, _ := newTestFactory(cloudConfig("http://localhost"))
	err := runBranchCmd(t, f, "delete", "feature/x")
	if err == nil {
		t.Fatal("expected error when deleting branch on Cloud")
	}
	if !strings.Contains(err.Error(), "Data Center") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------- set-default ----------

func TestBranchSetDefaultDC(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || !strings.Contains(r.URL.Path, "/settings/default-branch") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body["id"] != "refs/heads/develop" {
			t.Errorf("expected id ensureRef'd, got %v", body["id"])
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	f, stdout, stderr := newTestFactory(dcConfig(srv.URL))
	if err := runBranchCmd(t, f, "set-default", "develop"); err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Set default branch to develop") {
		t.Errorf("expected success message, got: %s", stdout.String())
	}
}

func TestBranchSetDefaultRejectsCloud(t *testing.T) {
	f, _, _ := newTestFactory(cloudConfig("http://localhost"))
	err := runBranchCmd(t, f, "set-default", "develop")
	if err == nil {
		t.Fatal("expected error when set-default on Cloud")
	}
	if !strings.Contains(err.Error(), "Data Center") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------- protect ----------

func TestBranchProtectList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "/restrictions") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"values": []map[string]any{
				{
					"id":   42,
					"type": "NO_DELETES",
					"matcher": map[string]any{
						"id":        "refs/heads/main",
						"displayId": "main",
						"type":      map[string]any{"id": "BRANCH"},
					},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	f, stdout, stderr := newTestFactory(dcConfig(srv.URL))
	if err := runBranchCmd(t, f, "protect", "list"); err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "42") || !strings.Contains(out, "NO_DELETES") || !strings.Contains(out, "main") {
		t.Errorf("expected restriction row in output, got: %s", out)
	}
}

func TestBranchProtectListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"values": []any{}})
	}))
	t.Cleanup(srv.Close)

	f, stdout, _ := newTestFactory(dcConfig(srv.URL))
	if err := runBranchCmd(t, f, "protect", "list"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "No branch restrictions") {
		t.Errorf("expected empty message, got: %s", stdout.String())
	}
}

func TestBranchProtectAdd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.Contains(r.URL.Path, "/restrictions") {
			http.NotFound(w, r)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		typeObj, _ := body["type"].(map[string]any)
		if typeObj["id"] != "FAST_FORWARD_ONLY" {
			t.Errorf("expected FAST_FORWARD_ONLY type, got %v", typeObj["id"])
		}
		matcher, _ := body["matcher"].(map[string]any)
		if matcher["id"] != "refs/heads/main" {
			t.Errorf("expected matcher id to be ensureBranchRef'd, got %v", matcher["id"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   99,
			"type": "FAST_FORWARD_ONLY",
			"matcher": map[string]any{
				"id":        "refs/heads/main",
				"displayId": "main",
				"type":      map[string]any{"id": "BRANCH"},
			},
		})
	}))
	t.Cleanup(srv.Close)

	f, stdout, stderr := newTestFactory(dcConfig(srv.URL))
	if err := runBranchCmd(t, f, "protect", "add", "main", "--type", "fast-forward-only"); err != nil {
		t.Fatalf("unexpected error: %v (stderr=%s)", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Added restriction 99") {
		t.Errorf("expected success message, got: %s", stdout.String())
	}
}

func TestBranchProtectAddRejectsBadType(t *testing.T) {
	f, _, _ := newTestFactory(dcConfig("http://localhost"))
	err := runBranchCmd(t, f, "protect", "add", "main", "--type", "bogus")
	if err == nil {
		t.Fatal("expected error for bad restriction type")
	}
	if !strings.Contains(err.Error(), "unsupported restriction type") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBranchProtectRemove(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || !strings.Contains(r.URL.Path, "/restrictions/42") {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	f, stdout, _ := newTestFactory(dcConfig(srv.URL))
	if err := runBranchCmd(t, f, "protect", "remove", "42"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Removed restriction 42") {
		t.Errorf("expected success message, got: %s", stdout.String())
	}
}

func TestBranchProtectRemoveRejectsBadID(t *testing.T) {
	f, _, _ := newTestFactory(dcConfig("http://localhost"))
	err := runBranchCmd(t, f, "protect", "remove", "not-a-number")
	if err == nil {
		t.Fatal("expected error for non-numeric id")
	}
	if !strings.Contains(err.Error(), "invalid restriction id") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBranchProtectRejectsCloud(t *testing.T) {
	for _, sub := range [][]string{
		{"protect", "list"},
		{"protect", "add", "main", "--type", "no-creates"},
		{"protect", "remove", "1"},
	} {
		t.Run(strings.Join(sub, " "), func(t *testing.T) {
			f, _, _ := newTestFactory(cloudConfig("http://localhost"))
			err := runBranchCmd(t, f, sub...)
			if err == nil {
				t.Fatal("expected error on Cloud")
			}
			if !strings.Contains(err.Error(), "Data Center") {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

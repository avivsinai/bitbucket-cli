package secret

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/99designs/keyring"
)

func TestBuildConfig_DarwinTrustFlags(t *testing.T) {
	cfg, err := buildConfig()
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}

	if runtime.GOOS == "darwin" {
		if !cfg.KeychainTrustApplication {
			t.Errorf("KeychainTrustApplication should be true on darwin")
		}
		if !cfg.KeychainAccessibleWhenUnlocked {
			t.Errorf("KeychainAccessibleWhenUnlocked should be true on darwin")
		}
	} else {
		if cfg.KeychainTrustApplication {
			t.Errorf("KeychainTrustApplication should be false on %s", runtime.GOOS)
		}
		if cfg.KeychainAccessibleWhenUnlocked {
			t.Errorf("KeychainAccessibleWhenUnlocked should be false on %s", runtime.GOOS)
		}
	}

	if cfg.ServiceName != serviceName {
		t.Errorf("ServiceName = %q, want %q", cfg.ServiceName, serviceName)
	}
}

func TestParseTimeoutEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want time.Duration
		ok   bool
	}{
		{name: "empty", raw: "", want: 0, ok: false},
		{name: "duration_seconds", raw: "60s", want: 60 * time.Second, ok: true},
		{name: "duration_minutes", raw: "2m", want: 2 * time.Minute, ok: true},
		{name: "plain_seconds", raw: "60", want: 60 * time.Second, ok: true},
		{name: "zero", raw: "0", want: 0, ok: false},
		{name: "negative", raw: "-1", want: 0, ok: false},
		{name: "garbage", raw: "nope", want: 0, ok: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseTimeoutEnv(tt.raw)
			if ok != tt.ok {
				t.Fatalf("ok=%v want %v (got=%v)", ok, tt.ok, got)
			}
			if ok && got != tt.want {
				t.Fatalf("got=%v want %v", got, tt.want)
			}
		})
	}
}

func TestKeyringTimeout_EnvOverride(t *testing.T) {
	t.Setenv(envTimeout, "2m")
	if got := keyringTimeout(); got != 2*time.Minute {
		t.Fatalf("got=%v want %v", got, 2*time.Minute)
	}
}

func TestTokenFromEnv(t *testing.T) {
	tests := []struct {
		name string
		env  string
		want string
	}{
		{name: "unset", env: "", want: ""},
		{name: "set", env: "my-token", want: "my-token"},
		{name: "trimmed", env: "  spaced-token  ", want: "spaced-token"},
		{name: "whitespace_only", env: "   ", want: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(EnvToken, tt.env)
			if got := TokenFromEnv(); got != tt.want {
				t.Fatalf("got=%q want %q", got, tt.want)
			}
		})
	}
}

func TestConfigureFileBackend_HeadlessNoPassphrase(t *testing.T) {
	// Simulate headless: clear DISPLAY/DBUS, set SSH_TTY.
	t.Setenv("SSH_TTY", "/dev/pts/0")
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")
	t.Setenv("CI", "")
	// Ensure no passphrase env vars are set.
	t.Setenv(envPassphrase, "")
	t.Setenv("KEYRING_FILE_PASSWORD", "")
	t.Setenv("KEYRING_PASSWORD", "")

	cfg := &keyring.Config{ServiceName: serviceName}
	opts := openOptions{allowFile: true}

	err := configureFileBackend(cfg, opts)
	if err == nil {
		t.Fatal("expected error for headless env without passphrase")
	}
	if !strings.Contains(err.Error(), envPassphrase) {
		t.Fatalf("error should mention %s, got: %v", envPassphrase, err)
	}
	if !strings.Contains(err.Error(), EnvToken) {
		t.Fatalf("error should mention %s, got: %v", EnvToken, err)
	}
}

func TestDefaultBackends_WindowsIsWinCredOnly(t *testing.T) {
	t.Parallel()

	got := defaultBackendsFor("windows", false)
	if len(got) != 1 || got[0] != keyring.WinCredBackend {
		t.Fatalf("windows default backends = %v, want [wincred]", got)
	}
	if usesFileBackend(got) {
		t.Fatal("windows default path must not include the file backend")
	}
	if chunkSizeForBackends(got) != winCredMaxBlobSize {
		t.Fatalf("windows default must enable WinCred chunking, got %d", chunkSizeForBackends(got))
	}

	headless := defaultBackendsFor("windows", true)
	if len(headless) != 1 || headless[0] != keyring.WinCredBackend {
		t.Fatalf("windows headless backends = %v, want [wincred]", headless)
	}
}

func TestResolveAllowedBackends_DefaultExcludesFile(t *testing.T) {
	t.Setenv(envBackend, "")
	t.Setenv(envAllowInsecure, "")

	got := resolveAllowedBackends(openOptions{})
	if usesFileBackend(got) {
		t.Fatal("default path must not enable the file backend without opt-in")
	}
	if chunkSizeForBackends(got) != chunkSizeForBackends(defaultBackends()) {
		t.Fatal("default chunk limit should match platform native backends")
	}
}

func TestResolveAllowedBackends_AllowFileDoesNotReplaceNative(t *testing.T) {
	t.Setenv(envBackend, "")
	t.Setenv(envAllowInsecure, "")

	got := resolveAllowedBackends(openOptions{allowFile: true})
	native := defaultBackends()
	if len(got) != len(native)+1 {
		t.Fatalf("allowFile should append file, got %v (native %v)", got, native)
	}
	for i, backend := range native {
		if got[i] != backend {
			t.Fatalf("allowFile reordered native backends: got %v, native %v", got, native)
		}
	}
	if got[len(got)-1] != keyring.FileBackend {
		t.Fatalf("file should be last, got %v", got)
	}
	if runtime.GOOS == "windows" && got[0] != keyring.WinCredBackend {
		t.Fatal("windows must keep WinCred first; file is not a silent size-error fallback")
	}
}

func TestChunkSizeForBackends(t *testing.T) {
	t.Parallel()

	if got := chunkSizeForBackends([]keyring.BackendType{keyring.WinCredBackend}); got != winCredMaxBlobSize {
		t.Fatalf("wincred = %d, want %d", got, winCredMaxBlobSize)
	}
	if got := chunkSizeForBackends([]keyring.BackendType{keyring.WinCredBackend, keyring.FileBackend}); got != winCredMaxBlobSize {
		t.Fatalf("wincred+file = %d, want %d (file must not disable chunking)", got, winCredMaxBlobSize)
	}
	if got := chunkSizeForBackends([]keyring.BackendType{keyring.FileBackend}); got != 0 {
		t.Fatalf("file-only = %d, want 0", got)
	}
	if got := chunkSizeForBackends(defaultBackendsFor("linux", false)); got != 0 {
		t.Fatalf("linux default = %d, want 0", got)
	}
	if got := chunkSizeForBackends(defaultBackendsFor("darwin", false)); got != 0 {
		t.Fatalf("darwin default = %d, want 0", got)
	}
}

func TestBuildConfig_DefaultDoesNotRequireInsecureEnv(t *testing.T) {
	t.Setenv(envBackend, "")
	t.Setenv(envAllowInsecure, "")
	t.Setenv(envPassphrase, "")
	t.Setenv(envFileDir, "")

	cfg, err := buildConfig()
	if err != nil {
		t.Fatalf("buildConfig: %v", err)
	}
	if usesFileBackend(cfg.AllowedBackends) {
		t.Fatal("default buildConfig must not enable the file backend")
	}
}

func TestOpen_FileBackendDoesNotEnableWinCredChunking(t *testing.T) {
	t.Setenv(envBackend, "file")
	t.Setenv(envAllowInsecure, "1")
	t.Setenv(envPassphrase, "test-pass")
	dir := t.TempDir()
	t.Setenv(envFileDir, dir)

	store, err := Open(WithAllowFileFallback(true), WithPassphrase("test-pass"), WithFileDir(dir))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if store.maxChunkSize != 0 {
		t.Fatalf("file-only Open set maxChunkSize=%d, want 0", store.maxChunkSize)
	}
}

func TestConfigureFileBackend_HeadlessWithPassphrase(t *testing.T) {
	// Simulate headless with passphrase provided.
	t.Setenv("SSH_TTY", "/dev/pts/0")
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("DBUS_SESSION_BUS_ADDRESS", "")

	cfg := &keyring.Config{ServiceName: serviceName}
	opts := openOptions{allowFile: true, passphrase: "test-pass"}

	err := configureFileBackend(cfg, opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.FilePasswordFunc == nil {
		t.Fatal("FilePasswordFunc should be set")
	}
}

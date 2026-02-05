package secret

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/99designs/keyring"
)

const serviceName = "bkt"

const (
	envAllowInsecure = "BKT_ALLOW_INSECURE_STORE"
	envPassphrase    = "BKT_KEYRING_PASSPHRASE"
	envBackend       = "KEYRING_BACKEND"
	envFileDir       = "KEYRING_FILE_DIR"
)

// keyringTimeout is the maximum time to wait for keyring operations.
// This prevents indefinite hangs when GUI-based keyrings (KWallet, gnome-keyring)
// try to show unlock prompts in headless/SSH environments.
const keyringTimeout = 3 * time.Second

// ErrKeyringTimeout indicates a keyring operation timed out.
var ErrKeyringTimeout = errors.New("keyring operation timed out")

// isHeadless returns true if the environment is likely unable to show GUI prompts.
// This specifically targets SSH sessions without X11/Wayland forwarding, where
// GUI-based keyring prompts (KWallet, gnome-keyring) would hang indefinitely.
// Local TTY sessions with D-Bus are NOT considered headless, as gnome-keyring
// can work via D-Bus even without a display.
func isHeadless() bool {
	hasDisplay := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	hasDBus := os.Getenv("DBUS_SESSION_BUS_ADDRESS") != ""

	// SSH session without display forwarding - this is the main hang case
	isSSH := os.Getenv("SSH_TTY") != "" || os.Getenv("SSH_CLIENT") != "" || os.Getenv("SSH_CONNECTION") != ""
	if isSSH && !hasDisplay {
		return true
	}

	// No display AND no D-Bus session (container, cron, systemd service, etc.)
	// If D-Bus is available, keyring may work without GUI prompts
	return !hasDisplay && !hasDBus
}

// Store wraps access to the configured keyring backend.
type Store struct {
	kr keyring.Keyring
}

type openOptions struct {
	allowFile       bool
	passphrase      string
	allowedBackends []keyring.BackendType
	fileDir         string
}

// Option customises how the secret store is opened.
type Option func(*openOptions)

// WithAllowFileFallback permits the encrypted file backend when no native
// keyring backend is available.
func WithAllowFileFallback(enable bool) Option {
	return func(o *openOptions) {
		o.allowFile = enable
	}
}

// WithPassphrase supplies a passphrase for the encrypted file backend.
func WithPassphrase(pass string) Option {
	return func(o *openOptions) {
		if pass != "" {
			o.passphrase = pass
		}
	}
}

// WithFileDir sets the directory for the encrypted file backend.
func WithFileDir(dir string) Option {
	return func(o *openOptions) {
		if dir != "" {
			o.fileDir = dir
		}
	}
}

// Open initialises the keyring-backed secret store.
func Open(opts ...Option) (*Store, error) {
	cfg := keyring.Config{
		ServiceName: serviceName,
	}

	settings := openOptions{}

	if envEnabled(os.Getenv(envAllowInsecure)) {
		settings.allowFile = true
	}
	if pass := strings.TrimSpace(os.Getenv(envPassphrase)); pass != "" {
		settings.passphrase = pass
	}
	if dir := strings.TrimSpace(os.Getenv(envFileDir)); dir != "" {
		settings.fileDir = dir
	}

	for _, opt := range opts {
		opt(&settings)
	}

	cfg.AllowedBackends = resolveAllowedBackends(settings)

	if usesFileBackend(cfg.AllowedBackends) {
		if err := configureFileBackend(&cfg, settings); err != nil {
			return nil, err
		}
	}

	kr, err := openKeyringWithTimeout(cfg)
	if err != nil {
		if errors.Is(err, ErrKeyringTimeout) {
			hint := "keyring prompt may be blocked (headless/SSH environment?)"
			return nil, fmt.Errorf("open keyring: %w; %s. Use --allow-insecure-store or set %s=1", err, hint, envAllowInsecure)
		}
		if errors.Is(err, keyring.ErrNoAvailImpl) && !usesFileBackend(cfg.AllowedBackends) {
			return nil, fmt.Errorf("open keyring: %w (set %s=1 or rerun with --allow-insecure-store to permit encrypted file fallback)", err, envAllowInsecure)
		}
		return nil, fmt.Errorf("open keyring: %w", err)
	}

	return &Store{kr: kr}, nil
}

// openKeyringWithTimeout opens the keyring with a timeout to prevent hangs
// when GUI-based keyrings try to show prompts in headless environments.
func openKeyringWithTimeout(cfg keyring.Config) (keyring.Keyring, error) {
	type result struct {
		kr  keyring.Keyring
		err error
	}

	ch := make(chan result, 1)
	go func() {
		kr, err := keyring.Open(cfg)
		ch <- result{kr, err}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), keyringTimeout)
	defer cancel()

	select {
	case res := <-ch:
		return res.kr, res.err
	case <-ctx.Done():
		return nil, ErrKeyringTimeout
	}
}

// Set writes a secret value.
func (s *Store) Set(key, value string) error {
	if s == nil || s.kr == nil {
		return errors.New("secret store not initialized")
	}

	return s.withTimeout(func() error {
		return s.kr.Set(keyring.Item{
			Key:   key,
			Data:  []byte(value),
			Label: fmt.Sprintf("bkt %s", key),
		})
	})
}

// Get retrieves a secret value.
func (s *Store) Get(key string) (string, error) {
	if s == nil || s.kr == nil {
		return "", errors.New("secret store not initialized")
	}

	var item keyring.Item
	err := s.withTimeout(func() error {
		var getErr error
		item, getErr = s.kr.Get(key)
		return getErr
	})
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return "", os.ErrNotExist
		}
		return "", err
	}

	return string(item.Data), nil
}

// Delete removes a stored secret.
func (s *Store) Delete(key string) error {
	if s == nil || s.kr == nil {
		return errors.New("secret store not initialized")
	}

	err := s.withTimeout(func() error {
		return s.kr.Remove(key)
	})
	if errors.Is(err, keyring.ErrKeyNotFound) {
		return nil
	}
	return err
}

// withTimeout runs fn with a timeout to prevent keyring operations from hanging.
func (s *Store) withTimeout(fn func() error) error {
	ch := make(chan error, 1)
	go func() {
		ch <- fn()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), keyringTimeout)
	defer cancel()

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		return fmt.Errorf("%w; keyring prompt may be blocked (headless/SSH environment?). Use --allow-insecure-store or set %s=1", ErrKeyringTimeout, envAllowInsecure)
	}
}

// TokenKey returns the keyring identifier for a host token.
func TokenKey(hostKey string) string {
	return fmt.Sprintf("host/%s/token", hostKey)
}

// IsNoKeyringError reports whether the error indicates that no native keyring
// backend is available on the system.
func IsNoKeyringError(err error) bool {
	return errors.Is(err, keyring.ErrNoAvailImpl)
}

func resolveAllowedBackends(opts openOptions) []keyring.BackendType {
	if len(opts.allowedBackends) > 0 {
		return opts.allowedBackends
	}

	if backendEnv := strings.TrimSpace(os.Getenv(envBackend)); backendEnv != "" {
		return parseBackendList(backendEnv, opts.allowFile)
	}

	backends := defaultBackends()
	if opts.allowFile {
		backends = append(backends, keyring.FileBackend)
	}
	return backends
}

func defaultBackends() []keyring.BackendType {
	switch runtime.GOOS {
	case "darwin":
		return []keyring.BackendType{keyring.KeychainBackend}
	case "windows":
		return []keyring.BackendType{keyring.WinCredBackend}
	default:
		// In headless environments (SSH without X11, containers, etc.),
		// skip GUI-based backends that would hang waiting for unlock prompts.
		if isHeadless() {
			return []keyring.BackendType{
				keyring.KeyCtlBackend,
				keyring.PassBackend,
			}
		}
		return []keyring.BackendType{
			keyring.SecretServiceBackend,
			keyring.KWalletBackend,
			keyring.KeyCtlBackend,
			keyring.PassBackend,
		}
	}
}

func parseBackendList(raw string, allowFile bool) []keyring.BackendType {
	parts := strings.Split(raw, ",")
	var backends []keyring.BackendType
	for _, part := range parts {
		switch strings.TrimSpace(strings.ToLower(part)) {
		case "keychain":
			backends = append(backends, keyring.KeychainBackend)
		case "wincred":
			backends = append(backends, keyring.WinCredBackend)
		case "secret-service", "secretservice":
			backends = append(backends, keyring.SecretServiceBackend)
		case "kwallet":
			backends = append(backends, keyring.KWalletBackend)
		case "keyctl":
			backends = append(backends, keyring.KeyCtlBackend)
		case "pass":
			backends = append(backends, keyring.PassBackend)
		case "file":
			backends = append(backends, keyring.FileBackend)
		}
	}
	if !allowFile {
		filtered := backends[:0]
		for _, backend := range backends {
			if backend == keyring.FileBackend {
				continue
			}
			filtered = append(filtered, backend)
		}
		backends = filtered
	}
	return backends
}

func configureFileBackend(cfg *keyring.Config, opts openOptions) error {
	passphrase := opts.passphrase
	if passphrase == "" {
		if pwd := os.Getenv("KEYRING_FILE_PASSWORD"); pwd != "" {
			passphrase = pwd
		} else if pwd := os.Getenv("KEYRING_PASSWORD"); pwd != "" {
			passphrase = pwd
		}
	}

	if passphrase != "" {
		cfg.FilePasswordFunc = keyring.FixedStringPrompt(passphrase)
	} else {
		cfg.FilePasswordFunc = keyring.TerminalPrompt
	}

	dir := opts.fileDir
	if dir == "" {
		if userDir, err := os.UserConfigDir(); err == nil {
			dir = filepath.Join(userDir, serviceName, "secrets")
		}
	}

	if dir != "" {
		cfg.FileDir = dir
	}
	return nil
}

func usesFileBackend(backends []keyring.BackendType) bool {
	for _, backend := range backends {
		if backend == keyring.FileBackend {
			return true
		}
	}
	return false
}

func envEnabled(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

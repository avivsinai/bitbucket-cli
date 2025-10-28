package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

const currentVersion = 1

var (
	// ErrContextNotFound is returned when a requested context is missing.
	ErrContextNotFound = errors.New("context not found")
	// ErrHostNotFound is returned when a requested host entry is missing.
	ErrHostNotFound = errors.New("host not found")
)

// Config models persisted CLI state.
type Config struct {
	Version       int                 `yaml:"version"`
	ActiveContext string              `yaml:"active_context,omitempty"`
	Contexts      map[string]*Context `yaml:"contexts,omitempty"`
	Hosts         map[string]*Host    `yaml:"hosts,omitempty"`

	path string
	mu   sync.RWMutex
}

// Context captures user-scoped defaults that reference a host.
type Context struct {
	Host        string `yaml:"host"`
	ProjectKey  string `yaml:"project_key,omitempty"`
	Workspace   string `yaml:"workspace,omitempty"`
	DefaultRepo string `yaml:"default_repo,omitempty"`
}

// Host stores connection and credential details for a Bitbucket instance.
type Host struct {
	Kind     string `yaml:"kind"` // dc | cloud
	BaseURL  string `yaml:"base_url"`
	Username string `yaml:"username,omitempty"`
	Token    string `yaml:"token,omitempty"`
}

// Load retrieves configuration from disk, returning default values when the
// file does not exist. The config file is named config.yml.
func Load() (*Config, error) {
	path, err := resolvePath()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Version:  currentVersion,
		Contexts: make(map[string]*Context),
		Hosts:    make(map[string]*Host),
		path:     path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	if cfg.Contexts == nil {
		cfg.Contexts = make(map[string]*Context)
	}
	if cfg.Hosts == nil {
		cfg.Hosts = make(map[string]*Host)
	}

	return cfg, nil
}

// Save persists the configuration atomically.
func (c *Config) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.path == "" {
		path, err := resolvePath()
		if err != nil {
			return err
		}
		c.path = path
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	if c.Version == 0 {
		c.Version = currentVersion
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	tmpFile, err := os.CreateTemp(dir, ".config-*.yml")
	if err != nil {
		return fmt.Errorf("create temp config: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp config: %w", err)
	}

	if err := tmpFile.Chmod(0o600); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("chmod temp config: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp config: %w", err)
	}

	if err := os.Rename(tmpFile.Name(), c.path); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// Path returns the absolute config file path.
func (c *Config) Path() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.path
}

// SetContext upserts a named context.
func (c *Config) SetContext(name string, ctx *Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Contexts == nil {
		c.Contexts = make(map[string]*Context)
	}
	c.Contexts[name] = ctx
}

// Context retrieves a named context.
func (c *Config) Context(name string) (*Context, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ctx, ok := c.Contexts[name]
	if !ok {
		return nil, ErrContextNotFound
	}
	return ctx, nil
}

// DeleteContext removes a named context and clears the active context if needed.
func (c *Config) DeleteContext(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.Contexts, name)
	if c.ActiveContext == name {
		c.ActiveContext = ""
	}
}

// SetActiveContext updates the active context name after verifying it exists.
func (c *Config) SetActiveContext(name string) error {
	if name == "" {
		c.mu.Lock()
		c.ActiveContext = ""
		c.mu.Unlock()
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.Contexts[name]; !ok {
		return ErrContextNotFound
	}
	c.ActiveContext = name
	return nil
}

// SetHost upserts host credentials by key.
func (c *Config) SetHost(key string, host *Host) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Hosts == nil {
		c.Hosts = make(map[string]*Host)
	}
	c.Hosts[key] = host
}

// Host retrieves host credentials by key.
func (c *Config) Host(key string) (*Host, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	h, ok := c.Hosts[key]
	if !ok {
		return nil, ErrHostNotFound
	}
	return h, nil
}

// DeleteHost removes a host entry. Contexts referencing the host should be
// cleaned up by the caller.
func (c *Config) DeleteHost(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.Hosts, key)
}

func resolvePath() (string, error) {
	base := os.Getenv("BKT_CONFIG_DIR")
	if base == "" {
		dir, err := os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("resolve config dir: %w", err)
		}
		base = filepath.Join(dir, "bkt")
	}
	return filepath.Join(base, "config.yml"), nil
}

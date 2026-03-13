// Package config manages dts-cli configuration, including endpoint contexts and user settings.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for dts-cli.
type Config struct {
	CurrentContext string              `yaml:"currentContext"`
	Contexts       map[string]*Context `yaml:"contexts"`
	Settings       Settings            `yaml:"settings"`
}

// Context represents a DTS endpoint connection configuration.
type Context struct {
	URL          string `yaml:"url"`
	TaskHub      string `yaml:"taskHub"`
	Subscription string `yaml:"subscription,omitempty"`
	Scheduler    string `yaml:"scheduler,omitempty"`
	TenantID     string `yaml:"tenantId,omitempty"`
	Description  string `yaml:"description,omitempty"`
}

// Settings holds user preferences.
type Settings struct {
	AuthMode                  string `yaml:"authMode"`
	TimeMode                  string `yaml:"timeMode"`
	Theme                     string `yaml:"theme"`
	RefreshInterval           int    `yaml:"refreshInterval"`
	PageSize                  int    `yaml:"pageSize"`
	EnableAgents              bool   `yaml:"enableAgents"`
	EnableSchedules           bool   `yaml:"enableSchedules"`
	HideAgentsFromEntities    bool   `yaml:"hideAgentsFromEntities"`
	HideSchedulesFromEntities bool   `yaml:"hideSchedulesFromEntities"`
}

// DefaultConfig returns a new config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Contexts: make(map[string]*Context),
		Settings: Settings{
			AuthMode:                  "default",
			TimeMode:                  "local",
			Theme:                     "dark",
			RefreshInterval:           30,
			PageSize:                  100,
			EnableAgents:              true,
			EnableSchedules:           true,
			HideAgentsFromEntities:    true,
			HideSchedulesFromEntities: true,
		},
	}
}

// ConfigDir returns the configuration directory path.
func ConfigDir() string {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "dts-cli")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "dts-cli")
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg != "" {
		return filepath.Join(xdg, "dts-cli")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "dts-cli")
}

// ConfigFilePath returns the full path to the config file.
func ConfigFilePath() string {
	return filepath.Join(ConfigDir(), "config.yaml")
}

// Load reads the config from disk. If the file doesn't exist, returns defaults.
func Load() (*Config, error) {
	path := ConfigFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if cfg.Contexts == nil {
		cfg.Contexts = make(map[string]*Context)
	}
	return cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	path := ConfigFilePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

// CurrentCtx returns the current active context, or nil if none is set.
func (c *Config) CurrentCtx() *Context {
	if c.CurrentContext == "" {
		return nil
	}
	return c.Contexts[c.CurrentContext]
}

// AddContext registers a new endpoint context.
func (c *Config) AddContext(name string, ctx *Context) {
	c.Contexts[name] = ctx
	if c.CurrentContext == "" {
		c.CurrentContext = name
	}
}

// RemoveContext removes an endpoint context.
func (c *Config) RemoveContext(name string) {
	delete(c.Contexts, name)
	if c.CurrentContext == name {
		c.CurrentContext = ""
		// Pick first available
		for k := range c.Contexts {
			c.CurrentContext = k
			break
		}
	}
}

// UseLocalTime returns true if times should be formatted in local timezone.
func (c *Config) UseLocalTime() bool {
	return c.Settings.TimeMode == "local"
}

// Package config manages slack-cli's persisted configuration: named auth
// profiles (bot/user tokens) and which one is active.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Profile holds one named set of Slack credentials.
type Profile struct {
	Token   string `yaml:"token"`
	Team    string `yaml:"team,omitempty"`     // optional human label, e.g. "acme-workspace"
	BaseURL string `yaml:"base_url,omitempty"` // optional override, e.g. for testing
}

// Config is the on-disk shape of the config file.
type Config struct {
	Active   string             `yaml:"active"`
	Profiles map[string]Profile `yaml:"profiles"`
}

// Dir returns the directory slack-cli stores its config in, honoring
// SLACK_CLI_CONFIG_DIR and XDG_CONFIG_HOME before falling back to
// ~/.config/slack-cli.
func Dir() (string, error) {
	if d := os.Getenv("SLACK_CLI_CONFIG_DIR"); d != "" {
		return d, nil
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "slack-cli"), nil
}

// Path returns the full path to the config file.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads the config file, returning an empty Config if it doesn't exist.
func Load() (*Config, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Config{Profiles: map[string]Profile{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return &cfg, nil
}

// Save writes the config file with owner-only permissions, since it holds
// bearer tokens.
func (c *Config) Save() error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	path, err := Path()
	if err != nil {
		return err
	}
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// SetProfile creates or overwrites a named profile, activating it if it is
// the first profile or activate is true.
func (c *Config) SetProfile(name string, prof Profile, activate bool) {
	if c.Profiles == nil {
		c.Profiles = map[string]Profile{}
	}
	c.Profiles[name] = prof
	if activate || c.Active == "" {
		c.Active = name
	}
}

// RemoveProfile deletes a named profile, clearing Active if it pointed at it.
func (c *Config) RemoveProfile(name string) {
	delete(c.Profiles, name)
	if c.Active == name {
		c.Active = ""
	}
}

// Get returns a named profile, or the active one if name is empty.
func (c *Config) Get(name string) (Profile, bool) {
	if name == "" {
		name = c.Active
	}
	if name == "" {
		return Profile{}, false
	}
	prof, ok := c.Profiles[name]
	return prof, ok
}

package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const configFileName = ".autogit.yaml"

// Config holds all autogit configuration settings.
type Config struct {
	Provider string       `yaml:"provider"`
	Claude   ClaudeConfig `yaml:"claude"`
	OpenAI   OpenAIConfig `yaml:"openai"`
}

// ClaudeConfig holds Anthropic Claude provider settings.
type ClaudeConfig struct {
	Model string `yaml:"model"`
}

// OpenAIConfig holds OpenAI-compatible provider settings.
type OpenAIConfig struct {
	BaseURL string `yaml:"base_url"`
	Model   string `yaml:"model"`
}

// DefaultConfig returns a Config populated with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Provider: "claude",
		Claude:   ClaudeConfig{Model: "claude-opus-4-6"},
		OpenAI:   OpenAIConfig{BaseURL: "https://api.openai.com/v1", Model: "gpt-4o"},
	}
}

// configPath returns the absolute path to the config file.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, configFileName), nil
}

// Load reads ~/.autogit.yaml and returns a Config. If the file does not exist,
// DefaultConfig is returned. Invalid YAML or an invalid base_url returns an error.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes cfg to ~/.autogit.yaml, creating or overwriting the file.
func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// ApplyOverrides updates cfg in-place. A non-empty provider replaces cfg.Provider.
// A non-empty model is applied to the active provider's model field only.
func (c *Config) ApplyOverrides(provider, model string) {
	if provider != "" {
		c.Provider = provider
	}
	if model != "" {
		switch c.Provider {
		case "claude":
			c.Claude.Model = model
		case "openai":
			c.OpenAI.Model = model
		}
	}
}

// validate normalises and validates the config, returning an error for any
// field that would prevent autogit from functioning correctly.
func (c *Config) validate() error {
	if c.Provider != "claude" && c.Provider != "openai" {
		return fmt.Errorf("unknown provider %q, supported: claude, openai", c.Provider)
	}

	c.OpenAI.BaseURL = strings.TrimRight(c.OpenAI.BaseURL, "/")

	if c.OpenAI.BaseURL != "" {
		u, err := url.Parse(c.OpenAI.BaseURL)
		if err != nil {
			return fmt.Errorf("invalid base_url %q: %w", c.OpenAI.BaseURL, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("base_url must start with http:// or https://, got %q", c.OpenAI.BaseURL)
		}
	}

	return nil
}

// ConfigExists reports whether ~/.autogit.yaml exists on disk.
func ConfigExists() bool {
	path, err := configPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Path returns the absolute path to the config file. If the home directory
// cannot be determined it returns a human-readable fallback.
func Path() string {
	path, err := configPath()
	if err != nil {
		return "~/" + configFileName
	}
	return path
}

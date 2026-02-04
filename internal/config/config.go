package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var (
	// ErrProfileNotFound is returned when a profile doesn't exist
	ErrProfileNotFound = errors.New("profile not found")
	// ErrConfigNotFound is returned when config file doesn't exist
	ErrConfigNotFound = errors.New("config file not found")
)

// DefaultConfigPath returns the default config file path
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".azure2aws", "config.yaml"), nil
}

// EnsureConfigDir ensures the config directory exists with proper permissions
func EnsureConfigDir(configPath string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	return nil
}

// LoadConfig loads configuration from the specified path
func LoadConfig(path string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, ErrConfigNotFound
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := NewConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Ensure profiles map is initialized
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]Profile)
	}

	return cfg, nil
}

// LoadOrCreateConfig loads config or creates a new one if it doesn't exist
func LoadOrCreateConfig(path string) (*Config, error) {
	cfg, err := LoadConfig(path)
	if err != nil {
		if errors.Is(err, ErrConfigNotFound) {
			return NewConfig(), nil
		}
		return nil, err
	}
	return cfg, nil
}

// SaveConfig saves configuration to the specified path
func SaveConfig(cfg *Config, path string) error {
	// Ensure directory exists
	if err := EnsureConfigDir(path); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with secure permissions (0600)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetProfile returns a merged profile (with defaults applied)
func (c *Config) GetProfile(name string) (*MergedProfile, error) {
	profile, exists := c.Profiles[name]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrProfileNotFound, name)
	}

	merged := &MergedProfile{
		Name:     name,
		URL:      profile.URL,
		AppID:    profile.AppID,
		Username: profile.Username,
		RoleARN:  profile.RoleARN,
		Output:   profile.Output,
	}

	if profile.Region != "" {
		merged.Region = profile.Region
	} else {
		merged.Region = c.Defaults.Region
	}

	if profile.SessionDuration > 0 {
		merged.SessionDuration = profile.SessionDuration
	} else {
		merged.SessionDuration = c.Defaults.SessionDuration
	}

	return merged, nil
}

// SetProfile adds or updates a profile
func (c *Config) SetProfile(name string, profile Profile) {
	if c.Profiles == nil {
		c.Profiles = make(map[string]Profile)
	}
	c.Profiles[name] = profile
}

// DeleteProfile removes a profile
func (c *Config) DeleteProfile(name string) error {
	if _, exists := c.Profiles[name]; !exists {
		return fmt.Errorf("%w: %s", ErrProfileNotFound, name)
	}
	delete(c.Profiles, name)
	return nil
}

// ListProfiles returns all profile names
func (c *Config) ListProfiles() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	return names
}

// HasProfile checks if a profile exists
func (c *Config) HasProfile(name string) bool {
	_, exists := c.Profiles[name]
	return exists
}

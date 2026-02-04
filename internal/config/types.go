package config

// Config represents the main configuration structure
type Config struct {
	Defaults Defaults           `yaml:"defaults"`
	Profiles map[string]Profile `yaml:"profiles"`
}

// Defaults contains default settings applied to all profiles
type Defaults struct {
	Region          string `yaml:"region"`
	SessionDuration int    `yaml:"session_duration"`
}

// Profile represents an Azure AD SAML profile configuration
type Profile struct {
	// Azure AD configuration
	URL      string `yaml:"url"`      // Azure AD app URL
	AppID    string `yaml:"app_id"`   // Azure AD application ID
	Username string `yaml:"username"` // Username/email

	// AWS configuration
	RoleARN string `yaml:"role_arn,omitempty"` // Preferred AWS role ARN
	Region  string `yaml:"region,omitempty"`   // Override default region
	Output  string `yaml:"output,omitempty"`   // AWS CLI output format (json, text, table)

	// Optional overrides
	SessionDuration int `yaml:"session_duration,omitempty"` // Override default session duration
}

// MergedProfile returns a profile with defaults applied
type MergedProfile struct {
	Name            string
	URL             string
	AppID           string
	Username        string
	RoleARN         string
	Region          string
	Output          string
	SessionDuration int
}

// NewConfig creates a new configuration with sensible defaults
func NewConfig() *Config {
	return &Config{
		Defaults: Defaults{
			Region:          "us-east-1",
			SessionDuration: 3600, // 1 hour
		},
		Profiles: make(map[string]Profile),
	}
}

package aws

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/ini.v1"
)

type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Expiration      time.Time
	Region          string
	Output          string
	AssumedRoleARN  string
}

func DefaultCredentialsPath() (string, error) {
	if envPath := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); envPath != "" {
		return envPath, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".aws", "credentials"), nil
}

func DefaultConfigPath() (string, error) {
	if envPath := os.Getenv("AWS_CONFIG_FILE"); envPath != "" {
		return envPath, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".aws", "config"), nil
}

func SaveCredentials(profile string, creds *Credentials) error {
	credPath, err := DefaultCredentialsPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(credPath), 0700); err != nil {
		return fmt.Errorf("failed to create credentials directory: %w", err)
	}

	cfg, err := ini.LooseLoad(credPath)
	if err != nil {
		return fmt.Errorf("failed to load credentials file: %w", err)
	}

	section, err := cfg.NewSection(profile)
	if err != nil {
		section = cfg.Section(profile)
	}

	section.Key("aws_access_key_id").SetValue(creds.AccessKeyID)
	section.Key("aws_secret_access_key").SetValue(creds.SecretAccessKey)
	section.Key("aws_session_token").SetValue(creds.SessionToken)
	section.Key("x_security_token_expires").SetValue(creds.Expiration.Format(time.RFC3339))

	if err := cfg.SaveTo(credPath); err != nil {
		return fmt.Errorf("failed to save credentials file: %w", err)
	}

	if err := os.Chmod(credPath, 0600); err != nil {
		return fmt.Errorf("failed to set credentials file permissions: %w", err)
	}

	if err := SaveAWSConfig(profile, creds.Region, creds.Output); err != nil {
		return fmt.Errorf("failed to save AWS config: %w", err)
	}

	return nil
}

func SaveAWSConfig(profile, region, output string) error {
	configPath, err := DefaultConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	cfg, err := ini.LooseLoad(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config file: %w", err)
	}

	sectionName := profile
	if profile != "default" {
		sectionName = "profile " + profile
	}

	section, err := cfg.NewSection(sectionName)
	if err != nil {
		section = cfg.Section(sectionName)
	}

	if region != "" {
		section.Key("region").SetValue(region)
	}

	if output != "" {
		section.Key("output").SetValue(output)
	} else {
		section.Key("output").SetValue("json")
	}

	if err := cfg.SaveTo(configPath); err != nil {
		return fmt.Errorf("failed to save config file: %w", err)
	}

	if err := os.Chmod(configPath, 0600); err != nil {
		return fmt.Errorf("failed to set config file permissions: %w", err)
	}

	return nil
}

// LoadCredentials loads AWS credentials from the credentials file
func LoadCredentials(profile string) (*Credentials, error) {
	credPath, err := DefaultCredentialsPath()
	if err != nil {
		return nil, err
	}

	cfg, err := ini.Load(credPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials file: %w", err)
	}

	section, err := cfg.GetSection(profile)
	if err != nil {
		return nil, fmt.Errorf("profile %s not found: %w", profile, err)
	}

	creds := &Credentials{
		AccessKeyID:     section.Key("aws_access_key_id").String(),
		SecretAccessKey: section.Key("aws_secret_access_key").String(),
		SessionToken:    section.Key("aws_session_token").String(),
		Region:          section.Key("region").String(),
	}

	// Parse expiration time if present
	if expStr := section.Key("x_security_token_expires").String(); expStr != "" {
		if exp, err := time.Parse(time.RFC3339, expStr); err == nil {
			creds.Expiration = exp
		}
	}

	return creds, nil
}

// CredentialsExpired checks if credentials for a profile are expired
func CredentialsExpired(profile string) bool {
	creds, err := LoadCredentials(profile)
	if err != nil {
		return true // If we can't load, assume expired
	}

	// If no expiration set, assume expired
	if creds.Expiration.IsZero() {
		return true
	}

	return IsExpired(creds.Expiration)
}

// DeleteCredentials removes credentials for a profile
func DeleteCredentials(profile string) error {
	credPath, err := DefaultCredentialsPath()
	if err != nil {
		return err
	}

	cfg, err := ini.Load(credPath)
	if err != nil {
		return fmt.Errorf("failed to load credentials file: %w", err)
	}

	cfg.DeleteSection(profile)

	if err := cfg.SaveTo(credPath); err != nil {
		return fmt.Errorf("failed to save credentials file: %w", err)
	}

	return nil
}

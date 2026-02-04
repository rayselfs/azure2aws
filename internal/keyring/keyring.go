package keyring

import (
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	// ServiceName is the keyring service name for azure2aws
	ServiceName = "azure2aws"
)

var (
	// ErrPasswordNotFound is returned when password is not found in keyring
	ErrPasswordNotFound = errors.New("password not found in keyring")
	// ErrKeyringUnavailable is returned when keyring is not available
	ErrKeyringUnavailable = errors.New("keyring is not available on this system")
)

// Keyring provides password storage operations
type Keyring struct {
	serviceName string
}

// New creates a new Keyring instance
func New() *Keyring {
	return &Keyring{
		serviceName: ServiceName,
	}
}

// NewWithService creates a new Keyring with a custom service name (useful for testing)
func NewWithService(serviceName string) *Keyring {
	return &Keyring{
		serviceName: serviceName,
	}
}

// SavePassword stores a password for the given profile
func (k *Keyring) SavePassword(profile, password string) error {
	if err := keyring.Set(k.serviceName, profile, password); err != nil {
		return fmt.Errorf("failed to save password: %w", err)
	}
	return nil
}

// GetPassword retrieves a password for the given profile
func (k *Keyring) GetPassword(profile string) (string, error) {
	password, err := keyring.Get(k.serviceName, profile)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrPasswordNotFound
		}
		return "", fmt.Errorf("failed to get password: %w", err)
	}
	return password, nil
}

// DeletePassword removes a password for the given profile
func (k *Keyring) DeletePassword(profile string) error {
	if err := keyring.Delete(k.serviceName, profile); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return ErrPasswordNotFound
		}
		return fmt.Errorf("failed to delete password: %w", err)
	}
	return nil
}

// HasPassword checks if a password exists for the given profile
func (k *Keyring) HasPassword(profile string) bool {
	_, err := k.GetPassword(profile)
	return err == nil
}

// IsAvailable checks if the keyring is available on this system
func (k *Keyring) IsAvailable() bool {
	// Try to perform a no-op operation to check availability
	// We use a test key that we immediately clean up
	testKey := "__azure2aws_keyring_test__"
	testValue := "test"

	err := keyring.Set(k.serviceName, testKey, testValue)
	if err != nil {
		return false
	}

	// Clean up test key
	_ = keyring.Delete(k.serviceName, testKey)
	return true
}

// Package-level convenience functions

// SavePassword stores a password using the default service name
func SavePassword(profile, password string) error {
	return New().SavePassword(profile, password)
}

// GetPassword retrieves a password using the default service name
func GetPassword(profile string) (string, error) {
	return New().GetPassword(profile)
}

// DeletePassword removes a password using the default service name
func DeletePassword(profile string) error {
	return New().DeletePassword(profile)
}

// HasPassword checks if a password exists using the default service name
func HasPassword(profile string) bool {
	return New().HasPassword(profile)
}

// IsAvailable checks if keyring is available using the default service name
func IsAvailable() bool {
	return New().IsAvailable()
}

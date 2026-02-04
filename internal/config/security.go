package config

import (
	"fmt"
	"os"
	"runtime"
)

// SecureFilePermissions ensures a file has secure permissions (0600)
// This is a no-op on Windows
func SecureFilePermissions(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, nothing to secure
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if permissions are already correct
	mode := info.Mode().Perm()
	if mode == 0600 {
		return nil
	}

	// Fix permissions
	if err := os.Chmod(path, 0600); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// SecureDirPermissions ensures a directory has secure permissions (0700)
// This is a no-op on Windows
func SecureDirPermissions(path string) error {
	if runtime.GOOS == "windows" {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist, nothing to secure
		}
		return fmt.Errorf("failed to stat directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	// Check if permissions are already correct
	mode := info.Mode().Perm()
	if mode == 0700 {
		return nil
	}

	// Fix permissions
	if err := os.Chmod(path, 0700); err != nil {
		return fmt.Errorf("failed to set directory permissions: %w", err)
	}

	return nil
}

// CheckFilePermissions verifies a file has secure permissions
// Returns true if permissions are secure, false otherwise
func CheckFilePermissions(path string) (bool, error) {
	if runtime.GOOS == "windows" {
		return true, nil // Skip on Windows
	}

	info, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("failed to stat file: %w", err)
	}

	mode := info.Mode().Perm()
	// Allow 0600 or 0400 (read-only)
	return mode == 0600 || mode == 0400, nil
}

// WarnInsecurePermissions logs a warning if file permissions are insecure
// Returns an error description if insecure, empty string if secure
func WarnInsecurePermissions(path string) string {
	if runtime.GOOS == "windows" {
		return ""
	}

	info, err := os.Stat(path)
	if err != nil {
		return ""
	}

	mode := info.Mode().Perm()
	if mode&0077 != 0 {
		return fmt.Sprintf("Warning: %s has insecure permissions %04o (should be 0600)", path, mode)
	}

	return ""
}

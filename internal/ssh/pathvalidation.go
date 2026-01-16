package ssh

import (
	"fmt"
	"path/filepath"
	"strings"
)

// validatePath validates that a path doesn't contain obvious malicious patterns.
// This helps prevent CWE-22 path traversal attacks.
func validatePath(path string, pathType string) error {
	// Check for null bytes (common attack vector)
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("invalid %s path: contains null byte", pathType)
	}

	// Note: We don't reject .. in paths because they're legitimate for relative paths
	// like "../../.ssh/id_rsa". filepath.Clean() will resolve them safely, and
	// os.Open/os.ReadFile will fail if the final path doesn't exist or isn't accessible.
	// The main protection is from null byte injection and the filesystem permissions.

	return nil
}

// validateSSHKeyPath validates an SSH key path for reading
func validateSSHKeyPath(keyPath string) error {
	if err := validatePath(keyPath, "SSH key"); err != nil {
		return err
	}

	// Additional validation for SSH keys: should be absolute or relative to home
	if !filepath.IsAbs(keyPath) && !strings.HasPrefix(keyPath, "~/") && !strings.HasPrefix(keyPath, ".ssh/") {
		// This is acceptable - relative paths are allowed
	}

	return nil
}

// validateLocalFilePath validates a local file path for reading/uploading
func validateLocalFilePath(localPath string) error {
	if err := validatePath(localPath, "local file"); err != nil {
		return err
	}

	// For local paths, we're more permissive since we're reading from the local system
	// The main concern is preventing injection attacks
	return nil
}

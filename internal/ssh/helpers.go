package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// getUserHomeDir returns the user's home directory with proper error handling.
// This centralizes home directory retrieval to avoid code duplication.
func getUserHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return home, nil
}

// getDefaultKnownHostsFile returns the default path to the known_hosts file (~/.ssh/known_hosts).
func getDefaultKnownHostsFile() (string, error) {
	home, err := getUserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "known_hosts"), nil
}

// expandHomePath expands paths starting with ~/ to absolute paths.
// For other paths, returns them unchanged.
func expandHomePath(path string) (string, error) {
	if !strings.HasPrefix(path, "~/") {
		return path, nil
	}

	home, err := getUserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, path[2:]), nil
}

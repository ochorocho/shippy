package deploy

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tinnie/internal/ssh"
)

// ReleaseManager manages release directories on the remote server
type ReleaseManager struct {
	client     *ssh.Client
	deployPath string
}

// NewReleaseManager creates a new release manager
func NewReleaseManager(client *ssh.Client, deployPath string) *ReleaseManager {
	return &ReleaseManager{
		client:     client,
		deployPath: deployPath,
	}
}

// CreateRelease creates a new release directory
// Folder structure follows Deployer/Capistrano conventions:
// deploy_path/
//
//	├── current -> releases/20240109120000
//	├── releases/
//	│   ├── 20240109120000/
//	│   └── 20240109110000/
//	└── shared/
//	    ├── .env
//	    └── var/
func (r *ReleaseManager) CreateRelease() (string, error) {
	timestamp := time.Now().Format("20060102150405")
	releaseName := timestamp
	releasesDir := filepath.Join(r.deployPath, "releases")
	releasePath := filepath.Join(releasesDir, releaseName)

	// Create releases directory
	if err := r.client.MkdirAll(releasesDir); err != nil {
		return "", fmt.Errorf("failed to create releases directory: %w", err)
	}

	// Create the release directory
	if err := r.client.MkdirAll(releasePath); err != nil {
		return "", fmt.Errorf("failed to create release directory: %w", err)
	}

	return releasePath, nil
}

// UpdateCurrentSymlink updates the "current" symlink to point to the new release
func (r *ReleaseManager) UpdateCurrentSymlink(releasePath string) error {
	currentPath := filepath.Join(r.deployPath, "current")
	releaseName := filepath.Base(releasePath)

	// Use ln -sfn to force create/update the symlink
	// -s: create symbolic link
	// -f: force (remove existing destination file)
	// -n: treat destination as normal file if it's a symlink to directory
	cmd := fmt.Sprintf("ln -sfn releases/%s %s", releaseName, currentPath)
	if _, err := r.client.RunCommand(cmd); err != nil {
		// Provide helpful error message with manual fix instructions
		return fmt.Errorf("failed to update 'current' symlink: %w\n\nTo fix manually, SSH to the server and run:\n  cd %s\n  rm -f current\n  ln -s releases/%s current",
			err, r.deployPath, releaseName)
	}

	return nil
}

// CleanupOldReleases removes old releases, keeping only the specified number
func (r *ReleaseManager) CleanupOldReleases(keep int) error {
	if keep <= 0 {
		keep = 5 // Default to keeping 5 releases
	}

	releasesPath := filepath.Join(r.deployPath, "releases")

	// List releases
	output, err := r.client.RunCommand(fmt.Sprintf("ls -1 %s 2>/dev/null || true", releasesPath))
	if err != nil {
		return fmt.Errorf("failed to list releases: %w", err)
	}

	releases := []string{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			releases = append(releases, line)
		}
	}

	// Sort releases (newest first)
	sort.Slice(releases, func(i, j int) bool {
		return releases[i] > releases[j]
	})

	// Remove old releases
	if len(releases) > keep {
		toRemove := releases[keep:]
		for _, release := range toRemove {
			releasePath := filepath.Join(releasesPath, release)
			cmd := fmt.Sprintf("rm -rf %s", releasePath)
			if _, err := r.client.RunCommand(cmd); err != nil {
				return fmt.Errorf("failed to remove old release %s: %w", release, err)
			}
		}
	}

	return nil
}

// CreateSharedSymlinks creates symlinks from the release to shared files/directories
func (r *ReleaseManager) CreateSharedSymlinks(releasePath string, sharedPaths []string) error {
	sharedDir := filepath.Join(r.deployPath, "shared")

	// Create shared directory
	if err := r.client.MkdirAll(sharedDir); err != nil {
		return fmt.Errorf("failed to create shared directory: %w", err)
	}

	for _, sharedPath := range sharedPaths {
		// Create the shared file/directory if it doesn't exist
		sharedFullPath := filepath.Join(sharedDir, sharedPath)
		sharedDir := filepath.Dir(sharedFullPath)
		if err := r.client.MkdirAll(sharedDir); err != nil {
			return fmt.Errorf("failed to create shared directory for %s: %w", sharedPath, err)
		}

		// Check if it's a directory path (ends with /)
		isDir := strings.HasSuffix(sharedPath, "/")
		if isDir {
			// Create shared directory
			if err := r.client.MkdirAll(sharedFullPath); err != nil {
				return fmt.Errorf("failed to create shared directory %s: %w", sharedPath, err)
			}
		} else {
			// Create shared file if it doesn't exist
			cmd := fmt.Sprintf("test -e %s || touch %s", sharedFullPath, sharedFullPath)
			if _, err := r.client.RunCommand(cmd); err != nil {
				return fmt.Errorf("failed to create shared file %s: %w", sharedPath, err)
			}
		}

		// Create symlink in release directory
		releaseTargetPath := filepath.Join(releasePath, sharedPath)
		releaseTargetDir := filepath.Dir(releaseTargetPath)

		// Create parent directory in release
		if err := r.client.MkdirAll(releaseTargetDir); err != nil {
			return fmt.Errorf("failed to create directory in release for %s: %w", sharedPath, err)
		}

		// Calculate relative path from release to shared
		relPath, err := filepath.Rel(releaseTargetDir, sharedFullPath)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}

		// Remove existing file/directory in release if it exists
		cmd := fmt.Sprintf("rm -rf %s", releaseTargetPath)
		if _, err := r.client.RunCommand(cmd); err != nil {
			return fmt.Errorf("failed to remove existing path %s: %w", releaseTargetPath, err)
		}

		// Create symlink
		cmd = fmt.Sprintf("ln -nfs %s %s", relPath, releaseTargetPath)
		if _, err := r.client.RunCommand(cmd); err != nil {
			return fmt.Errorf("failed to create symlink for %s: %w", sharedPath, err)
		}
	}

	return nil
}

// GetCurrentRelease returns the path of the current release
func (r *ReleaseManager) GetCurrentRelease() (string, error) {
	currentPath := filepath.Join(r.deployPath, "current")
	cmd := fmt.Sprintf("readlink -f %s", currentPath)

	output, err := r.client.RunCommand(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to read current symlink: %w", err)
	}

	return strings.TrimSpace(output), nil
}

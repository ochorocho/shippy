package backup

import (
	"archive/tar"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"shippy/internal/ssh"
)

// DownloadShared downloads specific files/directories from the remote server's
// shared directory and adds them to the ZIP. Only paths listed in files are included.
// Uses tar-over-SSH for efficient bulk transfer.
func DownloadShared(client *ssh.Client, deployPath string, files []string, zb *ZipBuilder, verbose bool) error {
	sharedPath := filepath.Join(deployPath, "shared")

	// Check if shared directory exists
	output, err := client.RunCommand(fmt.Sprintf("test -d %s && echo exists || echo missing", sharedPath))
	if err != nil || strings.TrimSpace(output) != "exists" {
		return fmt.Errorf("shared directory does not exist: %s", sharedPath)
	}

	// Build tar command with only the specified paths
	// Each path is relative to the shared directory
	quotedPaths := make([]string, len(files))
	for i, f := range files {
		// Remove trailing slash for consistency (tar handles both files and dirs)
		quotedPaths[i] = fmt.Sprintf("'%s'", strings.TrimSuffix(f, "/"))
	}

	tarCmd := fmt.Sprintf("tar cf - -C %s %s 2>/dev/null", sharedPath, strings.Join(quotedPaths, " "))

	// Stream as tar archive via SSH
	pr, pw := io.Pipe()

	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		err := client.RunCommandWithOutput(tarCmd, pw, io.Discard)
		errCh <- err
	}()

	// Read tar entries and add to ZIP
	tr := tar.NewReader(pr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar stream: %w", err)
		}

		// Skip directories and non-regular files
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Clean up the path and add under shared/ prefix
		cleanPath := strings.TrimPrefix(header.Name, "./")
		zipPath := filepath.Join("shared", cleanPath)

		if err := zb.AddFile(zipPath, tr); err != nil {
			return fmt.Errorf("failed to add shared file '%s': %w", cleanPath, err)
		}
	}

	// Wait for SSH command to complete
	if err := <-errCh; err != nil {
		return fmt.Errorf("tar streaming failed: %w", err)
	}

	return nil
}

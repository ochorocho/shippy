package rsync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ochorocho/shippy/internal/errors"
	"github.com/ochorocho/shippy/internal/ssh"
	"github.com/ochorocho/shippy/internal/ui"
)

// cacheManifestFile maps relPath→sha256 of the cache contents. It lives outside
// .cache/ so the rsync promote never copies it into a release.
const cacheManifestFile = ".shippy/cache-manifest.json"

// remoteClient is the subset of *ssh.Client the syncer needs. As an interface
// it lets tests assert the exact remote commands (and their shell quoting)
// using fabricated file lists and a recording fake, with no SSH connection.
type remoteClient interface {
	RunCommand(cmd string) (string, error)
	MkdirAll(path string) error
	UploadFile(localPath, remotePath string, mode os.FileMode) error
}

// Syncer handles file synchronization over SSH
type Syncer struct {
	client     remoteClient
	remotePath string
	verbose    bool
	deployPath string // Base deploy path for cache directory
}

// NewSyncer creates a new file syncer
func NewSyncer(client remoteClient, remotePath string, verbose bool, deployPath string) *Syncer {
	return &Syncer{
		client:     client,
		remotePath: remotePath,
		verbose:    verbose,
		deployPath: deployPath,
	}
}

// Sync synchronizes files to the remote server using a cache directory
func (s *Syncer) Sync(files []FileInfo) error {
	out := ui.New()

	fmt.Printf("\n")
	// #nosec G104 -- Printf errors in UI output can be safely ignored
	out.Blue.Printf("→ Analyzing %d files for synchronization\n", len(files))

	// Use remote .cache directory as intermediate upload target
	cachePath := filepath.Join(s.deployPath, ".cache")

	// Create cache directory on remote
	if err := s.client.MkdirAll(cachePath); err != nil {
		return fmt.Errorf("failed to create remote cache directory: %w", err)
	}

	// Determine which files need uploading by comparing with remote cache
	// Build a map of remote files for efficient lookup
	out.Info("  Building remote cache index...")
	remoteFiles, err := s.getRemoteFileIndex(cachePath)
	if err != nil {
		// If we can't get index, upload all files
		out.Info("  Warning: couldn't read remote cache, uploading all files: %v", err)
		remoteFiles = make(map[string]RemoteFileInfo)
	}

	filesToUpload := []FileInfo{}
	skipped := 0

	// Build a set of local file paths for deletion detection
	localFiles := make(map[string]bool)
	for _, file := range files {
		localFiles[file.RelPath] = true
	}

	// Compared by checksum, not mtime: uploads don't preserve mtimes and CI
	// checkouts restamp every file, which forced a full re-upload every deploy.
	manifest := s.readCacheManifest()

	out.Info("  Comparing local files with remote cache...")
	for _, file := range files {
		remoteInfo, exists := remoteFiles[file.RelPath]

		needsUpload := !exists ||
			remoteInfo.Size != file.Size ||
			file.Checksum == "" ||
			manifest[file.RelPath] != file.Checksum

		if needsUpload {
			filesToUpload = append(filesToUpload, file)
		} else {
			skipped++
		}
	}

	// Find files that exist in remote cache but not locally (need deletion)
	filesToDelete := []string{}
	for remotePath := range remoteFiles {
		if !localFiles[remotePath] {
			filesToDelete = append(filesToDelete, remotePath)
		}
	}

	fmt.Printf("\n")
	if len(filesToDelete) > 0 {
		out.Success("Found %d new/changed files, %d unchanged (cached), %d to delete", len(filesToUpload), skipped, len(filesToDelete))
	} else if skipped > 0 {
		out.Success("Found %d new/changed files, %d unchanged (cached)", len(filesToUpload), skipped)
	} else {
		out.Success("Found %d files to upload", len(filesToUpload))
	}

	// Upload only changed/new files
	if err := s.uploadWithProgress(filesToUpload, cachePath, out); err != nil {
		return err
	}

	// Delete files from cache that no longer exist locally
	if len(filesToDelete) > 0 {
		fmt.Printf("\n")
		// #nosec G104 -- Printf errors in UI output can be safely ignored
		out.Blue.Printf("→ Removing %d deleted files from cache\n", len(filesToDelete))

		deleted := 0
		for _, fileToDelete := range filesToDelete {
			remoteCachePath := filepath.Join(cachePath, fileToDelete)

			if s.verbose {
				// #nosec G104 -- Printf errors in UI output can be safely ignored
				out.Yellow.Printf("  Deleting: %s\n", fileToDelete)
			}

			// Delete the file from cache
			deleteCmd := fmt.Sprintf("rm -f %s", ssh.Quote(remoteCachePath))
			if _, err := s.client.RunCommand(deleteCmd); err != nil {
				// Non-fatal: log but continue
				if s.verbose {
					out.Info("  Warning: failed to delete %s: %v", fileToDelete, err)
				}
			} else {
				deleted++
			}
		}

		out.Success("Removed %d files from cache", deleted)
	}

	// Cache now matches the local tree; persist the manifest (best-effort).
	if err := s.writeCacheManifest(files); err != nil {
		out.Info("  Warning: could not write cache manifest (next deploy re-uploads everything): %v", err)
	}

	// Now copy from cache to release directory (rsync handles deletions)
	fmt.Printf("\n")
	out.Info("  Syncing cache to release directory (with deletions)...")

	// Use rsync on the remote server to copy from cache to release
	// --delete removes files that only exist in release (not in cache)
	copyCmd := fmt.Sprintf("rsync -rlt --no-perms --delete %s/ %s/", ssh.Quote(cachePath), ssh.Quote(s.remotePath))
	output, err := s.client.RunCommand(copyCmd)
	if err != nil {
		return fmt.Errorf("failed to copy from cache to release: %w (output: %s)", err, output)
	}

	out.Success("Release directory synchronized")

	return nil
}

// RemoteFileInfo holds remote file metadata
type RemoteFileInfo struct {
	Size int64
}

// getRemoteFileIndex builds an index of all files in remote cache with their size
func (s *Syncer) getRemoteFileIndex(cachePath string) (map[string]RemoteFileInfo, error) {
	// Use find + stat to get all file info in one command
	// Output format: path<tab>size
	cmd := fmt.Sprintf("cd %s && find . -type f -exec stat -c '%%n\t%%s' {} + 2>/dev/null || true", ssh.Quote(cachePath))
	output, err := s.client.RunCommand(cmd)
	if err != nil {
		return nil, err
	}

	index := make(map[string]RemoteFileInfo)
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			continue
		}

		// Remove leading "./" from path
		path := strings.TrimPrefix(parts[0], "./")

		var size int64
		// #nosec G104 -- Sscanf errors are acceptable here, defaults to 0
		fmt.Sscanf(parts[1], "%d", &size)

		index[path] = RemoteFileInfo{Size: size}
	}

	return index, nil
}

func (s *Syncer) manifestPath() string {
	return filepath.Join(s.deployPath, cacheManifestFile)
}

// readCacheManifest returns the stored checksums. Any failure yields an empty
// map, which re-uploads everything.
func (s *Syncer) readCacheManifest() map[string]string {
	cmd := fmt.Sprintf("cat %s 2>/dev/null || true", ssh.Quote(s.manifestPath()))
	output, err := s.client.RunCommand(cmd)
	if err != nil {
		return map[string]string{}
	}

	manifest := make(map[string]string)
	if err := json.Unmarshal([]byte(output), &manifest); err != nil {
		return map[string]string{}
	}
	return manifest
}

// writeCacheManifest uploads the checksum manifest for the synced files.
func (s *Syncer) writeCacheManifest(files []FileInfo) error {
	manifest := make(map[string]string, len(files))
	for _, file := range files {
		manifest[file.RelPath] = file.Checksum
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal cache manifest: %w", err)
	}

	tmp, err := os.CreateTemp("", "shippy-cache-manifest-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temp manifest: %w", err)
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("failed to write temp manifest: %w", err)
	}

	return s.client.UploadFile(tmp.Name(), s.manifestPath(), 0o600)
}

// uploadWithProgress uploads files with progress reporting
func (s *Syncer) uploadWithProgress(filesToUpload []FileInfo, cachePath string, out *ui.Output) error {
	if len(filesToUpload) == 0 {
		out.Info("  No files to upload - all files are cached")
		return nil
	}

	fmt.Printf("\n")
	// #nosec G104 -- Printf errors in UI output can be safely ignored
	out.Blue.Printf("→ Uploading %d changed/new files to cache\n", len(filesToUpload))
	fmt.Printf("\n")

	transferred := 0
	var totalSize int64

	for i, file := range filesToUpload {
		remoteCachePath := filepath.Join(cachePath, file.RelPath)

		if s.verbose {
			// Verbose mode: show each file
			// #nosec G104 -- Printf errors in UI output can be safely ignored
			out.Yellow.Printf("  [%d/%d] Uploading: %s\n", i+1, len(filesToUpload), file.RelPath)
		} else {
			// Progress bar mode: show progress with current file
			out.PrintProgressBar(i+1, len(filesToUpload), file.RelPath)
		}

		if err := s.client.UploadFile(file.FullPath, remoteCachePath, file.Mode); err != nil {
			if !s.verbose {
				out.ClearLine()
			}
			return errors.FileUploadError(file.RelPath, err)
		}

		transferred++
		totalSize += file.Size
	}

	if !s.verbose {
		out.ClearLine()
	}

	fmt.Printf("\n")
	out.Success("Uploaded %d files to cache (%.2f MB)", transferred, float64(totalSize)/(1024*1024))
	return nil
}

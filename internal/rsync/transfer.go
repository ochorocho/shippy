package rsync

import (
	"fmt"
	"path/filepath"

	"shippy/internal/cache"
	"shippy/internal/ssh"
	"shippy/internal/ui"
)

// Syncer handles file synchronization over SSH
type Syncer struct {
	client     *ssh.Client
	remotePath string
	verbose    bool
	cache      *cache.Cache
}

// NewSyncer creates a new file syncer
func NewSyncer(client *ssh.Client, remotePath string, verbose bool, c *cache.Cache) *Syncer {
	return &Syncer{
		client:     client,
		remotePath: remotePath,
		verbose:    verbose,
		cache:      c,
	}
}

// Sync synchronizes files to the remote server
func (s *Syncer) Sync(files []FileInfo) error {
	out := ui.New()

	fmt.Printf("\n")
	out.Blue.Printf("→ Syncing %d files to remote server\n", len(files))

	// Check cache to determine which files need uploading
	filesToUpload := make([]FileInfo, 0, len(files))
	skipped := 0

	if s.cache != nil {
		for _, file := range files {
			needsUpload, err := s.cache.NeedsUploadWithChecksum(file.RelPath, file.FullPath, file.Size, file.ModTime)
			if err != nil {
				// If cache check fails, upload the file to be safe
				filesToUpload = append(filesToUpload, file)
				continue
			}

			if needsUpload {
				filesToUpload = append(filesToUpload, file)
			} else {
				skipped++
			}
		}

		if skipped > 0 {
			out.Info("  %d files unchanged (cached), %d files to upload", skipped, len(filesToUpload))
		}
	} else {
		// No cache, upload all files
		filesToUpload = files
	}

	fmt.Printf("\n")

	// Create base directory
	if err := s.client.MkdirAll(s.remotePath); err != nil {
		return fmt.Errorf("failed to create remote directory: %w", err)
	}

	transferred := 0
	var totalSize int64

	for i, file := range filesToUpload {
		remotePath := filepath.Join(s.remotePath, file.RelPath)

		if s.verbose {
			// Verbose mode: show each file
			out.Yellow.Printf("  [%d/%d] Uploading: %s\n", i+1, len(filesToUpload), file.RelPath)
		} else {
			// Progress bar mode: show progress with current file
			out.PrintProgressBar(i+1, len(filesToUpload), file.RelPath)
		}

		if err := s.client.UploadFile(file.FullPath, remotePath, file.Mode); err != nil {
			if !s.verbose {
				out.ClearLine()
			}
			return fmt.Errorf("failed to upload %s: %w", file.RelPath, err)
		}

		// Mark as uploaded in cache
		if s.cache != nil {
			if err := s.cache.MarkUploaded(file.RelPath, file.FullPath, file.Size, file.ModTime); err != nil {
				// Non-fatal: log but continue
				if s.verbose {
					out.Info("  Warning: failed to update cache for %s: %v", file.RelPath, err)
				}
			}
		}

		transferred++
		totalSize += file.Size
	}

	if !s.verbose {
		out.ClearLine()
	}

	fmt.Printf("\n")
	if skipped > 0 {
		out.Success("Sync complete: %d files transferred, %d files skipped (%.2f MB)", transferred, skipped, float64(totalSize)/(1024*1024))
	} else {
		out.Success("Sync complete: %d files transferred (%.2f MB)", transferred, float64(totalSize)/(1024*1024))
	}

	return nil
}

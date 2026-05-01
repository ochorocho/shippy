package backup

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ZipBuilder creates a ZIP archive incrementally
type ZipBuilder struct {
	file   *os.File
	writer *zip.Writer
	path   string
	closed bool
}

// NewZipBuilder creates a new ZIP archive at the given path
func NewZipBuilder(path string) (*ZipBuilder, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create output directory '%s': %w", dir, err)
	}

	// #nosec G304 -- Path comes from user config or CLI flag
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create ZIP file: %w", err)
	}

	return &ZipBuilder{
		file:   f,
		writer: zip.NewWriter(f),
		path:   path,
	}, nil
}

// CreateEntry creates a new file entry in the ZIP and returns a writer for its contents
func (z *ZipBuilder) CreateEntry(name string) (io.Writer, error) {
	return z.writer.Create(name)
}

// AddFile adds a file from a reader with the given name and size
func (z *ZipBuilder) AddFile(name string, r io.Reader) error {
	w, err := z.writer.Create(name)
	if err != nil {
		return fmt.Errorf("failed to create ZIP entry '%s': %w", name, err)
	}
	if _, err := io.Copy(w, r); err != nil {
		return fmt.Errorf("failed to write ZIP entry '%s': %w", name, err)
	}
	return nil
}

// Close finalizes the ZIP archive. Safe to call multiple times.
func (z *ZipBuilder) Close() error {
	if z.closed {
		return nil
	}
	z.closed = true

	if err := z.writer.Close(); err != nil {
		_ = z.file.Close()
		return fmt.Errorf("failed to finalize ZIP: %w", err)
	}
	return z.file.Close()
}

// Size returns the size of the ZIP file on disk
func (z *ZipBuilder) Size() (int64, error) {
	info, err := os.Stat(z.path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

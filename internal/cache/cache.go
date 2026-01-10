package cache

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// FileEntry represents a cached file entry
type FileEntry struct {
	Path         string    `json:"path"`
	Checksum     string    `json:"checksum"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	LastUploaded time.Time `json:"last_uploaded"`
}

// Cache manages file transfer cache
type Cache struct {
	cacheDir string
	entries  map[string]*FileEntry
}

// New creates a new cache manager
func New(projectRoot string) (*Cache, error) {
	cacheDir := filepath.Join(projectRoot, ".cache")

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &Cache{
		cacheDir: cacheDir,
		entries:  make(map[string]*FileEntry),
	}, nil
}

// Load loads the cache from disk for a specific host
func (c *Cache) Load(hostName string) error {
	cachePath := filepath.Join(c.cacheDir, fmt.Sprintf("%s.json", hostName))

	data, err := os.ReadFile(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache doesn't exist yet, start fresh
			return nil
		}
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	var entries []*FileEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to parse cache file: %w", err)
	}

	// Build map from entries
	c.entries = make(map[string]*FileEntry)
	for _, entry := range entries {
		c.entries[entry.Path] = entry
	}

	return nil
}

// Save saves the cache to disk for a specific host
func (c *Cache) Save(hostName string) error {
	cachePath := filepath.Join(c.cacheDir, fmt.Sprintf("%s.json", hostName))

	// Convert map to slice
	entries := make([]*FileEntry, 0, len(c.entries))
	for _, entry := range c.entries {
		entries = append(entries, entry)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// NeedsUpload checks if a file needs to be uploaded
func (c *Cache) NeedsUpload(filePath string, size int64, modTime time.Time) (bool, error) {
	entry, exists := c.entries[filePath]
	if !exists {
		// File not in cache, needs upload
		return true, nil
	}

	// Check if file has changed
	if entry.Size != size || entry.ModTime != modTime {
		// Size or modification time changed, needs upload
		return true, nil
	}

	// File appears unchanged
	return false, nil
}

// NeedsUploadWithChecksum checks if a file needs upload using checksum verification
func (c *Cache) NeedsUploadWithChecksum(filePath string, fullPath string, size int64, modTime time.Time) (bool, error) {
	entry, exists := c.entries[filePath]
	if !exists {
		// File not in cache, needs upload
		return true, nil
	}

	// Quick check: size or modification time changed
	if entry.Size != size || entry.ModTime != modTime {
		// Changed, needs upload
		return true, nil
	}

	// Size and modTime match, but verify with checksum for accuracy
	checksum, err := calculateChecksum(fullPath)
	if err != nil {
		// If we can't calculate checksum, assume it needs upload
		return true, err
	}

	if entry.Checksum != checksum {
		// Checksum changed, needs upload
		return true, nil
	}

	// File is identical to cached version
	return false, nil
}

// MarkUploaded marks a file as uploaded and updates the cache
func (c *Cache) MarkUploaded(filePath string, fullPath string, size int64, modTime time.Time) error {
	checksum, err := calculateChecksum(fullPath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	c.entries[filePath] = &FileEntry{
		Path:         filePath,
		Checksum:     checksum,
		Size:         size,
		ModTime:      modTime,
		LastUploaded: time.Now(),
	}

	return nil
}

// GetStats returns cache statistics
func (c *Cache) GetStats() (total int, totalSize int64) {
	total = len(c.entries)
	for _, entry := range c.entries {
		totalSize += entry.Size
	}
	return
}

// Clear clears all cache entries
func (c *Cache) Clear() {
	c.entries = make(map[string]*FileEntry)
}

// calculateChecksum calculates MD5 checksum of a file
func calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

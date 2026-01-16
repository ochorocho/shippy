package rsync

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// FileInfo represents a file to be synced
type FileInfo struct {
	RelPath  string
	FullPath string
	Size     int64
	Mode     os.FileMode
	ModTime  time.Time
	Checksum string
}

// SyncOptions contains options for file synchronization
type SyncOptions struct {
	SourceDir       string
	ExcludePatterns []string
	IncludePatterns []string
	UseGitignore    bool
}

// Scanner scans files in a directory and determines which should be synced
type Scanner struct {
	opts           SyncOptions
	excludes       []gitignore.Pattern
	includes       []gitignore.Pattern
	gitignoreCache map[string][]gitignore.Pattern // Cache of .gitignore patterns by directory
}

// NewScanner creates a new file scanner
func NewScanner(opts SyncOptions) (*Scanner, error) {
	s := &Scanner{
		opts:           opts,
		gitignoreCache: make(map[string][]gitignore.Pattern),
	}

	// Parse exclude patterns
	for _, pattern := range opts.ExcludePatterns {
		s.excludes = append(s.excludes, gitignore.ParsePattern(pattern, nil))
	}

	// Parse include patterns - prefix with ! to create negation patterns
	// This allows them to override .gitignore exclusions
	for _, pattern := range opts.IncludePatterns {
		negatePattern := pattern
		if !strings.HasPrefix(pattern, "!") {
			negatePattern = "!" + pattern
		}
		s.includes = append(s.includes, gitignore.ParsePattern(negatePattern, nil))
	}

	// Load root .gitignore if requested
	if opts.UseGitignore {
		if err := s.loadGitignoreFromDir(opts.SourceDir); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// loadGitignoreFromDir loads .gitignore patterns from a specific directory
func (s *Scanner) loadGitignoreFromDir(dir string) error {
	// Check if already cached
	if _, exists := s.gitignoreCache[dir]; exists {
		return nil
	}

	gitignorePath := filepath.Join(dir, ".gitignore")
	var patterns []gitignore.Pattern

	// #nosec G304 -- gitignorePath is constructed from scanned directory within project
	file, err := os.Open(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No .gitignore file in this directory, that's okay
			s.gitignoreCache[dir] = patterns
			return nil
		}
		return fmt.Errorf("failed to open .gitignore: %w", err)
	}
	defer file.Close()

	// Calculate relative path from source directory to this .gitignore directory
	// This ensures patterns are scoped to their directory, not applied globally
	relDir, err := filepath.Rel(s.opts.SourceDir, dir)
	if err != nil {
		return fmt.Errorf("failed to get relative path for .gitignore domain: %w", err)
	}

	// Convert to forward slashes and split into path components for domain
	var domain []string
	if relDir != "." {
		domain = strings.Split(filepath.ToSlash(relDir), "/")
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Parse pattern with domain so it only applies relative to this directory
		pattern := gitignore.ParsePattern(line, domain)
		patterns = append(patterns, pattern)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to parse .gitignore: %w", err)
	}

	// Add patterns to global excludes
	s.excludes = append(s.excludes, patterns...)

	// Cache for this directory
	s.gitignoreCache[dir] = patterns

	return nil
}

// Scan scans the source directory and returns a list of files to sync
func (s *Scanner) Scan() ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(s.opts.SourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the source directory itself
		if path == s.opts.SourceDir {
			return nil
		}

		// When entering a directory, check for .gitignore and load it
		if info.IsDir() && s.opts.UseGitignore {
			if err := s.loadGitignoreFromDir(path); err != nil {
				return fmt.Errorf("failed to load .gitignore from %s: %w", path, err)
			}
		}

		// Get relative path
		relPath, err := filepath.Rel(s.opts.SourceDir, path)
		if err != nil {
			return err
		}

		// Convert to forward slashes for pattern matching
		relPath = filepath.ToSlash(relPath)

		// Check if file should be included
		if !s.shouldInclude(relPath, info.IsDir()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories in the file list (we only transfer files)
		if info.IsDir() {
			return nil
		}

		// Skip symlinks - they should not be transferred
		// Symlinks in source are usually development artifacts
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// Calculate checksum for regular files
		checksum := ""
		if info.Mode().IsRegular() {
			checksum, err = calculateChecksum(path)
			if err != nil {
				return fmt.Errorf("failed to calculate checksum for %s: %w", path, err)
			}
		}

		files = append(files, FileInfo{
			RelPath:  relPath,
			FullPath: path,
			Size:     info.Size(),
			Mode:     info.Mode(),
			ModTime:  info.ModTime(),
			Checksum: checksum,
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// shouldInclude determines if a file should be included based on patterns
func (s *Scanner) shouldInclude(relPath string, isDir bool) bool {
	pathParts := strings.Split(relPath, "/")

	// Check if any exclude pattern matches
	excluded := false
	for _, pattern := range s.excludes {
		result := pattern.Match(pathParts, isDir)
		if result == gitignore.Exclude {
			excluded = true
			break
		}
	}

	// If excluded, check if any include pattern overrides the exclusion
	if excluded {
		for _, pattern := range s.includes {
			result := pattern.Match(pathParts, isDir)
			// Include result (negation pattern) overrides exclusion
			if result == gitignore.Include {
				return true
			}
		}
		return false // Excluded and no override
	}

	// Not excluded - include by default
	return true
}

// calculateChecksum calculates SHA256 checksum of a file
func calculateChecksum(path string) (string, error) {
	// #nosec G304 -- Path comes from scanned local files within project directory
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

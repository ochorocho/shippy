package rsync

import (
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
	SourceDir string
	// ExcludePatterns are carve-outs: they always win over includes. This is the
	// built-in junk list plus any user-defined exclude patterns.
	ExcludePatterns []string
	// IncludePatterns form the allowlist. Deployment is deny-by-default: a path is
	// only synced when it (or an ancestor directory) matches an include pattern.
	IncludePatterns []string
}

// Scanner scans files in a directory and determines which should be synced.
//
// Selection is deny-by-default (an allowlist): nothing is synced unless it
// matches an include pattern. Excludes are carve-outs that win over includes, so
// junk (.git/, node_modules/, ...) never ships even inside an included directory.
type Scanner struct {
	opts     SyncOptions
	excludes []gitignore.Pattern
	includes []gitignore.Pattern
	// rawIncludes keeps the original include pattern strings (without the leading
	// "!"/"/") so we can decide whether to descend into a directory that is an
	// ancestor of an anchored include target (e.g. descend into "public" for the
	// include "public/index.php").
	rawIncludes []string
}

// NewScanner creates a new file scanner
func NewScanner(opts SyncOptions) (*Scanner, error) {
	s := &Scanner{opts: opts}

	// Parse exclude patterns (carve-outs)
	for _, pattern := range opts.ExcludePatterns {
		s.excludes = append(s.excludes, gitignore.ParsePattern(pattern, nil))
	}

	// Parse include patterns (allowlist). They are stored as gitignore negation
	// patterns (prefixed with "!") so the matcher reports gitignore.Include on a
	// match. The original strings are kept in rawIncludes for directory descent.
	for _, pattern := range opts.IncludePatterns {
		s.rawIncludes = append(s.rawIncludes, includeLiteral(pattern))
		negatePattern := pattern
		if !strings.HasPrefix(pattern, "!") {
			negatePattern = "!" + pattern
		}
		s.includes = append(s.includes, gitignore.ParsePattern(negatePattern, nil))
	}

	return s, nil
}

// includeLiteral strips the leading "!" (negation) and "/" (anchor) from an
// include pattern, leaving the path used for ancestor-directory descent checks.
func includeLiteral(pattern string) string {
	pattern = strings.TrimPrefix(pattern, "!")
	pattern = strings.TrimPrefix(pattern, "/")
	return pattern
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

		// Get relative path
		relPath, err := filepath.Rel(s.opts.SourceDir, path)
		if err != nil {
			return err
		}

		// Convert to forward slashes for pattern matching
		relPath = filepath.ToSlash(relPath)

		// Directories are never transferred themselves; we only decide whether to
		// walk into them. Descend only when the directory could contain an included
		// file, so a broad allowlist still prunes junk (e.g. node_modules/).
		if info.IsDir() {
			if !s.descendInto(relPath) {
				return filepath.SkipDir
			}
			return nil
		}

		// Deny-by-default: a file is synced only when the allowlist includes it.
		if !s.shouldInclude(relPath, false) {
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

// shouldInclude determines whether a path should be synced. Selection is
// deny-by-default:
//
//  1. Carve-outs (exclude patterns) win — if the path is excluded, it never ships.
//  2. Otherwise the path ships only when the allowlist (include patterns) matches.
//  3. Anything else is denied.
func (s *Scanner) shouldInclude(relPath string, isDir bool) bool {
	pathParts := strings.Split(relPath, "/")

	// Carve-outs always win over includes.
	if s.isExcluded(pathParts, isDir) {
		return false
	}

	// Allowlist: ship only when an include pattern matches.
	for _, pattern := range s.includes {
		if pattern.Match(pathParts, isDir) == gitignore.Include {
			return true
		}
	}

	// Deny by default.
	return false
}

// isExcluded reports whether the path matches a carve-out pattern, using
// gitignore last-match-wins semantics (a later "!" negation can re-include).
func (s *Scanner) isExcluded(pathParts []string, isDir bool) bool {
	excluded := false
	for _, pattern := range s.excludes {
		switch pattern.Match(pathParts, isDir) {
		case gitignore.Exclude:
			excluded = true
		case gitignore.Include:
			excluded = false
		}
	}
	return excluded
}

// descendInto reports whether the walk should enter a directory. It descends
// when the directory could contain an included file, so a carved-out directory
// (e.g. node_modules/) is pruned while ancestors of an include target are not.
func (s *Scanner) descendInto(dirRelPath string) bool {
	pathParts := strings.Split(dirRelPath, "/")

	// Carve-outs prune the whole subtree.
	if s.isExcluded(pathParts, true) {
		return false
	}

	// The directory itself is directly included (e.g. include "public/").
	for _, pattern := range s.includes {
		if pattern.Match(pathParts, true) == gitignore.Include {
			return true
		}
	}

	// The directory is an ancestor of an include target. A single-segment include
	// (bare name like "vendor" or "public/") can match at any depth, so we must
	// descend everywhere; a multi-segment include ("public/index.php") only
	// requires descending into its literal parent directories.
	for _, raw := range s.rawIncludes {
		literal := strings.TrimSuffix(raw, "/")
		if !strings.Contains(literal, "/") {
			return true
		}
		if isSegmentPrefix(pathParts, raw) {
			return true
		}
	}

	return false
}

// isSegmentPrefix reports whether dirParts is a leading path-segment prefix of
// the include pattern's literal path (e.g. ["public"] is a prefix of
// "public/index.php", so we descend into "public").
func isSegmentPrefix(dirParts []string, pattern string) bool {
	patternParts := strings.Split(strings.TrimSuffix(pattern, "/"), "/")
	if len(dirParts) >= len(patternParts) {
		return false
	}
	for i, part := range dirParts {
		if part != patternParts[i] {
			return false
		}
	}
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

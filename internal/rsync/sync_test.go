package rsync

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

func makeScanner(excludePatterns, includePatterns []string) *Scanner {
	s := &Scanner{}
	for _, p := range excludePatterns {
		s.excludes = append(s.excludes, gitignore.ParsePattern(p, nil))
	}
	for _, p := range includePatterns {
		s.rawIncludes = append(s.rawIncludes, includeLiteral(p))
		neg := p
		if len(p) == 0 || p[0] != '!' {
			neg = "!" + p
		}
		s.includes = append(s.includes, gitignore.ParsePattern(neg, nil))
	}
	return s
}

// TestShouldInclude verifies the deny-by-default (allowlist) selection model:
// a path ships only when an include matches and no carve-out excludes it.
func TestShouldInclude(t *testing.T) {
	tests := []struct {
		name     string
		excludes []string
		includes []string
		path     string
		isDir    bool
		want     bool
	}{
		{
			name: "no patterns ships nothing (deny by default)",
			path: "src/main.go",
			want: false,
		},
		{
			name:     "included file ships",
			includes: []string{"public/"},
			path:     "public/index.php",
			want:     true,
		},
		{
			name:     "file outside allowlist is denied",
			includes: []string{"public/"},
			path:     "config/settings.php",
			want:     false,
		},
		{
			name:     "included directory ships its subtree at any depth",
			includes: []string{"public/"},
			path:     "public/css/theme/main.css",
			want:     true,
		},
		{
			name:     "bare-name include ships vendor without gitignore",
			includes: []string{"vendor/"},
			path:     "vendor/acme/lib.php",
			want:     true,
		},
		{
			name:     "root file include ships",
			includes: []string{"composer.json"},
			path:     "composer.json",
			want:     true,
		},
		{
			name:     "carve-out wins over include",
			includes: []string{"public/"},
			excludes: []string{"public/typo3temp/"},
			path:     "public/typo3temp/cache.php",
			want:     false,
		},
		{
			name:     "junk-list carve-out wins over broad include",
			includes: []string{"*"},
			excludes: []string{"node_modules/"},
			path:     "public/theme/node_modules/pkg/index.js",
			want:     false,
		},
		{
			name:     "wildcard include ships everything not carved out",
			includes: []string{"*"},
			path:     "some/deep/file.txt",
			want:     true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := makeScanner(tc.excludes, tc.includes)
			got := s.shouldInclude(tc.path, tc.isDir)
			if got != tc.want {
				t.Errorf("shouldInclude(%q, dir=%v) = %v, want %v", tc.path, tc.isDir, got, tc.want)
			}
		})
	}
}

// TestDescendInto verifies directory pruning: carved-out dirs are pruned, and
// ancestors of an include target are descended into (the SkipDir footgun fix).
func TestDescendInto(t *testing.T) {
	tests := []struct {
		name     string
		excludes []string
		includes []string
		dir      string
		want     bool
	}{
		{
			name:     "descend into directly included dir",
			includes: []string{"public/"},
			dir:      "public",
			want:     true,
		},
		{
			name:     "descend into ancestor of anchored nested include",
			includes: []string{"public/index.php"},
			dir:      "public",
			want:     true,
		},
		{
			name:     "do not descend into unrelated dir for anchored include",
			includes: []string{"public/index.php"},
			dir:      "config",
			want:     false,
		},
		{
			name:     "prune carved-out dir even when included",
			includes: []string{"*"},
			excludes: []string{"node_modules/"},
			dir:      "node_modules",
			want:     false,
		},
		{
			name: "no includes prunes everything",
			dir:  "public",
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := makeScanner(tc.excludes, tc.includes)
			if got := s.descendInto(tc.dir); got != tc.want {
				t.Errorf("descendInto(%q) = %v, want %v", tc.dir, got, tc.want)
			}
		})
	}
}

// TestScanAllowlist is an end-to-end scan over a real temp tree. It proves the
// SkipDir footgun is fixed: an anchored nested include (public/index.php) is
// actually reached, and non-included paths are pruned.
func TestScanAllowlist(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, root, "public/index.php", "a")
	mustWrite(t, root, "public/css/main.css", "b")
	mustWrite(t, root, "config/settings.php", "c")
	mustWrite(t, root, "composer.json", "d")
	mustWrite(t, root, ".git/HEAD", "e")
	mustWrite(t, root, "node_modules/pkg/index.js", "f")

	t.Run("nested file include is reached (footgun regression)", func(t *testing.T) {
		got := scanRelPaths(t, root, []string{"node_modules/"}, []string{"public/index.php"})
		want := []string{"public/index.php"}
		assertPaths(t, got, want)
	})

	t.Run("directory allowlist ships subtree and prunes the rest", func(t *testing.T) {
		got := scanRelPaths(t, root,
			[]string{".git/", "node_modules/"},
			[]string{"public/", "composer.json"},
		)
		want := []string{"composer.json", "public/css/main.css", "public/index.php"}
		assertPaths(t, got, want)
	})

	t.Run("empty allowlist ships nothing", func(t *testing.T) {
		got := scanRelPaths(t, root, nil, nil)
		assertPaths(t, got, []string{})
	})
}

func scanRelPaths(t *testing.T, root string, excludes, includes []string) []string {
	t.Helper()
	s, err := NewScanner(SyncOptions{
		SourceDir:       root,
		ExcludePatterns: excludes,
		IncludePatterns: includes,
	})
	if err != nil {
		t.Fatalf("NewScanner: %v", err)
	}
	files, err := s.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	paths := make([]string, 0, len(files))
	for _, f := range files {
		paths = append(paths, f.RelPath)
	}
	sort.Strings(paths)
	return paths
}

func assertPaths(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("scanned %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("scanned %v, want %v", got, want)
		}
	}
}

func mustWrite(t *testing.T, root, rel, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

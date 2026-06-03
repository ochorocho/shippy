package rsync

import (
	"testing"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

func makeScanner(excludePatterns, includePatterns []string) *Scanner {
	s := &Scanner{gitignoreCache: make(map[string][]gitignore.Pattern)}
	for _, p := range excludePatterns {
		s.excludes = append(s.excludes, gitignore.ParsePattern(p, nil))
	}
	for _, p := range includePatterns {
		neg := p
		if len(p) == 0 || p[0] != '!' {
			neg = "!" + p
		}
		s.includes = append(s.includes, gitignore.ParsePattern(neg, nil))
	}
	return s
}

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
			name:    "no patterns includes everything",
			path:    "src/main.go",
			want:    true,
		},
		{
			name:     "excluded file is skipped",
			excludes: []string{"*.log"},
			path:     "app.log",
			want:     false,
		},
		{
			name:     "non-matching exclude keeps file",
			excludes: []string{"*.log"},
			path:     "src/main.go",
			want:     true,
		},
		{
			name:     "negation after exclude re-includes file",
			excludes: []string{"vendor", "!vendor/acme"},
			path:     "vendor/acme",
			isDir:    true,
			want:     true,
		},
		{
			name:     "later exclude overrides earlier negation",
			excludes: []string{"!vendor/acme", "vendor"},
			path:     "vendor/acme",
			isDir:    true,
			want:     false,
		},
		{
			name:     "config include overrides exclude",
			excludes: []string{"*.env"},
			includes: []string{".env.example"},
			path:     ".env.example",
			want:     true,
		},
		{
			name:     "config include does not affect non-excluded file",
			excludes: []string{"*.log"},
			includes: []string{"keep.txt"},
			path:     "keep.txt",
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

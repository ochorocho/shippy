package ssh

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetUserHomeDir(t *testing.T) {
	home, err := getUserHomeDir()
	if err != nil {
		t.Fatalf("getUserHomeDir() failed: %v", err)
	}

	if home == "" {
		t.Error("getUserHomeDir() returned empty string")
	}

	// Verify it's an absolute path
	if !filepath.IsAbs(home) {
		t.Errorf("getUserHomeDir() returned non-absolute path: %s", home)
	}

	// Verify it matches os.UserHomeDir()
	expectedHome, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("os.UserHomeDir() failed: %v", err)
	}

	if home != expectedHome {
		t.Errorf("getUserHomeDir() = %s, want %s", home, expectedHome)
	}
}

func TestGetDefaultKnownHostsFile(t *testing.T) {
	knownHosts, err := getDefaultKnownHostsFile()
	if err != nil {
		t.Fatalf("getDefaultKnownHostsFile() failed: %v", err)
	}

	if knownHosts == "" {
		t.Error("getDefaultKnownHostsFile() returned empty string")
	}

	// Verify it's an absolute path
	if !filepath.IsAbs(knownHosts) {
		t.Errorf("getDefaultKnownHostsFile() returned non-absolute path: %s", knownHosts)
	}

	// Verify it ends with .ssh/known_hosts
	expectedSuffix := filepath.Join(".ssh", "known_hosts")
	if !filepath.IsAbs(knownHosts) || filepath.Base(filepath.Dir(knownHosts)) != ".ssh" || filepath.Base(knownHosts) != "known_hosts" {
		t.Errorf("getDefaultKnownHostsFile() = %s, expected to end with %s", knownHosts, expectedSuffix)
	}

	// Verify it starts with home directory
	home, err := getUserHomeDir()
	if err != nil {
		t.Fatalf("getUserHomeDir() failed: %v", err)
	}

	expectedPath := filepath.Join(home, ".ssh", "known_hosts")
	if knownHosts != expectedPath {
		t.Errorf("getDefaultKnownHostsFile() = %s, want %s", knownHosts, expectedPath)
	}
}

func TestExpandHomePath(t *testing.T) {
	home, err := getUserHomeDir()
	if err != nil {
		t.Fatalf("getUserHomeDir() failed: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "home directory with file",
			input:    "~/.ssh/id_rsa",
			expected: filepath.Join(home, ".ssh", "id_rsa"),
			wantErr:  false,
		},
		{
			name:     "home directory only",
			input:    "~/",
			expected: home,
			wantErr:  false,
		},
		{
			name:     "home directory with subdirectory",
			input:    "~/Documents/test.txt",
			expected: filepath.Join(home, "Documents", "test.txt"),
			wantErr:  false,
		},
		{
			name:     "absolute path (no expansion)",
			input:    "/usr/local/bin",
			expected: "/usr/local/bin",
			wantErr:  false,
		},
		{
			name:     "relative path (no expansion)",
			input:    "relative/path/file.txt",
			expected: "relative/path/file.txt",
			wantErr:  false,
		},
		{
			name:     "empty path",
			input:    "",
			expected: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandHomePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("expandHomePath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("expandHomePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandHomePathEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "tilde without slash (no expansion)",
			input:    "~test",
			expected: "~test",
			wantErr:  false,
		},
		{
			name:     "tilde in middle (no expansion)",
			input:    "/path/~/file",
			expected: "/path/~/file",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := expandHomePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("expandHomePath(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("expandHomePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

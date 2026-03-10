package upload

import (
	"testing"
)

func TestParseGitRemoteURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantHost    string
		wantProject string
		wantErr     bool
	}{
		{
			name:        "SSH format simple",
			url:         "git@gitlab.com:namespace/project.git",
			wantHost:    "gitlab.com",
			wantProject: "namespace/project",
		},
		{
			name:        "SSH format with subgroup",
			url:         "git@gitlab.example.com:group/subgroup/project.git",
			wantHost:    "gitlab.example.com",
			wantProject: "group/subgroup/project",
		},
		{
			name:        "SSH format without .git suffix",
			url:         "git@gitlab.com:namespace/project",
			wantHost:    "gitlab.com",
			wantProject: "namespace/project",
		},
		{
			name:        "HTTPS format simple",
			url:         "https://gitlab.com/namespace/project.git",
			wantHost:    "gitlab.com",
			wantProject: "namespace/project",
		},
		{
			name:        "HTTPS format with subgroup",
			url:         "https://gitlab.example.com/group/subgroup/project.git",
			wantHost:    "gitlab.example.com",
			wantProject: "group/subgroup/project",
		},
		{
			name:        "HTTPS format without .git suffix",
			url:         "https://gitlab.com/namespace/project",
			wantHost:    "gitlab.com",
			wantProject: "namespace/project",
		},
		{
			name:        "GitHub SSH",
			url:         "git@github.com:user/repo.git",
			wantHost:    "github.com",
			wantProject: "user/repo",
		},
		{
			name:        "GitHub HTTPS",
			url:         "https://github.com/user/repo.git",
			wantHost:    "github.com",
			wantProject: "user/repo",
		},
		{
			name:    "invalid SSH format",
			url:     "git@gitlab.com",
			wantErr: true,
		},
		{
			name:    "HTTPS no path",
			url:     "https://gitlab.com/",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseGitRemoteURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseGitRemoteURL(%q) expected error, got nil", tt.url)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseGitRemoteURL(%q) unexpected error: %v", tt.url, err)
			}
			if info.Host != tt.wantHost {
				t.Errorf("Host = %q, want %q", info.Host, tt.wantHost)
			}
			if info.ProjectPath != tt.wantProject {
				t.Errorf("ProjectPath = %q, want %q", info.ProjectPath, tt.wantProject)
			}
		})
	}
}

func TestGitRemoteInfo_ProjectName(t *testing.T) {
	tests := []struct {
		projectPath string
		wantName    string
	}{
		{"namespace/project", "project"},
		{"group/subgroup/project", "project"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		t.Run(tt.projectPath, func(t *testing.T) {
			info := &GitRemoteInfo{ProjectPath: tt.projectPath}
			got := info.ProjectName()
			if got != tt.wantName {
				t.Errorf("ProjectName() = %q, want %q", got, tt.wantName)
			}
		})
	}
}

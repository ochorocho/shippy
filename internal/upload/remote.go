package upload

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/go-git/go-git/v5"
)

// GitRemoteInfo holds parsed git remote information
type GitRemoteInfo struct {
	Host        string // e.g., "gitlab.com"
	ProjectPath string // e.g., "namespace/project-name"
}

// ParseGitRemoteFromRepo reads the "origin" remote URL from a git repository
// and parses the host and project path.
func ParseGitRemoteFromRepo(repoPath string) (*GitRemoteInfo, error) {
	repo, err := git.PlainOpenWithOptions(repoPath, &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open git repository: %w", err)
	}

	remote, err := repo.Remote("origin")
	if err != nil {
		return nil, fmt.Errorf("failed to get 'origin' remote: %w", err)
	}

	urls := remote.Config().URLs
	if len(urls) == 0 {
		return nil, fmt.Errorf("origin remote has no URLs configured")
	}

	return ParseGitRemoteURL(urls[0])
}

// ParseGitRemoteURL parses a git remote URL (SSH or HTTPS) into host and project path.
func ParseGitRemoteURL(remoteURL string) (*GitRemoteInfo, error) {
	// Handle SSH format: git@gitlab.com:namespace/project.git
	if strings.HasPrefix(remoteURL, "git@") {
		parts := strings.SplitN(remoteURL, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid SSH git URL: %s", remoteURL)
		}
		host := strings.TrimPrefix(parts[0], "git@")
		path := strings.TrimSuffix(parts[1], ".git")
		return &GitRemoteInfo{Host: host, ProjectPath: path}, nil
	}

	// Handle HTTPS format: https://gitlab.com/namespace/project.git
	u, err := url.Parse(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse git URL: %w", err)
	}

	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	if path == "" {
		return nil, fmt.Errorf("git URL has no project path: %s", remoteURL)
	}

	return &GitRemoteInfo{Host: u.Host, ProjectPath: path}, nil
}

// ProjectName returns the last segment of the project path (the project name)
func (g *GitRemoteInfo) ProjectName() string {
	parts := strings.Split(g.ProjectPath, "/")
	return parts[len(parts)-1]
}

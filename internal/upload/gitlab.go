package upload

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// GitLabUploader uploads files to GitLab's Generic Packages registry
type GitLabUploader struct {
	baseURL     string // e.g., "https://gitlab.com"
	projectPath string // e.g., "namespace/project-name"
	token       string
	client      *http.Client
}

// NewGitLabUploader creates a new GitLab uploader
func NewGitLabUploader(host, projectPath, token string) *GitLabUploader {
	return &GitLabUploader{
		baseURL:     fmt.Sprintf("https://%s", host),
		projectPath: projectPath,
		token:       token,
		client: &http.Client{
			Timeout: 30 * time.Minute,
		},
	}
}

// Upload uploads a file to the GitLab Generic Packages registry
func (g *GitLabUploader) Upload(filePath string, opts UploadOptions) (*UploadResult, error) {
	// #nosec G304 -- File path comes from user CLI argument
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	fileName := filepath.Base(filePath)
	encodedPath := url.PathEscape(g.projectPath)

	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/packages/generic/%s/%s/%s",
		g.baseURL,
		encodedPath,
		url.PathEscape(opts.PackageName),
		url.PathEscape(opts.PackageVersion),
		url.PathEscape(fileName),
	)

	req, err := http.NewRequest("PUT", apiURL, file)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", g.token)
	req.ContentLength = fileInfo.Size()

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload failed with HTTP %d: %s", resp.StatusCode, string(body))
	}

	packageURL := fmt.Sprintf("%s/%s/-/packages", g.baseURL, g.projectPath)

	return &UploadResult{
		URL:     packageURL,
		Message: fmt.Sprintf("Uploaded %s (%.2f MB) to %s/%s", fileName, float64(fileInfo.Size())/(1024*1024), opts.PackageName, opts.PackageVersion),
	}, nil
}

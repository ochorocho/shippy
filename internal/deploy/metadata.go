package deploy

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"shippy/internal/ssh"
)

const metadataFileName = ".shippy-release.json"

// ReleaseMetadata contains metadata about a deployment release
type ReleaseMetadata struct {
	GitCommit  string    `json:"git_commit,omitempty"`
	GitTag     string    `json:"git_tag,omitempty"`
	GitBranch  string    `json:"git_branch,omitempty"`
	DeployedAt time.Time `json:"deployed_at"`
	DeployedBy string    `json:"deployed_by,omitempty"`
}

// WriteReleaseMetadata gathers local git info and writes it to the release directory on the remote server
func WriteReleaseMetadata(client *ssh.Client, releasePath string) error {
	meta := ReleaseMetadata{
		DeployedAt: time.Now(),
	}

	// Gather git info locally (best-effort, don't fail if git isn't available)
	if commit, err := gitCommand("rev-parse", "--short=7", "HEAD"); err == nil {
		meta.GitCommit = commit
	}
	if tag, err := gitCommand("describe", "--tags", "--exact-match", "HEAD"); err == nil {
		meta.GitTag = tag
	}
	if branch, err := gitCommand("rev-parse", "--abbrev-ref", "HEAD"); err == nil {
		meta.GitBranch = branch
	}
	if user, err := gitCommand("config", "user.name"); err == nil {
		meta.DeployedBy = user
	}

	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal release metadata: %w", err)
	}

	metaPath := filepath.Join(releasePath, metadataFileName)
	cmd := fmt.Sprintf("cat > %s << 'SHIPPY_EOF'\n%s\nSHIPPY_EOF", metaPath, string(data))
	if _, err := client.RunCommand(cmd); err != nil {
		return fmt.Errorf("failed to write release metadata: %w", err)
	}

	return nil
}

// ReadReleaseMetadata reads and parses release metadata from a remote release directory
func ReadReleaseMetadata(client *ssh.Client, releasePath string) (*ReleaseMetadata, error) {
	metaPath := filepath.Join(releasePath, metadataFileName)
	cmd := fmt.Sprintf("cat %s 2>/dev/null", metaPath)
	output, err := client.RunCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to read release metadata: %w", err)
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return nil, fmt.Errorf("empty metadata file")
	}

	var meta ReleaseMetadata
	if err := json.Unmarshal([]byte(output), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse release metadata: %w", err)
	}

	return &meta, nil
}

// gitCommand runs a git command and returns the trimmed output
func gitCommand(args ...string) (string, error) {
	// #nosec G204 -- "git" is hardcoded; args come from internal callers, never user input
	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

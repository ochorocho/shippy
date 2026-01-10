package errors

import "fmt"

// SSHError wraps an SSH-related error with helpful context
func SSHError(operation string, err error) error {
	return fmt.Errorf("%s: %w\n\nPossible causes:\n  • SSH connection was lost\n  • Network connectivity issues\n  • Server is unreachable\n  • SSH key authentication failed\n\nTry:\n  • Check network connection\n  • Verify SSH credentials\n  • Test SSH connection manually: ssh user@host", operation, err)
}

// FileUploadError wraps a file upload error with helpful context
func FileUploadError(file string, err error) error {
	return fmt.Errorf("failed to upload '%s': %w\n\nPossible causes:\n  • Disk space full on remote server\n  • Permission denied\n  • Network timeout\n\nTry:\n  • Check remote disk space: df -h\n  • Verify write permissions\n  • Retry the deployment", file, err)
}

// DeploymentError wraps a deployment failure with rollback instructions
func DeploymentError(step string, err error, releasePath string) error {
	return fmt.Errorf("deployment failed during '%s': %w\n\nTo rollback manually:\n  1. SSH to the server\n  2. cd to deployment directory\n  3. Run: rm -rf %s\n  4. Verify 'current' symlink points to previous release", step, err, releasePath)
}

// CommandError wraps a remote command error with context
func CommandError(cmd string, output string, err error) error {
	if output != "" {
		return fmt.Errorf("command failed: '%s'\n\nOutput:\n%s\n\nError: %w", cmd, output, err)
	}
	return fmt.Errorf("command failed: '%s'\n\nError: %w", cmd, err)
}

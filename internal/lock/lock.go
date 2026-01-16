package lock

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"tinnie/internal/config"
	"tinnie/internal/ssh"
)

// LockInfo contains metadata about a deployment lock
type LockInfo struct {
	LockedBy  string    `json:"locked_by"`
	LockedAt  time.Time `json:"locked_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Message   string    `json:"message"`
	PID       int       `json:"pid"`
}

// Locker handles deployment locking on the remote server
type Locker struct {
	client     *ssh.Client
	deployPath string
	timeout    time.Duration
	lockPath   string
}

// NewLocker creates a new deployment locker
func NewLocker(client *ssh.Client, deployPath string, timeout time.Duration) *Locker {
	if timeout == 0 {
		timeout = time.Duration(config.DefaultLockTimeoutMinutes) * time.Minute // Default like Capistrano
	}

	return &Locker{
		client:     client,
		deployPath: deployPath,
		timeout:    timeout,
		lockPath:   deployPath + "/.tinnie/deploy.lock",
	}
}

// Acquire attempts to acquire the deployment lock
func (l *Locker) Acquire(message string) error {
	// Check if lock exists and is still valid
	locked, info, err := l.IsLocked()
	if err != nil {
		return fmt.Errorf("failed to check lock status: %w", err)
	}

	if locked {
		// Lock exists and is not expired
		timeLeft := time.Until(info.ExpiresAt)
		minutesLeft := int(timeLeft.Minutes())

		return fmt.Errorf("deployment is locked\n\n"+
			"  Locked by: %s\n"+
			"  Locked at: %s\n"+
			"  Expires:   %s (in %d minutes)\n"+
			"  Message:   %s\n\n"+
			"To unlock manually, run:\n"+
			"  tinnie unlock\n\n"+
			"Or wait for automatic expiry in %d minutes.",
			info.LockedBy,
			info.LockedAt.Format("2006-01-02 15:04:05"),
			info.ExpiresAt.Format("2006-01-02 15:04:05"),
			minutesLeft,
			info.Message,
			minutesLeft,
		)
	}

	// Create lock
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = "unknown"
	}

	lockInfo := LockInfo{
		LockedBy:  fmt.Sprintf("%s@%s", username, hostname),
		LockedAt:  time.Now(),
		ExpiresAt: time.Now().Add(l.timeout),
		Message:   message,
		PID:       os.Getpid(),
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(lockInfo, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lock info: %w", err)
	}

	// Ensure .tinnie directory exists
	tinnieDir := l.deployPath + "/.tinnie"
	if _, err := l.client.RunCommand(fmt.Sprintf("mkdir -p %s", tinnieDir)); err != nil {
		return fmt.Errorf("failed to create .tinnie directory: %w", err)
	}

	// Write lock file
	cmd := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", l.lockPath, string(data))
	if _, err := l.client.RunCommand(cmd); err != nil {
		return fmt.Errorf("failed to create lock file: %w", err)
	}

	return nil
}

// Release removes the deployment lock
func (l *Locker) Release() error {
	cmd := fmt.Sprintf("rm -f %s", l.lockPath)
	_, err := l.client.RunCommand(cmd)
	// Ignore errors - lock might not exist
	return err
}

// IsLocked checks if a valid (non-expired) lock exists
func (l *Locker) IsLocked() (bool, *LockInfo, error) {
	// Check if lock file exists
	output, err := l.client.RunCommand(fmt.Sprintf("cat %s 2>/dev/null || echo ''", l.lockPath))
	if err != nil {
		// Command failed but we got output - parse it anyway
		output = strings.TrimSpace(output)
		if output == "" {
			// No lock file exists
			return false, nil, nil
		}
	}

	output = strings.TrimSpace(output)
	if output == "" {
		// No lock file exists
		return false, nil, nil
	}

	// Parse lock file
	var info LockInfo
	if err := json.Unmarshal([]byte(output), &info); err != nil {
		// Lock file exists but is corrupted - treat as stale
		_ = l.ForceUnlock()
		return false, nil, nil
	}

	// Check if lock has expired
	if time.Now().After(info.ExpiresAt) {
		// Lock expired - remove it
		_ = l.ForceUnlock()
		return false, nil, nil
	}

	// Valid lock exists
	return true, &info, nil
}

// ForceUnlock removes the lock regardless of ownership or expiry
func (l *Locker) ForceUnlock() error {
	return l.Release()
}

// RenewLock extends the lock expiry time
func (l *Locker) RenewLock() error {
	// Read current lock
	locked, info, err := l.IsLocked()
	if err != nil {
		return err
	}

	if !locked {
		return fmt.Errorf("no active lock to renew")
	}

	// Update expiry time
	info.ExpiresAt = time.Now().Add(l.timeout)

	// Write updated lock
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lock info: %w", err)
	}

	cmd := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", l.lockPath, string(data))
	if _, err := l.client.RunCommand(cmd); err != nil {
		return fmt.Errorf("failed to renew lock: %w", err)
	}

	return nil
}

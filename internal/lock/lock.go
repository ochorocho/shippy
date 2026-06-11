package lock

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ochorocho/shippy/internal/config"
	"github.com/ochorocho/shippy/internal/ssh"
)

const acquireMaxAttempts = 5

// lockRetryBackoff is how long Acquire waits before re-checking a lock whose
// metadata is not yet readable (another deploy is mid-acquire between creating
// the lock directory and writing info.json).
const lockRetryBackoff = 100 * time.Millisecond

// LockInfo contains metadata about a deployment lock
type LockInfo struct {
	LockedBy  string    `json:"locked_by"`
	LockedAt  time.Time `json:"locked_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Message   string    `json:"message"`
	PID       int       `json:"pid"`
	Token     string    `json:"token"` // set per acquire; Release only removes the lock when it matches
}

// commandRunner is the subset of *ssh.Client that the locker needs. Defining
// it as an interface lets tests drive the locker with a fake that records the
// exact remote commands, without a live SSH connection.
type commandRunner interface {
	RunCommand(cmd string) (string, error)
}

// Locker handles deployment locking on the remote server
type Locker struct {
	client     commandRunner
	deployPath string
	timeout    time.Duration
	shippyDir  string
	lockDir    string
	infoPath   string
	token      string
}

// NewLocker creates a new deployment locker
func NewLocker(client commandRunner, deployPath string, timeout time.Duration) *Locker {
	if timeout == 0 {
		timeout = time.Duration(config.DefaultLockTimeoutMinutes) * time.Minute // Default like Capistrano
	}

	shippyDir := deployPath + "/.shippy"
	lockDir := shippyDir + "/deploy.lock"

	return &Locker{
		client:     client,
		deployPath: deployPath,
		timeout:    timeout,
		shippyDir:  shippyDir,
		lockDir:    lockDir,
		infoPath:   lockDir + "/info.json",
		token:      newToken(),
	}
}

func newToken() string {
	hostname, _ := os.Hostname()
	var randomBytes [8]byte
	// #nosec G104 -- a short read is impossible; a zero suffix is still usable.
	_, _ = rand.Read(randomBytes[:])
	return fmt.Sprintf("%s-%d-%d-%s", hostname, os.Getpid(), time.Now().UnixNano(), hex.EncodeToString(randomBytes[:]))
}

// Acquire attempts to acquire the deployment lock
func (l *Locker) Acquire(message string) error {
	var lastCreateFailure string

	for attempt := 0; attempt < acquireMaxAttempts; attempt++ {
		created, output, err := l.tryCreate(message)
		if err != nil {
			return err
		}
		if created {
			return nil
		}

		exists, err := l.lockDirExists()
		if err != nil {
			return err
		}
		if !exists {
			// mkdir failed but nothing is there: either the lock was released
			// between the create and this probe (a retry then wins) or the
			// create genuinely failed (permissions, full disk). Retry; if every
			// attempt fails this way, surface the real create output below.
			lastCreateFailure = strings.TrimSpace(output)
			continue
		}
		lastCreateFailure = ""

		info, err := l.readInfo()
		switch {
		case err == nil && info != nil && time.Now().Before(info.ExpiresAt):
			// A valid lock is held.
			return l.lockedError(info)
		case err == nil && info != nil:
			// Expired: steal it atomically and retry.
			l.steal()
		default:
			// Metadata not yet readable: another deploy is mid-acquire between
			// creating the directory and writing info.json. Wait and re-check
			// rather than stealing a lock that was just legitimately taken.
			time.Sleep(lockRetryBackoff)
		}
	}

	if lastCreateFailure != "" {
		return fmt.Errorf("failed to create deployment lock: %s", lastCreateFailure)
	}
	return fmt.Errorf("could not acquire deployment lock after %d attempts", acquireMaxAttempts)
}

// tryCreate attempts the atomic create. It returns (true, "", nil) when this
// process won the lock and (false, output, nil) when the create command failed
// (the caller inspects whether lockDir exists to tell contention from a real
// failure). The error return is only for a local failure (marshalling).
func (l *Locker) tryCreate(message string) (bool, string, error) {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	if username == "" {
		username = "unknown"
	}

	now := time.Now()
	lockInfo := LockInfo{
		LockedBy:  fmt.Sprintf("%s@%s", username, hostname),
		LockedAt:  now,
		ExpiresAt: now.Add(l.timeout),
		Message:   message,
		PID:       os.Getpid(),
		Token:     l.token,
	}

	data, err := json.MarshalIndent(lockInfo, "", "  ")
	if err != nil {
		return false, "", fmt.Errorf("failed to marshal lock info: %w", err)
	}

	// One command, narrowing the window with a lock dir but no info.json: mkdir
	// (no -p, so it fails if another racer won) then write the metadata. The
	// quoted 'EOF' stops the shell expanding the JSON; only paths are quoted.
	cmd := fmt.Sprintf("mkdir -p %s && mkdir %s && cat > %s << 'EOF'\n%s\nEOF",
		ssh.Quote(l.shippyDir), ssh.Quote(l.lockDir), ssh.Quote(l.infoPath), string(data))

	if output, err := l.client.RunCommand(cmd); err != nil {
		return false, output, nil
	}
	return true, "", nil
}

// lockDirExists reports whether the lock directory currently exists.
func (l *Locker) lockDirExists() (bool, error) {
	output, err := l.client.RunCommand(fmt.Sprintf("[ -d %s ] && echo yes || echo no", ssh.Quote(l.lockDir)))
	if err != nil {
		return false, fmt.Errorf("failed to check lock directory: %w", err)
	}
	return strings.TrimSpace(output) == "yes", nil
}

// steal renames the existing lock dir away, then removes it. rename is atomic,
// so exactly one racer wins; losers' rename fails harmlessly and they retry.
func (l *Locker) steal() {
	stalePath := l.lockDir + ".stale." + l.token
	if _, err := l.client.RunCommand(fmt.Sprintf("mv %s %s", ssh.Quote(l.lockDir), ssh.Quote(stalePath))); err != nil {
		return
	}
	_, _ = l.client.RunCommand(fmt.Sprintf("rm -rf %s", ssh.Quote(stalePath)))
}

func (l *Locker) lockedError(info *LockInfo) error {
	minutesLeft := int(time.Until(info.ExpiresAt).Minutes())
	return fmt.Errorf("deployment is locked\n\n"+
		"  Locked by: %s\n"+
		"  Locked at: %s\n"+
		"  Expires:   %s (in %d minutes)\n"+
		"  Message:   %s\n\n"+
		"To unlock manually, run:\n"+
		"  shippy unlock\n\n"+
		"Or wait for automatic expiry in %d minutes.",
		info.LockedBy,
		info.LockedAt.Format("2006-01-02 15:04:05"),
		info.ExpiresAt.Format("2006-01-02 15:04:05"),
		minutesLeft,
		info.Message,
		minutesLeft,
	)
}

// readInfo returns (nil, nil) when no metadata exists and (nil, err) when it is
// present but unparseable.
func (l *Locker) readInfo() (*LockInfo, error) {
	output, err := l.client.RunCommand(fmt.Sprintf("cat %s 2>/dev/null || echo ''", ssh.Quote(l.infoPath)))
	if err != nil {
		output = strings.TrimSpace(output)
		if output == "" {
			return nil, nil
		}
	}

	output = strings.TrimSpace(output)
	if output == "" {
		return nil, nil
	}

	var info LockInfo
	if err := json.Unmarshal([]byte(output), &info); err != nil {
		return nil, fmt.Errorf("corrupt lock metadata: %w", err)
	}
	return &info, nil
}

// Release removes the deployment lock
func (l *Locker) Release() error {
	info, err := l.readInfo()
	if err != nil || info == nil || info.Token != l.token {
		return nil
	}
	_, err = l.client.RunCommand(fmt.Sprintf("rm -rf %s", ssh.Quote(l.lockDir)))
	return err
}

// IsLocked checks if a valid (non-expired) lock exists
func (l *Locker) IsLocked() (bool, *LockInfo, error) {
	info, err := l.readInfo()
	if err != nil || info == nil {
		return false, nil, nil
	}

	if time.Now().After(info.ExpiresAt) {
		return false, nil, nil
	}

	return true, info, nil
}

// ForceUnlock removes the lock regardless of ownership or expiry
func (l *Locker) ForceUnlock() error {
	_, err := l.client.RunCommand(fmt.Sprintf("rm -rf %s", ssh.Quote(l.lockDir)))
	return err
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

	cmd := fmt.Sprintf("cat > %s << 'EOF'\n%s\nEOF", ssh.Quote(l.infoPath), string(data))
	if _, err := l.client.RunCommand(cmd); err != nil {
		return fmt.Errorf("failed to renew lock: %w", err)
	}

	return nil
}

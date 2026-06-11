package lock

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// errFake stands in for a non-zero remote exit (e.g. mkdir failing on an
// existing directory).
var errFake = errors.New("non-zero exit")

// fakeRunner records every command issued and returns canned responses.
type fakeRunner struct {
	commands []string
	respond  func(cmd string) (string, error)
}

func (f *fakeRunner) RunCommand(cmd string) (string, error) {
	f.commands = append(f.commands, cmd)
	if f.respond != nil {
		return f.respond(cmd)
	}
	return "", nil
}

func (f *fakeRunner) sawContaining(substr string) bool {
	for _, c := range f.commands {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}

// deployPath with a space proves the paths are shell-quoted: an unquoted space
// would split the command into multiple arguments and corrupt the operation.
const spacedDeployPath = "/var/www/my app"

func lockJSON(t *testing.T, info LockInfo) string {
	t.Helper()
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("marshal lock info: %v", err)
	}
	return string(data)
}

func TestAcquireCreatesLockDirAtomicallyAndQuotes(t *testing.T) {
	// Default response "" with nil error => mkdir chain "succeeds" => we win.
	f := &fakeRunner{}
	l := NewLocker(f, spacedDeployPath, 15*time.Minute)

	if err := l.Acquire("deploying"); err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	// The acquire is a single atomic command: mkdir lockDir (no -p) + write info.json.
	if !f.sawContaining("mkdir -p '/var/www/my app/.shippy' && mkdir '/var/www/my app/.shippy/deploy.lock' && cat > '/var/www/my app/.shippy/deploy.lock/info.json' << 'EOF'") {
		t.Errorf("expected atomic quoted mkdir+write, commands: %v", f.commands)
	}
	// The metadata must carry our token so Release can verify ownership.
	if !f.sawContaining(`"token": "` + l.token) {
		t.Errorf("expected token %q embedded in lock metadata, commands: %v", l.token, f.commands)
	}
}

func TestAcquireFailsWhenValidLockHeld(t *testing.T) {
	other := lockJSON(t, LockInfo{
		LockedBy:  "alice@host",
		ExpiresAt: time.Now().Add(time.Hour),
		Message:   "deploying",
		Token:     "someone-elses-token",
	})

	f := &fakeRunner{respond: func(cmd string) (string, error) {
		switch {
		case strings.HasPrefix(cmd, "mkdir -p"):
			return "mkdir: File exists", errFake // mkdir of lockDir fails: contended
		case strings.HasPrefix(cmd, "[ -d"):
			return "yes\n", nil // the lock directory really exists
		case strings.HasPrefix(cmd, "cat "):
			return other, nil
		}
		return "", nil
	}}
	l := NewLocker(f, spacedDeployPath, time.Minute)

	err := l.Acquire("deploying")
	if err == nil {
		t.Fatal("Acquire() = nil, want 'deployment is locked' error")
	}
	if !strings.Contains(err.Error(), "deployment is locked") {
		t.Errorf("Acquire() error = %v, want 'deployment is locked'", err)
	}
	// A valid lock must NOT be stolen.
	if f.sawContaining("mv ") {
		t.Errorf("valid lock was stolen (saw mv), commands: %v", f.commands)
	}
}

func TestAcquireStealsExpiredLock(t *testing.T) {
	expired := lockJSON(t, LockInfo{
		LockedBy:  "bob@host",
		ExpiresAt: time.Now().Add(-time.Hour),
		Token:     "expired-token",
	})

	createAttempts := 0
	f := &fakeRunner{respond: func(cmd string) (string, error) {
		switch {
		case strings.HasPrefix(cmd, "mkdir -p"):
			createAttempts++
			if createAttempts == 1 {
				return "mkdir: File exists", errFake // first attempt: contended
			}
			return "", nil // after the steal, we win
		case strings.HasPrefix(cmd, "[ -d"):
			return "yes\n", nil
		case strings.HasPrefix(cmd, "cat "):
			return expired, nil
		}
		return "", nil
	}}
	l := NewLocker(f, spacedDeployPath, time.Minute)

	if err := l.Acquire("deploying"); err != nil {
		t.Fatalf("Acquire() error = %v, want success after stealing expired lock", err)
	}
	// Steal is an atomic rename of the lock dir, with our token in the target.
	if !f.sawContaining("mv '/var/www/my app/.shippy/deploy.lock' '/var/www/my app/.shippy/deploy.lock.stale." + l.token) {
		t.Errorf("expected atomic steal-by-rename, commands: %v", f.commands)
	}
}

// A lock directory whose metadata is not yet readable (another deploy is
// mid-acquire between mkdir and writing info.json) must NOT be stolen.
func TestAcquireDoesNotStealLockWithMissingMetadata(t *testing.T) {
	f := &fakeRunner{respond: func(cmd string) (string, error) {
		switch {
		case strings.HasPrefix(cmd, "mkdir -p"):
			return "mkdir: File exists", errFake // always contended
		case strings.HasPrefix(cmd, "[ -d"):
			return "yes\n", nil // dir exists ...
		case strings.HasPrefix(cmd, "cat "):
			return "", nil // ... but info.json is not there yet
		}
		return "", nil
	}}
	l := NewLocker(f, spacedDeployPath, time.Minute)

	if err := l.Acquire("deploying"); err == nil {
		t.Fatal("Acquire() = nil, want failure when metadata never becomes readable")
	}
	if f.sawContaining("mv ") {
		t.Errorf("stole a lock with unreadable metadata, commands: %v", f.commands)
	}
}

// A create failure where the lock directory does NOT exist afterwards is a real
// remote failure (permissions, full disk, ...) and must surface, not retry.
func TestAcquireSurfacesRealCreateFailure(t *testing.T) {
	f := &fakeRunner{respond: func(cmd string) (string, error) {
		switch {
		case strings.HasPrefix(cmd, "mkdir -p"):
			return "mkdir: Permission denied", errFake
		case strings.HasPrefix(cmd, "[ -d"):
			return "no\n", nil // nothing was created
		}
		return "", nil
	}}
	l := NewLocker(f, spacedDeployPath, time.Minute)

	err := l.Acquire("deploying")
	if err == nil || !strings.Contains(err.Error(), "Permission denied") {
		t.Errorf("Acquire() error = %v, want the real create failure surfaced", err)
	}
	if f.sawContaining("mv ") {
		t.Errorf("attempted steal on a real failure, commands: %v", f.commands)
	}
}

func TestReleaseOnlyRemovesOwnLock(t *testing.T) {
	f := &fakeRunner{}
	l := NewLocker(f, spacedDeployPath, time.Minute)
	// info.json reports OUR token => Release should remove the dir.
	f.respond = func(cmd string) (string, error) {
		if strings.HasPrefix(cmd, "cat ") {
			return lockJSON(t, LockInfo{Token: l.token, ExpiresAt: time.Now().Add(time.Hour)}), nil
		}
		return "", nil
	}

	if err := l.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if !f.sawContaining("rm -rf '/var/www/my app/.shippy/deploy.lock'") {
		t.Errorf("expected quoted rm -rf of lock dir, commands: %v", f.commands)
	}
}

func TestReleaseSkipsForeignLock(t *testing.T) {
	f := &fakeRunner{respond: func(cmd string) (string, error) {
		if strings.HasPrefix(cmd, "cat ") {
			// A different token: another deploy holds the lock now.
			return `{"token":"another-process","expires_at":"2999-01-01T00:00:00Z"}`, nil
		}
		return "", nil
	}}
	l := NewLocker(f, spacedDeployPath, time.Minute)

	if err := l.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if f.sawContaining("rm -rf") {
		t.Errorf("Release removed a lock it does not own, commands: %v", f.commands)
	}
}

func TestForceUnlockRemovesDirUnconditionally(t *testing.T) {
	f := &fakeRunner{}
	l := NewLocker(f, spacedDeployPath, time.Minute)

	if err := l.ForceUnlock(); err != nil {
		t.Fatalf("ForceUnlock() error = %v", err)
	}
	if !f.sawContaining("rm -rf '/var/www/my app/.shippy/deploy.lock'") {
		t.Errorf("expected unconditional quoted rm -rf, commands: %v", f.commands)
	}
	// ForceUnlock must not read metadata first — it removes regardless.
	if f.sawContaining("cat ") {
		t.Errorf("ForceUnlock should not inspect metadata, commands: %v", f.commands)
	}
}

func TestIsLockedReadsValidLock(t *testing.T) {
	info := LockInfo{
		LockedBy:  "alice@host",
		LockedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Message:   "deploying",
		PID:       42,
	}
	f := &fakeRunner{respond: func(string) (string, error) { return lockJSON(t, info), nil }}
	l := NewLocker(f, spacedDeployPath, time.Minute)

	locked, got, err := l.IsLocked()
	if err != nil {
		t.Fatalf("IsLocked() error = %v", err)
	}
	if !locked {
		t.Fatal("IsLocked() = false, want true for a non-expired lock")
	}
	if got.LockedBy != "alice@host" || got.Message != "deploying" {
		t.Errorf("parsed lock info = %+v", got)
	}
	if !f.sawContaining("cat '/var/www/my app/.shippy/deploy.lock/info.json' 2>/dev/null") {
		t.Errorf("expected quoted cat of info.json, commands: %v", f.commands)
	}
}

func TestIsLockedIsSideEffectFreeForExpiredLock(t *testing.T) {
	expired := lockJSON(t, LockInfo{
		LockedBy:  "bob@host",
		ExpiresAt: time.Now().Add(-time.Hour),
	})
	f := &fakeRunner{respond: func(string) (string, error) { return expired, nil }}
	l := NewLocker(f, spacedDeployPath, time.Minute)

	locked, _, err := l.IsLocked()
	if err != nil {
		t.Fatalf("IsLocked() error = %v", err)
	}
	if locked {
		t.Error("IsLocked() = true, want false for an expired lock")
	}
	// IsLocked must be pure: no rm/mv side effects (Acquire/unlock handle cleanup).
	if f.sawContaining("rm ") || f.sawContaining("mv ") {
		t.Errorf("IsLocked mutated remote state, commands: %v", f.commands)
	}
}

func TestIsLockedNoLockFile(t *testing.T) {
	f := &fakeRunner{respond: func(string) (string, error) { return "", nil }}
	l := NewLocker(f, spacedDeployPath, time.Minute)

	locked, info, err := l.IsLocked()
	if err != nil {
		t.Fatalf("IsLocked() error = %v", err)
	}
	if locked || info != nil {
		t.Errorf("IsLocked() = (%v, %v), want (false, nil)", locked, info)
	}
}

package lock

import (
	"strings"
	"testing"
	"time"
)

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

func TestNewLockerDerivesLockPath(t *testing.T) {
	l := NewLocker(&fakeRunner{}, spacedDeployPath, 0)
	want := spacedDeployPath + "/.shippy/deploy.lock"
	if l.lockPath != want {
		t.Errorf("lockPath = %q, want %q", l.lockPath, want)
	}
}

func TestAcquireQuotesPaths(t *testing.T) {
	f := &fakeRunner{} // default response "" => no existing lock
	l := NewLocker(f, spacedDeployPath, 15*time.Minute)

	if err := l.Acquire("deploying"); err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	if !f.sawContaining("mkdir -p '/var/www/my app/.shippy'") {
		t.Errorf("expected quoted mkdir of .shippy dir, commands: %v", f.commands)
	}
	if !f.sawContaining("cat > '/var/www/my app/.shippy/deploy.lock' << 'EOF'") {
		t.Errorf("expected quoted heredoc write of lock file, commands: %v", f.commands)
	}
}

func TestReleaseQuotesPath(t *testing.T) {
	f := &fakeRunner{}
	l := NewLocker(f, spacedDeployPath, time.Minute)

	if err := l.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if !f.sawContaining("rm -f '/var/www/my app/.shippy/deploy.lock'") {
		t.Errorf("expected quoted rm of lock file, commands: %v", f.commands)
	}
}

func TestIsLockedReadsAndQuotes(t *testing.T) {
	future := time.Now().Add(time.Hour)
	lockJSON := `{"locked_by":"alice@host","locked_at":"` +
		time.Now().Format(time.RFC3339) + `","expires_at":"` +
		future.Format(time.RFC3339) + `","message":"deploying","pid":42}`

	f := &fakeRunner{respond: func(string) (string, error) { return lockJSON, nil }}
	l := NewLocker(f, spacedDeployPath, time.Minute)

	locked, info, err := l.IsLocked()
	if err != nil {
		t.Fatalf("IsLocked() error = %v", err)
	}
	if !locked {
		t.Fatal("IsLocked() = false, want true for a non-expired lock")
	}
	if info.LockedBy != "alice@host" || info.Message != "deploying" {
		t.Errorf("parsed lock info = %+v", info)
	}
	if !f.sawContaining("cat '/var/www/my app/.shippy/deploy.lock' 2>/dev/null") {
		t.Errorf("expected quoted cat of lock file, commands: %v", f.commands)
	}
}

func TestIsLockedExpiredIsRemoved(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	lockJSON := `{"locked_by":"bob@host","locked_at":"` +
		past.Add(-time.Hour).Format(time.RFC3339) + `","expires_at":"` +
		past.Format(time.RFC3339) + `","message":"old","pid":1}`

	f := &fakeRunner{respond: func(string) (string, error) { return lockJSON, nil }}
	l := NewLocker(f, spacedDeployPath, time.Minute)

	locked, _, err := l.IsLocked()
	if err != nil {
		t.Fatalf("IsLocked() error = %v", err)
	}
	if locked {
		t.Error("IsLocked() = true, want false for an expired lock")
	}
	// Expired lock must be cleaned up with a quoted rm.
	if !f.sawContaining("rm -f '/var/www/my app/.shippy/deploy.lock'") {
		t.Errorf("expected expired lock to be removed, commands: %v", f.commands)
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

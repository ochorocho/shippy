package deploy

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ochorocho/shippy/internal/ssh"
)

// fakeClient records remote commands and mkdir calls, returning canned output.
type fakeClient struct {
	commands []string
	mkdirs   []string
	respond  func(cmd string) (string, error)
	// dateOutput overrides the server's `date` result.
	dateOutput func() (string, error)
}

// fakeServerTime is what the fake server's `date` prints.
const fakeServerTime = "20300102030405"

func (f *fakeClient) RunCommand(cmd string) (string, error) {
	f.commands = append(f.commands, cmd)
	if strings.HasPrefix(cmd, "date ") && f.dateOutput == nil {
		return fakeServerTime + "\n", nil
	}
	if strings.HasPrefix(cmd, "date ") {
		return f.dateOutput()
	}
	if f.respond != nil {
		return f.respond(cmd)
	}
	return "", nil
}

func (f *fakeClient) MkdirAll(path string) error {
	f.mkdirs = append(f.mkdirs, path)
	return nil
}

func (f *fakeClient) saw(substr string) bool {
	for _, c := range f.commands {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}

func (f *fakeClient) madeDir(path string) bool {
	for _, d := range f.mkdirs {
		if d == path {
			return true
		}
	}
	return false
}

// A deploy path with a space proves shell quoting: unquoted, the space would
// split the path into two arguments and corrupt every command.
const spacedDeploy = "/srv/my app"

func TestCreateReleaseMakesDirs(t *testing.T) {
	f := &fakeClient{}
	r := NewReleaseManager(f, spacedDeploy)

	releasePath, err := r.CreateRelease("")
	if err != nil {
		t.Fatalf("CreateRelease() error = %v", err)
	}
	if !f.madeDir(spacedDeploy + "/releases") {
		t.Errorf("expected releases dir created, mkdirs: %v", f.mkdirs)
	}
	if !strings.HasPrefix(releasePath, spacedDeploy+"/releases/") {
		t.Errorf("releasePath = %q, want under %q/releases/", releasePath, spacedDeploy)
	}
	if !f.madeDir(releasePath) {
		t.Errorf("expected release dir %q created, mkdirs: %v", releasePath, f.mkdirs)
	}
}

// The release dir is created at dir_mode via umask, so the promote rsync never
// has to chmod it: a chmod by a deploy user outside the inherited group (e.g.
// www-data) makes the kernel drop the setgid bit.
func TestCreateReleaseCreatesReleaseDirUnderDirModeUmask(t *testing.T) {
	f := &fakeClient{}
	r := NewReleaseManager(f, spacedDeploy)

	releasePath, err := r.CreateRelease("2775")
	if err != nil {
		t.Fatalf("CreateRelease() error = %v", err)
	}
	want := "umask 0002 && mkdir -p " + ssh.Quote(releasePath)
	if !f.saw(want) {
		t.Errorf("expected %q, commands: %v", want, f.commands)
	}
	if f.madeDir(releasePath) {
		t.Errorf("release dir created without the umask, mkdirs: %v", f.mkdirs)
	}
}

// Two deploys in the same second must not reuse a release directory: reusing a
// populated dir lets the mtime-based cache->release rsync ship stale content.
// A numeric suffix disambiguates while preserving lexical (chronological) order.
func TestCreateReleaseAvoidsSameSecondCollision(t *testing.T) {
	probes := 0
	f := &fakeClient{respond: func(cmd string) (string, error) {
		if strings.HasPrefix(strings.TrimSpace(cmd), "test -e") {
			probes++
			if probes == 1 {
				return "exists\n", nil // base timestamp already taken by a prior deploy
			}
			return "", nil // the suffixed candidate is free
		}
		return "", nil
	}}
	r := NewReleaseManager(f, spacedDeploy)

	releasePath, err := r.CreateRelease("")
	if err != nil {
		t.Fatalf("CreateRelease() error = %v", err)
	}

	// Must fall back to a suffixed name rather than reuse the taken base dir.
	if name := filepath.Base(releasePath); !strings.Contains(name, ".") {
		t.Fatalf("expected a suffixed release name to avoid the collision, got %q", name)
	}
	if !f.madeDir(releasePath) {
		t.Fatalf("expected the unique release dir %q to be created, mkdirs: %v", releasePath, f.mkdirs)
	}
}

func TestUpdateCurrentSymlinkQuotes(t *testing.T) {
	f := &fakeClient{}
	r := NewReleaseManager(f, spacedDeploy)

	if err := r.UpdateCurrentSymlink(spacedDeploy + "/releases/20240105120000"); err != nil {
		t.Fatalf("UpdateCurrentSymlink() error = %v", err)
	}
	// Target stays relative and quoted; current symlink path is quoted.
	want := "ln -sfn 'releases/20240105120000' '/srv/my app/current'"
	if !f.saw(want) {
		t.Errorf("expected %q, commands: %v", want, f.commands)
	}
}

func TestCleanupOldReleasesQuotesAndKeeps(t *testing.T) {
	f := &fakeClient{respond: func(cmd string) (string, error) {
		if strings.HasPrefix(cmd, "ls -1") {
			return "20240101\n20240102\n20240103\n20240104\n", nil
		}
		return "", nil
	}}
	r := NewReleaseManager(f, spacedDeploy)

	if err := r.CleanupOldReleases(2); err != nil {
		t.Fatalf("CleanupOldReleases() error = %v", err)
	}
	if !f.saw("ls -1 '/srv/my app/releases'") {
		t.Errorf("expected quoted ls, commands: %v", f.commands)
	}
	// Keep newest 2 (04, 03); remove 02 and 01, each quoted.
	if !f.saw("rm -rf '/srv/my app/releases/20240102'") ||
		!f.saw("rm -rf '/srv/my app/releases/20240101'") {
		t.Errorf("expected quoted rm of old releases, commands: %v", f.commands)
	}
	if f.saw("20240103'") || f.saw("20240104'") {
		t.Errorf("kept releases must not be removed, commands: %v", f.commands)
	}
}

func TestCreateSharedSymlinksQuotes(t *testing.T) {
	f := &fakeClient{}
	r := NewReleaseManager(f, spacedDeploy)
	releasePath := spacedDeploy + "/releases/20240105120000"

	if err := r.CreateSharedSymlinks(releasePath, []string{".env"}); err != nil {
		t.Fatalf("CreateSharedSymlinks() error = %v", err)
	}

	// Shared file is created (quoted) and symlinked into the release (quoted).
	if !f.saw("test -e '/srv/my app/shared/.env' || touch '/srv/my app/shared/.env'") {
		t.Errorf("expected quoted shared-file creation, commands: %v", f.commands)
	}
	if !f.saw("rm -rf '/srv/my app/releases/20240105120000/.env'") {
		t.Errorf("expected quoted rm of existing target, commands: %v", f.commands)
	}
	if !f.saw("ln -nfs '../../shared/.env' '/srv/my app/releases/20240105120000/.env'") {
		t.Errorf("expected quoted relative symlink, commands: %v", f.commands)
	}
}

func TestGetCurrentReleaseQuotes(t *testing.T) {
	f := &fakeClient{respond: func(string) (string, error) {
		return spacedDeploy + "/releases/20240105120000\n", nil
	}}
	r := NewReleaseManager(f, spacedDeploy)

	got, err := r.GetCurrentRelease()
	if err != nil {
		t.Fatalf("GetCurrentRelease() error = %v", err)
	}
	if got != spacedDeploy+"/releases/20240105120000" {
		t.Errorf("GetCurrentRelease() = %q", got)
	}
	if !f.saw("readlink -f '/srv/my app/current'") {
		t.Errorf("expected quoted readlink, commands: %v", f.commands)
	}
}

func TestListReleasesMarksCurrent(t *testing.T) {
	f := &fakeClient{respond: func(cmd string) (string, error) {
		switch {
		case strings.HasPrefix(cmd, "ls -1"):
			return "20240101\n20240102\n", nil
		case strings.HasPrefix(cmd, "readlink"):
			return spacedDeploy + "/releases/20240102\n", nil
		default:
			return "", nil // no metadata files
		}
	}}
	r := NewReleaseManager(f, spacedDeploy)

	releases, err := r.ListReleases()
	if err != nil {
		t.Fatalf("ListReleases() error = %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("got %d releases, want 2", len(releases))
	}
	// Sorted newest-first; 20240102 is current.
	if releases[0].Name != "20240102" || !releases[0].IsCurrent {
		t.Errorf("releases[0] = %+v, want 20240102 current", releases[0])
	}
	if releases[1].IsCurrent {
		t.Errorf("releases[1] (%s) should not be current", releases[1].Name)
	}
}

func TestWriteReleaseMetadataQuotes(t *testing.T) {
	f := &fakeClient{}
	releasePath := spacedDeploy + "/releases/20240105120000"

	if err := WriteReleaseMetadata(f, releasePath); err != nil {
		t.Fatalf("WriteReleaseMetadata() error = %v", err)
	}
	if !f.saw("cat > '/srv/my app/releases/20240105120000/.shippy-release.json' << 'SHIPPY_EOF'") {
		t.Errorf("expected quoted metadata heredoc, commands: %v", f.commands)
	}
}

func TestReadReleaseMetadataQuotesAndParses(t *testing.T) {
	meta := `{"git_commit":"abc1234","git_tag":"v1.0.0","deployed_at":"2024-01-05T12:00:00Z"}`
	f := &fakeClient{respond: func(string) (string, error) { return meta, nil }}
	releasePath := spacedDeploy + "/releases/20240105120000"

	got, err := ReadReleaseMetadata(f, releasePath)
	if err != nil {
		t.Fatalf("ReadReleaseMetadata() error = %v", err)
	}
	if got.GitCommit != "abc1234" || got.GitTag != "v1.0.0" {
		t.Errorf("parsed metadata = %+v", got)
	}
	if !f.saw("cat '/srv/my app/releases/20240105120000/.shippy-release.json' 2>/dev/null") {
		t.Errorf("expected quoted metadata read, commands: %v", f.commands)
	}
}

func TestCreateReleaseIsNamedAfterTheServerTime(t *testing.T) {
	r := NewReleaseManager(&fakeClient{}, spacedDeploy)

	releasePath, err := r.CreateRelease("")
	if err != nil {
		t.Fatalf("CreateRelease() error = %v", err)
	}
	if name := filepath.Base(releasePath); name != fakeServerTime {
		t.Errorf("release name = %q, want the server time %q", name, fakeServerTime)
	}
}

func TestCreateReleaseFailsWithoutAUsableServerTime(t *testing.T) {
	tests := []struct {
		name string
		date func() (string, error)
	}{
		{"command fails", func() (string, error) { return "", errors.New("no date") }},
		{"unexpected output", func() (string, error) { return "Tue Jan  2\n", nil }},
		{"empty output", func() (string, error) { return "", nil }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeClient{dateOutput: tt.date}
			r := NewReleaseManager(f, spacedDeploy)

			if _, err := r.CreateRelease(""); err == nil {
				t.Fatal("CreateRelease() succeeded, want an error")
			}
			if len(f.mkdirs) > 1 {
				t.Errorf("created release dir despite the error: %v", f.mkdirs)
			}
		})
	}
}

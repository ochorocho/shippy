package rsync

import (
	"os"
	"strings"
	"testing"
	"time"
)

type recordedUpload struct {
	local  string
	remote string
	mode   os.FileMode
}

// fakeClient records remote commands, mkdir calls and uploads. UploadFile does
// not touch disk, so tests can use fabricated FileInfo entries.
type fakeClient struct {
	commands []string
	mkdirs   []string
	uploads  []recordedUpload
	respond  func(cmd string) (string, error)
}

func (f *fakeClient) RunCommand(cmd string) (string, error) {
	f.commands = append(f.commands, cmd)
	if f.respond != nil {
		return f.respond(cmd)
	}
	return "", nil
}

func (f *fakeClient) MkdirAll(path string) error {
	f.mkdirs = append(f.mkdirs, path)
	return nil
}

func (f *fakeClient) UploadFile(localPath, remotePath string, mode os.FileMode) error {
	f.uploads = append(f.uploads, recordedUpload{localPath, remotePath, mode})
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

const spacedDeploy = "/srv/my app"

func TestSyncQuotesRemoteCommands(t *testing.T) {
	// Remote cache already holds a stale file that no longer exists locally,
	// so it must be deleted; the local file is new, so it must be uploaded.
	f := &fakeClient{respond: func(cmd string) (string, error) {
		if strings.Contains(cmd, "find . -type f") {
			return "./old data.txt\t3\t100\n", nil
		}
		return "", nil
	}}

	releasePath := spacedDeploy + "/releases/20240105120000"
	s := NewSyncer(f, releasePath, false, spacedDeploy)

	files := []FileInfo{
		{
			RelPath:  "app/config.php",
			FullPath: "/local/app/config.php",
			Size:     5,
			Mode:     0o644,
			ModTime:  time.Unix(1000, 0),
		},
	}

	if err := s.Sync(files); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	cache := spacedDeploy + "/.cache"

	// Cache directory is created.
	found := false
	for _, d := range f.mkdirs {
		if d == cache {
			found = true
		}
	}
	if !found {
		t.Errorf("expected cache dir %q created, mkdirs: %v", cache, f.mkdirs)
	}

	// Index scan, stale-file deletion, and final copy are all quoted.
	if !f.saw("cd '/srv/my app/.cache' && find . -type f") {
		t.Errorf("expected quoted find index command, commands: %v", f.commands)
	}
	if !f.saw("rm -f '/srv/my app/.cache/old data.txt'") {
		t.Errorf("expected quoted deletion of stale cache file, commands: %v", f.commands)
	}
	if !f.saw("rsync -rlt --no-perms --delete '/srv/my app/.cache'/ '/srv/my app/releases/20240105120000'/") {
		t.Errorf("expected quoted rsync copy, commands: %v", f.commands)
	}

	// The new local file is uploaded to its path under the cache.
	if len(f.uploads) != 1 {
		t.Fatalf("expected 1 upload, got %d: %v", len(f.uploads), f.uploads)
	}
	wantRemote := cache + "/app/config.php"
	if f.uploads[0].remote != wantRemote {
		t.Errorf("upload remote = %q, want %q", f.uploads[0].remote, wantRemote)
	}
	if f.uploads[0].local != "/local/app/config.php" {
		t.Errorf("upload local = %q, want /local/app/config.php", f.uploads[0].local)
	}
}

func TestSyncSkipsUnchangedFiles(t *testing.T) {
	// Remote cache already has the file with matching size and a newer mtime,
	// so it should be skipped (no upload), and nothing should be deleted.
	f := &fakeClient{respond: func(cmd string) (string, error) {
		if strings.Contains(cmd, "find . -type f") {
			return "./app/config.php\t5\t2000\n", nil
		}
		return "", nil
	}}

	s := NewSyncer(f, spacedDeploy+"/releases/X", false, spacedDeploy)
	files := []FileInfo{
		{
			RelPath:  "app/config.php",
			FullPath: "/local/app/config.php",
			Size:     5,
			Mode:     0o644,
			ModTime:  time.Unix(1000, 0), // older than remote mtime 2000
		},
	}

	if err := s.Sync(files); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if len(f.uploads) != 0 {
		t.Errorf("expected no uploads for unchanged file, got %v", f.uploads)
	}
	if f.saw("rm -f") {
		t.Errorf("expected no deletions, commands: %v", f.commands)
	}
}

// TestSyncCacheToReleaseFlags verifies that the remote rsync command used to
// promote the cache into the release directory never sets file permissions from
// the source tree. Server permissions are owned by the server (default ACLs,
// umask) not by the deploying machine; mirroring source permissions via -a or
// --perms would silently override those ACLs and break group-write access for
// the web server process.
func TestSyncCacheToReleaseFlags(t *testing.T) {
	var rsyncCmd string
	f := &fakeClient{respond: func(cmd string) (string, error) {
		if strings.Contains(cmd, "rsync") {
			rsyncCmd = cmd
		}
		return "", nil
	}}

	s := NewSyncer(f, spacedDeploy+"/releases/20240105120000", false, spacedDeploy)
	files := []FileInfo{
		{RelPath: "index.php", FullPath: "/local/index.php", Size: 1, Mode: 0o644, ModTime: time.Unix(1000, 0)},
	}
	if err := s.Sync(files); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	if rsyncCmd == "" {
		t.Fatal("no rsync command was issued")
	}

	// --no-perms must be present so chmod() is never called on transferred files.
	if !strings.Contains(rsyncCmd, "--no-perms") {
		t.Errorf("rsync command must contain --no-perms to let server ACLs govern permissions, got: %s", rsyncCmd)
	}

	// -a (archive) implies -p/--perms which calls chmod() and overrides default
	// ACLs. Neither form must appear in the command.
	for _, forbidden := range []string{"-a ", "-a\t", " -a\n", "--archive", "--perms", " -p "} {
		if strings.Contains(rsyncCmd, forbidden) {
			t.Errorf("rsync command must not contain %q (would mirror source permissions and override server ACLs), got: %s", forbidden, rsyncCmd)
		}
	}

	// Recursive, links, and timestamps must be preserved.
	for _, required := range []string{"-rlt", "--delete"} {
		if !strings.Contains(rsyncCmd, required) {
			t.Errorf("rsync command must contain %q, got: %s", required, rsyncCmd)
		}
	}
}

func TestGetRemoteFileIndexParses(t *testing.T) {
	f := &fakeClient{respond: func(string) (string, error) {
		return "./a.txt\t10\t111\n./dir/b bin\t20\t222\n", nil
	}}
	s := NewSyncer(f, "/release", false, spacedDeploy)

	idx, err := s.getRemoteFileIndex(spacedDeploy + "/.cache")
	if err != nil {
		t.Fatalf("getRemoteFileIndex() error = %v", err)
	}
	if len(idx) != 2 {
		t.Fatalf("got %d entries, want 2: %v", len(idx), idx)
	}
	if idx["a.txt"].Size != 10 || idx["a.txt"].Mtime != 111 {
		t.Errorf("a.txt = %+v", idx["a.txt"])
	}
	if idx["dir/b bin"].Size != 20 || idx["dir/b bin"].Mtime != 222 {
		t.Errorf("dir/b bin = %+v", idx["dir/b bin"])
	}
}

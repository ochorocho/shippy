package rsync

import (
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"

	gokrsync "github.com/gokrazy/rsync"
)

// localRemote implements remoteClient by running commands and the rsync receiver
// on the LOCAL machine with the real rsync binary, into real temp directories.
// This exercises the whole Sync path (gokr-rsync client-sender -> real rsync
// receiver, then the real-rsync promote) exactly as in production, minus SSH.
type localRemote struct {
	commands []string
}

func (r *localRemote) RunCommand(cmd string) (string, error) {
	r.commands = append(r.commands, cmd)
	out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	return string(out), err
}

func (r *localRemote) MkdirAll(path string) error { return os.MkdirAll(path, 0o755) }

func (r *localRemote) RsyncSender(remoteCmd string, run func(io.ReadWriteCloser) error) error {
	cmd := exec.Command("sh", "-c", remoteCmd)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	runErr := run(&gokrsync.BothCloser{ReadCloser: io.NopCloser(stdout), WriteCloser: stdin})
	if waitErr := cmd.Wait(); waitErr != nil && runErr == nil {
		return waitErr
	}
	return runErr
}

// releaseSet lists files and symlinks (not directories) under root, as relative
// slash paths.
func releaseSet(t *testing.T, root string) []string {
	t.Helper()
	var got []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		got = append(got, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatalf("walk release: %v", err)
	}
	sort.Strings(got)
	return got
}

func scanFixture(t *testing.T, src string) ([]FileInfo, []string) {
	t.Helper()
	scanner, err := NewScanner(SyncOptions{
		SourceDir:       src,
		IncludePatterns: []string{"**"},
		ExcludePatterns: []string{"node_modules/"},
	})
	if err != nil {
		t.Fatalf("NewScanner: %v", err)
	}
	files, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	var rel []string
	for _, f := range files {
		rel = append(rel, f.RelPath)
	}
	sort.Strings(rel)
	return files, rel
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestSyncTransfersScannedSet is the oracle test: whatever the go-git scanner
// includes must land in the release directory exactly, and nothing else.
func TestSyncTransfersScannedSet(t *testing.T) {
	if _, err := exec.LookPath("rsync"); err != nil {
		t.Skip("rsync not installed")
	}

	src := t.TempDir()
	deploy := t.TempDir()
	release := filepath.Join(deploy, "releases", "20240105120000")

	writeFile(t, filepath.Join(src, "app/config.php"), "config")
	writeFile(t, filepath.Join(src, "dir/b bin"), "bb") // spaced name
	writeFile(t, filepath.Join(src, "index.php"), "index")
	writeFile(t, filepath.Join(src, "node_modules/huge.js"), "junk") // excluded
	if err := os.Symlink("../index.php", filepath.Join(src, "app/link.php")); err != nil {
		t.Fatal(err)
	}

	files, want := scanFixture(t, src)
	// Sanity: the excluded dir is not in the scanned set.
	for _, p := range want {
		if p == "node_modules/huge.js" {
			t.Fatalf("scanner included an excluded file: %v", want)
		}
	}

	s := NewSyncer(&localRemote{}, release, false, deploy, src)
	if err := s.Sync(files); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	got := releaseSet(t, release)
	if !equalStrings(got, want) {
		t.Errorf("release set != scanned set\n got: %v\nwant: %v", got, want)
	}

	// The symlink must be a symlink in the release, not a dereferenced copy.
	if fi, err := os.Lstat(filepath.Join(release, "app/link.php")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("app/link.php not a symlink in release: mode=%v err=%v", fi.Mode(), err)
	}
}

// TestSyncPrunesRemovedFiles verifies the promote's --files-from + --delete
// removes a file that is no longer in the scanned set on a subsequent deploy.
func TestSyncPrunesRemovedFiles(t *testing.T) {
	if _, err := exec.LookPath("rsync"); err != nil {
		t.Skip("rsync not installed")
	}

	src := t.TempDir()
	deploy := t.TempDir()
	release := filepath.Join(deploy, "current")

	writeFile(t, filepath.Join(src, "keep.php"), "keep")
	writeFile(t, filepath.Join(src, "dir/gone.php"), "gone")

	files, _ := scanFixture(t, src)
	s := NewSyncer(&localRemote{}, release, false, deploy, src)
	if err := s.Sync(files); err != nil {
		t.Fatalf("first Sync: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(release, "dir/gone.php")); err != nil {
		t.Fatalf("dir/gone.php should exist after first sync: %v", err)
	}

	// Remove the file from the source and re-deploy.
	if err := os.Remove(filepath.Join(src, "dir/gone.php")); err != nil {
		t.Fatal(err)
	}
	files, _ = scanFixture(t, src)
	if err := s.Sync(files); err != nil {
		t.Fatalf("second Sync: %v", err)
	}

	if _, err := os.Lstat(filepath.Join(release, "dir/gone.php")); !os.IsNotExist(err) {
		t.Errorf("dir/gone.php was not pruned from release (err=%v)", err)
	}
	if _, err := os.Lstat(filepath.Join(release, "keep.php")); err != nil {
		t.Errorf("keep.php should still exist: %v", err)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

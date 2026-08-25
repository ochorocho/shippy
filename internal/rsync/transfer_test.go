package rsync

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

type recordedUpload struct {
	local   string
	remote  string
	mode    os.FileMode
	content string // captured at call time; temp files are gone once Sync returns
}

// fakeClient records remote commands, mkdir calls and uploads. UploadFile does
// not touch disk, so tests can use fabricated FileInfo entries.
type fakeClient struct {
	commands  []string
	mkdirs    []string
	uploads   []recordedUpload
	respond   func(cmd string) (string, error)
	uploadErr func(remotePath string) error
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
	if f.uploadErr != nil {
		if err := f.uploadErr(remotePath); err != nil {
			return err
		}
	}
	content := ""
	if data, err := os.ReadFile(localPath); err == nil {
		content = string(data)
	}
	f.uploads = append(f.uploads, recordedUpload{localPath, remotePath, mode, content})
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
			return "./old data.txt\t3\n", nil
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
			Checksum: "sum-config",
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

	// Index scan, manifest read, stale-file deletion, and final copy are all quoted.
	if !f.saw("cd '/srv/my app/.cache' && find . -type f") {
		t.Errorf("expected quoted find index command, commands: %v", f.commands)
	}
	if !f.saw("cat '/srv/my app/.shippy/cache-manifest.json'") {
		t.Errorf("expected quoted manifest read command, commands: %v", f.commands)
	}
	if !f.saw("rm -f '/srv/my app/.cache/old data.txt'") {
		t.Errorf("expected quoted deletion of stale cache file, commands: %v", f.commands)
	}
	if !f.saw("rsync -rlt --no-perms --delete '/srv/my app/.cache'/ '/srv/my app/releases/20240105120000'/") {
		t.Errorf("expected quoted rsync copy, commands: %v", f.commands)
	}

	// The new local file is uploaded to its path under the cache, and the
	// refreshed manifest is uploaded next to it.
	wantRemote := cache + "/app/config.php"
	fileUploaded := false
	manifestUploaded := false
	for _, u := range f.uploads {
		if u.remote == wantRemote && u.local == "/local/app/config.php" {
			fileUploaded = true
		}
		if u.remote == spacedDeploy+"/.shippy/cache-manifest.json" {
			manifestUploaded = true
		}
	}
	if !fileUploaded {
		t.Errorf("expected upload of %q, uploads: %v", wantRemote, f.uploads)
	}
	if !manifestUploaded {
		t.Errorf("expected manifest upload, uploads: %v", f.uploads)
	}
}

// A fresh CI checkout restamps every file, so local mtimes are always newer
// than the cache. Unchanged content must still be skipped.
func TestSyncSkipsUnchangedFileWithNewerMtime(t *testing.T) {
	f := &fakeClient{respond: func(cmd string) (string, error) {
		if strings.Contains(cmd, "find . -type f") {
			return "./app/config.php\t5\n", nil
		}
		if strings.Contains(cmd, "cache-manifest.json") {
			return `{"app/config.php":"abc123"}`, nil
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
			ModTime:  time.Unix(2000, 0), // newer than remote mtime 1000 (fresh checkout)
			Checksum: "abc123",
		},
	}

	if err := s.Sync(files); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	for _, u := range f.uploads {
		if u.remote == spacedDeploy+"/.cache/app/config.php" {
			t.Errorf("unchanged file was re-uploaded despite matching checksum: %v", f.uploads)
		}
	}
}

// Same size with an older local mtime (a composer downgrade restoring archive
// mtimes) must still upload; the old mtime comparison shipped a stale file.
func TestSyncUploadsWhenContentChangesButSizeMatches(t *testing.T) {
	f := &fakeClient{respond: func(cmd string) (string, error) {
		if strings.Contains(cmd, "find . -type f") {
			return "./app/config.php\t5\n", nil
		}
		if strings.Contains(cmd, "cache-manifest.json") {
			return `{"app/config.php":"oldsum"}`, nil
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
			Checksum: "newsum",
		},
	}

	if err := s.Sync(files); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	uploaded := false
	for _, u := range f.uploads {
		if u.remote == spacedDeploy+"/.cache/app/config.php" {
			uploaded = true
		}
	}
	if !uploaded {
		t.Errorf("expected changed file to be uploaded, uploads: %v", f.uploads)
	}
}

// The manifest drives the next deploy's skip decisions, so it must map every
// synced file to its local checksum.
func TestSyncWritesManifestWithLocalChecksums(t *testing.T) {
	f := &fakeClient{}

	s := NewSyncer(f, spacedDeploy+"/releases/X", false, spacedDeploy)
	files := []FileInfo{
		{RelPath: "a.php", FullPath: "/local/a.php", Size: 1, Mode: 0o644, ModTime: time.Unix(1000, 0), Checksum: "sum-a"},
		{RelPath: "dir/b.php", FullPath: "/local/dir/b.php", Size: 2, Mode: 0o644, ModTime: time.Unix(1000, 0), Checksum: "sum-b"},
	}

	if err := s.Sync(files); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	var manifestJSON string
	for _, u := range f.uploads {
		if u.remote == spacedDeploy+"/.shippy/cache-manifest.json" {
			manifestJSON = u.content
		}
	}
	if manifestJSON == "" {
		t.Fatalf("no manifest upload recorded, uploads: %v", f.uploads)
	}

	manifest := map[string]string{}
	if err := json.Unmarshal([]byte(manifestJSON), &manifest); err != nil {
		t.Fatalf("manifest is not valid JSON: %v (content: %s)", err, manifestJSON)
	}
	want := map[string]string{"a.php": "sum-a", "dir/b.php": "sum-b"}
	if len(manifest) != len(want) {
		t.Errorf("manifest has %d entries, want %d: %v", len(manifest), len(want), manifest)
	}
	for relPath, checksum := range want {
		if manifest[relPath] != checksum {
			t.Errorf("manifest[%q] = %q, want %q", relPath, manifest[relPath], checksum)
		}
	}
}

// The manifest must only be written once every upload succeeded, otherwise it
// vouches for files that never reached the cache and they are skipped forever.
func TestSyncDoesNotWriteManifestWhenUploadFails(t *testing.T) {
	f := &fakeClient{uploadErr: func(remotePath string) error {
		if strings.HasSuffix(remotePath, "b.php") {
			return fmt.Errorf("disk full")
		}
		return nil
	}}

	s := NewSyncer(f, spacedDeploy+"/releases/X", false, spacedDeploy)
	files := []FileInfo{
		{RelPath: "a.php", FullPath: "/local/a.php", Size: 1, Mode: 0o644, ModTime: time.Unix(1000, 0), Checksum: "sum-a"},
		{RelPath: "b.php", FullPath: "/local/b.php", Size: 2, Mode: 0o644, ModTime: time.Unix(1000, 0), Checksum: "sum-b"},
	}

	if err := s.Sync(files); err == nil {
		t.Fatal("Sync() returned nil, want error when an upload fails")
	}
	for _, u := range f.uploads {
		if u.remote == spacedDeploy+"/.shippy/cache-manifest.json" {
			t.Errorf("manifest written despite failed upload, uploads: %v", f.uploads)
		}
	}
}

// A file the manifest vouches for but that is gone from the cache must be
// re-uploaded; the release is populated solely by rsync from the cache. Uses an
// empty file, whose size matches the zero value a cache miss yields, so only
// the presence check can catch it.
func TestSyncUploadsWhenManifestMatchesButCacheFileMissing(t *testing.T) {
	f := &fakeClient{respond: func(cmd string) (string, error) {
		if strings.Contains(cmd, "cache-manifest.json") {
			return `{".gitkeep":"abc123"}`, nil
		}
		return "", nil
	}}

	s := NewSyncer(f, spacedDeploy+"/releases/X", false, spacedDeploy)
	files := []FileInfo{
		{
			RelPath:  ".gitkeep",
			FullPath: "/local/.gitkeep",
			Size:     0,
			Mode:     0o644,
			ModTime:  time.Unix(1000, 0),
			Checksum: "abc123",
		},
	}

	if err := s.Sync(files); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	uploaded := false
	for _, u := range f.uploads {
		if u.remote == spacedDeploy+"/.cache/.gitkeep" {
			uploaded = true
		}
	}
	if !uploaded {
		t.Errorf("expected re-upload of file missing from cache, uploads: %v", f.uploads)
	}
}

// Without a manifest (first deploy after upgrade, or a corrupt file) nothing
// can be trusted as cached, so everything is re-uploaded.
func TestSyncUploadsAllWhenManifestMissing(t *testing.T) {
	f := &fakeClient{respond: func(cmd string) (string, error) {
		if strings.Contains(cmd, "find . -type f") {
			return "./app/config.php\t5\n", nil
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
			ModTime:  time.Unix(500, 0),
			Checksum: "abc123",
		},
	}

	if err := s.Sync(files); err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	uploaded := false
	for _, u := range f.uploads {
		if u.remote == spacedDeploy+"/.cache/app/config.php" {
			uploaded = true
		}
	}
	if !uploaded {
		t.Errorf("expected upload when manifest is missing, uploads: %v", f.uploads)
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
		{RelPath: "index.php", FullPath: "/local/index.php", Size: 1, Mode: 0o644, ModTime: time.Unix(1000, 0), Checksum: "sum-index"},
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
		return "./a.txt\t10\n./dir/b bin\t20\n", nil
	}}
	s := NewSyncer(f, "/release", false, spacedDeploy)

	idx, err := s.getRemoteFileIndex(spacedDeploy + "/.cache")
	if err != nil {
		t.Fatalf("getRemoteFileIndex() error = %v", err)
	}
	if len(idx) != 2 {
		t.Fatalf("got %d entries, want 2: %v", len(idx), idx)
	}
	if idx["a.txt"].Size != 10 {
		t.Errorf("a.txt = %+v", idx["a.txt"])
	}
	if idx["dir/b bin"].Size != 20 {
		t.Errorf("dir/b bin = %+v", idx["dir/b bin"])
	}
}

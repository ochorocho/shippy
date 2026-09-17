//go:build windows

package rsynctest

import (
	"os"
	"testing"
)

func CreateDummyDeviceFiles(t *testing.T, dir string) {
	t.Skip("Windows does not support special files (devices/FIFOs/sockets)")
}

func VerifyDummyDeviceFiles(t *testing.T, source, dest string) {
	t.Skip("Windows does not support special files (devices/FIFOs/sockets)")
}

// StatUidGid is not meaningful on Windows (no Unix file ownership).
//
// Callers are supposed to check os.Getuid()==-1 and skip.
// Hence, this function must compile but is never reached.
func StatUidGid(t *testing.T, fi os.FileInfo) (uid, gid int) {
	t.Fatal("BUG: StatUidGid not skipped on Windows")
	return 0, 0
}

// ReplaceSymlink atomically replaces newname with a symlink to oldname.
//
// On Unix, this function works atomically,
// on Windows it falls back to non-atomic behavior.
func ReplaceSymlink(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.Remove(newname); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if err := os.Symlink(oldname, newname); err != nil {
		t.Skipf("cannot create symlinks on this Windows host: %v", err)
	}
}

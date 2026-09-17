//go:build !windows

package rsynctest

import (
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/google/renameio/v2"
	"golang.org/x/sys/unix"
)

func CreateDummyDeviceFiles(t *testing.T, dir string) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	char := filepath.Join(dir, "char")
	// major 1, minor 5, like /dev/zero
	if err := unix.Mknod(char, 0600|syscall.S_IFCHR, int(unix.Mkdev(1, 5))); err != nil {
		t.Fatal(err)
	}

	block := filepath.Join(dir, "block")
	// major 242, minor 9, like /dev/nvme0
	if err := unix.Mknod(block, 0600|syscall.S_IFBLK, int(unix.Mkdev(242, 9))); err != nil {
		t.Fatal(err)
	}

	fifo := filepath.Join(dir, "fifo")
	if err := unix.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}

	sock := filepath.Join(dir, "sock")
	ln, err := net.Listen("unix", sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
}

func VerifyDummyDeviceFiles(t *testing.T, source, dest string) {
	{
		sourcest, err := os.Stat(filepath.Join(source, "char"))
		if err != nil {
			t.Fatal(err)
		}
		destst, err := os.Stat(filepath.Join(dest, "char"))
		if err != nil {
			t.Fatal(err)
		}
		if destst.Mode().Type()&os.ModeCharDevice == 0 {
			t.Fatalf("unexpected type: got %v, want character device", destst.Mode())
		}
		destsys, ok := destst.Sys().(*syscall.Stat_t)
		if !ok {
			t.Fatal("stat does not contain rdev")
		}
		sourcesys, ok := sourcest.Sys().(*syscall.Stat_t)
		if !ok {
			t.Fatal("stat does not contain rdev")
		}
		if got, want := destsys.Rdev, sourcesys.Rdev; got != want {
			t.Fatalf("unexpected rdev: got %v, want %v", got, want)
		}
	}

	{
		sourcest, err := os.Stat(filepath.Join(source, "block"))
		if err != nil {
			t.Fatal(err)
		}
		destst, err := os.Stat(filepath.Join(dest, "block"))
		if err != nil {
			t.Fatal(err)
		}
		if destst.Mode().Type()&os.ModeDevice == 0 ||
			destst.Mode().Type()&os.ModeCharDevice != 0 {
			t.Fatalf("unexpected type: got %v, want block device", destst.Mode())
		}
		destsys, ok := destst.Sys().(*syscall.Stat_t)
		if !ok {
			t.Fatal("stat does not contain rdev")
		}
		sourcesys, ok := sourcest.Sys().(*syscall.Stat_t)
		if !ok {
			t.Fatal("stat does not contain rdev")
		}
		if got, want := destsys.Rdev, sourcesys.Rdev; got != want {
			t.Fatalf("unexpected rdev: got %v, want %v", got, want)
		}
	}

	{
		st, err := os.Stat(filepath.Join(dest, "fifo"))
		if err != nil {
			t.Fatal(err)
		}
		if st.Mode().Type()&os.ModeNamedPipe == 0 {
			t.Fatalf("unexpected type: got %v, want fifo", st.Mode())
		}
	}

	{
		st, err := os.Stat(filepath.Join(dest, "sock"))
		if err != nil {
			t.Fatal(err)
		}
		if st.Mode().Type()&os.ModeSocket == 0 {
			t.Fatalf("unexpected type: got %v, want socket", st.Mode())
		}
	}
}

// StatUidGid returns the owner uid and gid backing fi.
//
// This check is encapsulated in StatUidGid so that
// on Windows, we can fail loudly (the check should be skipped).
func StatUidGid(t *testing.T, fi os.FileInfo) (uid, gid int) {
	stt, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatalf("FileInfo.Sys() is %T, want *syscall.Stat_t", fi.Sys())
	}
	return int(stt.Uid), int(stt.Gid)
}

// ReplaceSymlink atomically replaces newname with a symlink to oldname.
//
// On Unix, this function works atomically,
// on Windows it falls back to non-atomic behavior.
func ReplaceSymlink(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := renameio.Symlink(oldname, newname); err != nil {
		t.Fatal(err)
	}
}

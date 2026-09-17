package rsyncclient_test

// SHIPPY PATCH test: verify the --files-from sender addition against a real rsync
// receiver (the production shape: gokr-rsync client-sender pushing to a real
// remote rsync over a pipe). Deletion is intentionally NOT exercised here -- see
// ../SHIPPY_PATCHES.md: shippy prunes in its cache->release promote with real
// rsync, because gokr-rsync's sender deadlocks driving a --delete receiver.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gokrazy/rsync"
	"github.com/gokrazy/rsync/internal/rsynctest"
	"github.com/gokrazy/rsync/rsyncclient"
)

func shippyWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestShippyFilesFromTransfersExactList(t *testing.T) {
	rsyncBin := rsynctest.AnyRsync(t)

	src := t.TempDir()
	dst := t.TempDir()

	shippyWrite(t, filepath.Join(src, "app/config.php"), "config")
	shippyWrite(t, filepath.Join(src, "dir/b bin"), "bb")
	shippyWrite(t, filepath.Join(src, "index.php"), "index")
	// A file deliberately NOT in the list must not be transferred.
	shippyWrite(t, filepath.Join(src, "node_modules/huge.js"), "junk")
	if err := os.Symlink("../index.php", filepath.Join(src, "app/link.php")); err != nil {
		t.Fatal(err)
	}
	list := []string{"app/config.php", "dir/b bin", "index.php", "app/link.php"}

	listFile := filepath.Join(t.TempDir(), "list")
	if err := os.WriteFile(listFile, []byte(strings.Join(list, "\x00")), 0o600); err != nil {
		t.Fatal(err)
	}

	client, err := rsyncclient.New([]string{
		"-rlt", "--no-perms",
		"--files-from=" + listFile, "--from0",
	}, rsyncclient.WithSender())
	if err != nil {
		t.Fatalf("rsyncclient.New: %v", err)
	}

	cmd := exec.Command(rsyncBin, client.ServerCommandOptions(dst+"/")...)
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	rw := &rsync.BothCloser{ReadCloser: stdout, WriteCloser: stdin}
	if _, err := client.Run(context.Background(), rw, []string{src + "/"}); err != nil {
		t.Fatalf("client.Run: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("receiver rsync: %v", err)
	}

	for _, p := range list {
		if _, err := os.Lstat(filepath.Join(dst, p)); err != nil {
			t.Errorf("missing after transfer: %s: %v", p, err)
		}
	}
	if fi, err := os.Lstat(filepath.Join(dst, "app/link.php")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("app/link.php not a symlink: mode=%v err=%v", fi.Mode(), err)
	}
	// The unlisted file must be absent (only the list drives the transfer).
	if _, err := os.Lstat(filepath.Join(dst, "node_modules/huge.js")); !os.IsNotExist(err) {
		t.Errorf("unlisted node_modules/huge.js was transferred (err=%v)", err)
	}
}

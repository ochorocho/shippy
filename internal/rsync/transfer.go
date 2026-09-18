package rsync

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gokrazy/rsync/rsyncclient"

	"github.com/ochorocho/shippy/internal/ssh"
	"github.com/ochorocho/shippy/internal/ui"
)

// remoteClient is the subset of *ssh.Client the syncer needs. As an interface it
// lets tests substitute a fake that runs the rsync receiver locally, with no SSH
// connection.
type remoteClient interface {
	RunCommand(cmd string) (string, error)
	MkdirAll(path string) error
	RsyncSender(remoteCmd string, run func(io.ReadWriteCloser) error) error
}

// Syncer handles file synchronization over SSH using rsync.
type Syncer struct {
	client     remoteClient
	remotePath string // release directory (rsync promote destination)
	verbose    bool
	deployPath string // base deploy path (holds .cache/ and .shippy/)
	sourceDir  string // local source root the scanned RelPaths are relative to
}

// NewSyncer creates a new file syncer. sourceDir is the local directory the
// scanned files are relative to (config rsync_src).
func NewSyncer(client remoteClient, remotePath string, verbose bool, deployPath, sourceDir string) *Syncer {
	return &Syncer{
		client:     client,
		remotePath: remotePath,
		verbose:    verbose,
		deployPath: deployPath,
		sourceDir:  sourceDir,
	}
}

// Sync transfers the scanned files to the release directory using rsync:
//  1. rsync-push the exact file list into a persistent remote .cache/ (block
//     delta over a single stream; unchanged files are skipped natively);
//  2. promote .cache/ -> release with the real remote rsync using
//     --files-from + --delete, which makes the release exactly the scanned set
//     (pruning anything stale in the cache).
//
// The transfer list drives both steps, so filtering stays in shippy's go-git
// scanner (see NewScanner) rather than rsync's pattern matching.
func (s *Syncer) Sync(files []FileInfo) error {
	out := ui.New()

	fmt.Printf("\n")
	// #nosec G104 -- Printf errors in UI output can be safely ignored
	out.Blue.Printf("→ Transferring %d files via rsync\n", len(files))

	cachePath := filepath.Join(s.deployPath, ".cache")
	if err := s.client.MkdirAll(cachePath); err != nil {
		return fmt.Errorf("failed to create remote cache directory: %w", err)
	}

	// Write the NUL-separated transfer list once, reused for the push and the
	// promote.
	listFile, err := writeFilesList(files)
	if err != nil {
		return err
	}
	defer os.Remove(listFile)

	// 1. Push the listed files into the cache over the SSH connection.
	if err := s.pushToCache(cachePath, listFile, out); err != nil {
		return err
	}

	// 2. Prune the cache to exactly the scanned set. gokr-rsync cannot forward
	// --delete (its sender deadlocks an openrsync receiver), so files removed
	// from the project would otherwise linger in the cache and be promoted into
	// the release. See third_party/rsync/SHIPPY_PATCHES.md.
	if err := s.pruneCache(cachePath, files); err != nil {
		return err
	}

	// 3. Mirror cache -> release with the real remote rsync. A plain --delete
	// mirror makes the release exactly the (now-clean) cache.
	out.Info("  Promoting cache to release directory...")
	promote := fmt.Sprintf(
		"rsync -rlt --no-perms --delete %s/ %s/",
		ssh.Quote(cachePath), ssh.Quote(s.remotePath),
	)
	if output, err := s.client.RunCommand(promote); err != nil {
		return fmt.Errorf("failed to promote cache to release: %w (output: %s)", err, output)
	}

	out.Success("Release directory synchronized")
	return nil
}

// pruneCache removes files and symlinks from the remote cache that are not in
// the scanned set, so the subsequent mirror promote yields exactly that set.
// find is used with paths only (no stat), portable across GNU and BSD.
func (s *Syncer) pruneCache(cachePath string, files []FileInfo) error {
	want := make(map[string]bool, len(files))
	for _, f := range files {
		want[f.RelPath] = true
	}

	listCmd := fmt.Sprintf("cd %s && find . \\( -type f -o -type l \\) -print0 2>/dev/null", ssh.Quote(cachePath))
	output, err := s.client.RunCommand(listCmd)
	if err != nil {
		return fmt.Errorf("failed to list cache for pruning: %w", err)
	}

	var stale []string
	for _, p := range strings.Split(output, "\x00") {
		p = strings.TrimPrefix(p, "./")
		if p == "" || want[p] {
			continue
		}
		stale = append(stale, p)
	}
	if len(stale) == 0 {
		return nil
	}

	// Delete in batches to stay within remote argv limits.
	const batchSize = 100
	for start := 0; start < len(stale); start += batchSize {
		end := min(start+batchSize, len(stale))
		quoted := make([]string, 0, end-start)
		for _, p := range stale[start:end] {
			quoted = append(quoted, ssh.Quote(filepath.Join(cachePath, p)))
		}
		rm := "rm -f -- " + strings.Join(quoted, " ")
		if output, err := s.client.RunCommand(rm); err != nil {
			return fmt.Errorf("failed to prune stale cache files: %w (output: %s)", err, output)
		}
	}
	return nil
}

// pushToCache runs the rsync client-sender against the remote rsync receiver,
// transferring exactly the files in listFile into cachePath.
func (s *Syncer) pushToCache(cachePath, listFile string, out *ui.Output) error {
	client, err := rsyncclient.New([]string{
		"-rlt", "--no-perms",
		"--files-from=" + listFile, "--from0",
	}, rsyncclient.WithSender())
	if err != nil {
		return fmt.Errorf("failed to build rsync client: %w", err)
	}

	// The remote receiver is the host's real rsync, invoked as a server with the
	// options the client derives (the file list is sent in-band, not here).
	serverArgs := client.ServerCommandOptions(cachePath)
	quoted := make([]string, len(serverArgs))
	for i, a := range serverArgs {
		quoted[i] = ssh.Quote(a)
	}
	remoteCmd := "rsync " + strings.Join(quoted, " ")

	src := strings.TrimSuffix(filepath.Clean(s.sourceDir), "/") + "/"

	out.Info("  Uploading changed files to cache...")
	var result *rsyncclient.Result
	err = s.client.RsyncSender(remoteCmd, func(conn io.ReadWriteCloser) error {
		r, runErr := client.Run(context.Background(), conn, []string{src})
		result = r
		return runErr
	})
	if err != nil {
		return fmt.Errorf("rsync transfer failed: %w", err)
	}

	if result != nil && result.Stats != nil {
		out.Success("Transferred %.2f MB (%.2f MB over the wire)",
			float64(result.Stats.Size)/(1024*1024),
			float64(result.Stats.Written)/(1024*1024))
	}
	return nil
}

// writeFilesList writes the scanned relative paths to a NUL-separated temp file
// for rsync --files-from --from0. NUL separators keep paths with spaces or
// newlines intact.
func writeFilesList(files []FileInfo) (string, error) {
	tmp, err := os.CreateTemp("", "shippy-files-*.list")
	if err != nil {
		return "", fmt.Errorf("failed to create transfer list: %w", err)
	}
	defer tmp.Close()

	var b strings.Builder
	for _, f := range files {
		b.WriteString(f.RelPath)
		b.WriteByte(0)
	}
	if _, err := tmp.WriteString(b.String()); err != nil {
		// #nosec G104 -- best-effort cleanup of the temp file; the write error below is what matters
		os.Remove(tmp.Name())
		return "", fmt.Errorf("failed to write transfer list: %w", err)
	}
	return tmp.Name(), nil
}

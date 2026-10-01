package rsync

import (
	"bytes"
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
	fileMode   string // octal mode applied to deployed files (e.g. "0644")
	dirMode    string // octal mode applied to deployed directories (e.g. "2755")
}

// NewSyncer creates a new file syncer. sourceDir is the local directory the
// scanned files are relative to (config rsync_src). fileMode/dirMode are octal
// strings that normalize the deployed file/directory modes on the remote.
func NewSyncer(client remoteClient, remotePath string, verbose bool, deployPath, sourceDir, fileMode, dirMode string) *Syncer {
	return &Syncer{
		client:     client,
		remotePath: remotePath,
		verbose:    verbose,
		deployPath: deployPath,
		sourceDir:  sourceDir,
		fileMode:   fileMode,
		dirMode:    dirMode,
	}
}

// chmodArg returns the rsync --chmod spec that normalizes deployed modes:
// directories to s.dirMode, files to s.fileMode, plus F+X to keep the exec bit
// for files that carry it in the source. It returns "" when both modes are unset
// (tests on rsync builds without --chmod), so callers fall back to plain --perms.
// The setgid "s" flag and the group are NOT set here; --chmod sets the base bits
// and both are realized via filesystem inheritance from the setgid deploy tree
// (.cache/ and releases/ created under a setgid $siteroot). Always pair it with
// --perms so the remote umask does not strip the setgid/group-write bits.
func (s *Syncer) chmodArg() string {
	if s.fileMode == "" && s.dirMode == "" {
		return ""
	}
	return fmt.Sprintf("--chmod=D%s,F%s,F+X", s.dirMode, s.fileMode)
}

// Sync transfers the scanned files to the release directory using rsync:
//  1. rsync-push the exact file list into a persistent remote .cache/ (block
//     delta over a single stream; unchanged files are skipped natively, and
//     --delete prunes files removed from the project so the cache stays exactly
//     the scanned set);
//  2. mirror .cache/ -> the fresh release directory with the real remote rsync.
//
// The scanned file list drives the push, so filtering stays in shippy's go-git
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

	// Ensure the release directory (and its parents) exists before the promote.
	// The deployer normally creates it first, but a tridge rsync receiver only
	// creates the final path component, not intermediate parents, so the mirror
	// below would fail without this. Idempotent when the dir already exists.
	if err := s.client.MkdirAll(s.remotePath); err != nil {
		return fmt.Errorf("failed to create remote release directory: %w", err)
	}

	// Write the NUL-separated transfer list once, reused for the push and the
	// promote.
	listFile, err := writeFilesList(files)
	if err != nil {
		return err
	}
	defer os.Remove(listFile)

	// 1. Push the listed files into the cache over the SSH connection. The push
	// runs rsync --delete, so the remote rsync prunes files removed from the
	// project during the transfer, leaving the cache exactly the scanned set.
	if err := s.pushToCache(cachePath, listFile, len(files), out); err != nil {
		return err
	}

	// 2. Mirror cache -> release with the real remote rsync (into a fresh
	// timestamped release directory).
	out.Info("  Promoting cache to release directory...")
	// Normalize modes into the release. --chmod must be paired with --perms, or the
	// remote umask strips the setgid/group-write bits (D2775->2755, F664->644). The
	// real remote rsync applies --chmod (the gokr push client cannot); the setgid "s"
	// flag and group come from inheritance off the setgid release parent, not --chmod.
	// --perms + --chmod also re-normalizes modes on unchanged .cache entries, so a
	// pre-fix .cache self-heals on the next deploy. When both modes are empty (tests
	// on rsync builds without --chmod), fall back to plain --perms.
	modeArg := "--perms"
	if c := s.chmodArg(); c != "" {
		modeArg = "--perms " + ssh.Quote(c)
	}
	promote := fmt.Sprintf(
		"rsync -rlt %s --delete %s/ %s/",
		modeArg, ssh.Quote(cachePath), ssh.Quote(s.remotePath),
	)
	if output, err := s.client.RunCommand(promote); err != nil {
		return fmt.Errorf("failed to promote cache to release: %w (output: %s)", err, output)
	}

	out.Success("Release directory synchronized")
	return nil
}

// pushToCache runs the rsync client-sender against the remote rsync receiver,
// transferring exactly the files in listFile into cachePath.
func (s *Syncer) pushToCache(cachePath, listFile string, totalFiles int, out *ui.Output) error {
	// --info=name1 makes the client emit each transferred file's path on stdout
	// (see the vendored SHIPPY PATCH), which drives the progress UI. It does not
	// set --verbose, so it is not forwarded to the remote rsync server.
	progress := &progressWriter{out: out, verbose: s.verbose, total: totalFiles}

	// The client's logger (SendFileList/"building file list" debug lines) writes
	// to its stderr. Silence that noise in normal mode; surface it in verbose for
	// debugging. Genuine transfer failures come back as the Run error, not here.
	clientStderr := nopWriteCloser{io.Discard}
	if s.verbose {
		clientStderr = nopWriteCloser{os.Stderr}
	}
	// --perms forwards -p to the remote rsync receiver so it applies the source
	// modes (which the sender always transmits) into .cache/ instead of using the
	// remote umask.
	client, err := rsyncclient.New([]string{
		"-rlt", "--perms", "--delete", "--info=name1",
		"--files-from=" + listFile, "--from0",
	}, rsyncclient.WithSender(), rsyncclient.WithStdout(progress), rsyncclient.WithStderr(clientStderr))
	if err != nil {
		return fmt.Errorf("failed to build rsync client: %w", err)
	}

	// The remote receiver is the host's real rsync, invoked as a server with the
	// options the client derives (the file list is sent in-band, not here).
	serverArgs := client.ServerCommandOptions(cachePath)
	// The gokr client cannot emit --chmod, so inject it into the server command
	// directly (right after the leading --server). The remote's real rsync honors
	// it, normalizing .cache modes exactly as the promote leg does for the release.
	// Guarded on serverArgs[0]=="--server" so a vendored-option change can't misplace
	// the flag ahead of the positional ". <path>" trailer. Empty modes => no inject.
	if c := s.chmodArg(); c != "" && len(serverArgs) > 0 && serverArgs[0] == "--server" {
		injected := make([]string, 0, len(serverArgs)+1)
		injected = append(injected, serverArgs[0], c)
		injected = append(injected, serverArgs[1:]...)
		serverArgs = injected
	}
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
	if !s.verbose && progress.count > 0 {
		out.ClearLine()
	}
	if err != nil {
		return fmt.Errorf("rsync transfer failed: %w", err)
	}

	if result != nil && result.Stats != nil {
		out.Success("Uploaded %d files, %.2f MB (%.2f MB over the wire)",
			progress.count,
			float64(result.Stats.Size)/(1024*1024),
			float64(result.Stats.Written)/(1024*1024))
	}
	return nil
}

// progressWriter turns the client's per-file name stream (--info=name1) into
// shippy's UI: a progress bar in normal mode, one line per file in verbose mode.
// The client writes to it from a single goroutine during the transfer.
type progressWriter struct {
	out     *ui.Output
	verbose bool
	total   int
	count   int
	buf     []byte
}

func (w *progressWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimRight(string(w.buf[:i]), "\r")
		w.buf = w.buf[i+1:]
		if line == "" {
			continue
		}
		w.count++
		if w.verbose {
			// #nosec G104 -- UI output errors can be safely ignored
			w.out.Yellow.Printf("  Uploading: %s\n", line)
		} else {
			w.out.PrintProgressBar(w.count, w.total, line)
		}
	}
	return len(p), nil
}

func (w *progressWriter) Close() error { return nil }

// nopWriteCloser adapts an io.Writer to io.WriteCloser (rsyncclient options
// require a WriteCloser) with a no-op Close, so wrapping os.Stderr never closes
// it.
type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

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

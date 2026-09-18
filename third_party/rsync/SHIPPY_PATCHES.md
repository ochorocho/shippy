# Shippy patches to gokrazy/rsync

This is a **vendored copy** of `github.com/gokrazy/rsync` @ **v0.3.7** (BSD-3,
LICENSE retained), wired into shippy via `replace github.com/gokrazy/rsync =>
./third_party/rsync` in the top-level `go.mod`.

Shippy uses only the public `rsyncclient` package (client-**sender** mode) to
push files to a host's real `rsync --server` over shippy's existing SSH
connection. Upstream's sender is incomplete for that use case, so we carry the
minimal patch below. All patched lines are marked `SHIPPY PATCH` in the source.

## Patch: `--files-from` support on the client-sender

Lets the caller drive the transfer from an explicit file list instead of walking
the source tree. Shippy feeds the exact output of its go-git `.gitignore` scanner
this way, so filtering stays in shippy (upstream's sender filter matcher is a
stub that panics on wildcards) and the file set that ships is fully controlled.

Files touched:
- `internal/rsyncopts/rsyncopts.go` — enable `--files-from`/`--from0`/`--no-from0`
  in `gokrazyTable()` (were commented out); add `FilesFrom()` / `EolNulls()`
  accessors. (`ParseArguments` already set `relative_paths`/`implied_dirs` when
  `files_from` is set.)
- `internal/sender/flist.go` —
  - factor per-entry emission out of `walkFn` into a shared `emit()`;
  - factor source resolution out of `walk()` into `openSource()`;
  - add `emitList()`: build the flist from the explicit list plus its implied
    parent directories (deduped, lexically sorted root→leaf), emitting each via
    `emit()`; symlinks handled by the existing `Readlink` path;
  - add `readFilesFrom()` (splits on NUL when `--from0`, else newline);
  - `SendFileList()` branches to `emitList()` when `FilesFrom()` is set.

Validated by `rsyncclient/shippy_filesfrom_test.go` against a real rsync
receiver: exact-list transfer, nested + spaced names, symlink preserved, unlisted
files excluded.

## Patch: per-file name output for progress UI

`rsyncclient/rsyncclient.go` gains `WithStdout(io.WriteCloser)` (parallel to the
existing `WithStderr`) so shippy can capture what the client writes to stdout.

`internal/sender/sender.go` printed the transferred file name only when both
`--info=name` **and** `--info=progress` were set, coupling name output to the
per-file progress display. Real rsync prints names with `-v`/`--info=name`
alone. Patched to print on `--info=name` by itself, so shippy can pass
`--info=name1` (which does not set `--verbose`, hence is not forwarded to the
remote server) and render its own progress bar / file list from the name stream.

## Patch: cap literal token size at CHUNK_SIZE (32 KiB)

`internal/sender/sender.go` `sendFile()` read and wrote whole-file literal data
in 256 KiB chunks, each emitted as one uncompressed rsync token. rsync's
protocol caps an uncompressed token at `CHUNK_SIZE` (32 KiB); a *tridge* rsync
receiver (Linux, the common deploy target) rejects anything larger with
`invalid uncompressed token length 262144 [receiver]` and drops the connection.
openrsync (macOS) does not enforce the limit, which is why it only surfaced
against tridge. Upstream chose 256 KiB purely for throughput. Fixed to
`32 * 1024`.

Without this, pushing any non-trivial file (>32 KiB) to a real rsync server
fails. Covered by the large-file case in shippy's `internal/rsync` tests and the
BATS integration deploy.

## Deliberately NOT patched: `--delete` forwarding

`internal/rsyncopts/serveroptions.go` leaves the `--delete` forwarding commented
out (upstream default). gokr-rsync's sender does not drive the receiver's
deletion phase; forwarding `--delete` **deadlocks** against an openrsync (macOS)
`--server` receiver.

Shippy instead performs authoritative deletion in its **cache → release promote**
step, run by the *real* remote rsync (which supports `--files-from` + `--delete`
fully): `rsync -rlt --no-perms --delete --files-from=<list> --from0 cache/ release/`.
Consequence: the persistent `.cache/` may accumulate files removed from the
project over time (disk only) — releases are always exactly the scanned set.

## Updating / rebasing

Re-vendor a newer upstream, then re-apply the `SHIPPY PATCH` hunks (grep for the
marker). Keep the patch minimal. Re-run `go -C third_party/rsync test ./...` and
shippy's `internal/rsync` tests.

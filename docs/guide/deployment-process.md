---
title: Deployment Process
description: The nine steps Shippy performs on every deploy, and why the site only goes live at step seven.
---

# Deployment Process

When you run `shippy deploy <host>`, the following steps occur:

1. **Scan files** — Walks source directory, applies the deny-by-default allowlist (include/exclude
   patterns)
2. **Connect to server** — Establishes SSH connection
3. **Create release** — Creates a new timestamped release directory (e.g. `releases/20260109203841`).
   If that name is already taken because two deploys landed in the same second, a numeric suffix is
   appended (`20260109203841.1`), which keeps the lexical order that rollback and cleanup rely on
4. **Sync files** — Transfers the scanned files to the server with rsync and fills the new release
   directory from a persistent cache (see [below](#how-files-are-transferred))
5. **Create symlinks** — Links shared files/directories from `shared/` to the release
6. **Execute commands** — Runs commands **in the new release directory** (e.g. cache flush,
   migrations)
7. **Activate release** — Atomically updates `current` symlink to new release (site goes live)
8. **Post-release commands** — Runs `commands_post_release` against `current/` (e.g. flush opcache,
   leave maintenance mode). Skipped when none are configured
9. **Cleanup** — Removes old releases, keeps last N

::: info Commands run before the release goes live
Commands execute in the new release directory **before** it goes live. This ensures all preparation
(cache warming, migrations, etc.) completes successfully before the atomic switchover. The site only
becomes live when the `current` symlink is updated in step 7. Anything that has to happen once the
release is live belongs in [`commands_post_release`](./deployment-commands#post-release-commands).
:::

## Why this is zero-downtime

Steps 3 to 6 happen in a directory nobody is serving yet. Your visitors keep hitting the previous
release the entire time files are uploading and commands are running. Step 7 replaces a single
symlink, which is atomic at the filesystem level — there is no window in which the document root
points at a partially written release.

That also means a failure during steps 1–6 leaves production untouched: the broken release directory
is simply never activated.

## How files are transferred

Step 4 is an rsync transfer, driven by an rsync client compiled into the Shippy binary and the
`rsync` installed on the server. Two things happen:

1. **Push to the cache.** The scanned file list is streamed over the existing SSH connection into
   `<deploy_path>/.cache/`. This is a real rsync delta: files that are unchanged since the last
   deploy are skipped natively, changed files only send the blocks that differ, and `--delete`
   removes anything that has since left your allowlist. A progress bar shows the upload;
   `--verbose` prints every file instead.
2. **Promote to the release.** The server's own `rsync` mirrors `.cache/` into the fresh release
   directory. That is a local copy on the host, so it is fast even for large `vendor/` trees.

Deployed files and directories get normalized modes on the server rather than inheriting the modes
from your machine or the server's umask. By default files are `0664` and directories `2775`
(group-writable, setgid), and files that are executable in your source keep their exec bit. See
[Permissions](./configuration#file-and-directory-permissions).

Symbolic links inside the project are recreated as links on the server rather than dereferenced, so
the runtime symlinks Composer and TYPO3 create (for example under `public/_assets/`) keep working.

The cache is what makes the second and every later deploy cheap: only what changed since the previous
deploy crosses the network. It is safe to delete `.cache/` on the server; the next deploy simply
uploads everything again.

::: info Requirements on the server
`rsync` must be installed on the target host. Nothing else is required beyond the SSH account.
:::

## Previewing a deployment

`--dry-run` runs the scan and resolves the command list without connecting to the host, so you can
confirm the allowlist picks up what you expect before anything is transferred:

```bash
shippy deploy production --dry-run
```

## One deployment at a time

Shippy takes a lock on the host for the duration of a deploy, so a second deployment cannot start
while one is running. It is on by default and expires after 15 minutes — see
[Deployment Locking](./configuration#deployment-locking), and
[`shippy unlock`](../reference/commands#unlock) if a crashed run leaves a lock behind.

## Rolling back

Because the previous releases are still on disk (see `keep_releases` in
[Configuration](./configuration#directory-structure)), undoing a deployment is the same symlink
swap in reverse:

```bash
shippy rollback production              # Interactive release selection
shippy rollback production -n -1        # One version back
```

Any `rollback_commands` you configured — typically a cache flush and warmup — run as part of it. See
the [`rollback` reference](../reference/commands#rollback) for all flags.

## What Shippy does not do

Shippy deploys files and runs commands. It does not manage your web server, PHP-FPM pools, database
schema (beyond whatever your own commands do), or DNS. That is deliberate: the same nine steps
behave identically on any host you can reach over SSH.

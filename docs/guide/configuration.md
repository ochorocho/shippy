---
title: Configuration
description: Everything .shippy.yaml can express — hosts, template variables, shared paths, deployment locking and the on-server directory structure.
---

# Configuration

Shippy is configured by a single `.shippy.yaml` in your project root. `shippy init` generates one for
you; this page explains every part of it. For a terse key-by-key table, see the
[Configuration Options reference](../reference/configuration).

## Basic Structure

```yaml
hosts:
  <hostname>:
    # SSH connection
    hostname: <server domain or IP>
    port: <SSH port, default: 22>
    remote_user: <SSH username>
    ssh_key: <path to SSH private key>
    ssh_options: <map of SSH options, see below>

    # Deployment
    deploy_path: <absolute path on server>
    rsync_src: <local source directory>
    keep_releases: <number of releases to keep, default: 5>
    file_mode: <octal mode for deployed files, default: "0664">
    dir_mode: <octal mode for deployed directories, default: "2775">

    # File management
    shared: <list of shared paths>
    include: <allowlist - paths to deploy (deny-by-default)>
    exclude: <carve-outs that win over includes>

commands:
  - name: <command description>
    run: <command to execute>

commands_post_release:   # optional, run after the release is live
  - name: <command description>
    run: <command to execute>
```

The key under `hosts:` is the name you pass on the command line — `shippy deploy production` uses the
`hosts.production` entry.

Two parts of this file are large enough to have their own pages:

- **`include:` / `exclude:`** — Shippy deploys an explicit allowlist and ships nothing you haven't
  listed. See [File Selection](./file-selection).
- **`commands:` / `commands_post_release:` / `rollback_commands:`** — including per-host scoping and running inside a container.
  See [Deployment Commands](./deployment-commands).

## Which files get deployed

Nothing ships unless you list it under `include:`; `exclude:` then punches holes in that allowlist,
and a built-in junk list is carved out on top. This is covered in full on
[File Selection](./file-selection), including the pattern syntax, the default excludes and how to
migrate a configuration written for an older version.

```yaml
hosts:
  production:
    include:
      - "public/"            # Web root (ships the whole subtree)
      - "vendor/"            # Composer dependencies
      - "config/"            # TYPO3 configuration
      - "composer.json"
      - "composer.lock"

    exclude:
      - "public/typo3temp/"  # Carve-outs win over includes
```

## Template Variables

Use `{{key.path}}`{v-pre} syntax to reference values from composer.json:

```yaml
hosts:
  production:
    deploy_path: /var/www/{{name}}  # Uses composer.json "name" field
```

Access nested values:

```yaml
deploy_path: /var/www/{{extra.typo3/cms.web-dir}}
```

Provide a fallback value with `|` (used when the key is not found in composer.json):

```yaml
commands:
  - name: Clear cache
    run: ./{{config.bin-dir|vendor/bin}}/typo3 cache:flush
```

Run [`shippy config validate`](../reference/commands#validate-configuration) to see what these
expand to before deploying.

## Environment Variables

Use `${VAR}` syntax to reference environment variables:

```yaml
hosts:
  production:
    hostname: ${DEPLOY_HOST}
    remote_user: ${DEPLOY_USER}
```

Provide a fallback value with `|` (used when the variable is not set):

```yaml
hosts:
  production:
    deploy_path: ${DEPLOY_PATH|/var/www/html}
```

::: warning
If an environment variable is not set and no fallback is provided, deployment will fail with an
error.
:::

This is how you keep secrets and per-environment values out of the repository — see
[CI/CD](./ci-cd).

## Shared Files/Directories

Files and directories in the `shared:` list are symlinked from the `shared/` directory to each
release:

```yaml
shared:
  - .env                    # Shared file
  - var/log/                # Shared directory (note trailing slash)
  - public/fileadmin/
  - public/uploads/
```

Anything that must survive a deployment — user uploads, logs, sessions, the environment file — belongs
here. Everything else is replaced wholesale with each release.

## Deployment Locking

To prevent two deployments from running against the same host at once, Shippy writes a lock file to
the remote `deploy_path` at the start of a deploy and removes it when finished. Locking is **enabled
by default** with a 15-minute timeout, after which a stale lock (e.g. from a crashed deployment) is
considered expired and automatically overridden.

```yaml
# Global defaults (can be overridden per host)
lock_enabled: true    # Enable deployment locking (default: true)
lock_timeout: 15      # Minutes before a stale lock expires (default: 15)

hosts:
  production:
    hostname: example.com
    remote_user: deploy
    deploy_path: /var/www/myproject
    # lock_enabled: false   # Per-host override to disable locking
```

If a deployment fails and leaves a stale lock behind before the timeout elapses, clear it manually
with [`shippy unlock`](../reference/commands#unlock).

## Directory Structure

Shippy creates the following structure on the server (following Deployer/Capistrano conventions):

```
/var/www/myproject/
├── current -> releases/20240109120000    # Symlink to latest release
├── releases/
│   ├── 20240109120000/                   # Current release
│   ├── 20240109110000/                   # Previous release
│   └── 20240109100000/                   # Older release
└── shared/
    ├── .env                              # Shared files
    ├── var/
    │   ├── log/
    │   └── session/
    └── public/
        ├── fileadmin/
        └── uploads/
```

Point your web server's document root at `current/public` (or wherever your web directory sits inside
the project). Because `current` is a symlink that Shippy swaps atomically, no request ever sees a
half-written release.

`keep_releases` controls how many entries under `releases/` are retained after a successful deploy —
it defaults to `5`, and those are exactly the releases you can roll back to.

Shippy also keeps two hidden directories next to these: `.cache/` holds the rsync cache that makes
repeat deploys fast (see [How files are transferred](./deployment-process#how-files-are-transferred)),
and `.shippy/` holds the deployment lock. Both can be deleted when no deploy is running.

## File and directory permissions

Shippy sets the modes of deployed files and directories explicitly instead of copying them from your
machine or leaving them to the server's umask:

```yaml
hosts:
  production:
    file_mode: "0664"   # default
    dir_mode: "2775"    # default
```

- Files get `file_mode`, directories get `dir_mode`. Files that are executable in your source keep
  their exec bit.
- The defaults are group-writable, so the web-server group (e.g. `www-data`) can write to the
  deployed tree.
- Quote the values. Unquoted, YAML would not read `0664` as octal.
- Both can be set globally or per host; the per-host value wins. `shippy config validate` rejects
  anything that is not a valid octal mode.

::: info Setgid comes from the deploy tree
`dir_mode`'s leading `2` is the setgid bit, but the group itself is inherited from the filesystem.
For new files to land in the right group, make your `deploy_path` setgid and owned by that group
(for example `chgrp www-data` and `chmod 2775`).
:::

## Next steps

- [File Selection](./file-selection) — the allowlist that decides what ships
- [Deployment Commands](./deployment-commands) — command scoping and `command_context`
- [SSH Connections](./ssh) — keys, agent, ports, timeouts and host key verification
- [Example Configurations](./examples) — complete minimal and advanced files
- [Configuration Options](../reference/configuration) — every key in one table

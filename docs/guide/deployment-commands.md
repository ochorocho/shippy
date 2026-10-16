---
title: Deployment Commands
description: Commands that run in the new release before it goes live — scoping them per host with only/except, and running them inside a container with command_context.
---

# Deployment Commands

`commands` (and `rollback_commands`) run in the new release directory, in order, before the atomic
switchover. Each entry needs a `name` and a `run`:

```yaml
commands:
  - name: Install dependencies
    run: composer install --no-dev --optimize-autoloader

  - name: Database migrations
    run: ./vendor/bin/typo3 upgrade:run
```

::: info
Because they run before the `current` symlink moves, a failing command means the release never goes
live — see [Deployment Process](./deployment-process).
:::

## Post-release commands

`commands_post_release` run **after** the atomic switchover, against the `current/` path, which now
points at the live release. Use them for actions that must happen once the release is live, such as
flushing PHP's opcache or leaving maintenance mode:

```yaml
commands_post_release:
  - name: Flush PHP opcache
    run: ./vendor/bin/typo3 cache:flush
```

They have no defaults — nothing runs unless you configure it — and they support the same
[`only` / `except` scoping](#scoping-a-command-to-specific-hosts) and
[`command_context`](#running-commands-inside-a-container) as `commands`.

::: warning
The release is already live when these run. A failing post-release command fails the deploy, but it
does not roll the release back.
:::

## Scoping a command to specific hosts

By default a command runs for every host in `hosts:`. Use `only` / `except` (GitLab-CI style) to
scope a single command instead of duplicating the whole command list per host:

```yaml
commands:
  - name: Database migrations
    run: ./vendor/bin/typo3 upgrade:run
    # Only run this command for the listed host(s) (keys under `hosts:`).
    # Skipped everywhere else. Omit `only`/`except` entirely to run on every host.
    only:
      - production

  - name: Notify monitoring
    run: /usr/local/bin/notify-deploy
    # except is the inverse of only: runs everywhere except the listed hosts.
    except:
      - staging
```

- `only` and `except` accept a list of host names (the keys under `hosts:`).
- If a host matches both, `except` wins.
- Commands skipped for a host are logged as skipped during `deploy`/`rollback`.
- `shippy config validate` rejects `only`/`except` entries that reference a host name not defined
  under `hosts:`.
- `shippy config show <host>` lists only the commands that actually apply to that host;
  `shippy config validate` annotates every command with its resolved scope.

## Running commands inside a container

If your PHP sources are mounted into a container, `command_context` runs every command inside a
subcontext instead of directly on the remote host. When set, Shippy executes each command as:

```
<command_context> sh -c 'cd <release-dir> && <run>'
```

Shippy adds `sh -c` itself — give only the container-entry prefix, without a trailing shell
(`docker exec php85`, not `docker exec php85 bash`). The `cd` happens *inside* the context, so the
release directory must resolve to the same path there (the normal bind-mount case, e.g.
`/var/www:/var/www`).

`command_context` can be set globally, per host, or per command — the most specific one wins:

```yaml
# Global default for every host and command (optional; empty = run directly on the host)
command_context: docker exec -u www-data php85

hosts:
  production:
    hostname: example.com
    remote_user: deploy
    command_context: docker exec php84  # Overrides the global default for this host

commands:
  - name: Notify monitoring
    run: /usr/local/bin/notify-deploy
    # Per-command override. An empty string forces this command to run
    # directly on the host even when a global/per-host context is set.
    command_context: ""
```

Precedence: per-command > per-host (`hosts.<name>.command_context`) > global `command_context`.

## Checking what will run

`shippy config show <host>` prints the effective command list for one host, with commands filtered
out by `only`/`except` shown as comments:

```bash
shippy config show production
```

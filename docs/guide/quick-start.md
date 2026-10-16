---
title: Quick Start
description: From an empty configuration to a first zero-downtime deployment in four commands.
---

# Quick Start

This walks through a first deployment. It assumes Shippy is [installed](./installation) and that you
have SSH access to the target server with key authentication.

## 1. Initialize configuration

Run this in your TYPO3 project root:

```bash
shippy init
```

This will create a `.shippy.yaml` file with sensible TYPO3 defaults and read your project name from
`composer.json`.

## 2. Edit configuration

```bash
vim .shippy.yaml
```

Update at minimum:

- `hostname` — your server's domain or IP
- `remote_user` — SSH username
- `ssh_key` — path to your SSH private key
- `include` — the allowlist of paths to deploy (deny-by-default; adjust to your project layout)

::: tip
`ssh_key` is optional. If you leave it out, Shippy looks for `~/.ssh/id_ed25519`, `~/.ssh/id_rsa` and
`~/.ssh/id_ecdsa` in that order, and any key held by a running `ssh-agent` is offered as well — see
[SSH Connections](./ssh).
:::

::: warning `include` is not optional
Shippy deploys an allowlist: nothing ships unless it is listed under `include:`. Without one,
`shippy deploy` scans 0 files. See [File Selection](./file-selection).
:::

A minimal configuration looks like this:

```yaml
hosts:
  production:
    hostname: www.example.com
    remote_user: deploy
    deploy_path: /var/www/{{name}}
    rsync_src: ./
    ssh_key: ~/.ssh/id_rsa
    # Deny-by-default: list exactly what should ship
    include:
      - public/
      - vendor/
      - config/
      - composer.json
      - composer.lock
    shared:
      - .env
      - var/log/
      - var/session/
      - public/fileadmin/
      - public/uploads/

commands:
  - name: Clear TYPO3 cache
    run: ./vendor/bin/typo3 cache:flush

  - name: Run extension setup
    run: ./vendor/bin/typo3 extension:setup
```

## 3. Validate configuration

```bash
shippy config validate
```

This checks the YAML syntax and required fields, resolves the `composer.json` template variables, and
prints the processed configuration — so you can see exactly what `{{name}}`{v-pre} expanded to before
anything touches the server.

## 4. Deploy to production

Preview first — `--dry-run` resolves the allowlist and command list without connecting to the host,
so you can check the file count before anything is transferred:

```bash
shippy deploy production --dry-run
```

Then ship it:

```bash
shippy deploy production
```

Shippy scans your files, creates a new timestamped release, syncs, links the shared paths, runs your
commands, and finally switches the `current` symlink. Read
[Deployment Process](./deployment-process) for what happens at each step.

## Undo a deployment

If something is wrong, point `current` back at the previous release:

```bash
shippy rollback production -n -1
```

Or run `shippy rollback production` with no flags to pick from an interactive list of releases.

## Where to go next

- [File Selection](./file-selection) — building the `include:` allowlist for your project
- [Configuration](./configuration) — shared paths, locking, template and environment variables
- [SSH Connections](./ssh) — key detection, agent, ports, timeouts, host key checking
- [CI/CD](./ci-cd) — deploy from GitHub Actions or GitLab CI
- [Example Configurations](./examples) — a minimal and an advanced multi-host setup

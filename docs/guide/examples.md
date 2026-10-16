---
title: Example Configurations
description: A minimal TYPO3 configuration and an advanced multi-host setup with staging and production.
---

# Example Configurations

Two complete `.shippy.yaml` files to copy from. Every key used here is explained in
[Configuration](./configuration).

## Minimal TYPO3 Configuration

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

## Complete TYPO3 Include List

Under deny-by-default you must explicitly list every path a Composer-based TYPO3
installation needs to run. The block below is the complete allowlist for a
standard project, with each entry annotated. Copy it and delete the optional
lines that don't apply to your project.

```yaml
hosts:
  production:
    hostname: www.example.com
    remote_user: deploy
    deploy_path: /var/www/{{name}}
    rsync_src: ./

    include:
      # --- Required: a Composer TYPO3 install will not boot without these ---
      - public/          # Web root: index.php, typo3/, _assets/, installed extensions' Resources/Public
      - vendor/          # All Composer dependencies incl. typo3/cms-core and vendor/bin/typo3
                         # (the server does NOT run "composer install")
      - config/          # Site config (config/sites/*/config.yaml) + system config (config/system/*.php)
      - composer.json    # TYPO3 reads it for package metadata / extension autoloading
      - composer.lock    # Pins the installed set; used by post-deploy commands

      # --- Optional: uncomment the ones your project actually uses ---
      # - packages/      # Local site extensions kept in the repo (monorepo layout)
      # - .htaccess      # Root .htaccess, if you serve from the project root
      # - api/           # Additional entry points / sub-apps outside public/

    # Carve-outs: excludes always win over includes. Common junk (.git/,
    # node_modules/, var/cache/, var/log/, .DS_Store, ...) is already excluded
    # automatically, so you only need project-specific holes here.
    exclude:
      - public/typo3temp/   # Generated at runtime, never ship it

    # Runtime/persistent data — symlinked from shared/, never part of a release
    shared:
      - .env
      - var/log/
      - var/session/
      - public/fileadmin/
      - public/uploads/

commands:
  - name: Run extension setup
    run: ./vendor/bin/typo3 extension:setup

  - name: Run upgrade wizards
    run: ./vendor/bin/typo3 upgrade:run

  - name: Flush caches
    run: ./vendor/bin/typo3 cache:flush
```

::: tip Non-standard web directory
If your project uses a non-standard web directory (configured via the `extra.typo3/cms.web-dir` key
in `composer.json`, e.g. `web/` instead of `public/`), include that directory instead of `public/`.
Run `shippy deploy <host> --dry-run` to preview exactly which files the allowlist resolves to before
deploying.
:::

## Advanced Configuration

Two hosts with different retention, per-host excludes, and a fuller command list. Note the
`{{config.bin-dir|vendor/bin}}`{v-pre} template variable, which reads the binary directory from
`composer.json` and falls back to `vendor/bin`.

```yaml
hosts:
  staging:
    hostname: staging.example.com
    remote_user: deploy
    deploy_path: /var/www/{{name}}/staging
    rsync_src: ./
    ssh_key: ~/.ssh/id_rsa
    keep_releases: 3

    # Allowlist: exactly what ships (deny-by-default)
    include:
      - public/
      - vendor/
      - config/
      - composer.json
      - composer.lock

    # Carve-outs (win over includes). Common junk is already excluded.
    exclude:
      - public/typo3temp/
      - Tests/

    # Shared paths
    shared:
      - .env
      - var/log/
      - var/session/
      - public/fileadmin/
      - public/uploads/

  production:
    hostname: www.example.com
    remote_user: deploy
    deploy_path: /var/www/{{name}}/production
    rsync_src: ./
    ssh_key: ~/.ssh/id_rsa_production
    keep_releases: 10
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
    run: ./{{config.bin-dir|vendor/bin}}/typo3 cache:flush

  - name: Run extension setup
    run: ./{{config.bin-dir|vendor/bin}}/typo3 extension:setup

  - name: Database migrations
    run: ./{{config.bin-dir|vendor/bin}}/typo3 upgrade:run

  - name: Warmup caches
    run: ./{{config.bin-dir|vendor/bin}}/typo3 cache:warmup

rollback_commands:
  - name: Flush caches
    run: ./{{config.bin-dir|vendor/bin}}/typo3 cache:flush

  - name: Warmup caches
    run: ./{{config.bin-dir|vendor/bin}}/typo3 cache:warmup
```

Deploy either host by name:

```bash
shippy deploy staging
shippy deploy production
```

## Backup configuration

Add a `backup:` block to have `shippy backup <host>` produce a ZIP with a database dump and the
shared files worth keeping:

```yaml
backup:
  output: ./backups            # Local directory for ZIPs (default: cwd)

  # Files to download from the remote shared/ directory (paths are relative to shared/)
  files:
    - .env
    - public/fileadmin/
    - public/uploads/

  database:
    credentials: auto

    # Exclude tables from the dump (glob patterns)
    exclude_tables:
      - "cache_*"
      - "cf_*"
      - "sys_log"
      - "be_sessions"

    # DBMS-specific options
    options:
      single_transaction: "true"   # MySQL: consistent dump without table locks
```

The full set of `backup:` options — including manual database credentials and per-host overrides — is
documented under [`backup`](../reference/commands#backup).

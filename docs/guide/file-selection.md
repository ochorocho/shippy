---
title: File Selection
description: Shippy deploys an explicit allowlist — nothing ships unless you include it. How include, exclude and the default carve-outs interact.
---

# File Selection: Deny-by-Default (Allowlist)

Shippy deploys files using an **allowlist**: by default **nothing is deployed**
unless it is listed under `include:`. A project root usually contains far more
that should *not* ship (`.git/`, `node_modules/`, dumps, IDE files, `.env`) than
should — an allowlist is safer and less error-prone than trying to exclude every
unwanted path.

::: warning Coming from an older version?
This reverses the previous behaviour. Without an `include:` list, `shippy deploy` scans **0 files**.
See [Migrating from a previous version](#migrating-from-a-previous-version).
:::

## Include — the allowlist

**Include = the allowlist.** List exactly what should ship. A directory entry
ships that directory and everything beneath it:

```yaml
hosts:
  production:
    hostname: example.com
    remote_user: deploy
    deploy_path: /var/www/myproject
    rsync_src: ./

    include:
      - "public/"            # Web root (ships the whole subtree)
      - "vendor/"            # Composer dependencies
      - "config/"            # TYPO3 configuration
      - "composer.json"
      - "composer.lock"
```

For a complete, annotated allowlist for a Composer-based TYPO3 project, see
[Complete TYPO3 Include List](./examples#complete-typo3-include-list).

## Exclude — carve-outs

**Exclude = carve-outs.** Excludes always **win over includes**, so use them to
punch holes in an included directory:

```yaml
    exclude:
      - "public/typo3temp/"  # Drop generated temp files under an included dir
      - "*.log"              # Never ship log files
```

Common junk — `.git/`, `node_modules/`, `var/cache/`, `var/log/`, `.DS_Store`,
`Thumbs.db` and more (see [Default Excludes](#default-excludes)) — is carved out
automatically; you don't need to list it.

## Notes

- `include:`/`exclude:` are **not merged across hosts** — a per-host list
  replaces the global one entirely. Give each host its own allowlist (or define
  one at the global level and omit it per host).
- `.gitignore` is **not** consulted for deployment. You select what ships with
  `include:`; you don't rely on gitignore to deselect.
- Escape hatch: `include: ["*"]` ships (almost) everything; the built-in junk
  list still protects `.git/` etc.
- Symbolic links are shipped **as links** and recreated verbatim on the server,
  not followed. Composer and TYPO3 rely on such links at runtime (e.g.
  `public/_assets/`), so a link whose target lies outside the allowlist will
  dangle on the server until its target is included too.

## Pattern Syntax

- Patterns use gitignore-style syntax
- A single-segment pattern (`vendor/`, `*.log`) matches at any depth
- A multi-segment pattern (`public/index.php`) is anchored to the project root
- Trailing `/` means directory only
- `*` matches within a path segment; `**` matches across segments

## Default Excludes

These carve-out patterns are always applied and **win over your `include:`
allowlist**, so common junk never ships even inside an included directory:

- `.git/`
- `.gitignore`
- `.shippy.yaml`
- `.shippy.yaml.example`
- `node_modules/`
- `.env.local`
- `.env.*.local`
- `var/cache/`
- `var/log/`
- `var/transient/`
- `.DS_Store`
- `Thumbs.db`

## Previewing what will ship

Before the first real deployment, check what the allowlist actually resolves to — no connection to
the server is made:

```bash
shippy deploy production --dry-run
```

## Migrating from a previous version

Earlier versions shipped **everything by default** and used `exclude:` (and
`.gitignore`) to remove unwanted files. Deny-by-default reverses this:

- **Add an `include:` allowlist to every host** (or globally). Without it,
  `shippy deploy` scans **0 files** and warns you.
- Existing `exclude:` entries still work, now as carve-outs on top of your
  includes.
- `.gitignore` no longer affects what is deployed — anything you relied on
  gitignore to exclude is already excluded by default; anything gitignored that
  you still need (e.g. `vendor/`) simply goes in `include:`.
- Run `shippy config validate` and review `shippy deploy`'s "Found N files to
  sync" count before your first real deploy.

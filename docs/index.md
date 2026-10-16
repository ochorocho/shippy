---
layout: home

hero:
  name: Shippy
  text: Deployments for PHP, in a single binary
  tagline: Zero-downtime, atomic releases for Composer-based projects. Inspired by Deployer and Capistrano — without the runtime.
  image:
    src: /logo.svg
    alt: Shippy
  actions:
    - theme: brand
      text: Get started
      link: /guide/quick-start
    - theme: alt
      text: What is Shippy?
      link: /guide/
    - theme: alt
      text: View on GitHub
      link: https://github.com/ochorocho/shippy

features:
  # Icons are inline SVG (VPFeature renders a string icon with v-html), drawn in the
  # logo's idiom: flat fills, no gradients, ink body with an #FF8700 accent. Colours come
  # from CSS custom properties so they invert with the theme — see theme/custom.css.
  - icon: |
      <svg viewBox="0 0 32 32" aria-hidden="true" focusable="false">
        <circle cx="16" cy="16" r="13" fill="var(--shippy-icon-ink)"/>
        <path d="M16 7.8V16h5.6" fill="none" stroke="var(--shippy-orange)" stroke-width="2.8" stroke-linecap="round" stroke-linejoin="round"/>
      </svg>
    title: Zero-downtime releases
    details: Files are synced into a fresh release directory while the old one keeps serving. Going live is one atomic symlink swap.
    link: /guide/deployment-process
    linkText: How a deploy runs
  - icon: |
      <svg viewBox="0 0 32 32" aria-hidden="true" focusable="false">
        <path d="M16 6.5A9.5 9.5 0 1 1 6.5 16" fill="none" stroke="var(--shippy-icon-ink)" stroke-width="3.2" stroke-linecap="round"/>
        <path d="M17.6 1.2v10.6L9 6.5Z" fill="var(--shippy-orange)"/>
      </svg>
    title: Rollback in seconds
    details: The last N releases stay on disk. Pick one from an interactive list, or step back with a single offset flag.
    link: /reference/commands#rollback
    linkText: rollback reference
  - icon: |
      <svg viewBox="0 0 32 32" aria-hidden="true" focusable="false">
        <rect x="3.5" y="4" width="11" height="17" rx="2.2" fill="var(--shippy-icon-ink)"/>
        <rect x="17.5" y="4" width="11" height="17" rx="2.2" fill="var(--shippy-icon-ink)" opacity="0.45"/>
        <rect x="2" y="23" width="28" height="6" rx="2.2" fill="var(--shippy-orange)"/>
      </svg>
    title: Shared files survive
    details: Uploads, logs, sessions and .env are symlinked from shared/ into every release, so nothing is lost between deployments.
    link: /guide/configuration#shared-files-directories
    linkText: Shared paths
  - icon: |
      <svg viewBox="0 0 32 32" aria-hidden="true" focusable="false">
        <path d="M16 3l12 6.5-12 6.5L4 9.5Z" fill="var(--shippy-orange)"/>
        <path d="M4 9.5l12 6.5v13L4 22.5Z" fill="var(--shippy-icon-ink)"/>
        <path d="M28 9.5v13l-12 6.5V16Z" fill="var(--shippy-icon-ink)" opacity="0.55"/>
      </svg>
    title: One static binary
    details: Written in Go. The server only needs SSH and rsync, and nothing is added to your composer.json.
    link: /guide/installation
    linkText: Install it
  - icon: |
      <svg viewBox="0 0 32 32" aria-hidden="true" focusable="false">
        <path d="M3.5 4.5h25L19 16.5h-6Z" fill="var(--shippy-icon-ink)"/>
        <path d="M13 16.5h6v8.6L13 28.5Z" fill="var(--shippy-orange)"/>
        <circle cx="5" cy="22" r="1.7" fill="var(--shippy-icon-ink)" opacity="0.45"/>
        <circle cx="27.5" cy="20.5" r="1.7" fill="var(--shippy-icon-ink)" opacity="0.45"/>
      </svg>
    title: Ships only what you list
    details: Deny-by-default file selection. Nothing leaves your machine unless it is on the include allowlist, so dumps, .env files and build junk cannot slip onto the server.
    link: /guide/file-selection
    linkText: File selection
  # The TYPO3 logo, a trademark of the TYPO3 Association, used here to identify the CMS.
  # Path data and #FF8700 are verbatim from typo3/sysext/backend/Resources/Public/Images/
  # typo3_logo_orange.svg in TYPO3/typo3; the transform only centres and uniformly scales
  # it (source viewBox 43.201 42.122 83.098 84.172, centre 84.75/84.208) so it carries the
  # same optical weight as the drawn icons beside it. The mark itself is not altered.
  - icon: |
      <svg viewBox="0 0 32 32" aria-hidden="true" focusable="false">
        <g transform="translate(16 16) scale(0.303) translate(-84.75 -84.208)">
          <path fill="#FF8700" d="M106.074 100.128c-1.247.368-2.242.506-3.549.506-10.689 0-26.389-37.359-26.389-49.793 0-4.577 1.083-6.104 2.613-7.415-13.084 1.527-28.784 6.329-33.806 12.433-1.085 1.529-1.743 3.926-1.743 6.98 0 19.41 20.718 63.455 35.332 63.455 6.765.001 18.164-11.112 27.542-26.166M99.25 42.122c13.52 0 27.049 2.18 27.049 9.812 0 15.483-9.819 34.246-14.832 34.246-8.942 0-20.065-24.867-20.065-37.301.001-5.67 2.181-6.757 7.848-6.757"/>
        </g>
      </svg>
    title: TYPO3 out of the box
    details: shippy init writes a config that already knows var/log/, public/fileadmin/ and the TYPO3 console commands.
    link: /guide/quick-start
    linkText: Quick start
---

## Install and ship

Three commands to install, four to deploy.

::: code-group

```bash [Homebrew]
brew tap ochorocho/shippy https://github.com/ochorocho/shippy
brew trust ochorocho/shippy
brew install shippy
```

```bash [Go]
go install github.com/ochorocho/shippy@latest
```

```bash [Docker]
docker run --rm ghcr.io/ochorocho/shippy:latest --help
```

:::

```bash
shippy init                     # write .shippy.yaml from your composer.json
vim .shippy.yaml                # add hostname, remote_user, ssh_key, include
shippy config validate          # check syntax, fields and template variables
shippy deploy production --dry-run   # preview the file list, no connection
shippy deploy production        # ship it
```

## One file describes the whole deployment

No scripting DSL, no plugins — a host, an allowlist of what ships, the paths that must survive, and
the commands to run before the release goes live.

```yaml
hosts:
  production:
    hostname: www.example.com
    remote_user: deploy
    deploy_path: /var/www/{{name}}   # {{name}} comes from composer.json
    rsync_src: ./
    ssh_key: ~/.ssh/id_rsa
    keep_releases: 10                # ten releases to roll back to
    include:                         # nothing ships unless it is listed here
      - public/
      - vendor/
      - config/
      - composer.json
      - composer.lock
    shared:                          # symlinked into every release
      - .env
      - var/log/
      - public/fileadmin/
      - public/uploads/

commands:
  - name: Clear TYPO3 cache
    run: ./{{config.bin-dir|vendor/bin}}/typo3 cache:flush

  - name: Database migrations
    run: ./{{config.bin-dir|vendor/bin}}/typo3 upgrade:run
    only: [production]               # scope a command to specific hosts
```

[Full configuration guide →](/guide/configuration)

## What it looks like on the server

The familiar Deployer/Capistrano layout — point your web server at `current/` and never touch it
again.

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

## Nine steps, and the site only moves at step seven

1. **Scan files** — walks the source directory, applying the deny-by-default allowlist
2. **Connect to server** — establishes the SSH connection
3. **Create release** — a new timestamped directory, e.g. `releases/20260109203841`
4. **Sync files** — an rsync delta transfer: only changed blocks travel, and the release is filled from a persistent cache on the server
5. **Create symlinks** — links the shared files and directories
6. **Execute commands** — cache flush, migrations, warmup — inside the new release
7. **Activate release** — atomically repoints `current`; the site goes live
8. **Post-release commands** — opcache flush, leaving maintenance mode — against the now-live `current/`
9. **Cleanup** — removes old releases, keeps the last N

A failure in steps 1–6 never reaches production: the broken release is simply never activated. A lock
on the host keeps a second deployment from starting while one is in flight.

[Read the full sequence →](/guide/deployment-process)

## Ships from your pipeline too

::: code-group

```yaml [GitHub Actions]
# .github/workflows/deploy.yml
name: Deploy
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: ochorocho/shippy-action@v0.0.1
        with:
          args: deploy production
```

```yaml [GitLab CI]
# .gitlab-ci.yml
deploy:
  stage: deploy
  image:
    name: ghcr.io/ochorocho/shippy:latest
    # ENTRYPOINT is "shippy", so it is cleared to run regular shell commands.
    entrypoint: [""]
  rules:
    - if: $CI_COMMIT_BRANCH == "main"
  script:
    - shippy deploy production
```

:::

[CI/CD guide →](/guide/ci-cd)

## Also in the box

- **`shippy backup`** — a ZIP with a database dump and your shared files, with TYPO3 credentials
  auto-detected and cache tables excluded
- **`shippy gitlab:upload`** — push that archive to the GitLab package registry straight from a
  scheduled pipeline
- **`shippy rollback -l`** — list every release with its date, git commit and tag before you choose
- **`shippy config show <host>`** — the fully resolved config for one host, templates expanded and
  per-host overrides applied
- **`shippy unlock`** — clear a lock left behind by a crashed deployment

[Command reference →](/reference/commands)

---
title: What is Shippy?
description: A minimal, opinionated deployment tool for Composer-based PHP projects, inspired by Deployer and Capistrano.
---

# What is Shippy?

Shippy is a minimal, opinionated deployment tool for Composer based PHP projects, inspired by
[Deployer](https://deployer.org/) and [Capistrano](https://capistranorb.com/).

It ships as a single Go binary. The target server needs nothing beyond an SSH account and `rsync`:
Shippy connects over SSH, streams your project to the server with rsync, mirrors it into a fresh
timestamped release directory, links the shared files, runs your commands, and only then flips the
`current` symlink. If a
deployment turns out to be wrong, `shippy rollback` points that symlink back at any previous release.

## Features

- **Zero-downtime deployments** with atomic releases
- **Release management** — keeps last N releases with easy rollback
- **Deployment locking** — prevents concurrent deployments to the same host
- **Shared files/directories** — persistent data between releases
- **Template variables** from composer.json
- **Pure Go implementation** — single binary, no local dependencies
- **rsync transfers** — one delta stream per deploy, so unchanged files never travel twice
- **Deny-by-default file selection** — explicit allowlist, nothing ships unless you include it
- **SSH-based** deployment with key authentication
- **Colored output** — clear, beautiful deployment progress
- **TYPO3 optimized** — sensible defaults for TYPO3 projects

## How it compares

Deployer and Capistrano are excellent, and Shippy borrows their on-server layout wholesale — the same
`current` / `releases/` / `shared/` convention, so the two are interchangeable from the server's point
of view. The differences are deliberate:

- **No runtime on the deploying machine.** Deployer is a PHP application that lives in your
  `composer.json`; Capistrano is a Ruby gem. Shippy is one static binary with the rsync client built
  in, which makes it equally simple to run from a laptop, a GitHub Actions runner, or a GitLab CI job.
- **A small configuration surface.** One `.shippy.yaml` with hosts and commands, rather than a
  scripting DSL. Everything Shippy does is in the config file.
- **Sensible TYPO3 defaults.** `shippy init` writes a configuration that already knows about
  `var/log/`, `public/fileadmin/` and the TYPO3 console commands — though nothing about Shippy is
  TYPO3-specific.

## Requirements

- SSH access to the target server with key authentication configured
- `rsync` installed on the target server (Shippy drives it over the SSH connection; see
  [How files are transferred](./deployment-process#how-files-are-transferred))
- Go 1.26 or higher, only for building from source

Prebuilt binaries are available via Homebrew, `go install`, and a Docker image, so Go is only needed
if you build from source. See [Installation](./installation).

## Next steps

- [Installation](./installation) — get the binary
- [Quick Start](./quick-start) — first deployment in four commands
- [Configuration](./configuration) — everything `.shippy.yaml` can express
- [File Selection](./file-selection) — the allowlist that decides what ships
- [Commands](../reference/commands) — full command and flag reference

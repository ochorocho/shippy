# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Shippy is a minimal, opinionated deployment tool for PHP projects written in pure Go. It provides zero-downtime deployments with atomic releases via SSH, inspired by Deployer and Capistrano. Single binary, no external dependencies.

## Build & Test Commands

```bash
make build            # Build binary → dist/shippy (injects version via ldflags)
make build-release    # Cross-compile for darwin/linux (amd64/arm64)
make test             # Run BATS integration tests: bats tests/*.bats
make clean            # Remove dist/
make version          # Show version that would be built

# Run a single test file
bats tests/test_deploy.bats

# Go unit tests
go test ./...
go test ./internal/ssh/...   # Single package
```

## Architecture

Layered architecture with clear dependency direction:

```
cmd/           CLI layer (Cobra commands, user interaction)
  ↓
internal/
  deploy/      Orchestration: 8-step deployment workflow (Deployer, ReleaseManager)
  ↓
  ssh/         SSH client, remote command execution, host key verification
  rsync/       .gitignore-aware file scanning + SFTP transfer
  lock/        Distributed deployment locking (JSON lock file on remote)
  config/      YAML config parsing with per-host overrides + template variables
  composer/    composer.json parsing for template variables
  ui/          Colored output + Bubble Tea interactive host selector
  errors/      Context-rich error wrapping with recovery hints
```

**Key flow:** Config YAML → `Deployer.Deploy()` → scan files → SSH connect → create release dir → sync files → symlink shared → run commands → activate (atomic symlink) → cleanup old releases.

## Configuration

- Config file: `.shippy.yaml` (see `.shippy.yaml.example` for all options)
- Supports per-host overrides, template variables from `composer.json` using `{{key.path}}` syntax
- Default: 5 TYPO3 post-deploy commands auto-added if none specified

## Testing

- **Integration tests:** BATS (Bash Automated Testing System) in `tests/`
- **Go unit tests:** `internal/ssh/` has Go test files
- **Test infrastructure:** Docker Compose in `tests/` provides SSH + Apache + MariaDB
- **Fixtures:** `tests/config-test/` (config files), `tests/ssh_keys/` (test keys), `tests/www/` (test project)

## Key Patterns

- **Receiver naming:** Single letter (`c *Client`, `d *Deployer`, `l *Locker`)
- **Error handling:** Errors wrapped with `fmt.Errorf("...: %w", err)` + custom helpers in `internal/errors/` that add troubleshooting hints
- **Version injection:** Via ldflags at build time (see Makefile)
- **Release model:** Capistrano-style timestamped directories with `current` symlink
- **No external binaries:** Pure Go SSH/SFTP, no rsync binary needed

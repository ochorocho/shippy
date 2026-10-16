---
title: Contributing
description: Project structure, local test setup and how to contribute to Shippy.
---

# Contributing

Contributions are welcome! Please feel free to submit a Pull Request on
[github.com/ochorocho/shippy](https://github.com/ochorocho/shippy).

## Project Structure

```
shippy/
├── cmd/                  # One file per CLI command (cobra)
│   ├── root.go           # Root command, --config, --version
│   ├── init.go           # shippy init
│   ├── deploy.go         # shippy deploy
│   ├── rollback.go       # shippy rollback
│   ├── backup.go         # shippy backup
│   ├── gitlab_upload.go  # shippy gitlab:upload
│   ├── config.go         # shippy config validate / show
│   ├── unlock.go         # shippy unlock
│   └── env.go            # shippy env
├── internal/
│   ├── config/           # YAML parser, template + env substitution, command scoping
│   ├── composer/         # composer.json parser
│   ├── rsync/
│   │   ├── sync.go       # Allowlist file scanner (deny-by-default)
│   │   └── transfer.go   # rsync push to the remote cache, promote to the release
│   ├── ssh/              # SSH client, agent, host keys, command executor
│   ├── deploy/           # Deployment orchestrator and release management
│   ├── lock/             # Remote deployment lock
│   ├── backup/           # Database dump + shared files ZIP
│   ├── upload/           # GitLab package registry upload
│   └── ui/               # Coloured output, progress bar, release selector
├── third_party/
│   └── rsync/            # Vendored gokrazy/rsync client with a --files-from patch
├── tests/                # bats integration tests and the Docker test target
├── Formula/
│   └── shippy.rb         # Homebrew formula (prebuilt binary, per arch)
├── scripts/
│   └── update-formula.sh # Bumps formula version + per-arch sha256 for a tag
├── Dockerfile            # Release image, ENTRYPOINT is shippy
├── main.go
├── go.mod
└── README.md
```

The rsync client lives in-process: `go.mod` replaces `github.com/gokrazy/rsync` with the vendored copy
under `third_party/rsync`, so the release binaries stay static and need no rsync on the deploying
machine. The server side is the host's regular `rsync`.

## Testing

Build with Go 1.26 (see `go.mod`):

```bash
go build -o shippy
```

Start the test instance (an Apache/PHP container with `sshd` and `rsync`):

```bash
cd tests/
docker compose up
```

```bash
cd tests/typo3/
composer install
```

Deploy against it:

```bash
../../shippy deploy production
```

Test the SSH connection:

```bash
ssh -i tests/ssh_keys/shippy_key root@127.0.0.1 -p 2424
```

## Releasing

::: info Maintainers
After tagging a release, run `make brew-formula` to bump `Formula/shippy.rb` to the latest tag (the
Release workflow does this automatically on tagged builds).
:::

## License

Shippy is released under the [MIT License](https://github.com/ochorocho/shippy/blob/main/LICENSE).

## This website

The site you are reading is built with [VitePress](https://vitepress.dev/) and lives in a separate
repository. Corrections to the documentation are just as welcome as code — every page has an
*Edit this page on GitHub* link at the bottom.

---
title: Installation
description: Install Shippy with Homebrew, go install, a prebuilt binary, or the Docker image.
---

# Installation

Shippy is a single static binary. Pick whichever channel fits your machine or pipeline — they all
produce the same `shippy` executable.

## Using Homebrew

```bash
brew tap ochorocho/shippy https://github.com/ochorocho/shippy
brew trust ochorocho/shippy
brew install shippy
```

To upgrade later:

```bash
brew upgrade shippy
```

The formula installs a prebuilt binary for your architecture, so there is no Go toolchain involved.

## From Source

```bash
git clone https://github.com/ochorocho/shippy.git
cd shippy
go build -o shippy
sudo mv shippy /usr/local/bin/
```

## Using Go Install

```bash
go install github.com/ochorocho/shippy@latest
```

## Prebuilt binaries

Every [release](https://github.com/ochorocho/shippy/releases) ships binaries with a matching
`.checksum` file for macOS and Linux, on both `amd64` and `arm64`:

| File | Platform |
| --- | --- |
| `shippy-darwin-arm64` | macOS, Apple Silicon |
| `shippy-darwin-amd64` | macOS, Intel |
| `shippy-linux-arm64` | Linux, arm64 |
| `shippy-linux-amd64` | Linux, x86_64 |

```bash
curl -sSL -o shippy \
  https://github.com/ochorocho/shippy/releases/latest/download/shippy-linux-amd64
chmod +x shippy
sudo mv shippy /usr/local/bin/
```

## Docker

A prebuilt image ships `shippy` on `PATH` and uses it as the `ENTRYPOINT`, so arguments go straight
to the binary:

```bash
docker run --rm ghcr.io/ochorocho/shippy:latest --help
docker run --rm -v "$PWD:/app" -w /app ghcr.io/ochorocho/shippy:latest config validate
```

The image is published to both GitHub Container Registry (`ghcr.io/ochorocho/shippy`) and Docker Hub
(`ochorocho/shippy`), for `linux/amd64` and `linux/arm64`, tagged `latest` and per version
(`0.2.4`, `0.2`, `0`).

::: tip GitLab CI
Because the entrypoint is `shippy`, a CI runner that wants to execute ordinary shell commands in the
image has to clear it with `entrypoint: [""]`. See [CI/CD](./ci-cd) for the full job.
:::

## What the server needs

Nothing is installed on the target host. Shippy only needs:

- an SSH account with key authentication
- `rsync` on the server — Shippy's built-in rsync client talks to it over the SSH connection to
  transfer files. It is present on most Linux distributions; on Debian/Ubuntu it is `apt install rsync`.

## Verify the installation

```bash
shippy --help
```

Then continue with the [Quick Start](./quick-start).

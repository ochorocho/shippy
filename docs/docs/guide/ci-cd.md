---
title: CI/CD
description: Run Shippy from GitHub Actions or GitLab CI, including scheduled backups.
---

# CI/CD

Deploy from a pipeline. Both examples assume a `.shippy.yaml` in your repository and an SSH key for
the target server provided as a secret/variable (see [SSH Connections](./ssh)).

## GitHub Actions

Install and run shippy with the
[setup-shippy action](https://github.com/marketplace/actions/setup-shippy):

```yaml
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

## GitLab CI

Use the prebuilt Docker image, which ships shippy on `PATH`:

```yaml
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

The image's `ENTRYPOINT` is the `shippy` binary itself. GitLab runs the job's `script:` through a
shell inside the container, so the entrypoint has to be cleared with `entrypoint: [""]`; without it
the runner's shell is passed to `shippy` as an argument and the job fails immediately. The same image
is also published to Docker Hub as `ochorocho/shippy:latest`.

## Keeping secrets out of the repository

Reference CI variables from `.shippy.yaml` with the `${VAR}` syntax described in
[Environment Variables](./configuration#environment-variables):

```yaml
hosts:
  production:
    hostname: ${DEPLOY_HOST}
    remote_user: ${DEPLOY_USER}
    ssh_key: ${DEPLOY_SSH_KEY_PATH|~/.ssh/id_ed25519}
    deploy_path: ${DEPLOY_PATH|/var/www/html}
```

Write the private key to that path in a `before_script` (or an action step) from your CI secret
store, and make sure the pipeline knows the host key — `StrictHostKeyChecking: "accept-new"` is the
pragmatic setting for ephemeral runners.

## Scheduled backups

`shippy backup` plus `shippy gitlab:upload` makes a nightly off-server backup a single pipeline. Inside
GitLab CI the upload authenticates automatically with `CI_JOB_TOKEN`:

```yaml
# .gitlab-ci.yml
nightly_backup:
  stage: backup
  image:
    name: ghcr.io/ochorocho/shippy:latest
    # ENTRYPOINT is "shippy", so it is cleared to run regular shell commands.
    entrypoint: [""]
  rules:
    - if: $CI_PIPELINE_SOURCE == "schedule"
  script:
    - shippy backup production
    - shippy gitlab:upload backups/backup-production-*.zip
```

Uploaded archives appear in **Project → Deploy → Package Registry**, grouped by
`<package-name>/<package-version>`. See the
[`backup`](../reference/commands#backup) and
[`gitlab:upload`](../reference/commands#upload-to-gitlab) references for the available options.

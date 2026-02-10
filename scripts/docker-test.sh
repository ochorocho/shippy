#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
IMAGE_NAME="shippy-test"

echo "==> Building linux binaries..."
make -C "${PROJECT_DIR}" build-release

echo "==> Building Docker image..."
docker buildx build --platform linux/amd64 -t "${IMAGE_NAME}" "${PROJECT_DIR}"

echo "==> Running BATS tests inside Docker container..."
docker run --rm \
    --platform linux/amd64 \
    -v "${PROJECT_DIR}/tests:/workspace/tests:ro" \
    --entrypoint bash \
    "${IMAGE_NAME}" -c '
set -euo pipefail

echo "--- Installing BATS ---"
apt update
apt install -y bats bats-assert bats-support bats-file

echo "--- Setting up binary path ---"
mkdir -p /workspace/dist
ln -s /usr/local/bin/shippy /workspace/dist/shippy

echo "--- Running tests (quick only, integration tests require Docker-in-Docker) ---"
cd /workspace
bats --filter-tags "!integration" tests/*.bats
'

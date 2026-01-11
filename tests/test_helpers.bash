#!/usr/bin/env bash

# Shared helper functions for Bats tests
#
# ⚠️  IMPORTANT: Tests MUST run serially
# Tests share resources (Docker containers, temporary directories, SSH connections)
# and will conflict if run in parallel.
#
# DO NOT use: bats --jobs N
# USE: bats tests/

# Common setup for all tests
common_setup() {
  # Set binary path
  export BIN="${BATS_TEST_DIRNAME}/../dist/tinnie"

  # Load Bats libraries
  TEST_BREW_PREFIX="$(brew --prefix 2>/dev/null || true)"
  export BATS_LIB_PATH="${BATS_LIB_PATH}:${TEST_BREW_PREFIX}/lib:/usr/lib/bats"
  bats_load_library bats-assert
  bats_load_library bats-file
  bats_load_library bats-support
}

# Check if Docker is available
docker_available() {
  command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1
}

# Check if SSH keys exist for testing
ssh_keys_available() {
  [[ -f "${BATS_TEST_DIRNAME}/ssh_keys/tinnie_key" ]]
}

# Skip test if Docker is not available
require_docker() {
  if ! docker_available; then
    skip "Docker is not available or not running"
  fi
}

# Skip test if SSH keys are not available
require_ssh_keys() {
  if ! ssh_keys_available; then
    skip "SSH test keys not found"
  fi
}

# Create a temporary test directory
create_test_dir() {
  local test_dir="${BATS_TMPDIR}/tinnie-test-${BATS_TEST_NUMBER}"
  mkdir -p "${test_dir}"
  echo "${test_dir}"
}

# Clean up temporary test directory
cleanup_test_dir() {
  local test_dir="$1"
  if [[ -d "${test_dir}" ]]; then
    rm -rf "${test_dir}"
  fi
}

# Create a minimal composer.json for testing
create_test_composer_json() {
  local target_dir="${1:-.}"
  cat > "${target_dir}/composer.json" <<'EOF'
{
  "name": "test/tinnie-project",
  "description": "Test TYPO3 project for Tinnie",
  "type": "project",
  "require": {
    "typo3/cms-core": "^13.4"
  },
  "config": {
    "bin-dir": "vendor/bin"
  }
}
EOF
}

# Create a minimal .tinnie.yaml for testing
create_minimal_config() {
  local target_dir="${1:-.}"
  cat > "${target_dir}/.tinnie.yaml" <<'EOF'
hosts:
  production:
    hostname: 127.0.0.1
    remote_user: root
    deploy_path: /var/www/html
    port: 2424
    ssh_key: ../ssh_keys/tinnie_key
EOF
}

# Assert that output contains a specific pattern
assert_output_contains() {
  local pattern="$1"
  if [[ ! "${output}" =~ ${pattern} ]]; then
    echo "Expected output to contain: ${pattern}"
    echo "Actual output:"
    echo "${output}"
    return 1
  fi
}

# Assert that output does not contain a specific pattern
assert_output_not_contains() {
  local pattern="$1"
  if [[ "${output}" =~ ${pattern} ]]; then
    echo "Expected output to NOT contain: ${pattern}"
    echo "Actual output:"
    echo "${output}"
    return 1
  fi
}

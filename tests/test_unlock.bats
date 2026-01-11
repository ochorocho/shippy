#!/usr/bin/env bats

# Tests for tinnie unlock command

# bats file_tags=quick

bats_require_minimum_version 1.5.0

set -eu -o pipefail

setup() {
  set -eu -o pipefail
  load test_helpers
  common_setup

  export TEST_DIR="${BATS_TMPDIR}/tinnie-unlock-test"
  mkdir -p "${TEST_DIR}"
  cd "${TEST_DIR}"
  create_test_composer_json
}

teardown() {
  cd "${BATS_TEST_DIRNAME}"
  rm -rf "${TEST_DIR}"
}

@test "Should show help for unlock command" {
  run -0 ${BIN} unlock --help
  assert_success
  assert_output --partial "Removes the deployment lock"
  assert_output --partial "WARNING:"
  assert_output --partial "Example:"
}

@test "Should fail when config file does not exist" {
  run -1 ${BIN} unlock production --config /nonexistent/config.yaml
  assert_failure
  assert_output --partial "failed to read config file"
}

@test "Should fail when host does not exist in config" {
  run -1 ${BIN} unlock nonexistent --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_failure
  assert_output --partial "not found"
}

@test "Should fail when no host is provided and config has no hosts" {
  # Create empty config with no hosts
  cat > .tinnie.yaml <<'EOF'
hosts: {}
EOF

  run -1 ${BIN} unlock --config .tinnie.yaml
  assert_failure
}

@test "Should accept host argument" {
  # This will fail because we can't connect, but it should accept the argument
  run -1 ${BIN} unlock production --config ${BATS_TEST_DIRNAME}/config-test/multi-host.yaml
  assert_failure
  # Should attempt to connect (showing it accepted the argument)
  assert_output --partial "Connecting to"
}

@test "Should load configuration when unlocking" {
  run ${BIN} unlock production --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_output --partial "Loading configuration from"
}

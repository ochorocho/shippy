#!/usr/bin/env bats

# Tests for shippy deploy command (without full integration)

# bats file_tags=quick

bats_require_minimum_version 1.5.0

set -eu -o pipefail

setup() {
  set -eu -o pipefail
  load test_helpers
  common_setup

  export TEST_DIR="${BATS_TMPDIR}/shippy-deploy-test"
  mkdir -p "${TEST_DIR}"
  cd "${TEST_DIR}"
  create_test_composer_json
}

teardown() {
  cd "${BATS_TEST_DIRNAME}"
  rm -rf "${TEST_DIR}"
}

@test "Should show help for deploy command" {
  run -0 ${BIN} deploy --help
  assert_success
  assert_output --partial "Deploy to a target host"
  assert_output --partial "deployment process:"
  assert_output --partial "Example:"
}

@test "Should show verbose flag in help" {
  run -0 ${BIN} deploy --help
  assert_success
  assert_output --partial "--verbose"
  assert_output --partial "-v"
}

@test "Should fail when config file does not exist" {
  run -1 ${BIN} deploy production --config /nonexistent/config.yaml
  assert_failure
  assert_output --partial "failed to read config file"
}

@test "Should fail when host does not exist in config" {
  run -1 ${BIN} deploy nonexistent --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_failure
  assert_output --partial "not found"
}

@test "Should fail when composer.json does not exist" {
  rm -f composer.json
  run -1 ${BIN} deploy production --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_failure
  assert_output --partial "composer.json"
}

@test "Should load configuration when deploying" {
  run ${BIN} deploy production --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_output --partial "Loading configuration from"
}

@test "Should accept host argument" {
  # This will fail because we can't connect, but it should accept the argument
  run ${BIN} deploy production --config ${BATS_TEST_DIRNAME}/config-test/multi-host.yaml
  # Should attempt to process, not fail on argument parsing
  assert_output --partial "Loading configuration"
}

@test "Should show deployment steps in help" {
  run -0 ${BIN} deploy --help
  assert_success
  assert_output --partial "1. Scans files"
  assert_output --partial "2. Creates a new release"
  assert_output --partial "3. Syncs files"
  assert_output --partial "4. Creates symlinks"
  assert_output --partial "5. Updates the"
  assert_output --partial "6. Executes post-deployment commands"
  assert_output --partial "7. Cleans up old releases"
}

@test "Should mention interactive selection in help" {
  run -0 ${BIN} deploy --help
  assert_success
  assert_output --partial "interactive"
}

@test "Should fail with invalid config syntax" {
  run -1 ${BIN} deploy production --config ${BATS_TEST_DIRNAME}/config-test/invalid-syntax.yaml
  assert_failure
}

@test "Should fail with missing required fields in config" {
  run -1 ${BIN} deploy production --config ${BATS_TEST_DIRNAME}/config-test/missing-fields.yaml
  assert_failure
  assert_output --partial "hostname is required"
}

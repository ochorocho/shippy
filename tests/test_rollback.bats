#!/usr/bin/env bats

# Tests for shippy rollback command

# bats file_tags=quick

bats_require_minimum_version 1.5.0

set -eu -o pipefail

setup() {
  set -eu -o pipefail
  load test_helpers
  common_setup

  export TEST_DIR="${BATS_TMPDIR}/shippy-rollback-test"
  mkdir -p "${TEST_DIR}"
  cd "${TEST_DIR}"
  create_test_composer_json
}

teardown() {
  cd "${BATS_TEST_DIRNAME}"
  rm -rf "${TEST_DIR}"
}

@test "Should show help for rollback command" {
  run -0 ${BIN} rollback --help
  assert_success
  assert_output --partial "Rollback to a previous release"
  assert_output --partial "Example:"
}

@test "Should show rollback usage in help" {
  run -0 ${BIN} rollback --help
  assert_success
  assert_output --partial "rollback [host]"
}

@test "Should fail when config file does not exist" {
  run -1 ${BIN} rollback production --config /nonexistent/config.yaml
  assert_failure
  assert_output --partial "failed to read config file"
}

@test "Should fail when host does not exist in config" {
  run -1 ${BIN} rollback nonexistent --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_failure
  assert_output --partial "not found"
}

@test "Should accept host argument" {
  # This will fail because we can't connect, but it should accept the argument
  run -1 ${BIN} rollback production --config ${BATS_TEST_DIRNAME}/config-test/multi-host.yaml
  assert_failure
  # Should attempt to connect (showing it accepted the argument)
  assert_output --partial "Connecting to"
}

@test "Should load configuration when rolling back" {
  run ${BIN} rollback production --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_output --partial "Loading configuration from"
}

@test "Should show --release flag in help" {
  run -0 ${BIN} rollback --help
  assert_success
  assert_output --partial "--release"
  assert_output --partial "-r"
}

@test "Should show --offset flag in help" {
  run -0 ${BIN} rollback --help
  assert_success
  assert_output --partial "--offset"
  assert_output --partial "-n"
}

@test "Should show examples with flags in help" {
  run -0 ${BIN} rollback --help
  assert_success
  assert_output --partial "-n -1"
  assert_output --partial "-r 20260109120000"
}

@test "Should show --list flag in help" {
  run -0 ${BIN} rollback --help
  assert_success
  assert_output --partial "--list"
  assert_output --partial "-l"
}

@test "Should fail when both --release and --offset are provided" {
  run -1 ${BIN} rollback production --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml -r 20260109120000 -n -1
  assert_failure
  assert_output --partial "cannot use --release and --offset together"
}

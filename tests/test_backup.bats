#!/usr/bin/env bats

# Tests for shippy backup command

# bats file_tags=quick

bats_require_minimum_version 1.5.0

set -eu -o pipefail

setup() {
  set -eu -o pipefail
  load test_helpers
  common_setup

  export TEST_DIR="${BATS_TMPDIR}/shippy-backup-test"
  mkdir -p "${TEST_DIR}"
  cd "${TEST_DIR}"
  create_test_composer_json
}

teardown() {
  cd "${BATS_TEST_DIRNAME}"
  rm -rf "${TEST_DIR}"
}

# Help text tests

@test "Should show help for backup command" {
  run -0 ${BIN} backup --help
  assert_success
  assert_output --partial "Create a ZIP backup"
  assert_output --partial "database export"
  assert_output --partial "shared files"
  assert_output --partial "Example:"
}

@test "Should show backup usage in help" {
  run -0 ${BIN} backup --help
  assert_success
  assert_output --partial "backup [host]"
}

@test "Should show verbose flag in help" {
  run -0 ${BIN} backup --help
  assert_success
  assert_output --partial "--verbose"
  assert_output --partial "-v"
}

@test "Should show output flag in help" {
  run -0 ${BIN} backup --help
  assert_success
  assert_output --partial "--output"
  assert_output --partial "-o"
}

@test "Should show skip-database flag in help" {
  run -0 ${BIN} backup --help
  assert_success
  assert_output --partial "--skip-database"
}

@test "Should show skip-shared flag in help" {
  run -0 ${BIN} backup --help
  assert_success
  assert_output --partial "--skip-shared"
}

# Error cases

@test "Should fail when config file does not exist" {
  run -1 ${BIN} backup production --config /nonexistent/config.yaml
  assert_failure
  assert_output --partial "failed to read config file"
}

@test "Should fail when host does not exist in config" {
  run -1 ${BIN} backup nonexistent --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_failure
  assert_output --partial "not found"
}

@test "Should fail when composer.json does not exist" {
  rm -f composer.json
  run -1 ${BIN} backup production --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_failure
  assert_output --partial "composer.json"
}

# Config loading

@test "Should load configuration when backing up" {
  run ${BIN} backup production --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_output --partial "Loading configuration from"
}

@test "Should accept host argument" {
  run ${BIN} backup production --config ${BATS_TEST_DIRNAME}/config-test/multi-host.yaml
  assert_output --partial "Loading configuration"
}

# Config validation with backup section

@test "Should validate config with backup section" {
  run -0 ${BIN} config validate --config ${BATS_TEST_DIRNAME}/config-test/backup.yaml
  assert_success
}

@test "Should validate config with manual database credentials" {
  run -0 ${BIN} config validate --config ${BATS_TEST_DIRNAME}/config-test/backup-manual-creds.yaml
  assert_success
}

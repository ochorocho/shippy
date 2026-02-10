#!/usr/bin/env bats

# Tests for root shippy command

# bats file_tags=quick

bats_require_minimum_version 1.5.0

set -eu -o pipefail

setup() {
  set -eu -o pipefail
  load test_helpers
  common_setup
}

@test "Should show help when run without arguments" {
  run -0 ${BIN}
  assert_output --partial "Shippy is a minimal, opinionated deployment tool for Composer based PHP projects"
  assert_output --partial "Usage:"
  assert_output --partial "Available Commands:"
}

@test "Should show help with --help flag" {
  run -0 ${BIN} --help
  assert_output --partial "Shippy is a minimal, opinionated deployment tool for Composer based PHP projects"
  assert_output --partial "Usage:"
  assert_output --partial "Available Commands:"
}

@test "Should show help with -h flag" {
  run -0 ${BIN} -h
  assert_output --partial "Shippy is a minimal, opinionated deployment tool for Composer based PHP projects"
  assert_output --partial "Usage:"
}

@test "Should list all available commands in help" {
  run -0 ${BIN} --help
  assert_output --partial "config"
  assert_output --partial "deploy"
  assert_output --partial "env"
  assert_output --partial "init"
  assert_output --partial "rollback"
  assert_output --partial "unlock"
}

@test "Should show error for invalid command" {
  run -1 ${BIN} invalid-command
  assert_failure
}

@test "Should accept --config flag" {
  # Create a test directory with config
  TEST_DIR=$(create_test_dir)
  cd "${TEST_DIR}"
  create_test_composer_json
  create_minimal_config

  run -0 ${BIN} --config .shippy.yaml config validate
  assert_success

  cleanup_test_dir "${TEST_DIR}"
}

@test "Should show error when config file does not exist" {
  run -1 ${BIN} --config /nonexistent/config.yaml config validate
  assert_failure
  assert_output --partial "failed to read config file"
}

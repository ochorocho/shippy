#!/usr/bin/env bats

# Tests for tinnie env command

# bats file_tags=quick

bats_require_minimum_version 1.5.0

set -eu -o pipefail

setup() {
  set -eu -o pipefail
  load test_helpers
  common_setup
}

@test "Should list all environment variables" {
  run -0 ${BIN} env
  assert_success
  assert_output --partial "Environment Variables"
}

@test "Should show environment variable count" {
  run -0 ${BIN} env
  assert_success
  assert_output --partial "Total:"
  assert_output --partial "variables"
}

@test "Should include PATH variable" {
  run -0 ${BIN} env
  assert_success
  assert_output --partial "PATH="
}

@test "Should include HOME variable" {
  run -0 ${BIN} env
  assert_success
  assert_output --partial "HOME="
}

@test "Should show help with --help flag" {
  run -0 ${BIN} env --help
  assert_success
  assert_output --partial "Print all environment variables"
  assert_output --partial "Usage:"
}

@test "Should include custom environment variable" {
  export TINNIE_TEST_VAR="test_value"
  run -0 ${BIN} env
  assert_success
  assert_output --partial "TINNIE_TEST_VAR=test_value"
}

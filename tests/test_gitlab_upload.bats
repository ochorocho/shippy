#!/usr/bin/env bats

# Tests for shippy gitlab:upload command

# bats file_tags=quick

bats_require_minimum_version 1.5.0

set -eu -o pipefail

setup() {
  set -eu -o pipefail
  load test_helpers
  common_setup
}

# Help text tests

@test "Should show help for gitlab:upload command" {
  run -0 ${BIN} gitlab:upload --help
  assert_success
  assert_output --partial "Upload a backup ZIP (or any file) to the GitLab Generic Packages"
  assert_output --partial "Example:"
}

@test "Should show gitlab:upload usage in help" {
  run -0 ${BIN} gitlab:upload --help
  assert_success
  assert_output --partial "gitlab:upload <file>"
}

@test "Should show token flag in help" {
  run -0 ${BIN} gitlab:upload --help
  assert_success
  assert_output --partial "--token"
  assert_output --partial "-t"
}

@test "Should show package-name flag in help" {
  run -0 ${BIN} gitlab:upload --help
  assert_success
  assert_output --partial "--package-name"
}

@test "Should show package-version flag in help" {
  run -0 ${BIN} gitlab:upload --help
  assert_success
  assert_output --partial "--package-version"
}

# Error cases

@test "Should fail when no file argument is provided" {
  run -1 ${BIN} gitlab:upload
  assert_failure
}

@test "Should fail when file does not exist" {
  run -1 ${BIN} gitlab:upload /nonexistent/file.zip
  assert_failure
  assert_output --partial "File not found"
}

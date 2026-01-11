#!/usr/bin/env bats

# Tests for shippy config command

# bats file_tags=quick

bats_require_minimum_version 1.5.0

set -eu -o pipefail

setup() {
  set -eu -o pipefail
  load test_helpers
  common_setup

  # Set up test directory with composer.json for validation tests
  export TEST_DIR="${BATS_TMPDIR}/shippy-config-test"
  mkdir -p "${TEST_DIR}"
  cd "${TEST_DIR}"
  create_test_composer_json
}

teardown() {
  cd "${BATS_TEST_DIRNAME}"
  rm -rf "${TEST_DIR}"
}

# Config command root tests

@test "Should show help for config command" {
  run -0 ${BIN} config --help
  assert_success
  assert_output --partial "Configuration management commands"
  assert_output --partial "validate"
  assert_output --partial "show"
}

# Config validate tests

@test "Should validate minimal config successfully" {
  run -0 ${BIN} config validate --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_success
  assert_output --partial "Validating configuration"
  assert_output --partial "Config file loaded successfully"
  assert_output --partial "Config structure is valid"
}

@test "Should validate full config successfully" {
  run -0 ${BIN} config validate --config ${BATS_TEST_DIRNAME}/config-test/full.yaml
  assert_success
  assert_output --partial "Config structure is valid"
}

@test "Should validate multi-host config successfully" {
  run -0 ${BIN} config validate --config ${BATS_TEST_DIRNAME}/config-test/multi-host.yaml
  assert_success
  assert_output --partial "Config structure is valid"
}

@test "Should fail validation for invalid YAML syntax" {
  run -1 ${BIN} config validate --config ${BATS_TEST_DIRNAME}/config-test/invalid-syntax.yaml
  assert_failure
  assert_output --partial "Failed to load config"
}

@test "Should fail validation for missing required fields" {
  run -1 ${BIN} config validate --config ${BATS_TEST_DIRNAME}/config-test/missing-fields.yaml
  assert_failure
  assert_output --partial "Validation failed"
  assert_output --partial "hostname is required"
}

@test "Should fail validation for nonexistent config file" {
  run -1 ${BIN} config validate --config /nonexistent/config.yaml
  assert_failure
  assert_output --partial "failed to read config file"
}

@test "Should show help for validate command" {
  run -0 ${BIN} config validate --help
  assert_success
  assert_output --partial "Validate the configuration file"
}

# Config show tests

@test "Should show complete configuration" {
  run -0 ${BIN} config show --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_success
  assert_output --partial "Complete Configuration"
  assert_output --partial "hosts"
  assert_output --partial "production"
}

@test "Should show configuration for specific host" {
  run -0 ${BIN} config show production --config ${BATS_TEST_DIRNAME}/config-test/multi-host.yaml
  assert_success
  assert_output --partial "production"
}

@test "Should show configuration with defaults applied" {
  run -0 ${BIN} config show --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_success
  # Should show default values
  assert_output --partial "keep_releases"
  assert_output --partial "lock_enabled"
}

@test "Should fail showing config for nonexistent host" {
  run -1 ${BIN} config show nonexistent --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_failure
  assert_output --partial "not found"
}

@test "Should show help for show command" {
  run -0 ${BIN} config show --help
  assert_success
  assert_output --partial "Display the complete configuration"
  assert_output --partial "Examples:"
}

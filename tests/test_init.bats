#!/usr/bin/env bats

# Tests for tinnie init command

# bats file_tags=quick

bats_require_minimum_version 1.5.0

set -eu -o pipefail

setup() {
  set -eu -o pipefail
  load test_helpers
  common_setup

  # Create test directory for each test
  export TEST_DIR="${BATS_TMPDIR}/tinnie-init-test-${BATS_TEST_NUMBER}"
  mkdir -p "${TEST_DIR}"
  cd "${TEST_DIR}"
}

teardown() {
  cd "${BATS_TEST_DIRNAME}"
  rm -rf "${TEST_DIR}"
}

@test "Should create .tinnie.yaml in empty directory" {
  run -0 ${BIN} init
  assert_success
  assert_output --partial "Tinnie - Initialize Configuration"
  assert_file_exist ".tinnie.yaml"
}

@test "Should warn when .tinnie.yaml already exists" {
  # Create initial config
  echo "hosts:" > .tinnie.yaml

  run -0 ${BIN} init
  assert_success
  assert_output --partial "Configuration file already exists"
  assert_output --partial "Use --force to overwrite"
}

@test "Should overwrite existing config with --force flag" {
  echo "hosts:" > .tinnie.yaml

  run -0 ${BIN} init --force
  assert_success
  assert_file_exist ".tinnie.yaml"
}

@test "Should overwrite existing config with -f flag" {
  echo "hosts:" > .tinnie.yaml

  run -0 ${BIN} init -f
  assert_success
  assert_file_exist ".tinnie.yaml"
}

@test "Should create valid YAML configuration" {
  create_test_composer_json
  run -0 ${BIN} init
  assert_success
  assert_file_exist ".tinnie.yaml"

  # Verify the generated config is valid
  run -0 ${BIN} config validate
  assert_success
}

@test "Generated config should contain production host" {
  run -0 ${BIN} init
  assert_success
  assert_file_exist ".tinnie.yaml"

  run cat .tinnie.yaml
  assert_output --partial "production:"
  assert_output --partial "hostname:"
  assert_output --partial "remote_user:"
  assert_output --partial "deploy_path:"
}

@test "Generated config should mark optional fields" {
  run -0 ${BIN} init
  assert_success

  run cat .tinnie.yaml
  assert_output --partial "# Optional:"
}

@test "Generated config should comment optional values" {
  run -0 ${BIN} init
  assert_success

  run cat .tinnie.yaml
  # Check for commented optional fields
  assert_output --partial "# composer:"
  assert_output --partial "# rsync_src:"
}

@test "Should show help for init command" {
  run -0 ${BIN} init --help
  assert_success
  assert_output --partial "Initialize a new .tinnie.yaml configuration file"
  assert_output --partial "Example:"
}

#!/usr/bin/env bats

# Tests for shippy init command

# bats file_tags=quick

bats_require_minimum_version 1.5.0

set -eu -o pipefail

setup() {
  set -eu -o pipefail
  load test_helpers
  common_setup

  # Create test directory for each test
  export TEST_DIR="${BATS_TMPDIR}/shippy-init-test-${BATS_TEST_NUMBER}"
  mkdir -p "${TEST_DIR}"
  cd "${TEST_DIR}"
}

teardown() {
  cd "${BATS_TEST_DIRNAME}"
  rm -rf "${TEST_DIR}"
}

@test "Should create .shippy.yaml in empty directory" {
  create_test_composer_json
  run -0 ${BIN} init
  assert_success
  assert_output --partial "Shippy - Initialize Configuration"
  assert_file_exist ".shippy.yaml"
}

@test "Should create .shippy.yaml with composer.json present" {
  create_test_composer_json
  run -0 ${BIN} init
  assert_success
  assert_file_exist ".shippy.yaml"
  assert_output --partial "composer.json loaded"
}

@test "Should warn when .shippy.yaml already exists" {
  # Create initial config
  echo "hosts:" > .shippy.yaml

  run -0 ${BIN} init
  assert_success
  assert_output --partial "Configuration file already exists"
  assert_output --partial "Use --force to overwrite"
}

@test "Should overwrite existing config with --force flag" {
  create_test_composer_json
  # Create initial config
  echo "hosts:" > .shippy.yaml

  run -0 ${BIN} init --force
  assert_success
  assert_file_exist ".shippy.yaml"
}

@test "Should overwrite existing config with -f flag" {
  create_test_composer_json
  # Create initial config
  echo "hosts:" > .shippy.yaml

  run -0 ${BIN} init -f
  assert_success
  assert_file_exist ".shippy.yaml"
}

@test "Should create valid YAML configuration" {
  create_test_composer_json
  run -0 ${BIN} init
  assert_success
  assert_file_exist ".shippy.yaml"

  # Verify the generated config is valid
  run -0 ${BIN} config validate
  assert_success
}

@test "Generated config should contain production host" {
  create_test_composer_json
  run -0 ${BIN} init
  assert_success
  assert_file_exist ".shippy.yaml"

  run cat .shippy.yaml
  assert_output --partial "production:"
  assert_output --partial "hostname:"
  assert_output --partial "remote_user:"
  assert_output --partial "deploy_path:"
}

@test "Generated config should mark optional fields" {
  create_test_composer_json
  run -0 ${BIN} init
  assert_success

  run cat .shippy.yaml
  assert_output --partial "# Optional:"
}

@test "Generated config should comment optional values" {
  create_test_composer_json
  run -0 ${BIN} init
  assert_success

  run cat .shippy.yaml
  # Check for commented optional fields
  assert_output --partial "# composer:"
  assert_output --partial "# rsync_src:"
}

@test "Should show help for init command" {
  run -0 ${BIN} init --help
  assert_success
  assert_output --partial "Initialize a new .shippy.yaml configuration file"
  assert_output --partial "Example:"
}

@test "Generated config should use project name from composer.json" {
  # Create composer.json with specific project name
  cat > composer.json <<'EOF'
{
  "name": "myvendor/myproject",
  "type": "project"
}
EOF

  run -0 ${BIN} init
  assert_success
  assert_output --partial "Project: myvendor/myproject"

  run cat .shippy.yaml
  # Config uses {{name}} template variable instead of hardcoded project name
  assert_output --partial "{{name}}"
}

@test "Should fail without composer.json" {
  # Don't create composer.json - expect failure
  run -1 ${BIN} init
  assert_failure
  assert_output --partial "Failed to load composer.json"
  assert_output --partial "Make sure you run this command in a directory with a composer.json file"
}

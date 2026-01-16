#!/usr/bin/env bats

# Tests for SSH host key checking with different StrictHostKeyChecking modes
# Tests the three modes: "no", "yes", and "accept-new"

# bats file_tags=integration,slow,ssh
# bats test_tags=bats:serial

set -eu -o pipefail

setup() {
  set -eu -o pipefail

  export DEPLOYMENT_SOURCE="$BATS_TEST_DIRNAME/typo3"
  export BIN="$DEPLOYMENT_SOURCE/../../dist/tinnie"
  export TEST_KNOWN_HOSTS="$BATS_TEST_TMPDIR/known_hosts"

  TEST_BREW_PREFIX="$(brew --prefix 2>/dev/null || true)"
  export BATS_LIB_PATH="${BATS_LIB_PATH}:${TEST_BREW_PREFIX}/lib:/usr/lib/bats"
  bats_load_library bats-assert
  bats_load_library bats-file
  bats_load_library bats-support

  # Start Docker SSH server
  run docker compose -f "${BATS_TEST_DIRNAME}/docker-compose.yaml" up -d --build
  assert_success

  # Wait for SSH to be ready
  sleep 2
}

teardown() {
  set -eu -o pipefail
  run docker compose -f "${BATS_TEST_DIRNAME}/docker-compose.yaml" down
  rm -f "$TEST_KNOWN_HOSTS"
}

@test "StrictHostKeyChecking=no should disable host key verification" {
  set -eu -o pipefail

  # Remove any existing host key
  ssh-keygen -R "[127.0.0.1]:2424" -f "$TEST_KNOWN_HOSTS" 2>/dev/null || true

  # Test with StrictHostKeyChecking=no
  cd "$DEPLOYMENT_SOURCE"
  export HOST_KEY_CHECKING="no"

  run ${BIN} config show production
  assert_success
  # Strip ANSI color codes and check for the value
  run bash -c "${BIN} config show production 2>&1 | sed 's/\x1b\[[0-9;]*m//g'"
  assert_output --partial "StrictHostKeyChecking: no"
}

@test "StrictHostKeyChecking=no should show insecure warning" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  # Create a test config with StrictHostKeyChecking=no
  cat > "$BATS_TEST_TMPDIR/test_no.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/tinnie_key
    ssh_options:
      StrictHostKeyChecking: no
EOF

  # Run unlock command to test SSH connection (doesn't require deployment)
  run ${BIN} unlock test --config "$BATS_TEST_TMPDIR/test_no.yaml"

  # Should show warning about insecure mode
  assert_output --partial "Host key checking: disabled (insecure)"
  assert_output --partial "WARNING: SSH host key verification disabled"
}

@test "StrictHostKeyChecking=accept-new should auto-accept unknown hosts" {
  set -eu -o pipefail

  # Remove any existing host key
  ssh-keygen -R "[127.0.0.1]:2424" -f "$TEST_KNOWN_HOSTS" 2>/dev/null || true

  cd "$DEPLOYMENT_SOURCE"

  # Create a test config with StrictHostKeyChecking=accept-new
  cat > "$BATS_TEST_TMPDIR/test_accept_new.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/tinnie_key
    ssh_options:
      StrictHostKeyChecking: accept-new
      UserKnownHostsFile: $TEST_KNOWN_HOSTS
EOF

  # Run unlock command to test SSH connection
  run ${BIN} unlock test --config "$BATS_TEST_TMPDIR/test_accept_new.yaml"

  # Should show accept-new mode message
  assert_output --partial "Host key checking: accept-new"
  assert_output --partial "Permanently added"

  # Verify host key was saved to known_hosts
  assert_file_exist "$TEST_KNOWN_HOSTS"
  run cat "$TEST_KNOWN_HOSTS"
  assert_output --partial "[127.0.0.1]:2424"
}

@test "StrictHostKeyChecking=accept-new should verify existing hosts" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  # First connection: create known_hosts with correct key
  ssh-keygen -R "[127.0.0.1]:2424" -f "$TEST_KNOWN_HOSTS" 2>/dev/null || true

  cat > "$BATS_TEST_TMPDIR/test_accept_new2.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/tinnie_key
    ssh_options:
      StrictHostKeyChecking: accept-new
      UserKnownHostsFile: $TEST_KNOWN_HOSTS
EOF

  # First connection - should accept and save
  run ${BIN} unlock test --config "$BATS_TEST_TMPDIR/test_accept_new2.yaml"
  assert_output --partial "Permanently added"

  # Second connection - should verify existing key (no prompt)
  run ${BIN} unlock test --config "$BATS_TEST_TMPDIR/test_accept_new2.yaml"
  refute_output --partial "Permanently added"
  assert_output --partial "Host key checking: accept-new"
}

@test "Environment variable HOST_KEY_CHECKING should control mode" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  # Test with HOST_KEY_CHECKING=accept-new
  export HOST_KEY_CHECKING="accept-new"
  run bash -c "${BIN} config show production 2>&1 | sed 's/\x1b\[[0-9;]*m//g'"
  assert_success
  assert_output --partial "StrictHostKeyChecking: accept-new"

  # Test with HOST_KEY_CHECKING=no
  export HOST_KEY_CHECKING="no"
  run bash -c "${BIN} config show production 2>&1 | sed 's/\x1b\[[0-9;]*m//g'"
  assert_success
  assert_output --partial "StrictHostKeyChecking: no"

  # Test with HOST_KEY_CHECKING=yes
  export HOST_KEY_CHECKING="yes"
  run bash -c "${BIN} config show production 2>&1 | sed 's/\x1b\[[0-9;]*m//g'"
  assert_success
  assert_output --partial "StrictHostKeyChecking: yes"
}

@test "Default should be accept-new when HOST_KEY_CHECKING not set" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  # Unset environment variable to test fallback
  unset HOST_KEY_CHECKING || true

  run bash -c "${BIN} config show production 2>&1 | sed 's/\x1b\[[0-9;]*m//g'"
  assert_success
  assert_output --partial "StrictHostKeyChecking: accept-new"
}

@test "Known hosts file should default to ~/.ssh/known_hosts" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  cat > "$BATS_TEST_TMPDIR/test_default_known_hosts.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/tinnie_key
    ssh_options:
      StrictHostKeyChecking: accept-new
EOF

  # Run unlock to trigger SSH connection
  run ${BIN} unlock test --config "$BATS_TEST_TMPDIR/test_default_known_hosts.yaml"

  # Should show default known_hosts path
  assert_output --partial "Known hosts file: $HOME/.ssh/known_hosts"
}

@test "Custom known_hosts file should be respected" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  CUSTOM_KNOWN_HOSTS="$BATS_TEST_TMPDIR/custom_known_hosts"
  rm -f "$CUSTOM_KNOWN_HOSTS"

  cat > "$BATS_TEST_TMPDIR/test_custom_known_hosts.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/tinnie_key
    ssh_options:
      StrictHostKeyChecking: accept-new
      UserKnownHostsFile: $CUSTOM_KNOWN_HOSTS
EOF

  # Run unlock to trigger SSH connection
  run ${BIN} unlock test --config "$BATS_TEST_TMPDIR/test_custom_known_hosts.yaml"

  # Should show custom known_hosts path
  assert_output --partial "Known hosts file: $CUSTOM_KNOWN_HOSTS"

  # Custom file should exist and contain host key
  assert_file_exist "$CUSTOM_KNOWN_HOSTS"
  run cat "$CUSTOM_KNOWN_HOSTS"
  assert_output --partial "[127.0.0.1]:2424"
}

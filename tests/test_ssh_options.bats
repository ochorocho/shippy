#!/usr/bin/env bats

# Tests for SSH options: ConnectTimeout, ServerAliveInterval, Compression
# These tests verify that the new SSH options are properly parsed and applied

# bats file_tags=integration,slow,ssh
# bats test_tags=bats:serial

set -eu -o pipefail

setup() {
  set -eu -o pipefail

  export DEPLOYMENT_SOURCE="$BATS_TEST_DIRNAME/typo3"
  export BIN="$DEPLOYMENT_SOURCE/../../dist/shippy"

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
}

@test "ConnectTimeout should accept integer seconds" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  cat > "$BATS_TEST_TMPDIR/test_connect_timeout.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/shippy_key
    ssh_options:
      StrictHostKeyChecking: no
      ConnectTimeout: 30
EOF

  # Run config show to verify the option is recognized
  run bash -c "${BIN} config show test --config '$BATS_TEST_TMPDIR/test_connect_timeout.yaml' 2>&1 | sed 's/\x1b\[[0-9;]*m//g'"
  assert_success
  assert_output --partial "ConnectTimeout: 30"
}

@test "ConnectTimeout should accept duration format (30s, 5m)" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  cat > "$BATS_TEST_TMPDIR/test_connect_timeout_duration.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/shippy_key
    ssh_options:
      StrictHostKeyChecking: no
      ConnectTimeout: 1m
EOF

  # Run config show to verify the option is recognized
  run bash -c "${BIN} config show test --config '$BATS_TEST_TMPDIR/test_connect_timeout_duration.yaml' 2>&1 | sed 's/\x1b\[[0-9;]*m//g'"
  assert_success
  assert_output --partial "ConnectTimeout: 1m"
}

@test "ConnectTimeout should work with actual connection" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  cat > "$BATS_TEST_TMPDIR/test_connect_timeout_real.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/shippy_key
    ssh_options:
      StrictHostKeyChecking: no
      ConnectTimeout: 10
EOF

  # Run unlock command to test actual SSH connection
  run ${BIN} unlock test --config "$BATS_TEST_TMPDIR/test_connect_timeout_real.yaml"
  assert_success
}

@test "ServerAliveInterval should be recognized" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  cat > "$BATS_TEST_TMPDIR/test_server_alive_interval.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/shippy_key
    ssh_options:
      StrictHostKeyChecking: no
      ServerAliveInterval: 30
EOF

  # Run config show to verify the option is recognized
  run bash -c "${BIN} config show test --config '$BATS_TEST_TMPDIR/test_server_alive_interval.yaml' 2>&1 | sed 's/\x1b\[[0-9;]*m//g'"
  assert_success
  assert_output --partial "ServerAliveInterval: 30"
}

@test "ServerAliveInterval with duration format should work" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  cat > "$BATS_TEST_TMPDIR/test_server_alive_duration.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/shippy_key
    ssh_options:
      StrictHostKeyChecking: no
      ServerAliveInterval: 1m
EOF

  # Run config show to verify the option is recognized
  run bash -c "${BIN} config show test --config '$BATS_TEST_TMPDIR/test_server_alive_duration.yaml' 2>&1 | sed 's/\x1b\[[0-9;]*m//g'"
  assert_success
  assert_output --partial "ServerAliveInterval: 1m"
}

@test "ServerAliveCountMax should be recognized" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  cat > "$BATS_TEST_TMPDIR/test_server_alive_count.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/shippy_key
    ssh_options:
      StrictHostKeyChecking: no
      ServerAliveInterval: 30
      ServerAliveCountMax: 5
EOF

  # Run config show to verify the option is recognized
  run bash -c "${BIN} config show test --config '$BATS_TEST_TMPDIR/test_server_alive_count.yaml' 2>&1 | sed 's/\x1b\[[0-9;]*m//g'"
  assert_success
  assert_output --partial "ServerAliveInterval: 30"
  assert_output --partial "ServerAliveCountMax: 5"
}

@test "ServerAliveInterval should work with actual connection" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  cat > "$BATS_TEST_TMPDIR/test_server_alive_real.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/shippy_key
    ssh_options:
      StrictHostKeyChecking: no
      ServerAliveInterval: 5
      ServerAliveCountMax: 3
EOF

  # Run unlock command to test actual SSH connection with keepalive
  run ${BIN} unlock test --config "$BATS_TEST_TMPDIR/test_server_alive_real.yaml"
  assert_success
}

@test "Compression=yes should be recognized" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  cat > "$BATS_TEST_TMPDIR/test_compression_yes.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/shippy_key
    ssh_options:
      StrictHostKeyChecking: no
      Compression: yes
EOF

  # Run config show to verify the option is recognized
  run bash -c "${BIN} config show test --config '$BATS_TEST_TMPDIR/test_compression_yes.yaml' 2>&1 | sed 's/\x1b\[[0-9;]*m//g'"
  assert_success
  assert_output --partial "Compression: yes"
}

@test "Compression=true should be recognized" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  cat > "$BATS_TEST_TMPDIR/test_compression_true.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/shippy_key
    ssh_options:
      StrictHostKeyChecking: no
      Compression: true
EOF

  # Run config show to verify the option is recognized
  run bash -c "${BIN} config show test --config '$BATS_TEST_TMPDIR/test_compression_true.yaml' 2>&1 | sed 's/\x1b\[[0-9;]*m//g'"
  assert_success
  assert_output --partial "Compression: true"
}

@test "Compression should work with actual connection" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  cat > "$BATS_TEST_TMPDIR/test_compression_real.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/shippy_key
    ssh_options:
      StrictHostKeyChecking: no
      Compression: yes
EOF

  # Run unlock command to test actual SSH connection with compression
  run ${BIN} unlock test --config "$BATS_TEST_TMPDIR/test_compression_real.yaml"
  assert_success
}

@test "Multiple new SSH options should work together" {
  set -eu -o pipefail

  cd "$DEPLOYMENT_SOURCE"

  cat > "$BATS_TEST_TMPDIR/test_multiple_options.yaml" <<EOF
hosts:
  test:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: $DEPLOYMENT_SOURCE/../ssh_keys/shippy_key
    ssh_options:
      StrictHostKeyChecking: no
      ConnectTimeout: 15
      ServerAliveInterval: 30
      ServerAliveCountMax: 3
      Compression: yes
EOF

  # Run config show to verify all options are recognized
  run bash -c "${BIN} config show test --config '$BATS_TEST_TMPDIR/test_multiple_options.yaml' 2>&1 | sed 's/\x1b\[[0-9;]*m//g'"
  assert_success
  assert_output --partial "ConnectTimeout: 15"
  assert_output --partial "ServerAliveInterval: 30"
  assert_output --partial "ServerAliveCountMax: 3"
  assert_output --partial "Compression: yes"

  # Test actual connection with all options
  run ${BIN} unlock test --config "$BATS_TEST_TMPDIR/test_multiple_options.yaml"
  assert_success
}

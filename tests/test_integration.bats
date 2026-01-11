#!/usr/bin/env bats

# Full integration tests for shippy deployment
# These tests require Docker and SSH keys
# IMPORTANT: These tests share Docker resources and must run serially

# bats file_tags=integration,slow
# bats test_tags=bats:serial

# For debugging:
#   bats ./tests/test_integration.bats --show-output-of-passing-tests --verbose-run --print-output-on-failure

set -eu -o pipefail
setup() {
  set -eu -o pipefail

  export DEPLOYMENT_SOURCE="$BATS_TEST_DIRNAME/typo3"
  export DEPLOYMENT_TARGET="$BATS_TEST_DIRNAME/www"
  export BIN="$DEPLOYMENT_SOURCE/../../dist/tinnie"
  cd $DEPLOYMENT_SOURCE
  # composer install --no-interaction --no-progress --prefer-dist --optimize-autoloader --no-dev

  TEST_BREW_PREFIX="$(brew --prefix 2>/dev/null || true)"
  export BATS_LIB_PATH="${BATS_LIB_PATH}:${TEST_BREW_PREFIX}/lib:/usr/lib/bats"
  bats_load_library bats-assert
  bats_load_library bats-file
  bats_load_library bats-support

  run rm -rf "${DEPLOYMENT_TARGET}/*" "${DEPLOYMENT_TARGET}/.cache"
  run docker compose -f "${BATS_TEST_DIRNAME}/docker-compose.yaml" up -d --build
  assert_success
}

teardown() {
  set -eu -o pipefail
  run docker compose -f "${BATS_TEST_DIRNAME}/docker-compose.yaml" down
}

@test "Validate minimal config" {
  set -eu -o pipefail
  run ${BIN} config validate --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_success
}

@test "Deploy application with minimal config" {
  set -eu -o pipefail
  run ${BIN} deploy production
  assert_success

  assert_output --partial "Congratulations - TYPO3 Setup is done."
  assert_output --partial "[OK] Extension(s)"
  assert_output --partial "[OK] No wizards left to run."
  assert_output --partial "Updating language packs"
  assert_output --partial "Release activated - site is now live!"
  assert_output --partial "Kept last 2 releases"
}

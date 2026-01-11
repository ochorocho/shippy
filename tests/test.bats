#!/usr/bin/env bats

# For debugging:
#   bats ./tests/test.bats --show-output-of-passing-tests --verbose-run --print-output-on-failure

set -eu -o pipefail
setup() {
  set -eu -o pipefail

  export DEPLOYMENT_SOURCE="$BATS_TEST_DIRNAME/typo3"
  export DEPLOYMENT_TARGET="$BATS_TEST_DIRNAME/www"
  export BIN="$DEPLOYMENT_SOURCE/../../shippy"
  cd $DEPLOYMENT_SOURCE
  # composer install --no-interaction --no-progress --prefer-dist --optimize-autoloader --no-dev

  TEST_BREW_PREFIX="$(brew --prefix 2>/dev/null || true)"
  export BATS_LIB_PATH="${BATS_LIB_PATH}:${TEST_BREW_PREFIX}/lib:/usr/lib/bats"
  bats_load_library bats-assert
  bats_load_library bats-file
  bats_load_library bats-support

  run docker compose -f "${BATS_TEST_DIRNAME}/docker-compose.yaml" up -d --build
  run rm -rf "${DEPLOYMENT_TARGET}/*" "${DEPLOYMENT_TARGET}/.cache"
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
  run ${BIN} deploy production --config ${BATS_TEST_DIRNAME}/config-test/minimal.yaml
  assert_success

  # assert_output --partial "Uploaded 15310 files to cache"
  # assert_output --partial "[1/4] Install TYPO3"
  # assert_output --partial "[2/4] Run extension setup"
  assert_output --partial "[3/4] Database migrations"
  assert_output --partial "[4/4] Warmup caches"
  assert_output --partial "Release activated - site is now live!"
  assert_output --partial "Kept last 5 releases"
}

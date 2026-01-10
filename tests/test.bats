#!/usr/bin/env bats

# For debugging:
#   bats ./tests/test.bats --show-output-of-passing-tests --verbose-run --print-output-on-failure

set -eu -o pipefail
setup() {
  set -eu -o pipefail

  export DEPLOYMENT_SOURCE="$BATS_TEST_DIRNAME/typo3"
  export DEPLOYMENT_TARGET="$BATS_TEST_DIRNAME/www"

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

@test "install from directory" {
  set -eu -o pipefail
  run echo "Setup complete"
  assert_success
}

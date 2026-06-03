#!/usr/bin/env bats

# Full integration tests for shippy against a TYPO3 v14 project.
# Unlike the v13 fixture, v14 ships the core `configuration:show` command, so
# database credentials are read from TYPO3 directly rather than parsed from
# settings.php.
# These tests require Docker and SSH keys and must run serially.

# bats file_tags=integration,slow
# bats test_tags=bats:serial

# For debugging:
#   bats ./tests/test_integration_v14.bats --show-output-of-passing-tests --verbose-run --print-output-on-failure

set -eu -o pipefail
setup() {
  set -eu -o pipefail

  export DEPLOYMENT_SOURCE="$BATS_TEST_DIRNAME/typo3-v14"
  export DEPLOYMENT_TARGET="$BATS_TEST_DIRNAME/www"
  export BIN="$DEPLOYMENT_SOURCE/../../dist/shippy"
  cd $DEPLOYMENT_SOURCE

  TEST_BREW_PREFIX="$(brew --prefix 2>/dev/null || true)"
  export BATS_LIB_PATH="${BATS_LIB_PATH}:${TEST_BREW_PREFIX}/lib:/usr/lib/bats"
  bats_load_library bats-assert
  bats_load_library bats-file
  bats_load_library bats-support

  run docker compose -f "${BATS_TEST_DIRNAME}/docker-compose.yaml" up -d --build --wait
  assert_success

  # Reset the deploy target inside the container (files are root-owned on the
  # bind mount, so the non-root CI runner cannot delete them from the host).
  # This clears the v13 fixture's leftover release and rsync cache at
  # /var/www/html/.cache, which would otherwise corrupt the v14 vendor autoloader.
  run docker compose -f "${BATS_TEST_DIRNAME}/docker-compose.yaml" exec -T typo3-shippy-apache \
    rm -rf /var/www/html/releases /var/www/html/shared /var/www/html/current /var/www/html/.cache /var/www/html/.shippy
  assert_success
}

teardown() {
  set -eu -o pipefail
  run docker compose -f "${BATS_TEST_DIRNAME}/docker-compose.yaml" down
}

@test "Deploy TYPO3 v14 application" {
  set -eu -o pipefail
  run ${BIN} deploy production
  assert_success

  assert_output --partial "Release activated - site is now live!"
}

@test "Backup reads database credentials via typo3 configuration:show on v14" {
  set -eu -o pipefail

  # Deploy first so a bootstrappable release with the typo3 binary exists.
  run ${BIN} deploy production
  assert_success
  assert_output --partial "Release activated - site is now live!"

  out_dir="${BATS_TMPDIR}/shippy-backup-v14-out"
  rm -rf "${out_dir}"
  mkdir -p "${out_dir}"

  run ${BIN} backup production --output "${out_dir}"
  assert_success
  # The credential source must be the core command, not the settings.php fallback.
  assert_output --partial "via typo3 configuration:show"
  assert_output --partial "Database: mysql (db@typo3-shippy-mariadb:3308/db)"
  assert_output --partial "Database dumped successfully"
  assert_output --partial "Backup completed successfully!"

  # A ZIP archive was written to the output directory.
  run bash -c "ls ${out_dir}/backup-production-*.zip"
  assert_success
}

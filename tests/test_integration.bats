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
  export BIN="$DEPLOYMENT_SOURCE/../../dist/shippy"
  cd $DEPLOYMENT_SOURCE
  # composer install --no-interaction --no-progress --prefer-dist --optimize-autoloader --no-dev

  TEST_BREW_PREFIX="$(brew --prefix 2>/dev/null || true)"
  export BATS_LIB_PATH="${BATS_LIB_PATH}:${TEST_BREW_PREFIX}/lib:/usr/lib/bats"
  bats_load_library bats-assert
  bats_load_library bats-file
  bats_load_library bats-support

  run docker compose -f "${BATS_TEST_DIRNAME}/docker-compose.yaml" up -d --build --wait
  assert_success

  # Reset the deploy target between tests. Files on the bind-mounted target are
  # created by root inside the container, so the (non-root) CI runner cannot
  # delete them from the host — clean them inside the container instead. This
  # also clears the remote rsync cache at /var/www/html/.cache, which would
  # otherwise leak files between fixtures/tests.
  run docker compose -f "${BATS_TEST_DIRNAME}/docker-compose.yaml" exec -T typo3-shippy-apache \
    rm -rf /var/www/html/releases /var/www/html/shared /var/www/html/current /var/www/html/.cache /var/www/html/.shippy
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

@test "Backup database credentials fall back to settings.php when configuration:show is unavailable" {
  set -eu -o pipefail

  # Deploy first so a release with config/system/settings.php exists.
  run ${BIN} deploy production
  assert_success
  assert_output --partial "Release activated - site is now live!"

  out_dir="${BATS_TMPDIR}/shippy-backup-out"
  rm -rf "${out_dir}"
  mkdir -p "${out_dir}"

  # TYPO3 v13.4 has no `configuration:show` command, so extraction must fall
  # back to settings.php and still dump the database through the SSH tunnel.
  run ${BIN} backup production --output "${out_dir}"
  assert_success
  assert_output --partial "Database: mysql (db@typo3-shippy-mariadb:3308/db)"
  # v13.4 has no configuration:show, so credentials come from the config files.
  assert_output --partial "via config files"
  assert_output --partial "Database dumped successfully"
  assert_output --partial "Backup completed successfully!"

  # A ZIP archive was written to the output directory.
  run bash -c "ls ${out_dir}/backup-production-*.zip"
  assert_success
}

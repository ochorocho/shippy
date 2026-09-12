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

@test "Deploy with command_context wraps commands in a subcontext" {
  set -eu -o pipefail

  # command-context.yaml sets `command_context: env`, a transparent passthrough
  # present in the container. This exercises the full wrapping path
  #   env sh -c 'cd <release> && <run>'
  # through the real remote shell: if the `cd` or the sh -c quoting were wrong
  # the `test -f composer.json` guard would fail the deploy. The second command
  # uses `command_context: ""` to force it back onto the host, so both branches
  # are covered in one deploy.
  cd "$BATS_TEST_DIRNAME/config-test"

  run ${BIN} deploy production --config command-context.yaml
  assert_success

  # Both commands run `pwd` and must resolve to the release directory, proving
  # the `cd` took effect both inside the context and on the host.
  assert_output --partial "/var/www/html/releases/"
  assert_output --partial "All commands executed successfully"
  assert_output --partial "Release activated - site is now live!"
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

@test "Concurrent deploys contend for the lock - exactly one is rejected" {
  set -eu -o pipefail

  # Use the lightweight lock-race config (a single `sleep` command) so the
  # winning deploy holds the lock briefly without running the full TYPO3 setup.
  # ssh_key '../ssh_keys/shippy_key' resolves relative to this directory.
  cd "$BATS_TEST_DIRNAME/config-test"

  out1="${BATS_TMPDIR}/shippy-lock-race-1.log"
  out2="${BATS_TMPDIR}/shippy-lock-race-2.log"
  rm -f "$out1" "$out2"

  # Launch two deploys as simultaneously as possible so both race for the lock.
  ${BIN} deploy production --config lock-race.yaml >"$out1" 2>&1 &
  pid1=$!
  ${BIN} deploy production --config lock-race.yaml >"$out2" 2>&1 &
  pid2=$!

  wait "$pid1" || true
  wait "$pid2" || true

  # Exactly one run must win (deploy succeeds) and exactly one must be rejected
  # by the lock. Asserting both guards against a false pass where the winner
  # fails for an unrelated reason while the loser is still rejected.
  locked_count=0
  success_count=0
  for f in "$out1" "$out2"; do
    if grep -q "deployment is locked" "$f"; then
      locked_count=$((locked_count + 1))
    fi
    if grep -q "Deployment completed successfully" "$f"; then
      success_count=$((success_count + 1))
    fi
  done

  if [ "$locked_count" -ne 1 ] || [ "$success_count" -ne 1 ]; then
    echo "Expected exactly 1 success and 1 lock rejection, got success=${success_count} locked=${locked_count}"
    echo "----- run 1 -----"; cat "$out1"
    echo "----- run 2 -----"; cat "$out2"
  fi
  [ "$success_count" -eq 1 ]
  [ "$locked_count" -eq 1 ]
}

@test "Checksum cache: CI mtime reset skips uploads, changed same-size file is not skipped" {
  set -eu -o pipefail

  # Reproduces issue #34 end to end. The old code compared size+mtime, which:
  #   1. re-uploaded every file in CI (a fresh checkout resets all mtimes), and
  #   2. could skip a file that changed but kept the same size with an older
  #      mtime, silently deploying stale content.
  # The checksum manifest must fix both.
  cd "$BATS_TEST_DIRNAME/config-test"

  # Small, stable source tree. settings.php holds a 10-byte payload we later
  # rewrite in place to the same length.
  src="${BATS_TMPDIR}/shippy-cache-src"
  rm -rf "$src"
  mkdir -p "$src/app"
  printf 'alpha\n' > "$src/app/a.txt"
  printf 'beta\n'  > "$src/app/b.txt"
  printf "['x'=>1];\n" > "$src/app/settings.php"

  # Fast config: deploy the app/ tree only, no TYPO3 setup commands.
  cat > cache-test.yaml <<EOF
rsync_src: ${src}
keep_releases: 2
include:
  - app/
commands:
  - name: noop
    run: "true"
hosts:
  production:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: ../ssh_keys/shippy_key
    ssh_options:
      StrictHostKeyChecking: accept-new
EOF

  # First deploy: empty cache, all three files upload.
  run ${BIN} deploy production --config cache-test.yaml
  assert_success
  assert_output --partial "Found 3 files to upload"

  # Simulate a fresh CI checkout: every file gets a brand-new (newer) mtime.
  find "$src" -type f -exec touch {} +

  # Second deploy: identical content, so the checksum cache must skip all three
  # despite every mtime now being newer than the cache.
  run ${BIN} deploy production --config cache-test.yaml
  assert_success
  assert_output --partial "0 new/changed files, 3 unchanged (cached)"
  assert_output --partial "No files to upload - all files are cached"

  # Change one file to the SAME size but an OLDER mtime, the exact shape the old
  # mtime comparison wrongly skipped. Checksum must catch it.
  printf "['x'=>9];\n" > "$src/app/settings.php"
  touch -t 200001010000 "$src/app/settings.php"

  run ${BIN} deploy production --config cache-test.yaml
  assert_success
  assert_output --partial "1 new/changed files, 2 unchanged (cached)"

  # The activated release must have the new content, not the stale one.
  run docker compose -f "${BATS_TEST_DIRNAME}/docker-compose.yaml" exec -T typo3-shippy-apache \
    cat /var/www/html/current/app/settings.php
  assert_success
  assert_output --partial "['x'=>9];"

  # The manifest lives in .shippy/, never inside the promoted .cache/.
  run docker compose -f "${BATS_TEST_DIRNAME}/docker-compose.yaml" exec -T typo3-shippy-apache \
    test -f /var/www/html/.shippy/manifest.json
  assert_success

  rm -f cache-test.yaml
}

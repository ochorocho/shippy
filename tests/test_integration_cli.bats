#!/usr/bin/env bats

# End-to-end CLI tests for rollback, backup and unlock against the Docker host.
# Fixtures are small, so one stack is shared across this file.
# IMPORTANT: These tests share Docker resources and must run serially

# bats file_tags=integration,slow
# bats test_tags=bats:serial

bats_require_minimum_version 1.5.0

set -eu -o pipefail

setup_file() {
  # Fail instead of stalling CI if a command hangs.
  export BATS_TEST_TIMEOUT=120

  export COMPOSE_FILE_PATH="${BATS_TEST_DIRNAME}/docker-compose.yaml"
  docker compose -f "${COMPOSE_FILE_PATH}" up -d --build --wait >&3 2>&1

  # MariaDB data lives on tmpfs, so this table is gone with the stack.
  docker compose -f "${COMPOSE_FILE_PATH}" exec -T typo3-shippy-mariadb \
    mariadb -uroot -pdb db -e "CREATE TABLE IF NOT EXISTS backup_probe (id INT); INSERT INTO backup_probe VALUES (42);"
}

teardown_file() {
  docker compose -f "${COMPOSE_FILE_PATH}" down >&3 2>&1 || true
}

setup() {
  set -eu -o pipefail
  load test_helpers
  common_setup

  remote rm -rf /var/www/html/releases /var/www/html/shared /var/www/html/current /var/www/html/.cache /var/www/html/.shippy

  PROJECT_DIR="${BATS_TEST_TMPDIR}/project"
  mkdir -p "${PROJECT_DIR}/app"
  cd "${PROJECT_DIR}"
  create_test_composer_json

  cat > .shippy.yaml <<EOF
rsync_src: .
keep_releases: 5
include:
  - app/
commands:
  - name: noop
    run: "true"
rollback_commands:
  - name: record working directory
    run: "pwd > rollback_marker.txt"
  - name: staging only
    run: "touch skipped_marker.txt"
    only: [staging]
backup:
  database:
    credentials: manual
    driver: mysql
    host: typo3-shippy-mariadb
    port: 3308
    name: db
    user: db
    password: db
  files:
    - fileadmin/
hosts:
  production:
    hostname: 127.0.0.1
    port: 2424
    remote_user: root
    deploy_path: /var/www/html
    ssh_key: ${BATS_TEST_DIRNAME}/ssh_keys/shippy_key
    ssh_options:
      StrictHostKeyChecking: "no"
  staging:
    hostname: staging.invalid
    remote_user: root
    deploy_path: /var/www/html
EOF
}

remote() {
  docker compose -f "${COMPOSE_FILE_PATH}" exec -T typo3-shippy-apache "$@"
}

# Deploys a release whose content identifies it, e.g. `deploy_version v1`.
deploy_version() {
  printf '%s\n' "$1" > app/version.txt
  run ${BIN} deploy production
  assert_success
}

# Deploys v1, v2, v3 and sets RELEASE_V1..3 to their release names.
deploy_three_releases() {
  deploy_version v1
  deploy_version v2
  deploy_version v3
  local names
  names=($(remote ls -1 /var/www/html/releases | sort))
  [ "${#names[@]}" -eq 3 ]
  RELEASE_V1="${names[0]}"
  RELEASE_V2="${names[1]}"
  RELEASE_V3="${names[2]}"
}

assert_live_version() {
  run remote cat /var/www/html/current/app/version.txt
  assert_success
  assert_output "$1"
}

# Prints the entries of the only backup ZIP in the given directory.
zip_entries() {
  unzip -Z1 "$1"/backup-production-*.zip
}

# Rollback

@test "rollback --list shows all releases and marks the current one" {
  deploy_three_releases

  run -0 ${BIN} rollback production --list
  assert_output --partial "Total: 3 releases"
  assert_line --regexp "${RELEASE_V3}[[:space:]].*\(current\)"
  assert_line --partial "${RELEASE_V1}"
  refute_line --regexp "${RELEASE_V1}[[:space:]].*\(current\)"
}

@test "rollback -l does not switch the release" {
  deploy_three_releases

  run -0 ${BIN} rollback production -l
  assert_live_version v3
}

@test "rollback -n -1 switches to the previous release" {
  deploy_three_releases

  run -0 ${BIN} rollback production -n -1
  assert_output --partial "Rolled back to ${RELEASE_V2}"
  assert_live_version v2
}

@test "rollback --offset moves back and forward from the current release" {
  deploy_three_releases

  run -0 ${BIN} rollback production --offset=-2
  assert_live_version v1

  run -0 ${BIN} rollback production --offset=1
  assert_output --partial "Rolled back to ${RELEASE_V2}"
  assert_live_version v2
}

@test "rollback --release switches to the named release" {
  deploy_three_releases

  run -0 ${BIN} rollback production --release "${RELEASE_V1}"
  assert_output --partial "Rolled back to ${RELEASE_V1}"
  assert_live_version v1
}

@test "rollback -r switches to the named release" {
  deploy_three_releases

  run -0 ${BIN} rollback production -r "${RELEASE_V2}"
  assert_live_version v2
}

@test "shit alias performs a rollback" {
  deploy_three_releases

  run -0 ${BIN} shit production -n -1
  assert_live_version v2
}

@test "rollback runs rollback_commands in the selected release" {
  deploy_three_releases

  run -0 ${BIN} rollback production -n -1

  run remote cat "/var/www/html/releases/${RELEASE_V2}/rollback_marker.txt"
  assert_success
  assert_output "/var/www/html/releases/${RELEASE_V2}"
}

@test "rollback skips rollback_commands limited to other hosts" {
  deploy_three_releases

  run -0 ${BIN} rollback production -n -1
  assert_output --partial "Skipping staging only"

  run remote test -e "/var/www/html/releases/${RELEASE_V2}/skipped_marker.txt"
  assert_failure
}

@test "rollback reports a failing rollback command" {
  deploy_version v1
  deploy_version v2
  sed -i.bak 's/pwd > rollback_marker.txt/exit 7/' .shippy.yaml

  run -1 ${BIN} rollback production -n -1
  assert_output --partial "is live, but a rollback command failed"
  assert_live_version v1
}

@test "rollback -r refuses the current release" {
  deploy_three_releases

  run -1 ${BIN} rollback production -r "${RELEASE_V3}"
  assert_output --partial "already the current release"
  assert_live_version v3
}

@test "rollback -r refuses an unknown release" {
  deploy_three_releases

  run -1 ${BIN} rollback production -r 19700101000000
  assert_output --partial "does not exist"
  assert_live_version v3
}

@test "rollback -n refuses an offset past the oldest release" {
  deploy_three_releases

  run -1 ${BIN} rollback production -n -3
  assert_output --partial "no older release available"
  assert_live_version v3
}

@test "rollback -n refuses an offset past the newest release" {
  deploy_three_releases

  run -1 ${BIN} rollback production -n 1
  assert_output --partial "no newer release available"
  assert_live_version v3
}

@test "rollback without a target fails when only one release exists" {
  deploy_version v1

  run -1 ${BIN} rollback production
  assert_output --partial "Only one release exists"
}

@test "rollback without a target fails without a terminal instead of hanging" {
  deploy_three_releases

  run -1 ${BIN} rollback production < /dev/null
  assert_output --partial "interactive selection needs a terminal: use --release or --offset"
  assert_live_version v3
}

# Runs `shippy rollback production` in a pseudo-terminal and types the given keys
# once the selector is showing. Exits with the status of shippy.
rollback_in_terminal() {
  TERM=xterm KEYS="$1" expect -c '
    set timeout 60
    spawn '"${BIN}"' rollback production
    expect {
      "Select release" {}
      timeout { exit 124 }
      eof { exit 125 }
    }
    send -- $env(KEYS)
    expect {
      eof {}
      timeout { exit 124 }
    }
    exit [lindex [wait] 3]
  '
}

@test "interactive rollback switches to the release picked in a terminal" {
  deploy_three_releases

  run -0 rollback_in_terminal $'j\r'
  assert_output --partial "Rolled back to"
  assert_live_version v1
}

@test "interactive rollback leaves the release alone when cancelled in a terminal" {
  deploy_three_releases

  run -1 rollback_in_terminal 'q'
  assert_output --partial "selection cancelled"
  assert_live_version v3
}

@test "rollback fails when no releases exist" {
  run -1 ${BIN} rollback production -n -1
  assert_output --partial "No releases found"
}

# Backup

setup_backup_source() {
  remote sh -c 'mkdir -p /var/www/html/shared/fileadmin && echo hello > /var/www/html/shared/fileadmin/hello.txt'
  OUT_DIR="${BATS_TEST_TMPDIR}/out"
  mkdir -p "${OUT_DIR}"
}

@test "backup includes the database dump and configured files" {
  setup_backup_source

  run -0 ${BIN} backup production --output "${OUT_DIR}"

  run zip_entries "${OUT_DIR}"
  assert_success
  assert_line "database/dump.sql"
  assert_line "shared/fileadmin/hello.txt"

  run bash -c "unzip -p '${OUT_DIR}'/backup-production-*.zip database/dump.sql"
  assert_success
  assert_output --partial "backup_probe"
}

@test "backup --skip-database leaves out the dump" {
  setup_backup_source

  run -0 ${BIN} backup production --skip-database -o "${OUT_DIR}"
  refute_output --partial "Dumping database"

  run zip_entries "${OUT_DIR}"
  assert_success
  assert_line "shared/fileadmin/hello.txt"
  refute_line "database/dump.sql"
}

@test "backup --skip-shared leaves out the files" {
  setup_backup_source

  run -0 ${BIN} backup production --skip-shared -o "${OUT_DIR}"
  refute_output --partial "Downloading files"

  run zip_entries "${OUT_DIR}"
  assert_success
  assert_line "database/dump.sql"
  refute_line "shared/fileadmin/hello.txt"
}

@test "backup lists files only with --verbose and -v" {
  setup_backup_source

  run -0 ${BIN} backup production --skip-database -o "${OUT_DIR}"
  refute_output --partial "shared/fileadmin/hello.txt"

  run -0 ${BIN} backup production --skip-database --verbose -o "${OUT_DIR}"
  assert_output --partial "shared/fileadmin/hello.txt"

  run -0 ${BIN} backup production --skip-database -v -o "${OUT_DIR}"
  assert_output --partial "shared/fileadmin/hello.txt"
}

# Deploy

@test "deploy ships a changed file with equal size and mtime" {
  printf 'v1\n' > app/version.txt
  touch -t 202601010000 app/version.txt
  run -0 ${BIN} deploy production

  printf 'v2\n' > app/version.txt
  touch -t 202601010000 app/version.txt
  run -0 ${BIN} deploy production
  assert_live_version v2
}

@test "deploy works without a composer.json" {
  rm composer.json

  run -0 ${BIN} deploy production
  assert_output --partial "! composer.json not found at composer.json"
  assert_output --partial "Release activated"
}

@test "deploy reports changed files like new ones" {
  printf 'first\n' > app/version.txt
  run -0 ${BIN} deploy production

  printf 'second-longer\n' > app/version.txt
  run -0 ${BIN} deploy production --verbose
  assert_output --partial "Uploading: app/version.txt"
  assert_output --partial "Uploaded 1 files"
}

# Unlock

write_foreign_lock() {
  remote sh -c 'mkdir -p /var/www/html/.shippy/deploy.lock && cat > /var/www/html/.shippy/deploy.lock/info.json' <<'EOF'
{"locked_by":"someone@elsewhere","locked_at":"2026-01-01T00:00:00Z","expires_at":"2099-01-01T00:00:00Z","message":"stuck deploy","pid":1,"token":"foreign"}
EOF
}

@test "unlock removes a held lock so a blocked deploy can proceed" {
  write_foreign_lock
  printf 'v1\n' > app/version.txt

  run -1 ${BIN} deploy production
  assert_output --partial "deployment is locked"

  run -0 ${BIN} unlock production
  assert_output --partial "someone@elsewhere"
  assert_output --partial "Deployment lock removed successfully"

  run remote test -e /var/www/html/.shippy/deploy.lock
  assert_failure

  deploy_version v1
  assert_live_version v1
}

@test "unlock reports when there is no lock" {
  run -0 ${BIN} unlock production
  assert_output --partial "No active deployment lock found"
}

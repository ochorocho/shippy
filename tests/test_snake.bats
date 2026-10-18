#!/usr/bin/env bats

# Tests for the hidden snake easter egg

# bats file_tags=quick

bats_require_minimum_version 1.5.0

set -eu -o pipefail

setup() {
  set -eu -o pipefail
  load test_helpers
  common_setup
}

@test "Should hide snake from root help" {
  run -0 ${BIN} --help
  refute_output --partial "snake"
}

@test "Should refuse to play snake without an interactive terminal" {
  run -1 ${BIN} snake < /dev/null
  assert_output --partial "snake needs an interactive terminal"
}

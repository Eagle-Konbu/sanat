#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

@test "--version prints the sanat version" {
  run --separate-stderr "${SANAT_BIN}" --version

  [ "$status" -eq 0 ]
  [[ "$output" == "sanat version "* ]]
}

@test "--help prints usage" {
  run --separate-stderr "${SANAT_BIN}" --help

  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage:"* ]]
  [[ "$output" == *"sanat [flags] [pattern ...]"* ]]
}

@test "running with no args and no piped stdin shows help instead of hanging" {
  run --separate-stderr "${SANAT_BIN}" < /dev/null

  [ "$status" -eq 0 ]
  [[ "$output" == *"Usage:"* ]]
}

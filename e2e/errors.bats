#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

@test "formatting a path outside the working directory fails with a clear error" {
  outside_dir="$(mktemp -d)"
  cp "${BATS_TEST_DIRNAME}/_fixtures/format/input.go" "${outside_dir}/sample.go"

  run --separate-stderr "${SANAT_BIN}" "${outside_dir}/sample.go"

  [ "$status" -eq 0 ]
  [[ "$stderr" == *"path is outside working directory"* ]]

  rm -rf "${outside_dir}"
}

@test "a nonexistent file argument produces no output and exits successfully" {
  run --separate-stderr "${SANAT_BIN}" "${BATS_TEST_TMPDIR}/does-not-exist.go"

  [ "$status" -eq 0 ]
  [ -z "$output" ]
  [ -z "$stderr" ]
}

@test "malformed Go source reports a parse error on stderr for that file" {
  printf 'package broken\nfunc ( {\n' > "${BATS_TEST_TMPDIR}/broken.go"

  run --separate-stderr bash -c 'cd "$1" && exec "$2" broken.go' -- "${BATS_TEST_TMPDIR}" "${SANAT_BIN}"

  [ "$status" -eq 0 ]
  [[ "$stderr" == *"broken.go"* ]]
  [[ "$stderr" == *"expected"* ]]
}

#!/usr/bin/env bats

FIXTURES="${BATS_TEST_DIRNAME}/_fixtures/format"

@test "stdin is read and formatted output is written to stdout" {
  "${SANAT_BIN}" < "${FIXTURES}/input.go" > "${BATS_TEST_TMPDIR}/got.go"

  diff "${BATS_TEST_TMPDIR}/got.go" "${FIXTURES}/expected.go"
}

@test "stdin mode never touches any file on disk" {
  cp "${FIXTURES}/input.go" "${BATS_TEST_TMPDIR}/sample.go"

  "${SANAT_BIN}" < "${FIXTURES}/input.go" > "${BATS_TEST_TMPDIR}/got.go"

  diff "${BATS_TEST_TMPDIR}/sample.go" "${FIXTURES}/input.go"
}

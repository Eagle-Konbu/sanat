#!/usr/bin/env bats

FIXTURES="${BATS_TEST_DIRNAME}/_fixtures/format"

setup() {
  cp "${FIXTURES}/input.go" "${BATS_TEST_TMPDIR}/sample.go"
}

@test "-w rewrites the file in place" {
  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" -w sample.go)

  diff "${BATS_TEST_TMPDIR}/sample.go" "${FIXTURES}/expected.go"
}

@test "without -w, the file on disk is left untouched" {
  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" sample.go > /dev/null)

  diff "${BATS_TEST_TMPDIR}/sample.go" "${FIXTURES}/input.go"
}

@test "--write is a longhand alias for -w" {
  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" --write sample.go)

  diff "${BATS_TEST_TMPDIR}/sample.go" "${FIXTURES}/expected.go"
}

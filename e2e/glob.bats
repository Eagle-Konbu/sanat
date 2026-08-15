#!/usr/bin/env bats

FIXTURE_DIR="${BATS_TEST_DIRNAME}/_fixtures/globdir"

setup() {
  cp -r "${FIXTURE_DIR}" "${BATS_TEST_TMPDIR}/globdir"
}

@test "./... recursively formats nested Go files" {
  (cd "${BATS_TEST_TMPDIR}/globdir" && "${SANAT_BIN}" -w ./...)

  grep -q '^SELECT' "${BATS_TEST_TMPDIR}/globdir/a.go"
  grep -q '^SELECT' "${BATS_TEST_TMPDIR}/globdir/nested/b.go"
}

@test "a bare directory argument also recurses" {
  (cd "${BATS_TEST_TMPDIR}/globdir" && "${SANAT_BIN}" -w .)

  grep -q '^SELECT' "${BATS_TEST_TMPDIR}/globdir/a.go"
  grep -q '^SELECT' "${BATS_TEST_TMPDIR}/globdir/nested/b.go"
}

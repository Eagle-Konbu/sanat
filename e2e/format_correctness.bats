#!/usr/bin/env bats
# Spot-checks that the CLI produces the exact SQL formatting the internal
# formatter is unit-tested to produce. Not a substitute for the golden tests
# in internal/gofile/golden_test.go, just proof that CLI wiring doesn't
# mangle the result.

FIXTURES="${BATS_TEST_DIRNAME}/_fixtures/format"

@test "stdout output matches the expected formatted SQL exactly" {
  "${SANAT_BIN}" "${FIXTURES}/input.go" > "${BATS_TEST_TMPDIR}/got.go"

  diff "${BATS_TEST_TMPDIR}/got.go" "${FIXTURES}/expected.go"
}

@test "-w output matches the expected formatted SQL exactly" {
  cp "${FIXTURES}/input.go" "${BATS_TEST_TMPDIR}/sample.go"

  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" -w sample.go)

  diff "${BATS_TEST_TMPDIR}/sample.go" "${FIXTURES}/expected.go"
}

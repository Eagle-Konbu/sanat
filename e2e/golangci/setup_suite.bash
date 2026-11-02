#!/usr/bin/env bash
# bats-core special file: setup_suite() runs once before any .bats file in
# this directory. `bats e2e/` does not recurse into subdirectories, so this
# suite only runs when invoked explicitly (e.g. `bats e2e/golangci/`).
#
# Requires a stock golangci-lint v2 on PATH; it is only used to build the
# custom binary that embeds the sanat plugin from this checkout.

setup_suite() {
  export FIXTURE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/testdata"
  export CUSTOM_GCL="${BATS_SUITE_TMPDIR}/custom-gcl"

  (cd "${FIXTURE_DIR}" && golangci-lint custom --destination "${BATS_SUITE_TMPDIR}")
}

#!/usr/bin/env bash
# bats-core special file: setup_suite() runs once before any .bats file in
# this directory, when bats is invoked on the directory itself (e.g. `bats e2e/`).

setup_suite() {
  export SANAT_BIN="${BATS_SUITE_TMPDIR}/sanat"

  go build -o "${SANAT_BIN}" .
}

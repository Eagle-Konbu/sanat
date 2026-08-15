#!/usr/bin/env bats

FIXTURES="${BATS_TEST_DIRNAME}/_fixtures/format"

setup() {
  cp "${FIXTURES}/input.go" "${BATS_TEST_TMPDIR}/sample.go"
}

@test ".sanat.yml in the working directory is picked up automatically" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.yml" <<'EOF'
indent: 4
newline: false
EOF

  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" sample.go > got.go)

  grep -q '^    id,$' "${BATS_TEST_TMPDIR}/got.go"
  ! grep -q '^`$' "${BATS_TEST_TMPDIR}/got.go"
}

@test "an explicit flag overrides the config file value for that setting" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.yml" <<'EOF'
indent: 4
newline: false
EOF

  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" --indent 2 sample.go > got.go)

  grep -q '^  id,$' "${BATS_TEST_TMPDIR}/got.go"
}

#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

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

@test "keyword_case: lower in the config file lowercases operator/predicate keywords" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.yml" <<'EOF'
version: 1
keyword_case: lower
EOF

  cat > "${BATS_TEST_TMPDIR}/where.go" <<'EOF'
package sample

import "database/sql"

func query(db *sql.DB) {
	db.Query(`select id from users where active = ? and id > 1`, true)
}
EOF

  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" where.go > got.go)

  grep -q '^  and id > 1$' "${BATS_TEST_TMPDIR}/got.go"
}

@test "--keyword-case flag overrides the config file value" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.yml" <<'EOF'
version: 1
keyword_case: lower
EOF

  cat > "${BATS_TEST_TMPDIR}/where.go" <<'EOF'
package sample

import "database/sql"

func query(db *sql.DB) {
	db.Query(`select id from users where active = ? and id > 1`, true)
}
EOF

  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" --keyword-case upper where.go > got.go)

  grep -q '^  AND id > 1$' "${BATS_TEST_TMPDIR}/got.go"
}

@test "comma_style: leading in the config file places commas before the next item" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.yml" <<'EOF'
version: 1
comma_style: leading
EOF

  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" sample.go > got.go)

  grep -q '^, name$' "${BATS_TEST_TMPDIR}/got.go"
}

@test "an invalid --keyword-case flag value fails with a clear error" {
  run --separate-stderr "${SANAT_BIN}" --keyword-case sideways "${BATS_TEST_TMPDIR}/sample.go"

  [ "$status" -ne 0 ]
  [[ "$stderr" == *"keyword_case"* ]]
}

@test "an invalid keyword_case value in the config file fails with a clear error" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.yml" <<'EOF'
version: 1
keyword_case: sideways
EOF

  run --separate-stderr bash -c 'cd "$1" && exec "$2" sample.go' -- "${BATS_TEST_TMPDIR}" "${SANAT_BIN}"

  [ "$status" -ne 0 ]
  [[ "$stderr" == *"keyword_case"* ]]
}

@test "an unsupported config version fails with a clear error" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.yml" <<'EOF'
version: 99
EOF

  run --separate-stderr bash -c 'cd "$1" && exec "$2" sample.go' -- "${BATS_TEST_TMPDIR}" "${SANAT_BIN}"

  [ "$status" -ne 0 ]
  [[ "$stderr" == *"version"* ]]
}

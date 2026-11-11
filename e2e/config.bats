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

@test ".sanat.toml in the working directory is picked up automatically" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.toml" <<'EOF'
indent = 4
newline = false
EOF

  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" sample.go > got.go)

  grep -q '^    id,$' "${BATS_TEST_TMPDIR}/got.go"
  ! grep -q '^`$' "${BATS_TEST_TMPDIR}/got.go"
}

@test "an explicit flag overrides the toml config file value for that setting" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.toml" <<'EOF'
indent = 4
newline = false
EOF

  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" --indent 2 sample.go > got.go)

  grep -q '^  id,$' "${BATS_TEST_TMPDIR}/got.go"
}

@test ".sanat.yml takes priority over .sanat.toml when both are present" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.yml" <<'EOF'
indent: 2
EOF

  cat > "${BATS_TEST_TMPDIR}/.sanat.toml" <<'EOF'
indent = 4
EOF

  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" sample.go > got.go)

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

@test "keyword_case: lower in the toml config file lowercases operator/predicate keywords" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.toml" <<'EOF'
version = 1
keyword_case = "lower"
EOF

  cat > "${BATS_TEST_TMPDIR}/where.go" <<'EOF'
package sample

import "database/sql"

func query(db *sql.DB) {
	db.Query(`SELECT id FROM users WHERE active = ? AND id > 1`, true)
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

@test "an invalid keyword_case value in the toml config file fails with a clear error" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.toml" <<'EOF'
version = 1
keyword_case = "sideways"
EOF

  run --separate-stderr bash -c 'cd "$1" && exec "$2" sample.go' -- "${BATS_TEST_TMPDIR}" "${SANAT_BIN}"

  [ "$status" -ne 0 ]
  [[ "$stderr" == *"keyword_case"* ]]
}

@test "an unsupported config version in the toml config file fails with a clear error" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.toml" <<'EOF'
version = 99
EOF

  run --separate-stderr bash -c 'cd "$1" && exec "$2" sample.go' -- "${BATS_TEST_TMPDIR}" "${SANAT_BIN}"

  [ "$status" -ne 0 ]
  [[ "$stderr" == *"version"* ]]
}

# String literals are emitted as written, so sql_mode is only observable
# through where a literal ends: 'a\' is complete under no_backslash_escapes
# but unterminated (left unchanged) under default.
@test "sql_mode: no_backslash_escapes in the config file ends a literal at a backslash-quote" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.yml" <<'EOF'
version: 1
sql_mode: no_backslash_escapes
EOF

  cat > "${BATS_TEST_TMPDIR}/quote.go" <<'EOF'
package sample

import "database/sql"

func query(db *sql.DB) {
	db.Query(`select 'a\' from t`)
}
EOF

  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" quote.go > got.go)

  grep -qF "  'a\'" "${BATS_TEST_TMPDIR}/got.go"
}

@test "--sql-mode flag overrides the config file value" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.yml" <<'EOF'
version: 1
sql_mode: no_backslash_escapes
EOF

  cat > "${BATS_TEST_TMPDIR}/quote.go" <<'EOF'
package sample

import "database/sql"

func query(db *sql.DB) {
	db.Query(`select 'a\' from t`)
}
EOF

  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" --sql-mode default quote.go > got.go)

  grep -qF "select 'a\' from t" "${BATS_TEST_TMPDIR}/got.go"
}

@test "an invalid --sql-mode flag value fails with a clear error" {
  run --separate-stderr "${SANAT_BIN}" --sql-mode sideways "${BATS_TEST_TMPDIR}/sample.go"

  [ "$status" -ne 0 ]
  [[ "$stderr" == *"sql_mode"* ]]
  [[ "$stderr" == *"sideways"* ]]
}

# The jsonb "?" operator only parses as PostgreSQL, so formatting it shows
# which dialect was used: MySQL leaves the literal unchanged.
@test "dialect: postgresql in the config file formats PostgreSQL SQL" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.yml" <<'EOF'
version: 1
dialect: postgresql
EOF

  cat > "${BATS_TEST_TMPDIR}/jsonb.go" <<'EOF'
package sample

import "database/sql"

func query(db *sql.DB) {
	db.Query(`select id from t where data ? 'k' and id = $1`)
}
EOF

  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" jsonb.go > got.go)

  grep -qF '  AND id = $1' "${BATS_TEST_TMPDIR}/got.go"
}

@test "--dialect flag overrides the config file value" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.yml" <<'EOF'
version: 1
dialect: postgresql
EOF

  cat > "${BATS_TEST_TMPDIR}/jsonb.go" <<'EOF'
package sample

import "database/sql"

func query(db *sql.DB) {
	db.Query(`select id from t where data ? 'k' and id = $1`)
}
EOF

  (cd "${BATS_TEST_TMPDIR}" && "${SANAT_BIN}" --dialect mysql jsonb.go > got.go)

  grep -qF "select id from t where data ? 'k' and id = \$1" "${BATS_TEST_TMPDIR}/got.go"
}

@test "an unsupported config version fails with a clear error" {
  cat > "${BATS_TEST_TMPDIR}/.sanat.yml" <<'EOF'
version: 99
EOF

  run --separate-stderr bash -c 'cd "$1" && exec "$2" sample.go' -- "${BATS_TEST_TMPDIR}" "${SANAT_BIN}"

  [ "$status" -ne 0 ]
  [[ "$stderr" == *"version"* ]]
}

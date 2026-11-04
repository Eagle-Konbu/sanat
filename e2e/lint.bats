#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

FIXTURES="${BATS_TEST_DIRNAME}/_fixtures/lint"

setup() {
  cp -R "${FIXTURES}/." "${BATS_TEST_TMPDIR}/"
}

# lint_in runs `sanat lint` with the given arguments inside a fixture module.
lint_in() {
  local module="$1"
  shift
  run --separate-stderr bash -c 'cd "$1" && shift && exec "$@"' -- \
    "${BATS_TEST_TMPDIR}/${module}" "${SANAT_BIN}" lint "$@"
}

@test "lint exits 0 with no output for formatted SQL and non-SQL strings" {
  lint_in clean ./...

  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "lint without a pattern analyzes the current directory" {
  lint_in dirty

  [ "$status" -eq 1 ]
  [ "$output" = "dirty.go:3:21: SQL literal is not formatted (sanat)
dirty_test.go:5:19: SQL literal is not formatted (sanat)" ]
}

@test "lint exits 1 and reports file:line:col for each unformatted literal" {
  lint_in dirty ./...

  [ "$status" -eq 1 ]
  [ "$output" = "dirty.go:3:21: SQL literal is not formatted (sanat)
dirty_test.go:5:19: SQL literal is not formatted (sanat)
sub/sub.go:3:24: SQL literal is not formatted (sanat)" ]
  [[ "$stderr" == *"3 unformatted SQL literal(s) found"* ]]
}

@test "lint restricts analysis to the given package pattern" {
  lint_in dirty ./sub

  [ "$status" -eq 1 ]
  [ "$output" = "sub/sub.go:3:24: SQL literal is not formatted (sanat)" ]
}

@test "lint --tests=false skips _test.go files" {
  lint_in dirty --tests=false .

  [ "$status" -eq 1 ]
  [ "$output" = "dirty.go:3:21: SQL literal is not formatted (sanat)" ]
}

@test "lint never modifies files" {
  lint_in dirty ./...

  [ "$status" -eq 1 ]
  diff -r "${FIXTURES}/dirty" "${BATS_TEST_TMPDIR}/dirty"
}

@test "lint exits 2 when a package does not type-check" {
  lint_in broken ./...

  [ "$status" -eq 2 ]
  [ -z "$output" ]
  [[ "$stderr" == *"broken.go:5:"* ]]
}

@test "lint exits 2 when a pattern matches no package" {
  lint_in clean ./missing

  [ "$status" -eq 2 ]
  [ -z "$output" ]
}

@test "lint picks up .sanat.yml from the working directory" {
  cat > "${BATS_TEST_TMPDIR}/clean/.sanat.yml" <<'EOF'
version: 1
indent: 4
EOF

  lint_in clean ./...

  [ "$status" -eq 1 ]
  [[ "$output" == *"clean.go:5:21: SQL literal is not formatted (sanat)"* ]]
}

@test "lint flags override the config file" {
  cat > "${BATS_TEST_TMPDIR}/clean/.sanat.yml" <<'EOF'
version: 1
indent: 4
EOF

  lint_in clean --indent 2 ./...

  [ "$status" -eq 0 ]
  [ -z "$output" ]
}

@test "lint --keyword-case changes which literals are reported" {
  lint_in clean --keyword-case lower ./...

  [ "$status" -eq 1 ]
  [ "$output" = "clean.go:5:21: SQL literal is not formatted (sanat)" ]
}

@test "lint keyword_case in .sanat.toml changes which literals are reported" {
  cat > "${BATS_TEST_TMPDIR}/clean/.sanat.toml" <<'EOF'
version = 1
keyword_case = "lower"
EOF

  lint_in clean ./...

  [ "$status" -eq 1 ]
  [ "$output" = "clean.go:5:21: SQL literal is not formatted (sanat)" ]
}

@test "lint exits 2 on an invalid flag value" {
  lint_in clean --keyword-case sideways

  [ "$status" -eq 2 ]
  [[ "$stderr" == *"keyword_case"* ]]
}

@test "lint exits 2 on an unknown flag" {
  lint_in clean --write

  [ "$status" -eq 2 ]
  [[ "$stderr" == *"unknown flag"* ]]
}

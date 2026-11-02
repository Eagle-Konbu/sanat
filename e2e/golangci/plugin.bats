#!/usr/bin/env bats
bats_require_minimum_version 1.5.0

setup() {
  # golangci-lint's cache does not track the referenced sanat config file,
  # so each test gets a fresh cache to keep results independent.
  export GOLANGCI_LINT_CACHE="${BATS_TEST_TMPDIR}/golangci-cache"
  cd "${FIXTURE_DIR}" || return 1
}

@test "custom binary lists sanat as an enabled linter" {
  run "${CUSTOM_GCL}" linters

  [ "$status" -eq 0 ]
  [[ "$output" == *"sanat"* ]]
}

@test "reports an unformatted SQL literal" {
  run "${CUSTOM_GCL}" run ./unformatted/...

  [ "$status" -eq 1 ]
  [[ "$output" == *"unformatted.go:3:"* ]]
  [[ "$output" == *"SQL literal is not formatted (sanat)"* ]]
}

@test "passes when every SQL literal is formatted" {
  run "${CUSTOM_GCL}" run ./formatted/...

  [ "$status" -eq 0 ]
}

@test "reports a leading-comma literal under the default settings" {
  run "${CUSTOM_GCL}" run ./leading/...

  [ "$status" -eq 1 ]
  [[ "$output" == *"SQL literal is not formatted (sanat)"* ]]
}

@test "the config setting applies values from the referenced sanat config file" {
  run "${CUSTOM_GCL}" run -c golangci-with-config.yml ./leading/...

  [ "$status" -eq 0 ]
}

@test "an invalid setting fails before linting" {
  cat > "${BATS_TEST_TMPDIR}/golangci.yml" <<'EOF'
version: "2"
linters:
  default: none
  enable:
    - sanat
  settings:
    custom:
      sanat:
        type: module
        settings:
          comma-style: middle
EOF

  run "${CUSTOM_GCL}" run -c "${BATS_TEST_TMPDIR}/golangci.yml" ./formatted/...

  [ "$status" -ne 0 ]
  [ "$status" -ne 1 ]
  [[ "$output" == *"comma_style must be one of"* ]]
}

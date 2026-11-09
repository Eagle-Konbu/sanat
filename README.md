# sanat

[![CI](https://github.com/Eagle-Konbu/sanat/actions/workflows/ci.yml/badge.svg)](https://github.com/Eagle-Konbu/sanat/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/Eagle-Konbu/sanat/branch/main/graph/badge.svg)](https://codecov.io/gh/Eagle-Konbu/sanat)

Yet another CLI tool that automatically formats embedded SQL literals in Go source files.

## Overview

`sanat` scans Go source files for raw string literals (backtick strings) containing SQL, parses them, and reformats them into a consistent, readable style.

### Before

```go
db.Query(`select u.id, o.total from users u join orders o on u.id = o.user_id where o.total > ?`, 100)
```

### After

```go
db.Query(`
SELECT
  u.id,
  o.total
FROM
  users u
  JOIN
  orders o
    ON u.id = o.user_id
WHERE
  o.total > ?
`, 100)
```

## Features

- Formats SQL in raw string literals (backticks)
- Supports SELECT, INSERT, UPDATE, DELETE, and UNION statements
- Preserves placeholders (`?`)
- Skips non-SQL strings (plain text, fmt templates, URLs)
- Configurable indentation
- Stdin/stdout support for editor integration
- [golangci-lint plugin](#golangci-lint-plugin) that reports unformatted SQL literals

## Installation

### Homebrew

```bash
brew install Eagle-Konbu/tap/sanat
```

### Go

```bash
go install github.com/Eagle-Konbu/sanat@latest
```

## Usage

### Format files and print to stdout

```bash
sanat file.go
sanat ./...
```

### Format files in place

```bash
sanat -w file.go
sanat -w ./...
```

### Format from stdin

```bash
cat file.go | sanat > formatted.go
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `-w, --write` | `false` | Overwrite files in place |
| `--indent` | `2` | Indent width for SQL formatting |
| `--newline` | `true` | Add newline after opening backtick |
| `--keyword-case` | `upper` | Casing for operator/predicate keywords (`upper`, `lower`, `preserve`) |
| `--comma-style` | `trailing` | Comma placement in lists (`trailing`, `leading`) |
| `--sql-mode` | `default` | SQL mode controlling string-literal parsing (`default`, `no_backslash_escapes`) |
| `-c, --config` | | Path to config file |

## Standalone Lint

`sanat lint` checks Go packages for SQL literals that are not formatted, without modifying any files. It takes Go package patterns (default: `.`) and uses the same formatting flags and configuration file as the formatter.

```bash
sanat lint ./...
sanat lint ./internal/db
sanat lint --tests=false ./...   # skip _test.go files
```

Each finding is printed to stdout as `file:line:col: message`:

```
internal/db/users.go:12:15: SQL literal is not formatted (sanat)
```

| Exit status | Meaning |
|-------------|---------|
| `0` | No findings |
| `1` | Unformatted SQL literals were found |
| `2` | Analysis failed (invalid flags or config, or packages that fail to load or compile) |

To fix the reported literals, run `sanat -w` on the reported files (for example, `sanat -w internal/db/users.go`). Note that `sanat -w ./...` rewrites every SQL literal in all Go files under the directory, including `_test.go` files and files `lint` did not analyze.

### CI example (GitHub Actions)

```yaml
- uses: actions/checkout@v4

- uses: actions/setup-go@v5
  with:
    go-version: stable

- name: Install sanat
  run: go install github.com/Eagle-Konbu/sanat@latest

- name: Check SQL formatting
  run: sanat lint ./...
```

## Configuration File

Settings can also be defined in a configuration file placed in the current directory. The following files are searched in order:

1. `.sanat.yml`
2. `.sanat.yaml`
3. `.sanat.toml`

You can also specify a config file explicitly with the `--config` (`-c`) flag:

```bash
sanat -c path/to/config.yml file.go
```

CLI flags take precedence over configuration file values when explicitly specified.

### YAML example (`.sanat.yml`)

```yaml
version: 1
write: true
indent: 4
newline: false
keyword_case: upper
comma_style: trailing
sql_mode: default
```

### TOML example (`.sanat.toml`)

```toml
version = 1
write = true
indent = 4
newline = false
keyword_case = "upper"
comma_style = "trailing"
sql_mode = "default"
```

See [docs/formatter-spec.md](docs/formatter-spec.md#configuration) for the full list of configuration options.

## golangci-lint Plugin

sanat can run inside [golangci-lint](https://golangci-lint.run/) as a [module plugin](https://golangci-lint.run/plugins/module-plugins/). It reports every SQL raw string literal that `sanat` would reformat with the message `SQL literal is not formatted`, and attaches the formatted literal as a suggested fix (`./custom-gcl run --fix`).

### Requirements

- **golangci-lint v2 only.** The plugin is built for the v2 Module Plugin System. v2.15.0 is the minimum supported version (it is the version tested in CI); golangci-lint v1 is not supported.
- **A custom binary is required.** Module plugins are compiled into a dedicated golangci-lint binary built with `golangci-lint custom`. The stock `golangci-lint` binary (Homebrew, release archives, `golangci-lint-action` defaults, ...) cannot load the plugin.
- Building golangci-lint v2.15.0 requires Go 1.26 or later.

### 1. Build the custom binary

No sanat release includes the plugin package yet, so build it from a local checkout of this repository. Clone sanat next to your project and create `.custom-gcl.yml` in your project root:

```yaml
version: v2.15.0
plugins:
  - module: github.com/Eagle-Konbu/sanat
    import: github.com/Eagle-Konbu/sanat/plugin
    path: ../sanat # path to your sanat checkout, relative to where `golangci-lint custom` runs
```

Then build it with a stock golangci-lint v2 binary:

```bash
golangci-lint custom
```

This writes a `custom-gcl` binary to the current directory (configurable with `name` and `destination` in `.custom-gcl.yml`).

### 2. Enable sanat in `.golangci.yml`

```yaml
version: "2"

linters:
  enable:
    - sanat
  settings:
    custom:
      sanat:
        type: module
        description: Reports SQL raw string literals that are not formatted by sanat.
        settings:
          indent: 2
          keyword-case: upper
          # config: .sanat.yml
```

### 3. Run it

```bash
./custom-gcl run ./...
```

### Settings

| Key | Default | Description |
|-----|---------|-------------|
| `indent` | `2` | Indent width for SQL formatting |
| `newline` | `true` | Add newline after opening backtick |
| `keyword-case` | `upper` | Casing for operator/predicate keywords (`upper`, `lower`, `preserve`) |
| `comma-style` | `trailing` | Comma placement in lists (`trailing`, `leading`) |
| `sql-mode` | `default` | SQL mode controlling string-literal parsing (`default`, `no_backslash_escapes`) |
| `config` | | Path to a `.sanat.yml` / `.sanat.yaml` / `.sanat.toml` file |

Each option is resolved independently with this precedence:

1. The key set explicitly in `settings`
2. The value from the file referenced by `config`
3. The default shown above (the same as the `sanat` CLI)

Notes:

- `.sanat.*` files are **not** discovered automatically by the plugin; set `config` to share settings with the CLI. A relative `config` path is resolved against the directory `golangci-lint` is run from.
- The `write` option of a config file is ignored by the plugin.
- Unknown keys in `settings`, invalid values, and an unreadable or invalid `config` file make golangci-lint fail at startup. Unknown keys inside the referenced `config` file do not: they only print a warning and are ignored, so those options fall back to the defaults.
- golangci-lint's cache does not track the file referenced by `config`. After editing it, run `golangci-lint cache clean` (or `./custom-gcl cache clean`) so that cached results are not reused.

## Supported SQL

- `SELECT`
- `INSERT`
- `UPDATE`
- `DELETE`
- `UNION`

Strings that don't parse as valid SQL are left unchanged.

## How It Works

1. Parses Go source files using `go/parser`
2. Finds raw string literals (backtick strings)
3. Detects SQL by checking for keywords (SELECT, INSERT, UPDATE, DELETE)
4. Parses SQL using an in-house SQL parser (`internal/sqlfmt/parser`)
5. Reformats SQL with consistent indentation
6. Outputs modified Go source

## License

MIT

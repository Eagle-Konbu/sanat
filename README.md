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

## Installation

### Homebrew

```bash
brew install Eagle-Konbu/tap/sanat
```

### Go

```bash
go install github.com/Eagle-Konbu/sanat@latest
```

Or build from source:

```bash
git clone https://github.com/Eagle-Konbu/sanat.git
cd sanat
go build -o sanat ./cmd
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

## Supported SQL

- `SELECT` (with subqueries, JOINs, window functions, CTEs)
- `INSERT`
- `UPDATE`
- `DELETE`
- `UNION`

Strings that don't parse as valid SQL are left unchanged.

## How It Works

1. Parses Go source files using `go/parser`
2. Finds raw string literals (backtick strings)
3. Detects SQL by checking for keywords (SELECT, INSERT, UPDATE, DELETE)
4. Parses SQL using [Vitess](https://vitess.io/) SQL parser
5. Reformats SQL with consistent indentation
6. Outputs modified Go source

## License

MIT

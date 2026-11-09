package gofile_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/gofile"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt"
)

// goldenOptions returns the formatter options for a golden file: the
// PostgreSQL dialect for the pg_*.go files, and the defaults otherwise.
func goldenOptions(name string) *gofile.Options {
	opts := &gofile.Options{Indent: 2, Newline: true}

	if strings.HasPrefix(name, "pg_") {
		opts.Dialect = sqlfmt.DialectPostgreSQL
	}

	return opts
}

func TestGoldenFiles(t *testing.T) {
	inputDir := filepath.Join("..", "..", "testdata", "input")
	expectedDir := filepath.Join("..", "..", "testdata", "expected")

	entries, err := os.ReadDir(inputDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			inputPath := filepath.Join(inputDir, entry.Name())
			expectedPath := filepath.Join(expectedDir, entry.Name())

			src, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatal(err)
			}

			expected, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatal(err)
			}

			file, fset, literals, err := gofile.FindSQLLiterals(src, entry.Name())
			if err != nil {
				t.Fatal(err)
			}

			got, err := gofile.RewriteFile(fset, file, literals, goldenOptions(entry.Name()))
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(got, expected) {
				t.Errorf("output mismatch for %s\ngot:\n%s\nwant:\n%s", entry.Name(), got, expected)
			}
		})
	}
}

func TestGoldenFiles_ExpectedOutputNeedsNoFormatting(t *testing.T) {
	expectedDir := filepath.Join("..", "..", "testdata", "expected")

	entries, err := os.ReadDir(expectedDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" {
			continue
		}

		t.Run(entry.Name(), func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join(expectedDir, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}

			file, _, _, err := gofile.FindSQLLiterals(src, entry.Name())
			if err != nil {
				t.Fatal(err)
			}

			for _, result := range gofile.FormatLiterals(file, goldenOptions(entry.Name())) {
				t.Errorf("already formatted literal would change\ngot:  %s\nwant: %s", result.Node.Value, result.Formatted)
			}
		})
	}
}

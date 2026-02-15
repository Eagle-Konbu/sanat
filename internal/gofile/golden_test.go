package gofile_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/gofile"
)

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

			got, err := gofile.RewriteFile(fset, file, literals, gofile.Options{Indent: 2, Newline: true})
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(got, expected) {
				t.Errorf("output mismatch for %s\ngot:\n%s\nwant:\n%s", entry.Name(), got, expected)
			}
		})
	}
}

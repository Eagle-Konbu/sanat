package cmd_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Eagle-Konbu/sanat/cmd"
	"github.com/Eagle-Konbu/sanat/internal/config"
)

func assertConfig(t *testing.T, cfg config.Config, indent int, newline, write bool) {
	t.Helper()

	if cfg.Indent == nil || *cfg.Indent != indent {
		t.Errorf("indent: got %v, want %d", cfg.Indent, indent)
	}

	if cfg.Newline == nil || *cfg.Newline != newline {
		t.Errorf("newline: got %v, want %v", cfg.Newline, newline)
	}

	if cfg.Write == nil || *cfg.Write != write {
		t.Errorf("write: got %v, want %v", cfg.Write, write)
	}
}

func TestInit_CreateConfig(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		file    string
		indent  int
		newline bool
		write   bool
		wantMsg string
	}{
		{
			name:    "yaml with custom values",
			input:   "1\n4\nn\ny\n",
			file:    ".sanat.yml",
			indent:  4,
			newline: false,
			write:   true,
			wantMsg: "Created .sanat.yml",
		},
		{
			name:    "toml with custom values",
			input:   "2\n8\ny\nn\n",
			file:    ".sanat.toml",
			indent:  8,
			newline: true,
			write:   false,
			wantMsg: "Created .sanat.toml",
		},
		{
			name:    "all defaults",
			input:   "\n\n\n\n",
			file:    ".sanat.yml",
			indent:  2,
			newline: true,
			write:   false,
			wantMsg: "Created .sanat.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			var out bytes.Buffer

			if err := cmd.RunInitWith(dir, strings.NewReader(tt.input), &out); err != nil {
				t.Fatal(err)
			}

			path := filepath.Join(dir, tt.file)

			cfg, err := config.LoadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			assertConfig(t, cfg, tt.indent, tt.newline, tt.write)

			if !strings.Contains(out.String(), tt.wantMsg) {
				t.Errorf("expected %q in output, got %q", tt.wantMsg, out.String())
			}
		})
	}
}

func TestInit_ExistingFile(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{"existing yml", ".sanat.yml"},
		{"existing toml", ".sanat.toml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			if err := os.WriteFile(filepath.Join(dir, tt.file), []byte("indent: 2\n"), 0644); err != nil {
				t.Fatal(err)
			}

			input := "1\n2\ny\nn\n"

			var out bytes.Buffer

			err := cmd.RunInitWith(dir, strings.NewReader(input), &out)
			if err == nil {
				t.Fatal("expected error for existing config file")
			}

			if !strings.Contains(err.Error(), "already exists") {
				t.Errorf("expected 'already exists' error, got %q", err.Error())
			}
		})
	}
}

func TestInit_InvalidFormat(t *testing.T) {
	dir := t.TempDir()
	input := "3\n"

	var out bytes.Buffer

	err := cmd.RunInitWith(dir, strings.NewReader(input), &out)
	if err == nil {
		t.Fatal("expected error for invalid format choice")
	}

	if !errors.Is(err, cmd.ErrInvalidFormatChoice) {
		t.Errorf("expected ErrInvalidFormatChoice, got %q", err.Error())
	}
}

func TestInit_InvalidIndent(t *testing.T) {
	dir := t.TempDir()
	input := "1\nabc\n"

	var out bytes.Buffer

	err := cmd.RunInitWith(dir, strings.NewReader(input), &out)
	if err == nil {
		t.Fatal("expected error for invalid indent")
	}

	if !errors.Is(err, cmd.ErrInvalidIndentWidth) {
		t.Errorf("expected ErrInvalidIndentWidth, got %q", err.Error())
	}
}

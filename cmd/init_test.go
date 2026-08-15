package cmd_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Eagle-Konbu/sanat/cmd"
	"github.com/Eagle-Konbu/sanat/internal/config"
)

const sanatYML = ".sanat.yml"

var errNoMoreAnswers = errors.New("no more answers")

type fakePrompter struct {
	format  config.Format
	indent  int
	answers []bool
	idx     int
}

func (f *fakePrompter) SelectFormat() (config.Format, error) {
	return f.format, nil
}

func (f *fakePrompter) PromptIndent() (int, error) {
	return f.indent, nil
}

func (f *fakePrompter) PromptYesNo(_ string, _ bool) (bool, error) {
	if f.idx >= len(f.answers) {
		return false, fmt.Errorf("fakePrompter: %w at index %d", errNoMoreAnswers, f.idx)
	}

	ans := f.answers[f.idx]
	f.idx++

	return ans, nil
}

func assertConfig(t *testing.T, cfg config.Config, indent int, newline bool) {
	t.Helper()

	if cfg.Indent == nil || *cfg.Indent != indent {
		t.Errorf("indent: got %v, want %d", cfg.Indent, indent)
	}

	if cfg.Newline == nil || *cfg.Newline != newline {
		t.Errorf("newline: got %v, want %v", cfg.Newline, newline)
	}
}

func TestInit_CreateConfig(t *testing.T) {
	tests := []struct {
		name    string
		prompt  *fakePrompter
		file    string
		indent  int
		newline bool
		wantMsg string
	}{
		{
			name:    "yaml with custom values",
			prompt:  &fakePrompter{format: config.FormatYAML, indent: 4, answers: []bool{false}},
			file:    sanatYML,
			indent:  4,
			newline: false,
			wantMsg: "Created " + sanatYML,
		},
		{
			name:    "toml with custom values",
			prompt:  &fakePrompter{format: config.FormatTOML, indent: 8, answers: []bool{true}},
			file:    ".sanat.toml",
			indent:  8,
			newline: true,
			wantMsg: "Created .sanat.toml",
		},
		{
			name:    "all defaults",
			prompt:  &fakePrompter{format: config.FormatYAML, indent: 2, answers: []bool{true}},
			file:    sanatYML,
			indent:  2,
			newline: true,
			wantMsg: "Created " + sanatYML,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			var out bytes.Buffer

			if err := cmd.RunInitWith(dir, tt.prompt, &out); err != nil {
				t.Fatal(err)
			}

			path := filepath.Join(dir, tt.file)

			cfg, err := config.LoadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			assertConfig(t, cfg, tt.indent, tt.newline)

			if !strings.Contains(out.String(), tt.wantMsg) {
				t.Errorf("expected %q in output, got %q", tt.wantMsg, out.String())
			}
		})
	}
}

func TestInit_ExistingFile_Overwrite(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, sanatYML), []byte("indent: 2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	p := &fakePrompter{format: config.FormatYAML, indent: 4, answers: []bool{true, true}}

	var out bytes.Buffer

	if err := cmd.RunInitWith(dir, p, &out); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadFile(filepath.Join(dir, sanatYML))
	if err != nil {
		t.Fatal(err)
	}

	assertConfig(t, cfg, 4, true)

	if !strings.Contains(out.String(), "Created "+sanatYML) {
		t.Errorf("expected 'Created' message, got %q", out.String())
	}
}

func TestInit_ExistingFile_NoOverwrite(t *testing.T) {
	dir := t.TempDir()

	original := []byte("indent: 2\n")
	path := filepath.Join(dir, sanatYML)

	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatal(err)
	}

	p := &fakePrompter{format: config.FormatYAML, indent: 4, answers: []bool{false}}

	var out bytes.Buffer

	if err := cmd.RunInitWith(dir, p, &out); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(got, original) {
		t.Errorf("file was modified: got %q, want %q", got, original)
	}

	if out.String() != "" {
		t.Errorf("expected no output, got %q", out.String())
	}
}

func TestInit_ExistingFile_DifferentFormat(t *testing.T) {
	dir := t.TempDir()

	oldPath := filepath.Join(dir, sanatYML)
	if err := os.WriteFile(oldPath, []byte("indent: 2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	p := &fakePrompter{format: config.FormatTOML, indent: 4, answers: []bool{true, true}}

	var out bytes.Buffer

	if err := cmd.RunInitWith(dir, p, &out); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("old config file should have been removed")
	}

	newPath := filepath.Join(dir, ".sanat.toml")

	cfg, err := config.LoadFile(newPath)
	if err != nil {
		t.Fatal(err)
	}

	assertConfig(t, cfg, 4, true)

	if !strings.Contains(out.String(), "Created .sanat.toml") {
		t.Errorf("expected 'Created .sanat.toml' message, got %q", out.String())
	}
}

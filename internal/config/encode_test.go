package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/config"
)

func ptr[T any](v T) *T { return &v }

func assertField[T comparable](t *testing.T, name string, got *T, want *T) {
	t.Helper()

	if want == nil {
		return
	}

	if got == nil || *got != *want {
		t.Errorf("%s: got %v, want %v", name, got, *want)
	}
}

func TestEncode_RoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		cfg    config.Config
		format config.Format
	}{
		{
			name:   "yaml",
			cfg:    config.Config{Write: ptr(true), Indent: ptr(4), Newline: ptr(false)},
			format: config.FormatYAML,
		},
		{
			name:   "toml",
			cfg:    config.Config{Write: ptr(false), Indent: ptr(8), Newline: ptr(true)},
			format: config.FormatTOML,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := config.Encode(tt.cfg, tt.format)
			if err != nil {
				t.Fatal(err)
			}

			dir := t.TempDir()
			path := filepath.Join(dir, tt.format.Filename())

			if err := os.WriteFile(path, data, 0644); err != nil {
				t.Fatal(err)
			}

			got, err := config.LoadFile(path)
			if err != nil {
				t.Fatal(err)
			}

			assertField(t, "write", got.Write, tt.cfg.Write)
			assertField(t, "indent", got.Indent, tt.cfg.Indent)
			assertField(t, "newline", got.Newline, tt.cfg.Newline)
		})
	}
}

func TestFilename(t *testing.T) {
	tests := []struct {
		format config.Format
		want   string
	}{
		{config.FormatYAML, ".sanat.yml"},
		{config.FormatTOML, ".sanat.toml"},
	}

	for _, tt := range tests {
		if got := tt.format.Filename(); got != tt.want {
			t.Errorf("Format(%q).Filename() = %q, want %q", tt.format, got, tt.want)
		}
	}
}

func TestExists_Found(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, ".sanat.yml"), []byte("indent: 2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	name, ok := config.Exists(dir)
	if !ok {
		t.Fatal("expected config file to be found")
	}

	if name != ".sanat.yml" {
		t.Errorf("got %q, want .sanat.yml", name)
	}
}

func TestExists_NotFound(t *testing.T) {
	dir := t.TempDir()

	name, ok := config.Exists(dir)
	if ok {
		t.Errorf("expected no config file, found %q", name)
	}
}

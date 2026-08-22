package config_test

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/config"
)

func TestLoad_YAML(t *testing.T) {
	dir := t.TempDir()

	content := "indent: 4\nnewline: false\n"

	if err := os.WriteFile(filepath.Join(dir, ".sanat.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Indent == nil || *cfg.Indent != 4 {
		t.Errorf("indent: got %v, want 4", cfg.Indent)
	}

	if cfg.Newline == nil || *cfg.Newline != false {
		t.Errorf("newline: got %v, want false", cfg.Newline)
	}

	if cfg.Write != nil {
		t.Errorf("write: got %v, want nil", cfg.Write)
	}
}

func TestLoad_TOML(t *testing.T) {
	dir := t.TempDir()

	content := "indent = 8\nwrite = true\n"

	if err := os.WriteFile(filepath.Join(dir, ".sanat.toml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Indent == nil || *cfg.Indent != 8 {
		t.Errorf("indent: got %v, want 8", cfg.Indent)
	}

	if cfg.Write == nil || *cfg.Write != true {
		t.Errorf("write: got %v, want true", cfg.Write)
	}
}

func TestLoad_NoFile(t *testing.T) {
	dir := t.TempDir()

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Indent != nil || cfg.Newline != nil || cfg.Write != nil {
		t.Errorf("expected zero config, got %+v", cfg)
	}
}

func TestLoad_YAMLPriority(t *testing.T) {
	dir := t.TempDir()
	ymlContent := "indent: 2\n"
	tomlContent := "indent = 4\n"

	if err := os.WriteFile(filepath.Join(dir, ".sanat.yml"), []byte(ymlContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dir, ".sanat.toml"), []byte(tomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Indent == nil || *cfg.Indent != 2 {
		t.Errorf("indent: got %v, want 2 (yml should take priority)", cfg.Indent)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".sanat.yml"), []byte(":\ninvalid: ["), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(dir)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadFile_YAML(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "custom.yml")
	if err := os.WriteFile(path, []byte("indent: 6\nwrite: true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Indent == nil || *cfg.Indent != 6 {
		t.Errorf("indent: got %v, want 6", cfg.Indent)
	}

	if cfg.Write == nil || *cfg.Write != true {
		t.Errorf("write: got %v, want true", cfg.Write)
	}
}

func TestLoadFile_TOML(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "custom.toml")
	if err := os.WriteFile(path, []byte("indent = 3\nnewline = false\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Indent == nil || *cfg.Indent != 3 {
		t.Errorf("indent: got %v, want 3", cfg.Indent)
	}

	if cfg.Newline == nil || *cfg.Newline != false {
		t.Errorf("newline: got %v, want false", cfg.Newline)
	}
}

func TestLoadFile_NotFound(t *testing.T) {
	_, err := config.LoadFile("/nonexistent/path/config.yml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoad_PermissionError(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, ".sanat.yml")
	if err := os.WriteFile(path, []byte("indent: 2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Chmod(path, 0000); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(dir)
	if err == nil {
		t.Error("expected error for unreadable file")
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".sanat.toml"), []byte("= invalid"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(dir)
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

func TestLoad_Version_Supported(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".sanat.yml"), []byte("version: 1\nindent: 4\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Version == nil || *cfg.Version != 1 {
		t.Errorf("version: got %v, want 1", cfg.Version)
	}
}

func TestLoad_Version_Unsupported(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".sanat.yml"), []byte("version: 99\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := config.Load(dir)
	if !errors.Is(err, config.ErrUnsupportedVersion) {
		t.Errorf("got %v, want ErrUnsupportedVersion", err)
	}
}

func TestLoad_Indent_Invalid(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"zero", "indent: 0\n"},
		{"negative", "indent: -1\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, ".sanat.yml"), []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			_, err := config.Load(dir)
			if !errors.Is(err, config.ErrInvalidIndent) {
				t.Errorf("got %v, want ErrInvalidIndent", err)
			}
		})
	}
}

// assertValidatedStringField exercises a config field that's validated
// against a fixed set of valid string values: every value in valid must load
// successfully and populate the field (read via get), and an unrecognized
// value must fail loading with wantErr.
func assertValidatedStringField(
	t *testing.T,
	field string,
	valid []string,
	get func(config.Config) *string,
	wantErr error,
) {
	t.Helper()

	for _, v := range valid {
		t.Run(v, func(t *testing.T) {
			dir := t.TempDir()
			content := field + ": " + v + "\n"

			if err := os.WriteFile(filepath.Join(dir, ".sanat.yml"), []byte(content), 0644); err != nil {
				t.Fatal(err)
			}

			cfg, err := config.Load(dir)
			if err != nil {
				t.Fatal(err)
			}

			if got := get(cfg); got == nil || *got != v {
				t.Errorf("%s: got %v, want %q", field, got, v)
			}
		})
	}

	t.Run("invalid", func(t *testing.T) {
		dir := t.TempDir()
		content := field + ": sideways\n"

		if err := os.WriteFile(filepath.Join(dir, ".sanat.yml"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		_, err := config.Load(dir)
		if !errors.Is(err, wantErr) {
			t.Errorf("got %v, want %v", err, wantErr)
		}
	})
}

func TestLoad_KeywordCase(t *testing.T) {
	valid := []string{config.KeywordCaseUpper, config.KeywordCaseLower, config.KeywordCasePreserve}
	get := func(cfg config.Config) *string { return cfg.KeywordCase }

	assertValidatedStringField(t, "keyword_case", valid, get, config.ErrInvalidKeywordCase)
}

func TestLoad_CommaStyle(t *testing.T) {
	valid := []string{config.CommaStyleTrailing, config.CommaStyleLeading}
	get := func(cfg config.Config) *string { return cfg.CommaStyle }

	assertValidatedStringField(t, "comma_style", valid, get, config.ErrInvalidCommaStyle)
}

func TestLoad_SQLMode(t *testing.T) {
	valid := []string{config.SQLModeDefault, config.SQLModeNoBackslashEscapes}
	get := func(cfg config.Config) *string { return cfg.SQLMode }

	assertValidatedStringField(t, "sql_mode", valid, get, config.ErrInvalidSQLMode)
}

func TestLoad_UnknownField_WarnsButSucceeds(t *testing.T) {
	dir := t.TempDir()
	content := "version: 1\nindent: 4\nnot_a_real_field: true\n"

	if err := os.WriteFile(filepath.Join(dir, ".sanat.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	stderr := captureStderr(t, func() {
		cfg, err := config.Load(dir)
		if err != nil {
			t.Fatal(err)
		}

		if cfg.Indent == nil || *cfg.Indent != 4 {
			t.Errorf("indent: got %v, want 4", cfg.Indent)
		}
	})

	if !strings.Contains(stderr, "not_a_real_field") {
		t.Errorf("expected warning about unknown field in stderr, got %q", stderr)
	}
}

func TestLoad_MissingVersion_Warns(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".sanat.yml"), []byte("indent: 4\n"), 0644); err != nil {
		t.Fatal(err)
	}

	stderr := captureStderr(t, func() {
		if _, err := config.Load(dir); err != nil {
			t.Fatal(err)
		}
	})

	if !strings.Contains(stderr, "version") {
		t.Errorf("expected deprecation warning about missing version in stderr, got %q", stderr)
	}
}

func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stderr

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	os.Stderr = w

	t.Cleanup(func() {
		os.Stderr = orig

		w.Close()
		r.Close()
	})

	fn()

	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	os.Stderr = orig

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	return string(out)
}

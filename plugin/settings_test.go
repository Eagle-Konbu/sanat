package plugin_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/golangci/plugin-module-register/register"

	"github.com/Eagle-Konbu/sanat/internal/config"
	"github.com/Eagle-Konbu/sanat/internal/gofile"
	"github.com/Eagle-Konbu/sanat/plugin"
)

func defaultOptions() gofile.Options {
	return gofile.Options{
		Indent:      2,
		Newline:     true,
		KeywordCase: config.KeywordCaseUpper,
		CommaStyle:  config.CommaStyleTrailing,
		SQLMode:     config.SQLModeDefault,
		Dialect:     config.DialectMySQL,
	}
}

func writeConfigFile(t *testing.T, name, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func resolveOptions(t *testing.T, rawSettings any) gofile.Options {
	t.Helper()

	settings, err := register.DecodeSettings[plugin.Settings](rawSettings)
	if err != nil {
		t.Fatalf("DecodeSettings() error = %v", err)
	}

	opts, err := settings.Options()
	if err != nil {
		t.Fatalf("Options() error = %v", err)
	}

	return opts
}

func TestSettings_DefaultsMatchCLI(t *testing.T) {
	for name, raw := range map[string]any{
		"nil":       nil,
		"empty map": map[string]any{},
	} {
		t.Run(name, func(t *testing.T) {
			if got := resolveOptions(t, raw); got != defaultOptions() {
				t.Errorf("got %+v, want %+v", got, defaultOptions())
			}
		})
	}
}

func TestSettings_ExplicitValues(t *testing.T) {
	got := resolveOptions(t, map[string]any{
		"indent":       4,
		"newline":      false,
		"keyword-case": "lower",
		"comma-style":  "leading",
		"sql-mode":     "no_backslash_escapes",
		"dialect":      "postgresql",
	})

	want := gofile.Options{
		Indent:      4,
		Newline:     false,
		KeywordCase: config.KeywordCaseLower,
		CommaStyle:  config.CommaStyleLeading,
		SQLMode:     config.SQLModeNoBackslashEscapes,
		Dialect:     config.DialectPostgreSQL,
	}

	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestSettings_ConfigFileFillsUnsetKeys(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
	}{
		{".sanat.yml", "version: 1\nindent: 4\ncomma_style: leading\ndialect: postgresql\n"},
		{".sanat.toml", "version = 1\nindent = 4\ncomma_style = \"leading\"\ndialect = \"postgresql\"\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeConfigFile(t, tc.name, tc.content)

			got := resolveOptions(t, map[string]any{"config": path})

			want := defaultOptions()
			want.Indent = 4
			want.CommaStyle = config.CommaStyleLeading
			want.Dialect = config.DialectPostgreSQL

			if got != want {
				t.Errorf("got %+v, want %+v", got, want)
			}
		})
	}
}

func TestSettings_ExplicitValuesOverrideConfigFile(t *testing.T) {
	path := writeConfigFile(t, ".sanat.yml", "version: 1\nindent: 4\nnewline: false\nkeyword_case: lower\n")

	got := resolveOptions(t, map[string]any{
		"config":  path,
		"indent":  8,
		"newline": true,
	})

	want := defaultOptions()
	want.Indent = 8
	want.Newline = true
	want.KeywordCase = config.KeywordCaseLower

	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestNew_RejectsInvalidSettings(t *testing.T) {
	for _, tc := range []struct {
		name    string
		raw     map[string]any
		wantErr error
	}{
		{"zero indent", map[string]any{"indent": 0}, config.ErrInvalidIndent},
		{"negative indent", map[string]any{"indent": -1}, config.ErrInvalidIndent},
		{"keyword-case", map[string]any{"keyword-case": "camel"}, config.ErrInvalidKeywordCase},
		{"comma-style", map[string]any{"comma-style": "middle"}, config.ErrInvalidCommaStyle},
		{"sql-mode", map[string]any{"sql-mode": "ansi"}, config.ErrInvalidSQLMode},
		{"dialect", map[string]any{"dialect": "sqlite"}, config.ErrInvalidDialect},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := plugin.New(tc.raw)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("New() error = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestNew_RejectsMalformedSettings(t *testing.T) {
	for name, raw := range map[string]any{
		"unknown key":   map[string]any{"keyword_case": "lower"},
		"wrong type":    map[string]any{"indent": "four"},
		"not an object": "sanat",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := plugin.New(raw); err == nil {
				t.Error("New() error = nil, want error")
			}
		})
	}
}

func TestNew_RejectsInvalidConfigFile(t *testing.T) {
	invalidValue := writeConfigFile(t, ".sanat.yml", "version: 1\ncomma_style: middle\n")

	if _, err := plugin.New(map[string]any{"config": invalidValue}); !errors.Is(err, config.ErrInvalidCommaStyle) {
		t.Errorf("New() error = %v, want %v", err, config.ErrInvalidCommaStyle)
	}

	missing := filepath.Join(t.TempDir(), ".sanat.yml")

	if _, err := plugin.New(map[string]any{"config": missing}); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("New() error = %v, want %v", err, os.ErrNotExist)
	}
}

func TestNew_ExplicitSettingDoesNotMaskInvalidConfigFile(t *testing.T) {
	path := writeConfigFile(t, ".sanat.yml", "version: 1\nindent: 0\n")

	if _, err := plugin.New(map[string]any{"config": path, "indent": 4}); !errors.Is(err, config.ErrInvalidIndent) {
		t.Errorf("New() error = %v, want %v", err, config.ErrInvalidIndent)
	}
}

package plugin

import (
	"fmt"

	"github.com/Eagle-Konbu/sanat/internal/config"
	"github.com/Eagle-Konbu/sanat/internal/gofile"
)

// Settings mirrors `linters.settings.custom.sanat.settings` in .golangci.yml.
// Keys use kebab-case to match the CLI flags. Pointer fields distinguish an
// omitted key from an explicit zero value so that config file values and
// defaults only fill keys the user left out.
//
//nolint:tagliatelle // golangci-lint settings keys follow the CLI flag names
type Settings struct {
	// Config is the path to a .sanat.yml/.sanat.toml file. A relative path is
	// resolved against the working directory of the golangci-lint process.
	Config      string  `json:"config"`
	Indent      *int    `json:"indent"`
	Newline     *bool   `json:"newline"`
	KeywordCase *string `json:"keyword-case"`
	CommaStyle  *string `json:"comma-style"`
	SQLMode     *string `json:"sql-mode"`
}

// Options resolves each formatter option with the precedence: explicit
// setting, then the referenced config file, then the CLI defaults.
func (s Settings) Options() (gofile.Options, error) {
	fileCfg, err := s.loadConfigFile()
	if err != nil {
		return gofile.Options{}, err
	}

	defaults := config.Config{
		Indent:      ptr(config.DefaultIndent),
		Newline:     ptr(config.DefaultNewline),
		KeywordCase: ptr(config.DefaultKeywordCase),
		CommaStyle:  ptr(config.DefaultCommaStyle),
		SQLMode:     ptr(config.DefaultSQLMode),
	}

	merged := config.Config{
		Indent:      firstSet(s.Indent, fileCfg.Indent, defaults.Indent),
		Newline:     firstSet(s.Newline, fileCfg.Newline, defaults.Newline),
		KeywordCase: firstSet(s.KeywordCase, fileCfg.KeywordCase, defaults.KeywordCase),
		CommaStyle:  firstSet(s.CommaStyle, fileCfg.CommaStyle, defaults.CommaStyle),
		SQLMode:     firstSet(s.SQLMode, fileCfg.SQLMode, defaults.SQLMode),
	}

	if err := config.Validate(merged); err != nil {
		return gofile.Options{}, fmt.Errorf("sanat: invalid settings: %w", err)
	}

	return gofile.Options{
		Indent:      *merged.Indent,
		Newline:     *merged.Newline,
		KeywordCase: *merged.KeywordCase,
		CommaStyle:  *merged.CommaStyle,
		SQLMode:     *merged.SQLMode,
	}, nil
}

// loadConfigFile reads the config file only when one is referenced
// explicitly; .sanat files are never discovered implicitly because the
// directory golangci-lint runs from is not necessarily the project root.
func (s Settings) loadConfigFile() (config.Config, error) {
	if s.Config == "" {
		return config.Config{}, nil
	}

	cfg, err := config.LoadFile(s.Config)
	if err != nil {
		return config.Config{}, fmt.Errorf("sanat: loading config %q: %w", s.Config, err)
	}

	return cfg, nil
}

func firstSet[T any](values ...*T) *T {
	for _, v := range values {
		if v != nil {
			return v
		}
	}

	return nil
}

func ptr[T any](v T) *T {
	return &v
}

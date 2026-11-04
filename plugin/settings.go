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
	Dialect     *string `json:"dialect"`
}

// Options resolves each formatter option with the precedence: explicit
// setting, then the referenced config file, then the CLI defaults.
func (s Settings) Options() (gofile.Options, error) {
	fileCfg, err := s.loadConfigFile()
	if err != nil {
		return gofile.Options{}, err
	}

	opts := gofile.Options{
		Indent:      firstSet(s.Indent, fileCfg.Indent, config.DefaultIndent),
		Newline:     firstSet(s.Newline, fileCfg.Newline, config.DefaultNewline),
		KeywordCase: firstSet(s.KeywordCase, fileCfg.KeywordCase, config.DefaultKeywordCase),
		CommaStyle:  firstSet(s.CommaStyle, fileCfg.CommaStyle, config.DefaultCommaStyle),
		SQLMode:     firstSet(s.SQLMode, fileCfg.SQLMode, config.DefaultSQLMode),
		Dialect:     firstSet(s.Dialect, fileCfg.Dialect, config.DefaultDialect),
	}

	resolved := config.Config{
		Indent:      &opts.Indent,
		Newline:     &opts.Newline,
		KeywordCase: &opts.KeywordCase,
		CommaStyle:  &opts.CommaStyle,
		SQLMode:     &opts.SQLMode,
		Dialect:     &opts.Dialect,
	}
	if err := config.Validate(resolved); err != nil {
		return gofile.Options{}, fmt.Errorf("sanat: invalid settings: %w", err)
	}

	return opts, nil
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

//nolint:ireturn // returns the caller's type argument, not an interface
func firstSet[T any](explicit, fromFile *T, fallback T) T {
	if explicit != nil {
		return *explicit
	}

	if fromFile != nil {
		return *fromFile
	}

	return fallback
}

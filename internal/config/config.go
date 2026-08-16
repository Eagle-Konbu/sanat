package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// CurrentVersion is the latest supported config schema version.
const CurrentVersion = 1

const (
	KeywordCaseUpper    = "upper"
	KeywordCaseLower    = "lower"
	KeywordCasePreserve = "preserve"

	CommaStyleTrailing = "trailing"
	CommaStyleLeading  = "leading"
)

var (
	ErrUnsupportedVersion = errors.New("unsupported config version")
	ErrInvalidIndent      = errors.New("indent must be a positive integer")
	ErrInvalidKeywordCase = errors.New("keyword_case must be one of: upper, lower, preserve")
	ErrInvalidCommaStyle  = errors.New("comma_style must be one of: trailing, leading")
)

var knownFields = map[string]bool{
	"version":      true,
	"write":        true,
	"indent":       true,
	"newline":      true,
	"keyword_case": true,
	"comma_style":  true,
}

type Config struct {
	Version     *int    `toml:"version,omitempty"      yaml:"version,omitempty"`
	Write       *bool   `toml:"write,omitempty"        yaml:"write,omitempty"`
	Indent      *int    `toml:"indent,omitempty"       yaml:"indent,omitempty"`
	Newline     *bool   `toml:"newline,omitempty"      yaml:"newline,omitempty"`
	KeywordCase *string `toml:"keyword_case,omitempty" yaml:"keyword_case,omitempty"`
	CommaStyle  *string `toml:"comma_style,omitempty"  yaml:"comma_style,omitempty"`
}

var configFiles = []string{
	".sanat.yml",
	".sanat.yaml",
	".sanat.toml",
}

// LoadFile reads and decodes the config file at the given path.
func LoadFile(path string) (Config, error) {
	cleanPath := filepath.Clean(path)

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return Config{}, err
	}

	return decode(filepath.Base(cleanPath), data)
}

// Load searches for a config file in the given directory and decodes it.
// Returns a zero Config and nil error if no config file is found.
func Load(dir string) (Config, error) {
	for _, name := range configFiles {
		path := filepath.Clean(filepath.Join(dir, name))

		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}

			return Config{}, err
		}

		return decode(name, data)
	}

	return Config{}, nil
}

func decode(name string, data []byte) (Config, error) {
	var cfg Config

	switch filepath.Ext(name) {
	case ".toml":
		if err := toml.Unmarshal(data, &cfg); err != nil {
			return Config{}, err
		}
	default:
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, err
		}
	}

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	warn(name, data, cfg)

	return cfg, nil
}

func validate(cfg Config) error {
	for _, check := range []func(Config) error{
		validateVersion,
		validateIndent,
		validateKeywordCase,
		validateCommaStyle,
	} {
		if err := check(cfg); err != nil {
			return err
		}
	}

	return nil
}

func validateVersion(cfg Config) error {
	if cfg.Version != nil && *cfg.Version != CurrentVersion {
		return fmt.Errorf("%w: %d (supported: %d)", ErrUnsupportedVersion, *cfg.Version, CurrentVersion)
	}

	return nil
}

func validateIndent(cfg Config) error {
	if cfg.Indent != nil && *cfg.Indent <= 0 {
		return ErrInvalidIndent
	}

	return nil
}

func validateKeywordCase(cfg Config) error {
	if cfg.KeywordCase == nil {
		return nil
	}

	switch *cfg.KeywordCase {
	case KeywordCaseUpper, KeywordCaseLower, KeywordCasePreserve:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidKeywordCase, *cfg.KeywordCase)
	}
}

func validateCommaStyle(cfg Config) error {
	if cfg.CommaStyle == nil {
		return nil
	}

	switch *cfg.CommaStyle {
	case CommaStyleTrailing, CommaStyleLeading:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidCommaStyle, *cfg.CommaStyle)
	}
}

// warn prints deprecation and forward-compatibility warnings for the decoded
// config file. Validation errors are handled separately by validate; warn
// only reports conditions that should not block loading the config.
func warn(name string, data []byte, cfg Config) {
	if cfg.Version == nil {
		versionHint := fmt.Sprintf("version = %d", CurrentVersion)
		if ext := filepath.Ext(name); ext == ".yml" || ext == ".yaml" {
			versionHint = fmt.Sprintf("version: %d", CurrentVersion)
		}

		fmt.Fprintf(os.Stderr,
			"sanat: warning: %s is missing \"version\"; treating as version 0 (pre-schema). "+
				"Add \"%s\" to silence this warning.\n", name, versionHint)
	}

	for _, field := range unknownFields(name, data) {
		fmt.Fprintf(os.Stderr, "sanat: warning: %s: unknown config field %q\n", name, field)
	}
}

func unknownFields(name string, data []byte) []string {
	raw := map[string]any{}

	switch filepath.Ext(name) {
	case ".toml":
		if err := toml.Unmarshal(data, &raw); err != nil {
			return nil
		}
	default:
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil
		}
	}

	var unknown []string

	for k := range raw {
		if !knownFields[k] {
			unknown = append(unknown, k)
		}
	}

	sort.Strings(unknown)

	return unknown
}

package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Write   *bool `toml:"write"   yaml:"write"`
	Indent  *int  `toml:"indent"  yaml:"indent"`
	Newline *bool `toml:"newline" yaml:"newline"`
}

var configFiles = []string{
	".sanat.yml",
	".sanat.yaml",
	".sanat.toml",
}

// LoadFile reads and decodes the config file at the given path.
func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	return decode(filepath.Base(path), data)
}

// Load searches for a config file in the given directory and decodes it.
// Returns a zero Config and nil error if no config file is found.
func Load(dir string) (Config, error) {
	for _, name := range configFiles {
		path := filepath.Join(dir, name)

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

	return cfg, nil
}

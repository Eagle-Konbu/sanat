package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// Format represents the config file format.
type Format string

const (
	FormatYAML Format = "yaml"
	FormatTOML Format = "toml"
)

// Filename returns the config file name for the format.
func (f Format) Filename() string {
	switch f {
	case FormatTOML:
		return ".sanat.toml"
	default:
		return ".sanat.yml"
	}
}

// Encode serializes the Config into the given format.
func Encode(cfg Config, format Format) ([]byte, error) {
	switch format {
	case FormatTOML:
		return toml.Marshal(cfg)
	default:
		return yaml.Marshal(cfg)
	}
}

// Exists checks whether a config file already exists in dir.
// Returns the filename and true if found, or "" and false otherwise.
func Exists(dir string) (string, bool) {
	for _, name := range configFiles {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return name, true
		}
	}

	return "", false
}

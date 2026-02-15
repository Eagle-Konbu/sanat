package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_YAML(t *testing.T) {
	dir := t.TempDir()
	content := "indent: 4\nnewline: false\n"
	if err := os.WriteFile(filepath.Join(dir, ".sanat.yml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
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

	cfg, err := Load(dir)
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
	cfg, err := Load(dir)
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

	cfg, err := Load(dir)
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
	_, err := Load(dir)
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

	cfg, err := LoadFile(path)
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

	cfg, err := LoadFile(path)
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
	_, err := LoadFile("/nonexistent/path/config.yml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".sanat.toml"), []byte("= invalid"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

// Package plugin registers the sanat analyzer with the golangci-lint v2
// Module Plugin System. It is compiled into a custom golangci-lint binary
// built with `golangci-lint custom`; the stock binary cannot load it.
package plugin

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/Eagle-Konbu/sanat/internal/analyzer"
	"github.com/Eagle-Konbu/sanat/internal/gofile"
)

func init() {
	register.Plugin("sanat", New)
}

type sanatPlugin struct {
	opts gofile.Options
}

// New decodes the golangci-lint settings for sanat and returns the plugin.
// Invalid settings or an unreadable config file are reported here so that
// golangci-lint fails at startup instead of silently linting with defaults.
//
//nolint:ireturn // register.NewPlugin requires returning the LinterPlugin interface
func New(rawSettings any) (register.LinterPlugin, error) {
	settings, err := register.DecodeSettings[Settings](rawSettings)
	if err != nil {
		return nil, err
	}

	opts, err := settings.Options()
	if err != nil {
		return nil, err
	}

	return &sanatPlugin{opts: opts}, nil
}

func (p *sanatPlugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{analyzer.New(p.opts)}, nil
}

// GetLoadMode returns syntax mode because the analyzer only inspects the AST.
func (p *sanatPlugin) GetLoadMode() string {
	return register.LoadModeSyntax
}

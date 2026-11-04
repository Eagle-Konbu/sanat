package plugin_test

import (
	"testing"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis/analysistest"

	"github.com/Eagle-Konbu/sanat/plugin"
)

//nolint:ireturn // mirrors plugin.New
func newPlugin(t *testing.T, rawSettings any) register.LinterPlugin {
	t.Helper()

	p, err := plugin.New(rawSettings)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	return p
}

func TestPlugin_IsRegisteredAsSanat(t *testing.T) {
	constructor, err := register.GetPlugin("sanat")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := constructor(nil); err != nil {
		t.Fatalf("registered constructor error = %v", err)
	}
}

func TestPlugin_UsesSyntaxLoadMode(t *testing.T) {
	p := newPlugin(t, nil)

	if got := p.GetLoadMode(); got != register.LoadModeSyntax {
		t.Errorf("GetLoadMode() = %q, want %q", got, register.LoadModeSyntax)
	}
}

func TestPlugin_ReportsWithDefaultSettings(t *testing.T) {
	p := newPlugin(t, nil)

	analyzers, err := p.BuildAnalyzers()
	if err != nil {
		t.Fatal(err)
	}

	if len(analyzers) != 1 || analyzers[0].Name != "sanat" {
		t.Fatalf("BuildAnalyzers() = %v, want a single sanat analyzer", analyzers)
	}

	analysistest.Run(t, analysistest.TestData(), analyzers[0], "defaults")
}

func TestPlugin_AppliesSettingsToAnalyzer(t *testing.T) {
	// golangci-lint passes settings as a generic map decoded from YAML.
	p := newPlugin(t, map[string]any{
		"newline":     false,
		"comma-style": "leading",
	})

	analyzers, err := p.BuildAnalyzers()
	if err != nil {
		t.Fatal(err)
	}

	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), analyzers[0], "settings")
}

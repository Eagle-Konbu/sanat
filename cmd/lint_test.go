package cmd_test

import (
	"bytes"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Eagle-Konbu/sanat/cmd"
)

const lintGoMod = "module example.com/fixture\n\ngo 1.23\n"

const formattedSource = "package clean\n\n" +
	"const query = `\nSELECT\n  id,\n  name\nFROM\n  users\n`\n\n" +
	"const greeting = `hello, world`\n"

const unformattedSource = "package dirty\n\n" +
	"const query = `select id, name from users`\n"

var errBoom = errors.New("boom")

type lintResult struct {
	stdout string
	stderr string
	err    error
}

func (r lintResult) exitCode() int {
	return cmd.ExitCode(r.err)
}

// writeModule creates a Go module from files (relative path -> content) in a
// temporary directory and makes it the working directory.
func writeModule(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	files = maps.Clone(files)
	files["go.mod"] = lintGoMod

	for name, content := range files {
		path := filepath.Join(dir, name)

		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	t.Chdir(dir)

	return dir
}

func runLint(t *testing.T, args ...string) lintResult {
	t.Helper()

	var stdout, stderr bytes.Buffer

	err := cmd.ExecuteArgs(append([]string{"lint"}, args...), &stdout, &stderr)

	return lintResult{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func assertExitCode(t *testing.T, got lintResult, want int) {
	t.Helper()

	if code := got.exitCode(); code != want {
		t.Fatalf("exit code: got %d, want %d\nerr: %v\nstdout:\n%s\nstderr:\n%s",
			code, want, got.err, got.stdout, got.stderr)
	}
}

func assertStdout(t *testing.T, got lintResult, want string) {
	t.Helper()

	if got.stdout != want {
		t.Errorf("stdout:\ngot:\n%s\nwant:\n%s", got.stdout, want)
	}
}

func TestLint_CleanPackageExitsZero(t *testing.T) {
	writeModule(t, map[string]string{"clean.go": formattedSource})

	got := runLint(t)

	assertExitCode(t, got, 0)
	assertStdout(t, got, "")
}

func TestLint_UnformattedLiteralIsReported(t *testing.T) {
	writeModule(t, map[string]string{"dirty.go": unformattedSource})

	got := runLint(t)

	assertExitCode(t, got, 1)
	assertStdout(t, got, "dirty.go:3:15: SQL literal is not formatted (sanat)\n")

	if !strings.Contains(got.stderr, "1 unformatted SQL literal(s) found") {
		t.Errorf("stderr should summarize findings, got:\n%s", got.stderr)
	}
}

func TestLint_DoesNotModifyFiles(t *testing.T) {
	dir := writeModule(t, map[string]string{"dirty.go": unformattedSource})

	runLint(t)

	got, err := os.ReadFile(filepath.Join(dir, "dirty.go"))
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != unformattedSource {
		t.Errorf("file was modified:\n%s", got)
	}
}

func TestLint_FindingsAreSortedByPosition(t *testing.T) {
	writeModule(t, map[string]string{
		"b.go": "package dirty\n\nconst b = `select b from t`\n",
		"a.go": "package dirty\n\nconst a1 = `select a from t`\n\nconst a2 = `select a2 from t`\n",
	})

	got := runLint(t)

	assertExitCode(t, got, 1)
	assertStdout(t, got, ""+
		"a.go:3:12: SQL literal is not formatted (sanat)\n"+
		"a.go:5:12: SQL literal is not formatted (sanat)\n"+
		"b.go:3:11: SQL literal is not formatted (sanat)\n")
}

func TestLint_PackagePatterns(t *testing.T) {
	files := map[string]string{
		"clean.go":       formattedSource,
		"sub/dirty.go":   unformattedSource,
		"other/dirty.go": unformattedSource,
	}

	tests := []struct {
		name     string
		patterns []string
		want     string
		wantCode int
	}{
		{
			name:     "no pattern means current directory",
			patterns: nil,
			want:     "",
			wantCode: 0,
		},
		{
			name:     "single package",
			patterns: []string{"./sub"},
			want:     "sub/dirty.go:3:15: SQL literal is not formatted (sanat)\n",
			wantCode: 1,
		},
		{
			name:     "recursive pattern",
			patterns: []string{"./..."},
			want: "other/dirty.go:3:15: SQL literal is not formatted (sanat)\n" +
				"sub/dirty.go:3:15: SQL literal is not formatted (sanat)\n",
			wantCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeModule(t, files)

			got := runLint(t, tt.patterns...)

			assertExitCode(t, got, tt.wantCode)
			assertStdout(t, got, tt.want)
		})
	}
}

func TestLint_TestFiles(t *testing.T) {
	files := map[string]string{
		"dirty.go":      unformattedSource,
		"dirty_test.go": "package dirty\n\nconst testQuery = `select id from users`\n",
	}

	t.Run("included by default without duplicating non-test findings", func(t *testing.T) {
		writeModule(t, files)

		got := runLint(t)

		assertExitCode(t, got, 1)
		assertStdout(t, got, ""+
			"dirty.go:3:15: SQL literal is not formatted (sanat)\n"+
			"dirty_test.go:3:19: SQL literal is not formatted (sanat)\n")
	})

	t.Run("excluded with --tests=false", func(t *testing.T) {
		writeModule(t, files)

		got := runLint(t, "--tests=false")

		assertExitCode(t, got, 1)
		assertStdout(t, got, "dirty.go:3:15: SQL literal is not formatted (sanat)\n")
	})
}

func TestLint_FormattingOptions(t *testing.T) {
	lowerSource := "package lower\n\nconst query = `\nSELECT\n  id\nFROM\n  users\nWHERE\n  active = 1\n  and id > 2\n`\n"
	finding := "lower.go:3:15: SQL literal is not formatted (sanat)\n"

	tests := []struct {
		name   string
		config string
		args   []string
		want   string
	}{
		{
			name: "defaults report lowercase keywords",
			want: finding,
		},
		{
			name:   "config file is applied",
			config: "version: 1\nkeyword_case: lower\n",
			want:   "",
		},
		{
			name: "flag is applied",
			args: []string{"--keyword-case", "lower"},
			want: "",
		},
		{
			name:   "flag overrides config file",
			config: "version: 1\nkeyword_case: lower\n",
			args:   []string{"--keyword-case", "upper"},
			want:   finding,
		},
		{
			name:   "flag overrides one config value while others still apply",
			config: "version: 1\nkeyword_case: lower\nindent: 4\n",
			args:   []string{"--indent", "2"},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := map[string]string{"lower.go": lowerSource}
			if tt.config != "" {
				files[sanatYML] = tt.config
			}

			writeModule(t, files)

			got := runLint(t, tt.args...)

			assertStdout(t, got, tt.want)
		})
	}
}

func TestLint_FormattingFlagBeforeSubcommand(t *testing.T) {
	writeModule(t, map[string]string{
		"lower.go": "package lower\n\nconst query = `\nSELECT\n  id\nFROM\n  users\nWHERE\n  active = 1\n  and id > 2\n`\n",
	})

	var stdout, stderr bytes.Buffer

	err := cmd.ExecuteArgs([]string{"--keyword-case", "lower", "lint"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}
}

func TestLint_AnalysisFailures(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]string
		args       []string
		wantStderr string
	}{
		{
			name: "type error",
			files: map[string]string{
				"broken.go": "package broken\n\nvar x int = \"not an int\"\n",
			},
			wantStderr: "broken.go:3:13",
		},
		{
			name: "syntax error",
			files: map[string]string{
				"broken.go": "package broken\n\nfunc {\n",
			},
			wantStderr: "broken.go:3",
		},
		{
			name:       "nonexistent package",
			files:      map[string]string{"clean.go": formattedSource},
			args:       []string{"./missing"},
			wantStderr: "missing",
		},
		{
			name:       "invalid flag value",
			files:      map[string]string{"clean.go": formattedSource},
			args:       []string{"--keyword-case", "sideways"},
			wantStderr: "",
		},
		{
			name:       "invalid config file",
			files:      map[string]string{"clean.go": formattedSource, sanatYML: "version: 99\n"},
			wantStderr: "",
		},
		{
			name:       "unknown flag",
			files:      map[string]string{"clean.go": formattedSource},
			args:       []string{"--write"},
			wantStderr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeModule(t, tt.files)

			got := runLint(t, tt.args...)

			assertExitCode(t, got, 2)
			assertStdout(t, got, "")

			if !strings.Contains(got.stderr, tt.wantStderr) {
				t.Errorf("stderr should contain %q, got:\n%s", tt.wantStderr, got.stderr)
			}
		})
	}
}

func TestExitCode(t *testing.T) {
	if got := cmd.ExitCode(nil); got != 0 {
		t.Errorf("nil error: got %d, want 0", got)
	}

	if got := cmd.ExitCode(errBoom); got != 1 {
		t.Errorf("plain error: got %d, want 1", got)
	}
}

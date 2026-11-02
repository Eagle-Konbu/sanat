package cmd

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/checker"
	"golang.org/x/tools/go/packages"

	"github.com/Eagle-Konbu/sanat/internal/analyzer"
)

var (
	errNoPackages     = errors.New("matched no packages")
	errPackageErrors  = errors.New("packages contain errors")
	errAnalysisFailed = errors.New("analysis failed")
	errFindings       = errors.New("unformatted SQL literal(s) found")
)

var lintTestsFlag bool

func init() {
	rootCmd.AddCommand(newLintCmd())
}

func newLintCmd() *cobra.Command {
	lintCmd := &cobra.Command{
		Use:   "lint [flags] [pattern ...]",
		Short: "Report unformatted SQL literals in Go packages",
		Long: "Reports embedded SQL string literals that are not formatted, without modifying files.\n" +
			"Patterns are Go package patterns such as ./... and default to the current directory.\n\n" +
			"Exit status is 0 when no findings are reported, 1 when findings are reported,\n" +
			"and 2 when packages cannot be loaded or analyzed.",
		Args: cobra.ArbitraryArgs,
		RunE: runLint,
	}

	lintCmd.Flags().BoolVar(&lintTestsFlag, "tests", true, "analyze _test.go files")

	lintCmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return &exitError{code: exitCodeError, err: err}
	})

	return lintCmd
}

type lintFinding struct {
	position string
	message  string
	analyzer string
	filename string
	line     int
	column   int
}

func runLint(cmd *cobra.Command, args []string) error {
	if err := applyConfig(cmd); err != nil {
		return &exitError{code: exitCodeError, err: err}
	}

	patterns := args
	if len(patterns) == 0 {
		patterns = []string{"."}
	}

	findings, err := lintPackages(patterns, lintTestsFlag, cmd.ErrOrStderr())
	if err != nil {
		return &exitError{code: exitCodeError, err: err}
	}

	out := cmd.OutOrStdout()
	for _, f := range findings {
		if _, err := fmt.Fprintf(out, "%s: %s (%s)\n", f.position, f.message, f.analyzer); err != nil {
			return &exitError{code: exitCodeError, err: err}
		}
	}

	if len(findings) > 0 {
		return &exitError{code: exitCodeFindings, err: fmt.Errorf("%d %w", len(findings), errFindings)}
	}

	return nil
}

func lintPackages(patterns []string, includeTests bool, stderr io.Writer) ([]lintFinding, error) {
	pkgs, err := loadPackages(patterns, includeTests)
	if err != nil {
		return nil, err
	}

	if n := printPackageErrors(pkgs, stderr); n > 0 {
		return nil, fmt.Errorf("%w: %d error(s)", errPackageErrors, n)
	}

	graph, err := checker.Analyze([]*analysis.Analyzer{analyzer.New(opts())}, pkgs, nil)
	if err != nil {
		return nil, err
	}

	return collectFindings(graph)
}

func loadPackages(patterns []string, includeTests bool) ([]*packages.Package, error) {
	conf := &packages.Config{
		// The analyzer neither produces nor consumes facts, so only the
		// initial packages need syntax; dependencies come from export data.
		Mode:  packages.LoadSyntax | packages.NeedModule,
		Tests: includeTests,
	}

	pkgs, err := packages.Load(conf, patterns...)
	if err != nil {
		return nil, err
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("%s %w", strings.Join(patterns, " "), errNoPackages)
	}

	return pkgs, nil
}

// printPackageErrors reports load, parse, and type errors in the packages and
// their dependencies. Like go vet, sanat treats packages that do not compile as
// an analysis failure rather than silently skipping them.
func printPackageErrors(pkgs []*packages.Package, stderr io.Writer) int {
	seen := make(map[string]bool)

	packages.Visit(pkgs, nil, func(pkg *packages.Package) {
		for _, pkgErr := range pkg.Errors {
			msg := pkgErr.Error()
			if seen[msg] {
				continue
			}

			seen[msg] = true

			fmt.Fprintln(stderr, msg)
		}
	})

	return len(seen)
}

func collectFindings(graph *checker.Graph) ([]lintFinding, error) {
	baseDirs := workingDirs()

	// With tests enabled, a package's non-test files are analyzed both as the
	// package itself and as its test variant, so identical diagnostics are
	// merged by position.
	seen := make(map[string]bool)

	var findings []lintFinding

	for _, act := range graph.Roots {
		if act.Err != nil {
			return nil, fmt.Errorf("%w: %s: %w", errAnalysisFailed, act.Package.PkgPath, act.Err)
		}

		for _, diag := range act.Diagnostics {
			pos := act.Package.Fset.Position(diag.Pos)
			filename := relativePath(pos.Filename, baseDirs)
			position := fmt.Sprintf("%s:%d:%d", filename, pos.Line, pos.Column)

			if seen[position] {
				continue
			}

			seen[position] = true

			findings = append(findings, lintFinding{
				position: position,
				message:  diag.Message,
				analyzer: act.Analyzer.Name,
				filename: filename,
				line:     pos.Line,
				column:   pos.Column,
			})
		}
	}

	slices.SortFunc(findings, func(a, b lintFinding) int {
		return cmp.Or(
			cmp.Compare(a.filename, b.filename),
			cmp.Compare(a.line, b.line),
			cmp.Compare(a.column, b.column),
		)
	})

	return findings, nil
}

// workingDirs returns the working directory both as reported and with
// symlinks resolved, because the go command may report file names under
// either form (e.g. /var vs /private/var on macOS).
func workingDirs() []string {
	wd, err := os.Getwd()
	if err != nil {
		return nil
	}

	dirs := []string{wd}

	if resolved, err := filepath.EvalSymlinks(wd); err == nil && resolved != wd {
		dirs = append(dirs, resolved)
	}

	return dirs
}

func relativePath(filename string, baseDirs []string) string {
	for _, dir := range baseDirs {
		rel, err := filepath.Rel(dir, filename)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return rel
		}
	}

	return filename
}

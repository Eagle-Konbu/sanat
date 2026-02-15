package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Eagle-Konbu/sanat/internal/config"
	"github.com/Eagle-Konbu/sanat/internal/gofile"
)

var (
	writeFlag   bool
	indentFlag  int
	newlineFlag bool
)

var rootCmd = &cobra.Command{
	Use:               "sanat [flags] [pattern ...]",
	Short:             "Format SQL strings in Go source files",
	Long:              "Automatically formats embedded SQL string literals in Go source code.",
	RunE:              run,
	SilenceUsage:      true,
	CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
}

func SetVersion(v string) {
	rootCmd.Version = v
}

func init() {
	rootCmd.Flags().BoolVarP(&writeFlag, "write", "w", false, "overwrite files in place")
	rootCmd.Flags().IntVar(&indentFlag, "indent", 2, "indent width for SQL formatting")
	rootCmd.Flags().BoolVar(&newlineFlag, "newline", true, "add newline after opening backtick")
}

func Execute() error {
	return rootCmd.Execute()
}

func applyConfig(cmd *cobra.Command) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	cfg, err := config.Load(dir)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Config file values apply only when the flag was not explicitly set.
	if !cmd.Flags().Changed("write") && cfg.Write != nil {
		writeFlag = *cfg.Write
	}
	if !cmd.Flags().Changed("indent") && cfg.Indent != nil {
		indentFlag = *cfg.Indent
	}
	if !cmd.Flags().Changed("newline") && cfg.Newline != nil {
		newlineFlag = *cfg.Newline
	}
	return nil
}

func opts() gofile.Options {
	return gofile.Options{
		Indent:  indentFlag,
		Newline: newlineFlag,
	}
}

func run(cmd *cobra.Command, args []string) error {
	if err := applyConfig(cmd); err != nil {
		return err
	}

	if len(args) == 0 {
		return processStdin()
	}

	files, err := resolvePatterns(args)
	if err != nil {
		return err
	}

	for _, path := range files {
		if err := processFile(path); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
		}
	}
	return nil
}

func processStdin() error {
	src, err := io.ReadAll(os.Stdin)
	if err != nil {
		return err
	}
	file, fset, literals, err := gofile.FindSQLLiterals(src, "stdin.go")
	if err != nil {
		return err
	}
	out, err := gofile.RewriteFile(fset, file, literals, opts())
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(out)
	return err
}

func processFile(path string) error {
	src, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	file, fset, literals, err := gofile.FindSQLLiterals(src, path)
	if err != nil {
		return err
	}
	out, err := gofile.RewriteFile(fset, file, literals, opts())
	if err != nil {
		return err
	}

	if writeFlag {
		return os.WriteFile(path, out, 0644)
	}
	_, err = os.Stdout.Write(out)
	return err
}

var excludeDirs = map[string]bool{
	"vendor":   true,
	".git":     true,
	"testdata": true,
}

func resolvePatterns(patterns []string) ([]string, error) {
	var files []string
	for _, pattern := range patterns {
		resolved, err := resolvePattern(pattern)
		if err != nil {
			return nil, fmt.Errorf("resolving %q: %w", pattern, err)
		}
		files = append(files, resolved...)
	}
	return files, nil
}

func resolvePattern(pattern string) ([]string, error) {
	if strings.HasSuffix(pattern, "/...") {
		dir := strings.TrimSuffix(pattern, "/...")
		if dir == "." || dir == "" {
			dir = "."
		}
		return walkDir(dir)
	}

	info, err := os.Stat(pattern)
	if err == nil && info.IsDir() {
		return walkDir(pattern)
	}

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	var goFiles []string
	for _, m := range matches {
		if strings.HasSuffix(m, ".go") {
			goFiles = append(goFiles, m)
		}
	}
	return goFiles, nil
}

func walkDir(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && excludeDirs[d.Name()] {
			return filepath.SkipDir
		}
		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

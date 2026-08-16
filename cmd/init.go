package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Eagle-Konbu/sanat/internal/config"
)

var errAborted = errors.New("aborted")

func init() {
	rootCmd.AddCommand(newInitCmd())
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a sanat configuration file",
		Long:  "Interactively creates a .sanat.yml or .sanat.toml configuration file in the current directory.",
		Args:  cobra.NoArgs,
		RunE:  runInit,
	}
}

func runInit(cmd *cobra.Command, _ []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	return runInitWith(dir, interactivePrompter{}, cmd.OutOrStdout())
}

func runInitWith(dir string, p Prompter, out io.Writer) error {
	existingFile, err := confirmOverwrite(dir, p)
	if errors.Is(err, errAborted) {
		return nil
	}

	if err != nil {
		return err
	}

	format, err := p.SelectFormat()
	if err != nil {
		return err
	}

	indent, err := p.PromptIndent()
	if err != nil {
		return err
	}

	newline, err := p.PromptYesNo("Add newline after opening backtick?", true)
	if err != nil {
		return err
	}

	keywordCase, err := p.PromptKeywordCase()
	if err != nil {
		return err
	}

	commaStyle, err := p.PromptCommaStyle()
	if err != nil {
		return err
	}

	version := config.CurrentVersion

	cfg := config.Config{
		Version:     &version,
		Indent:      &indent,
		Newline:     &newline,
		KeywordCase: &keywordCase,
		CommaStyle:  &commaStyle,
	}

	return writeConfig(dir, cfg, format, existingFile, out)
}

// confirmOverwrite checks for an existing config file and asks whether to overwrite.
// Returns the existing filename (empty if none), or a nil error with empty string
// if the user declined to overwrite (caller should return nil).
func confirmOverwrite(dir string, p Prompter) (string, error) {
	name, ok := config.Exists(dir)
	if !ok {
		return "", nil
	}

	overwrite, err := p.PromptYesNo(name+" already exists. Overwrite?", false)
	if err != nil {
		return "", err
	}

	if !overwrite {
		return "", errAborted
	}

	return name, nil
}

func writeConfig(dir string, cfg config.Config, format config.Format, existingFile string, out io.Writer) error {
	data, err := config.Encode(cfg, format)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	filename := format.Filename()
	path := filepath.Join(dir, filename)

	if existingFile != "" && existingFile != filename {
		if err := os.Remove(filepath.Join(dir, existingFile)); err != nil {
			return fmt.Errorf("removing old config: %w", err)
		}
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Fprintf(out, "Created %s\n", filename)

	return nil
}

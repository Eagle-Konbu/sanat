package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Eagle-Konbu/sanat/internal/config"
)

var (
	errConfigAlreadyExists = errors.New("config file already exists")
	errInvalidFormatChoice = errors.New("invalid format choice: must be 1 or 2")
	errInvalidIndentWidth  = errors.New("invalid indent width: must be a positive integer")
	errInvalidYesNo        = errors.New("invalid input: expected y or n")
)

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

	return runInitWith(dir, os.Stdin, cmd.OutOrStdout())
}

func runInitWith(dir string, in io.Reader, out io.Writer) error {
	if name, ok := config.Exists(dir); ok {
		return fmt.Errorf("%w: %s", errConfigAlreadyExists, name)
	}

	reader := bufio.NewReader(in)

	format, err := promptFormat(reader, out)
	if err != nil {
		return err
	}

	indent, err := promptIndent(reader, out)
	if err != nil {
		return err
	}

	newline, err := promptYesNo(reader, out, "Add newline after opening backtick? (y/n)", true)
	if err != nil {
		return err
	}

	write, err := promptYesNo(reader, out, "Overwrite files in place? (y/n)", false)
	if err != nil {
		return err
	}

	cfg := config.Config{
		Write:   &write,
		Indent:  &indent,
		Newline: &newline,
	}

	data, err := config.Encode(cfg, format)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	filename := format.Filename()
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	fmt.Fprintf(out, "Created %s\n", filename)

	return nil
}

func promptFormat(reader *bufio.Reader, out io.Writer) (config.Format, error) {
	fmt.Fprintln(out, "Config file format:")
	fmt.Fprintln(out, "  1) yaml (.sanat.yml)")
	fmt.Fprintln(out, "  2) toml (.sanat.toml)")

	answer, err := prompt(reader, out, "Choose", "1")
	if err != nil {
		return "", err
	}

	switch answer {
	case "1":
		return config.FormatYAML, nil
	case "2":
		return config.FormatTOML, nil
	default:
		return "", errInvalidFormatChoice
	}
}

func promptIndent(reader *bufio.Reader, out io.Writer) (int, error) {
	answer, err := prompt(reader, out, "Indent width", "2")
	if err != nil {
		return 0, err
	}

	n, err := strconv.Atoi(answer)
	if err != nil || n <= 0 {
		return 0, errInvalidIndentWidth
	}

	return n, nil
}

func promptYesNo(reader *bufio.Reader, out io.Writer, question string, defaultVal bool) (bool, error) {
	def := "n"
	if defaultVal {
		def = "y"
	}

	answer, err := prompt(reader, out, question, def)
	if err != nil {
		return false, err
	}

	switch strings.ToLower(answer) {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	default:
		return false, errInvalidYesNo
	}
}

func prompt(reader *bufio.Reader, out io.Writer, question, defaultVal string) (string, error) {
	fmt.Fprintf(out, "%s [%s]: ", question, defaultVal)

	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return defaultVal, nil
	}

	return line, nil
}

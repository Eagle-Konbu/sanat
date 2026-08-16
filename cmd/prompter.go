package cmd

import (
	"errors"
	"strconv"

	"github.com/manifoldco/promptui"

	"github.com/Eagle-Konbu/sanat/internal/config"
)

var errInvalidIndentWidth = errors.New("must be a positive integer")

// Prompter abstracts interactive prompts so the init command
// can be tested without a real terminal.
type Prompter interface {
	SelectFormat() (config.Format, error)
	PromptIndent() (int, error)
	PromptYesNo(question string, defaultYes bool) (bool, error)
	PromptKeywordCase() (string, error)
	PromptCommaStyle() (string, error)
}

type interactivePrompter struct{}

func (interactivePrompter) SelectFormat() (config.Format, error) {
	sel := promptui.Select{
		Label: "Config file format",
		Items: []string{"yaml (.sanat.yml)", "toml (.sanat.toml)"},
	}

	idx, _, err := sel.Run()
	if err != nil {
		return "", err
	}

	switch idx {
	case 0:
		return config.FormatYAML, nil
	default:
		return config.FormatTOML, nil
	}
}

func (interactivePrompter) PromptIndent() (int, error) {
	p := promptui.Prompt{
		Label:   "Indent width",
		Default: "2",
		Validate: func(input string) error {
			n, err := strconv.Atoi(input)
			if err != nil || n <= 0 {
				return errInvalidIndentWidth
			}

			return nil
		},
	}

	result, err := p.Run()
	if err != nil {
		return 0, err
	}

	n, _ := strconv.Atoi(result)

	return n, nil
}

func (interactivePrompter) PromptYesNo(question string, defaultYes bool) (bool, error) {
	items := []string{"Yes", "No"}

	cursorPos := 0
	if !defaultYes {
		cursorPos = 1
	}

	sel := promptui.Select{
		Label:     question,
		Items:     items,
		CursorPos: cursorPos,
	}

	idx, _, err := sel.Run()
	if err != nil {
		return false, err
	}

	return idx == 0, nil
}

func (interactivePrompter) PromptKeywordCase() (string, error) {
	sel := promptui.Select{
		Label: "Keyword case (AS, AND, OR, IN, IS, LIKE, ...)",
		Items: []string{config.KeywordCaseUpper, config.KeywordCaseLower, config.KeywordCasePreserve},
	}

	_, result, err := sel.Run()
	if err != nil {
		return "", err
	}

	return result, nil
}

func (interactivePrompter) PromptCommaStyle() (string, error) {
	sel := promptui.Select{
		Label: "Comma style",
		Items: []string{config.CommaStyleTrailing, config.CommaStyleLeading},
	}

	_, result, err := sel.Run()
	if err != nil {
		return "", err
	}

	return result, nil
}

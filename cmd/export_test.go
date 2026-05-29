package cmd

import "io"

// RunInitWith is exported for testing.
var RunInitWith = runInitWith

// RunInitWithFunc is the function signature for runInitWith.
type RunInitWithFunc func(dir string, in io.Reader, out io.Writer) error

// Exported error sentinels for testing.
var (
	ErrConfigAlreadyExists = errConfigAlreadyExists
	ErrInvalidFormatChoice = errInvalidFormatChoice
	ErrInvalidIndentWidth  = errInvalidIndentWidth
	ErrInvalidYesNo        = errInvalidYesNo
)

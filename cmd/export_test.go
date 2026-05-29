package cmd

import "io"

// RunInitWith is exported for testing.
var RunInitWith = runInitWith

// RunInitWithFunc is the function signature for runInitWith.
type RunInitWithFunc func(dir string, p Prompter, out io.Writer) error

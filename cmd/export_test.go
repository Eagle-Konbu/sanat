package cmd

import (
	"io"

	"github.com/spf13/pflag"
)

// RunInitWith is exported for testing.
var RunInitWith = runInitWith

// RunInitWithFunc is the function signature for runInitWith.
type RunInitWithFunc func(dir string, p Prompter, out io.Writer) error

// ExecuteArgs runs the root command with args, writing to stdout and stderr.
// Flag values and their "changed" state persist across executions of the same
// command tree, so they are reset to defaults first.
func ExecuteArgs(args []string, stdout, stderr io.Writer) error {
	resetFlags(rootCmd.PersistentFlags())
	resetFlags(rootCmd.Flags())

	for _, sub := range rootCmd.Commands() {
		resetFlags(sub.Flags())
	}

	rootCmd.SetArgs(args)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)

	return rootCmd.Execute()
}

func resetFlags(flags *pflag.FlagSet) {
	flags.VisitAll(func(f *pflag.Flag) {
		_ = f.Value.Set(f.DefValue)
		f.Changed = false
	})
}

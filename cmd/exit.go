package cmd

import "errors"

const (
	exitCodeFailure  = 1
	exitCodeFindings = 1
	exitCodeError    = 2
)

// exitError carries the process exit code for errors whose code differs from
// the default failure code, so main can distinguish lint findings from
// analysis failures.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	return e.err.Error()
}

func (e *exitError) Unwrap() error {
	return e.err
}

// ExitCode returns the process exit code for an error returned by Execute.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}

	if exitErr, ok := errors.AsType[*exitError](err); ok {
		return exitErr.code
	}

	return exitCodeFailure
}

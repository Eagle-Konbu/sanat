package pgparser

import (
	"errors"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

// A statement kind that has no parser yet (MERGE, at the time of writing)
// can't reach attachWith through the public API, so this calls it directly
// to pin down that an unsupported kind is a parse error rather than a crash.
func TestAttachWith_unsupportedStatement(t *testing.T) {
	//nolint:nonamedreturns // the named result is mutated by the deferred recover
	attach := func() (err error) {
		defer recoverParseError(&err)

		attachWith(&pgast.Merge{}, &pgast.With{}, Position{Line: 1, Column: 1})

		return nil
	}

	err := attach()
	if _, ok := errors.AsType[*ParseError](err); !ok {
		t.Errorf("attachWith(*pgast.Merge) error = %v, want *ParseError", err)
	}
}

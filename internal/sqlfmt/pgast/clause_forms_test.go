package pgast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestFrameExclusion_String(t *testing.T) {
	tests := []struct {
		e    pgast.FrameExclusion
		want string
	}{
		{pgast.ExcludeDefault, ""},
		{pgast.ExcludeCurrentRow, "EXCLUDE CURRENT ROW"},
		{pgast.ExcludeGroup, "EXCLUDE GROUP"},
		{pgast.ExcludeTies, "EXCLUDE TIES"},
		{pgast.ExcludeNoOthers, "EXCLUDE NO OTHERS"},
	}

	for _, tt := range tests {
		assertEqual(t, tt.want, tt.e.String())
	}

	frame := &pgast.FrameClause{Start: &pgast.FramePoint{Type: pgast.CurrentRow}, Exclude: pgast.ExcludeTies}
	assertEqual(t, "ROWS CURRENT ROW EXCLUDE TIES", frame.String())
}

func TestOrder_String_using(t *testing.T) {
	o := &pgast.Order{Expr: col("a"), Direction: pgast.DescOrder, Using: &pgast.Operator{Symbol: ">"}, Nulls: pgast.NullsLast}
	assertEqual(t, "a USING > NULLS LAST", o.String())
}

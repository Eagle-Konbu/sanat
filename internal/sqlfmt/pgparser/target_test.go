package pgparser_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

func TestParseStatement_assignmentTargetIndirection(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"update subscript", "UPDATE t SET arr[1] = 2", "UPDATE t SET arr[1] = 2"},
		{"update slice", "UPDATE t SET arr[1:2] = '{3,4}'", "UPDATE t SET arr[1:2] = '{3,4}'"},
		{"update nested field", "UPDATE t SET c.a.b = 1", "UPDATE t SET c.a.b = 1"},
		{"update field of subscript", "UPDATE t SET arr[1].f = 1", "UPDATE t SET arr[1].f = 1"},
		{"update multi-column targets", "UPDATE t SET (a[1], b.f) = (1, 2)", "UPDATE t SET (a[1], b.f) = (1, 2)"},
		{"insert subscript column", "INSERT INTO t (arr[1], c.f) VALUES (1, 2)", "INSERT INTO t (arr[1], c.f) VALUES (1, 2)"},
		{
			"on conflict set subscript",
			"INSERT INTO t (a) VALUES (1) ON CONFLICT (a) DO UPDATE SET arr[2] = 1",
			"INSERT INTO t (a) VALUES (1) ON CONFLICT (a) DO UPDATE SET arr[2] = 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertStatementString(t, tt.in, tt.want)
		})
	}

	for _, in := range []string{"UPDATE t SET arr[1 = 2", "UPDATE t SET c. = 1", "INSERT INTO t (a[) VALUES (1)"} {
		t.Run("error "+in, func(t *testing.T) {
			if _, err := pgparser.ParseStatement(in); err == nil {
				t.Errorf("ParseStatement(%q) error = nil, want error", in)
			}
		})
	}
}

func TestParseUpdate_targetStructure(t *testing.T) {
	upd, err := pgparser.ParseUpdate("UPDATE t SET c.f = 1")
	if err != nil {
		t.Fatal(err)
	}

	want := &pgast.FieldSelectExpr{Expr: &pgast.ColName{Name: "c"}, Field: "f"}

	got, ok := upd.Set[0].Name.(*pgast.FieldSelectExpr)
	if !ok || got.Field != want.Field || got.Expr.String() != "c" {
		t.Errorf("Set[0].Name = %#v, want %#v", upd.Set[0].Name, want)
	}
}

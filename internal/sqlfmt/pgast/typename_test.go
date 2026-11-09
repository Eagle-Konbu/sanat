package pgast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestTimeZoneSpec_String(t *testing.T) {
	assertEqual(t, "", pgast.TimeZoneUnspecified.String())
	assertEqual(t, "with time zone", pgast.WithTimeZone.String())
	assertEqual(t, "without time zone", pgast.WithoutTimeZone.String())
}

func TestTypeName_String(t *testing.T) {
	tests := []struct {
		name string
		t    *pgast.TypeName
		want string
	}{
		{"simple", typeName("int"), "int"},
		{"qualified", &pgast.TypeName{Names: []pgast.ColIdent{"public", `"T"`}}, `public."T"`},
		{"modifiers", &pgast.TypeName{Names: []pgast.ColIdent{"numeric"}, Modifiers: []pgast.Expr{lit("10"), lit("2")}}, "numeric(10, 2)"},
		{
			"time zone",
			&pgast.TypeName{Names: []pgast.ColIdent{"timestamp"}, Modifiers: []pgast.Expr{lit("3")}, TimeZone: pgast.WithTimeZone},
			"timestamp(3) with time zone",
		},
		{"interval fields", &pgast.TypeName{Names: []pgast.ColIdent{"interval"}, IntervalFields: "day to second"}, "interval day to second"},
		{"array bounds", &pgast.TypeName{Names: []pgast.ColIdent{"int"}, ArrayBounds: []string{"", "3"}}, "int[][3]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.t.String())
		})
	}
}

func TestTypedLiteral_String(t *testing.T) {
	tests := []struct {
		name string
		l    *pgast.TypedLiteral
		want string
	}{
		{"simple", &pgast.TypedLiteral{Type: typeName("date"), Value: "'2020-01-01'"}, "date '2020-01-01'"},
		{
			"interval fields follow the string",
			&pgast.TypedLiteral{Type: &pgast.TypeName{Names: []pgast.ColIdent{"interval"}, IntervalFields: "day"}, Value: "'1'"},
			"interval '1' day",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertEqual(t, tt.want, tt.l.String())
		})
	}
}

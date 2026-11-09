package pgparser_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgparser"
)

// assertMergeRoundTrip checks that parsing input and re-stringifying the
// resulting AST reproduces the canonical SQL text in want.
func assertMergeRoundTrip(t *testing.T, input, want string) {
	t.Helper()

	merge, err := pgparser.ParseMerge(input)
	if err != nil {
		t.Fatalf("ParseMerge(%q) error = %v", input, err)
	}

	if got := merge.String(); got != want {
		t.Errorf("ParseMerge(%q).String() = %q, want %q", input, got, want)
	}
}

// mergePrefix is a minimal MERGE head that the WHEN clause tests append to.
const mergePrefix = "MERGE INTO t USING s ON t.id = s.id "

func TestParseMerge_target(t *testing.T) {
	const tail = " USING s ON t.id = s.id WHEN MATCHED THEN DELETE"

	tests := []struct {
		name, in, want string
	}{
		{"simple", "MERGE INTO t" + tail, "MERGE INTO t" + tail},
		{"trailing semicolon", "MERGE INTO t" + tail + ";", "MERGE INTO t" + tail},
		{"lower-case keywords", "merge into t using s on t.id = s.id when matched then delete", "MERGE INTO t" + tail},
		{"qualified table", "MERGE INTO public.t" + tail, "MERGE INTO public.t" + tail},
		{"quoted identifiers keep their quotes", `MERGE INTO "S"."T"` + tail, `MERGE INTO "S"."T"` + tail},
		{"explicit alias", "MERGE INTO target AS t" + tail, "MERGE INTO target t" + tail},
		{"implicit alias", "MERGE INTO target t" + tail, "MERGE INTO target t" + tail},
		{"only", "MERGE INTO ONLY t" + tail, "MERGE INTO ONLY t" + tail},
		{"only parenthesized name", "MERGE INTO ONLY (t)" + tail, "MERGE INTO ONLY t" + tail},
		{"include descendants", "MERGE INTO t *" + tail, "MERGE INTO t *" + tail},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertMergeRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseMerge_source(t *testing.T) {
	const tail = " WHEN MATCHED THEN DELETE"

	tests := []struct {
		name, in, want string
	}{
		{"table", "MERGE INTO t USING s ON t.id = s.id" + tail, "MERGE INTO t USING s ON t.id = s.id" + tail},
		{
			"aliased table",
			"MERGE INTO t USING public.src AS s ON t.id = s.id" + tail,
			"MERGE INTO t USING public.src s ON t.id = s.id" + tail,
		},
		{
			"subquery",
			"MERGE INTO t USING (SELECT id, v FROM src WHERE v > 0) AS s ON t.id = s.id" + tail,
			"MERGE INTO t USING (SELECT id, v FROM src WHERE v > 0) s ON t.id = s.id" + tail,
		},
		{
			"set operation subquery",
			"MERGE INTO t USING (SELECT 1 AS id UNION ALL SELECT 2) s ON t.id = s.id" + tail,
			"MERGE INTO t USING (SELECT 1 AS id UNION ALL SELECT 2) s ON t.id = s.id" + tail,
		},
		{
			"join",
			"MERGE INTO t USING a JOIN b ON a.id = b.a_id ON t.id = a.id" + tail,
			"MERGE INTO t USING a JOIN b ON a.id = b.a_id ON t.id = a.id" + tail,
		},
		{
			"lateral subquery",
			"MERGE INTO t USING LATERAL (SELECT 1 AS id) s ON t.id = s.id" + tail,
			"MERGE INTO t USING LATERAL (SELECT 1 AS id) s ON t.id = s.id" + tail,
		},
		{
			"compound join condition",
			"MERGE INTO t USING s ON t.id = s.id AND t.region = s.region" + tail,
			"MERGE INTO t USING s ON t.id = s.id AND t.region = s.region" + tail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertMergeRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseMerge_whenClauses(t *testing.T) {
	tests := []struct {
		name, when, want string
	}{
		{"matched update", "WHEN MATCHED THEN UPDATE SET v = s.v", ""},
		{"matched update multiple", "WHEN MATCHED THEN UPDATE SET v = s.v, n = n + 1", ""},
		{"matched update default", "WHEN MATCHED THEN UPDATE SET v = DEFAULT", ""},
		{"matched update multi-column", "WHEN MATCHED THEN UPDATE SET (a, b) = (s.a, s.b)", ""},
		{"matched update row", "WHEN MATCHED THEN UPDATE SET (a, b) = ROW(s.a, DEFAULT)", ""},
		{"matched update sub-select", "WHEN MATCHED THEN UPDATE SET (a, b) = (SELECT x, y FROM u)", ""},
		{"matched delete", "WHEN MATCHED THEN DELETE", ""},
		{"matched do nothing", "WHEN MATCHED THEN DO NOTHING", ""},
		{"matched with condition", "WHEN MATCHED AND s.deleted THEN DELETE", ""},
		{
			"compound condition",
			"WHEN MATCHED AND s.v > t.v AND NOT s.locked THEN UPDATE SET v = s.v",
			"",
		},
		{"not matched insert", "WHEN NOT MATCHED THEN INSERT VALUES (s.id, s.v)", ""},
		{"not matched insert columns", "WHEN NOT MATCHED THEN INSERT (id, v) VALUES (s.id, s.v)", ""},
		{"not matched insert default item", "WHEN NOT MATCHED THEN INSERT (id, v) VALUES (s.id, DEFAULT)", ""},
		{"not matched insert default values", "WHEN NOT MATCHED THEN INSERT DEFAULT VALUES", ""},
		{
			"not matched insert overriding system",
			"WHEN NOT MATCHED THEN INSERT (id) OVERRIDING SYSTEM VALUE VALUES (s.id)",
			"",
		},
		{
			"not matched insert overriding user",
			"WHEN NOT MATCHED THEN INSERT OVERRIDING USER VALUE VALUES (s.id)",
			"",
		},
		{
			"overriding lower case",
			"WHEN NOT MATCHED THEN INSERT overriding user value VALUES (s.id)",
			"WHEN NOT MATCHED THEN INSERT OVERRIDING USER VALUE VALUES (s.id)",
		},
		{"not matched do nothing", "WHEN NOT MATCHED THEN DO NOTHING", ""},
		{"not matched with condition", "WHEN NOT MATCHED AND s.v IS NOT NULL THEN INSERT VALUES (s.id)", ""},
		{"not matched by target insert", "WHEN NOT MATCHED BY TARGET THEN INSERT VALUES (s.id)", ""},
		{"not matched by target do nothing", "WHEN NOT MATCHED BY TARGET AND s.skip THEN DO NOTHING", ""},
		{"not matched by source update", "WHEN NOT MATCHED BY SOURCE THEN UPDATE SET active = FALSE", ""},
		{"not matched by source delete", "WHEN NOT MATCHED BY SOURCE AND t.expired THEN DELETE", ""},
		{"not matched by source do nothing", "WHEN NOT MATCHED BY SOURCE THEN DO NOTHING", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := tt.want
			if want == "" {
				want = tt.when
			}

			assertMergeRoundTrip(t, mergePrefix+tt.when, mergePrefix+want)
		})
	}
}

func TestParseMerge_multipleWhenClauses(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"upsert",
			mergePrefix + "WHEN MATCHED THEN UPDATE SET v = s.v WHEN NOT MATCHED THEN INSERT (id, v) VALUES (s.id, s.v)",
			mergePrefix + "WHEN MATCHED THEN UPDATE SET v = s.v WHEN NOT MATCHED THEN INSERT (id, v) VALUES (s.id, s.v)",
		},
		{
			"conditional matched clauses in order",
			mergePrefix + "WHEN MATCHED AND s.deleted THEN DELETE WHEN MATCHED THEN UPDATE SET v = s.v",
			mergePrefix + "WHEN MATCHED AND s.deleted THEN DELETE WHEN MATCHED THEN UPDATE SET v = s.v",
		},
		{
			"every match kind",
			mergePrefix +
				"WHEN MATCHED AND s.v IS NULL THEN DELETE " +
				"WHEN MATCHED THEN UPDATE SET v = s.v " +
				"WHEN NOT MATCHED BY TARGET AND s.v IS NULL THEN DO NOTHING " +
				"WHEN NOT MATCHED THEN INSERT VALUES (s.id, s.v) " +
				"WHEN NOT MATCHED BY SOURCE THEN UPDATE SET active = FALSE",
			mergePrefix +
				"WHEN MATCHED AND s.v IS NULL THEN DELETE " +
				"WHEN MATCHED THEN UPDATE SET v = s.v " +
				"WHEN NOT MATCHED BY TARGET AND s.v IS NULL THEN DO NOTHING " +
				"WHEN NOT MATCHED THEN INSERT VALUES (s.id, s.v) " +
				"WHEN NOT MATCHED BY SOURCE THEN UPDATE SET active = FALSE",
		},
		{
			"official docs example",
			"MERGE INTO customer_account ca USING recent_transactions t ON t.customer_id = ca.customer_id " +
				"WHEN MATCHED THEN UPDATE SET balance = balance + transaction_value " +
				"WHEN NOT MATCHED THEN INSERT (customer_id, balance) VALUES (t.customer_id, t.transaction_value)",
			"MERGE INTO customer_account ca USING recent_transactions t ON t.customer_id = ca.customer_id " +
				"WHEN MATCHED THEN UPDATE SET balance = balance + transaction_value " +
				"WHEN NOT MATCHED THEN INSERT (customer_id, balance) VALUES (t.customer_id, t.transaction_value)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertMergeRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseMerge_returning(t *testing.T) {
	const head = mergePrefix + "WHEN MATCHED THEN DELETE "

	tests := []struct {
		name, in, want string
	}{
		{"star", head + "RETURNING *", head + "RETURNING *"},
		{"qualified star", head + "RETURNING t.*", head + "RETURNING t.*"},
		{"merge_action", head + "RETURNING merge_action(), t.id", head + "RETURNING merge_action(), t.id"},
		{"implicit alias", head + "RETURNING merge_action() action", head + "RETURNING merge_action() AS action"},
		{"old alias", head + "RETURNING WITH (OLD AS o) o.v", head + "RETURNING WITH (OLD AS o) o.v"},
		{
			"old and new aliases",
			head + "RETURNING WITH (OLD AS o, NEW AS n) o.v AS old_v, n.v AS new_v",
			head + "RETURNING WITH (OLD AS o, NEW AS n) o.v AS old_v, n.v AS new_v",
		},
		{
			"new before old",
			head + "returning with (new as n, old as o) n.*",
			head + "RETURNING WITH (NEW AS n, OLD AS o) n.*",
		},
		{"old and new as columns", head + "RETURNING old.v, new.v", head + "RETURNING old.v, new.v"},
		{"trailing semicolon", head + "RETURNING *;", head + "RETURNING *"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertMergeRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseMerge_with(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"leading with",
			"WITH s AS (SELECT id, v FROM src) MERGE INTO t USING s ON t.id = s.id WHEN MATCHED THEN DELETE",
			"WITH s AS (SELECT id, v FROM src) MERGE INTO t USING s ON t.id = s.id WHEN MATCHED THEN DELETE",
		},
		{
			"leading with recursive",
			"WITH RECURSIVE s AS (SELECT 1 AS id) MERGE INTO t USING s ON t.id = s.id WHEN MATCHED THEN DELETE",
			"WITH RECURSIVE s AS (SELECT 1 AS id) MERGE INTO t USING s ON t.id = s.id WHEN MATCHED THEN DELETE",
		},
		{
			"data-modifying cte feeding merge",
			"WITH d AS (DELETE FROM staging RETURNING *) MERGE INTO t USING d ON t.id = d.id " +
				"WHEN NOT MATCHED THEN INSERT VALUES (d.id)",
			"WITH d AS (DELETE FROM staging RETURNING *) MERGE INTO t USING d ON t.id = d.id " +
				"WHEN NOT MATCHED THEN INSERT VALUES (d.id)",
		},
		{
			"with in source subquery",
			"MERGE INTO t USING (WITH x AS (SELECT 1 AS id) SELECT * FROM x) s ON t.id = s.id WHEN MATCHED THEN DELETE",
			"MERGE INTO t USING (WITH x AS (SELECT 1 AS id) SELECT * FROM x) s ON t.id = s.id WHEN MATCHED THEN DELETE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertMergeRoundTrip(t, tt.in, tt.want)
		})
	}
}

func TestParseStatement_mergeAsCTEBody(t *testing.T) {
	tests := []struct {
		name, in, want string
		wantType       pgast.Statement
	}{
		{
			"merge body feeding select",
			"WITH m AS (MERGE INTO t USING s ON t.id = s.id WHEN MATCHED THEN DELETE RETURNING t.id) SELECT * FROM m",
			"WITH m AS (MERGE INTO t USING s ON t.id = s.id WHEN MATCHED THEN DELETE RETURNING t.id) SELECT * FROM m",
			&pgast.Select{},
		},
		{
			"merge body with its own with",
			"WITH m AS (WITH s AS (SELECT 1 AS id) MERGE INTO t USING s ON t.id = s.id " +
				"WHEN MATCHED THEN DELETE RETURNING t.id) INSERT INTO log SELECT id FROM m",
			"WITH m AS (WITH s AS (SELECT 1 AS id) MERGE INTO t USING s ON t.id = s.id " +
				"WHEN MATCHED THEN DELETE RETURNING t.id) INSERT INTO log SELECT id FROM m",
			&pgast.Insert{},
		},
		{
			"merge body feeding merge",
			"WITH m AS (MERGE INTO a USING s ON a.id = s.id WHEN MATCHED THEN DELETE RETURNING s.id) " +
				"MERGE INTO b USING m ON b.id = m.id WHEN MATCHED THEN DELETE",
			"WITH m AS (MERGE INTO a USING s ON a.id = s.id WHEN MATCHED THEN DELETE RETURNING s.id) " +
				"MERGE INTO b USING m ON b.id = m.id WHEN MATCHED THEN DELETE",
			&pgast.Merge{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertStatementRoundTrip(t, tt.in, tt.want, tt.wantType)
		})
	}
}

// mergeStructureInput exercises every part of the Merge AST in one statement.
const mergeStructureInput = "WITH s AS (SELECT 1 AS id) MERGE INTO ONLY t AS tgt USING s ON tgt.id = s.id " +
	"WHEN MATCHED AND s.id > 0 THEN UPDATE SET v = 1 " +
	"WHEN NOT MATCHED BY SOURCE THEN DELETE " +
	"WHEN NOT MATCHED BY TARGET THEN INSERT (id) OVERRIDING USER VALUE VALUES (s.id) " +
	"WHEN NOT MATCHED THEN DO NOTHING " +
	"RETURNING WITH (NEW AS n) n.id"

func parseMergeStructureInput(t *testing.T) *pgast.Merge {
	t.Helper()

	merge, err := pgparser.ParseMerge(mergeStructureInput)
	if err != nil {
		t.Fatalf("ParseMerge(%q) error = %v", mergeStructureInput, err)
	}

	return merge
}

func TestParseMerge_structure(t *testing.T) {
	merge := parseMergeStructureInput(t)

	if merge.With == nil || len(merge.With.CTEs) != 1 {
		t.Errorf("With = %#v, want one CTE", merge.With)
	}

	if !merge.Target.Only || merge.Target.As.Name != "tgt" {
		t.Errorf("Target = %#v, want ONLY t aliased tgt", merge.Target)
	}

	wantAliases := []*pgast.ReturningAlias{{Kind: pgast.ReturningNew, Name: pgast.TableIdent{Name: "n", Raw: "n"}}}
	if merge.Returning == nil || !reflect.DeepEqual(merge.Returning.Aliases, wantAliases) {
		t.Errorf("Returning = %#v, want WITH (NEW AS n)", merge.Returning)
	}
}

func TestParseMerge_whenStructure(t *testing.T) {
	merge := parseMergeStructureInput(t)

	wantWhens := []struct {
		match        pgast.MergeMatchKind
		hasCondition bool
		action       pgast.MergeAction
	}{
		{pgast.MergeMatched, true, &pgast.MergeUpdate{}},
		{pgast.MergeNotMatchedBySource, false, pgast.MergeDelete{}},
		{pgast.MergeNotMatchedByTarget, false, &pgast.MergeInsert{}},
		{pgast.MergeNotMatched, false, pgast.MergeDoNothing{}},
	}

	if len(merge.Whens) != len(wantWhens) {
		t.Fatalf("len(Whens) = %d, want %d", len(merge.Whens), len(wantWhens))
	}

	for i, want := range wantWhens {
		got := merge.Whens[i]

		if got.Match != want.match {
			t.Errorf("Whens[%d].Match = %v, want %v", i, got.Match, want.match)
		}

		if (got.Condition != nil) != want.hasCondition {
			t.Errorf("Whens[%d].Condition = %v, want present = %v", i, got.Condition, want.hasCondition)
		}

		if reflect.TypeOf(got.Action) != reflect.TypeOf(want.action) {
			t.Errorf("Whens[%d].Action = %T, want %T", i, got.Action, want.action)
		}
	}
}

func TestParseMerge_insertStructure(t *testing.T) {
	merge := parseMergeStructureInput(t)

	ins, ok := merge.Whens[2].Action.(*pgast.MergeInsert)
	if !ok {
		t.Fatalf("Whens[2].Action = %T, want *pgast.MergeInsert", merge.Whens[2].Action)
	}

	if ins.Overriding != pgast.OverridingUserValue || len(ins.Columns) != 1 || len(ins.Values) != 1 {
		t.Errorf("Whens[2].Action = %#v, want one column, OVERRIDING USER VALUE, and one value", ins)
	}
}

func TestParseMerge_defaultValuesHasNilValues(t *testing.T) {
	merge, err := pgparser.ParseMerge(mergePrefix + "WHEN NOT MATCHED THEN INSERT DEFAULT VALUES")
	if err != nil {
		t.Fatalf("ParseMerge error = %v", err)
	}

	ins, ok := merge.Whens[0].Action.(*pgast.MergeInsert)
	if !ok {
		t.Fatalf("Action = %T, want *pgast.MergeInsert", merge.Whens[0].Action)
	}

	if ins.Values != nil || ins.Columns != nil || ins.Overriding != pgast.OverridingNone {
		t.Errorf("Action = %#v, want DEFAULT VALUES with no columns or OVERRIDING", ins)
	}
}

// MERGE's keywords are unreserved in PostgreSQL, so they remain usable as
// names everywhere, including inside a MERGE statement itself.
func TestParseStatement_mergeKeywordsAsIdentifiers(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{
			"columns",
			"SELECT merge, matched, source, target, overriding, system, value, old, new, user FROM t",
			"SELECT merge, matched, source, target, overriding, system, value, old, new, user FROM t",
		},
		{"bare select aliases", "SELECT 1 source, 2 target", "SELECT 1 AS source, 2 AS target"},
		{"table and alias", "SELECT * FROM merge matched", "SELECT * FROM merge matched"},
		{"function name", "SELECT value(x), source(y)", "SELECT value(x), source(y)"},
		{
			"names inside merge",
			"MERGE INTO target source USING merge matched ON source.value = matched.value " +
				"WHEN MATCHED THEN UPDATE SET value = matched.new " +
				"WHEN NOT MATCHED THEN INSERT (value, old) VALUES (matched.value, matched.old)",
			"MERGE INTO target source USING merge matched ON source.value = matched.value " +
				"WHEN MATCHED THEN UPDATE SET value = matched.new " +
				"WHEN NOT MATCHED THEN INSERT (value, old) VALUES (matched.value, matched.old)",
		},
		{
			"returning alias names",
			mergePrefix + "WHEN MATCHED THEN DELETE RETURNING WITH (OLD AS old, NEW AS new) old.v",
			mergePrefix + "WHEN MATCHED THEN DELETE RETURNING WITH (OLD AS old, NEW AS new) old.v",
		},
		{"delete alias", "DELETE FROM t AS source", "DELETE FROM t source"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertStatementString(t, tt.in, tt.want)
		})
	}
}

func TestParseMerge_errors(t *testing.T) {
	tests := []struct {
		name, in, wantMsg string
	}{
		{"empty", "", "expected MERGE, got EOF"},
		{"missing into", "MERGE t USING s ON TRUE WHEN MATCHED THEN DELETE", "expected INTO, got IDENT"},
		{"missing target", "MERGE INTO USING s ON TRUE WHEN MATCHED THEN DELETE", "expected identifier, got USING"},
		{"set alias", "MERGE INTO t set USING s ON TRUE WHEN MATCHED THEN DELETE", "expected USING, got SET"},
		{"missing using", "MERGE INTO t WHEN MATCHED THEN DELETE", "expected USING, got WHEN"},
		{"missing source", "MERGE INTO t USING ON TRUE WHEN MATCHED THEN DELETE", "expected identifier, got ON"},
		{"comma source list", "MERGE INTO t USING a, b ON TRUE WHEN MATCHED THEN DELETE", "expected ON, got ,"},
		{"missing on", "MERGE INTO t USING s WHEN MATCHED THEN DELETE", "expected ON, got WHEN"},
		{"missing condition", "MERGE INTO t USING s ON WHEN MATCHED THEN DELETE", "unexpected token WHEN"},
		{"no when clause", "MERGE INTO t USING s ON TRUE", "expected WHEN, got EOF"},
		{"no when clause before returning", "MERGE INTO t USING s ON TRUE RETURNING *", "expected WHEN, got RETURNING"},
		{"missing match kind", mergePrefix + "WHEN THEN DELETE", "expected NOT, got THEN"},
		{"not without matched", mergePrefix + "WHEN NOT THEN DELETE", "expected MATCHED, got THEN"},
		{"by without kind", mergePrefix + "WHEN NOT MATCHED BY THEN DO NOTHING", "expected SOURCE or TARGET after NOT MATCHED BY, got THEN"},
		{"matched by source", mergePrefix + "WHEN MATCHED BY SOURCE THEN DELETE", "expected THEN, got BY"},
		{"missing then", mergePrefix + "WHEN MATCHED DELETE", "expected THEN, got DELETE"},
		{"missing action", mergePrefix + "WHEN MATCHED THEN", "expected UPDATE, DELETE, or DO NOTHING, got EOF"},
		{"matched insert", mergePrefix + "WHEN MATCHED THEN INSERT VALUES (1)", "expected UPDATE, DELETE, or DO NOTHING, got INSERT"},
		{
			"not matched by source insert",
			mergePrefix + "WHEN NOT MATCHED BY SOURCE THEN INSERT VALUES (1)",
			"expected UPDATE, DELETE, or DO NOTHING, got INSERT",
		},
		{"not matched update", mergePrefix + "WHEN NOT MATCHED THEN UPDATE SET v = 1", "expected INSERT or DO NOTHING, got UPDATE"},
		{"not matched delete", mergePrefix + "WHEN NOT MATCHED THEN DELETE", "expected INSERT or DO NOTHING, got DELETE"},
		{
			"not matched by target delete",
			mergePrefix + "WHEN NOT MATCHED BY TARGET THEN DELETE",
			"expected INSERT or DO NOTHING, got DELETE",
		},
		{"do without nothing", mergePrefix + "WHEN MATCHED THEN DO", "expected NOTHING, got EOF"},
		{"update without set", mergePrefix + "WHEN MATCHED THEN UPDATE v = 1", "expected SET, got IDENT"},
		{"update with table", mergePrefix + "WHEN MATCHED THEN UPDATE t SET v = 1", "expected SET, got IDENT"},
		{"update with where", mergePrefix + "WHEN MATCHED THEN UPDATE SET v = 1 WHERE v = 2", "unexpected token WHERE after statement"},
		{"delete with where", mergePrefix + "WHEN MATCHED THEN DELETE WHERE v = 2", "unexpected token WHERE after statement"},
		{"insert into", mergePrefix + "WHEN NOT MATCHED THEN INSERT INTO t VALUES (1)", "expected VALUES, got INTO"},
		{"insert without values", mergePrefix + "WHEN NOT MATCHED THEN INSERT (id)", "expected VALUES, got EOF"},
		{"insert multiple rows", mergePrefix + "WHEN NOT MATCHED THEN INSERT VALUES (1), (2)", "unexpected token , after statement"},
		{"insert select", mergePrefix + "WHEN NOT MATCHED THEN INSERT SELECT 1", "expected VALUES, got SELECT"},
		{"insert empty values", mergePrefix + "WHEN NOT MATCHED THEN INSERT VALUES ()", "unexpected token )"},
		{
			"default values with columns",
			mergePrefix + "WHEN NOT MATCHED THEN INSERT (id) DEFAULT VALUES",
			"DEFAULT VALUES cannot follow a column list or OVERRIDING",
		},
		{
			"default values with overriding",
			mergePrefix + "WHEN NOT MATCHED THEN INSERT OVERRIDING SYSTEM VALUE DEFAULT VALUES",
			"DEFAULT VALUES cannot follow a column list or OVERRIDING",
		},
		{"default without values", mergePrefix + "WHEN NOT MATCHED THEN INSERT DEFAULT", "expected VALUES, got EOF"},
		{
			"overriding without kind",
			mergePrefix + "WHEN NOT MATCHED THEN INSERT OVERRIDING VALUE VALUES (1)",
			"expected SYSTEM or USER after OVERRIDING, got VALUE",
		},
		{
			"overriding quoted user",
			mergePrefix + `WHEN NOT MATCHED THEN INSERT OVERRIDING "user" VALUE VALUES (1)`,
			"expected SYSTEM or USER after OVERRIDING, got QUOTED_IDENT",
		},
		{
			"overriding without value",
			mergePrefix + "WHEN NOT MATCHED THEN INSERT OVERRIDING SYSTEM VALUES (1)",
			"expected VALUE, got VALUES",
		},
		{
			"overriding before columns",
			mergePrefix + "WHEN NOT MATCHED THEN INSERT OVERRIDING SYSTEM VALUE (id) VALUES (1)",
			"expected VALUES, got (",
		},
		{"empty returning", mergePrefix + "WHEN MATCHED THEN DELETE RETURNING", "unexpected token EOF"},
		{
			"returning with without aliases",
			mergePrefix + "WHEN MATCHED THEN DELETE RETURNING WITH () *",
			"expected OLD or NEW, got )",
		},
		{
			"returning alias without as",
			mergePrefix + "WHEN MATCHED THEN DELETE RETURNING WITH (OLD o) o.*",
			"expected AS, got IDENT",
		},
		{
			"returning alias with other kind",
			mergePrefix + "WHEN MATCHED THEN DELETE RETURNING WITH (CURRENT AS c) c.*",
			"expected OLD or NEW, got CURRENT",
		},
		{
			"returning with without list",
			mergePrefix + "WHEN MATCHED THEN DELETE RETURNING WITH (OLD AS o)",
			"unexpected token EOF",
		},
		{
			"returning with unclosed",
			mergePrefix + "WHEN MATCHED THEN DELETE RETURNING WITH (OLD AS o *",
			"expected ), got *",
		},
		{"trailing garbage", mergePrefix + "WHEN MATCHED THEN DELETE garbage", "unexpected token IDENT after statement"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := pgparser.ParseMerge(tt.in)

			parseErr, ok := errors.AsType[*pgparser.ParseError](err)
			if !ok {
				t.Fatalf("ParseMerge(%q) error = %v, want *pgparser.ParseError", tt.in, err)
			}

			if !strings.Contains(parseErr.Msg, tt.wantMsg) {
				t.Errorf("ParseMerge(%q) error message = %q, want it to contain %q", tt.in, parseErr.Msg, tt.wantMsg)
			}
		})
	}
}

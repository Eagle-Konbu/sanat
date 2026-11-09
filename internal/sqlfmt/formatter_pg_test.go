package sqlfmt_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt"
)

type pgCase struct {
	name string
	in   string
	want string
}

func pgOptions() sqlfmt.Options {
	return sqlfmt.Options{Indent: 2, Dialect: sqlfmt.DialectPostgreSQL}
}

// assertPGFormats checks that each case formats to want under the
// PostgreSQL dialect, and that formatting want again leaves it unchanged.
func assertPGFormats(t *testing.T, tests []pgCase) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sqlfmt.FormatSQLWithOptions(tt.in, pgOptions())
			if !ok {
				t.Fatalf("FormatSQLWithOptions(%q) ok = false, want true", tt.in)
			}

			assertSQL(t, got, tt.want)

			again, ok := sqlfmt.FormatSQLWithOptions(got, pgOptions())
			if !ok {
				t.Fatalf("reformatting the output: ok = false, want true")
			}

			assertSQL(t, again, got)
		})
	}
}

func TestFormatPostgreSQL_Select(t *testing.T) {
	assertPGFormats(t, []pgCase{
		{
			name: "clauses",
			in:   "select u.Id, count(*) as Cnt from public.Users u left join orders o on o.user_id = u.id where u.active and (u.x = $1 or u.y is not null) group by u.Id having count(*) > 1 order by 2 desc nulls last limit 10 offset 5 for update of u skip locked",
			want: join(
				"SELECT",
				"  u.Id,",
				"  COUNT(*) AS Cnt",
				"FROM",
				"  public.Users u",
				"  LEFT JOIN",
				"  orders o",
				"    ON o.user_id = u.id",
				"WHERE",
				"  u.active",
				"  AND (u.x = $1 OR u.y IS NOT NULL)",
				"GROUP BY",
				"  u.Id",
				"HAVING",
				"  COUNT(*) > 1",
				"ORDER BY",
				"  2 DESC NULLS LAST",
				"LIMIT",
				"  10",
				"OFFSET",
				"  5",
				"FOR UPDATE OF u SKIP LOCKED",
			),
		},
		{
			name: "distinct on and subqueries",
			in:   "select distinct on (a) a, b from t where exists (select 1 from s where s.a = t.a) and t.id in (select id from u) and t.k = any(array[1,2]) and t.v = any (select v from w)",
			want: join(
				"SELECT DISTINCT ON (a)",
				"  a,",
				"  b",
				"FROM",
				"  t",
				"WHERE",
				"  EXISTS (",
				"    SELECT",
				"      1",
				"    FROM",
				"      s",
				"    WHERE",
				"      s.a = t.a",
				"  )",
				"  AND t.id IN (",
				"    SELECT",
				"      id",
				"    FROM",
				"      u",
				"  )",
				"  AND t.k = ANY (ARRAY[1, 2])",
				"  AND t.v = ANY (",
				"    SELECT",
				"      v",
				"    FROM",
				"      w",
				"  )",
			),
		},
		{
			name: "distinct",
			in:   "select distinct a from t",
			want: join(
				"SELECT DISTINCT",
				"  a",
				"FROM",
				"  t",
			),
		},
		{
			name: "scalar subquery and case",
			in:   "select (select 1) as s, case x when 1 then (select 2) end from t",
			want: join(
				"SELECT",
				"  (",
				"    SELECT",
				"      1",
				"  ) AS s,",
				"  CASE x",
				"    WHEN 1 THEN (",
				"      SELECT",
				"        2",
				"    )",
				"  END",
				"FROM",
				"  t",
			),
		},
		{
			name: "window functions",
			in:   "select case when a then 'x' else 'y' end, sum(x) over (partition by a order by b rows between unbounded preceding and current row), rank() over w, sum(a) over (), sum(a) over (w partition by b range between 1 preceding and 1 following exclude ties) from t window w as (order by a)",
			want: join(
				"SELECT",
				"  CASE",
				"    WHEN a THEN 'x'",
				"    ELSE 'y'",
				"  END,",
				"  SUM(x) OVER (",
				"    PARTITION BY a",
				"    ORDER BY b",
				"    ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW",
				"  ),",
				"  RANK() OVER w,",
				"  SUM(a) OVER (),",
				"  SUM(a) OVER (",
				"    w",
				"    PARTITION BY b",
				"    RANGE BETWEEN 1 PRECEDING AND 1 FOLLOWING EXCLUDE TIES",
				"  )",
				"FROM",
				"  t",
				"WINDOW",
				"  w AS (",
				"    ORDER BY a",
				"  )",
			),
		},
		{
			name: "grouping elements",
			in:   "select a from t group by grouping sets ((a), ()), cube (a, b), rollup (a)",
			want: join(
				"SELECT",
				"  a",
				"FROM",
				"  t",
				"GROUP BY",
				"  GROUPING SETS (a, ()),",
				"  CUBE (a, b),",
				"  ROLLUP (a)",
			),
		},
		{
			name: "nested grouping elements stay unparenthesized",
			in:   "select a from t group by grouping sets (rollup (a, b), (c, d))",
			want: join(
				"SELECT",
				"  a",
				"FROM",
				"  t",
				"GROUP BY",
				"  GROUPING SETS (rollup(a, b), (c, d))",
			),
		},
		{
			name: "values",
			in:   "values (1, 2), (3, 4) order by 1",
			want: join(
				"VALUES",
				"  (1, 2),",
				"  (3, 4)",
				"ORDER BY",
				"  1",
			),
		},
		{
			name: "table",
			in:   "table users",
			want: join(
				"TABLE users",
			),
		},
		{
			name: "fetch with ties",
			in:   "select a from t fetch first 3 rows with ties",
			want: join(
				"SELECT",
				"  a",
				"FROM",
				"  t",
				"FETCH FIRST 3 ROWS WITH TIES",
			),
		},
		{
			name: "offset and fetch next",
			in:   "select a from t offset 5 fetch next row only",
			want: join(
				"SELECT",
				"  a",
				"FROM",
				"  t",
				"OFFSET",
				"  5",
				"FETCH NEXT ROW ONLY",
			),
		},
		{
			name: "limit all",
			in:   "select a from t limit all",
			want: join(
				"SELECT",
				"  a",
				"FROM",
				"  t",
				"LIMIT",
				"  ALL",
			),
		},
		{
			name: "multiple locks",
			in:   "select a from t for update for share of t, u nowait",
			want: join(
				"SELECT",
				"  a",
				"FROM",
				"  t",
				"FOR UPDATE",
				"FOR SHARE OF t, u NOWAIT",
			),
		},
		{
			name: "trailing semicolon",
			in:   "select * from t;",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  t",
			),
		},
		{
			name: "no select list",
			in:   "select from t",
			want: join(
				"SELECT",
				"FROM",
				"  t",
			),
		},
	})
}

func TestFormatPostgreSQL_With(t *testing.T) {
	assertPGFormats(t, []pgCase{
		{
			name: "recursive and materialized",
			in:   "with recursive t(n) as materialized (select 1 union all select n+1 from t where n < 5) select * from t",
			want: join(
				"WITH RECURSIVE",
				"  t (n) AS MATERIALIZED (",
				"    SELECT",
				"      1",
				"    UNION ALL",
				"    SELECT",
				"      n + 1",
				"    FROM",
				"      t",
				"    WHERE",
				"      n < 5",
				"  )",
				"SELECT",
				"  *",
				"FROM",
				"  t",
			),
		},
		{
			name: "data-modifying cte",
			in:   "with x as (insert into t values (1) returning id), y as not materialized (select 2) select * from x",
			want: join(
				"WITH",
				"  x AS (",
				"    INSERT INTO",
				"      t",
				"    VALUES",
				"      (1)",
				"    RETURNING",
				"      id",
				"  ),",
				"  y AS NOT MATERIALIZED (",
				"    SELECT",
				"      2",
				"  )",
				"SELECT",
				"  *",
				"FROM",
				"  x",
			),
		},
	})
}

func TestFormatPostgreSQL_SetOp(t *testing.T) {
	assertPGFormats(t, []pgCase{
		{
			name: "operand clauses need parens",
			in:   "(select 1 order by 1) union (select 2) except select 3",
			want: join(
				"(",
				"  SELECT",
				"    1",
				"  ORDER BY",
				"    1",
				")",
				"UNION",
				"SELECT",
				"  2",
				"EXCEPT",
				"SELECT",
				"  3",
			),
		},
		{
			name: "precedence needs parens",
			in:   "select 1 union all (select 2 union select 3) intersect select 4",
			want: join(
				"SELECT",
				"  1",
				"UNION ALL",
				"(",
				"  SELECT",
				"    2",
				"  UNION",
				"  SELECT",
				"    3",
				")",
				"INTERSECT",
				"SELECT",
				"  4",
			),
		},
		{
			name: "trailing clauses",
			in:   "select a from t union select a from u order by a limit 1",
			want: join(
				"SELECT",
				"  a",
				"FROM",
				"  t",
				"UNION",
				"SELECT",
				"  a",
				"FROM",
				"  u",
				"ORDER BY",
				"  a",
				"LIMIT",
				"  1",
			),
		},
	})
}

func TestFormatPostgreSQL_From(t *testing.T) {
	assertPGFormats(t, []pgCase{
		{
			name: "derived table, lateral function, joins",
			in:   "select * from (select 1) s, lateral generate_series(1,3) with ordinality as g(i, o) cross join t2 join t3 using (id)",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  (",
				"  SELECT",
				"    1",
				"  ) s,",
				"  LATERAL generate_series(1, 3) WITH ORDINALITY g(i, o)",
				"  CROSS JOIN",
				"  t2",
				"  JOIN",
				"  t3",
				"    USING (id)",
			),
		},
		{
			name: "lateral derived table",
			in:   "select * from t, lateral (select t.a) x",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  t,",
				"  LATERAL (",
				"  SELECT",
				"    t.a",
				"  ) x",
			),
		},
		{
			name: "only",
			in:   "select * from only t",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  ONLY t",
			),
		},
		{
			name: "include descendants",
			in:   "select * from t *",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  t *",
			),
		},
		{
			name: "tablesample",
			in:   "select * from t2 tablesample bernoulli (10) repeatable (1)",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  t2 TABLESAMPLE bernoulli (10) REPEATABLE (1)",
			),
		},
		{
			name: "rows from",
			in:   "select * from rows from (f(1), g(2)) as x(a, b)",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  ROWS FROM (f(1), g(2)) x(a, b)",
			),
		},
		{
			name: "parenthesized join",
			in:   "select * from (t3 join t4 on true) as j",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  (",
				"    t3",
				"    JOIN",
				"    t4",
				"      ON TRUE",
				"  ) j",
			),
		},
		{
			name: "using alias and natural join",
			in:   "select * from t5 join t6 using (id) as u natural join t7",
			want: join(
				"SELECT",
				"  *",
				"FROM",
				"  t5",
				"  JOIN",
				"  t6",
				"    USING (id) AS u",
				"  NATURAL JOIN",
				"  t7",
			),
		},
	})
}

func TestFormatPostgreSQL_Expressions(t *testing.T) {
	assertPGFormats(t, []pgCase{
		{
			name: "operators and indirection",
			in:   `select x[1], x[1:2], x[:], (r).f, (r).*, row(1, 2), array[[1], [2]], array(select 1), -a, - -a, ~ b, a operator(pg_catalog.+) b, a operator(+) b, a collate "C", a at time zone 'UTC', (a, b) overlaps (c, d), f(x => 1, y := 2), a || b, 2 ^ 3 from t`,
			want: join(
				"SELECT",
				"  x[1],",
				"  x[1:2],",
				"  x[:],",
				"  (r).f,",
				"  (r).*,",
				"  ROW(1, 2),",
				"  ARRAY[[1], [2]],",
				"  ARRAY(",
				"    SELECT",
				"      1",
				"  ),",
				"  -a,",
				"  - -a,",
				"  ~ b,",
				"  a OPERATOR(pg_catalog.+) b,",
				"  a OPERATOR(+) b,",
				`  a COLLATE "C",`,
				"  a AT TIME ZONE 'UTC',",
				"  (a, b) OVERLAPS (c, d),",
				"  f(x => 1, y := 2),",
				"  a || b,",
				"  2 ^ 3",
				"FROM",
				"  t",
			),
		},
		{
			name: "predicates",
			in:   "select a is true, a is not unknown, a is not distinct from b, a is nfc normalized, a is not normalized, a not ilike b, a ilike b, a similar to b escape c, a like 'x' escape '!', a like any (array['x']), a <> all (select 1), a not between symmetric 1 and 2, a between 1 and 2, a isnull from t",
			want: join(
				"SELECT",
				"  a IS TRUE,",
				"  a IS NOT UNKNOWN,",
				"  a IS NOT DISTINCT FROM b,",
				"  a IS NFC NORMALIZED,",
				"  a IS NOT NORMALIZED,",
				"  a NOT ILIKE b,",
				"  a ILIKE b,",
				"  a SIMILAR TO b ESCAPE c,",
				"  a LIKE 'x' ESCAPE '!',",
				"  a LIKE ANY (ARRAY['x']),",
				"  a <> ALL (",
				"    SELECT",
				"      1",
				"  ),",
				"  a NOT BETWEEN SYMMETRIC 1 AND 2,",
				"  a BETWEEN 1 AND 2,",
				"  a IS NULL",
				"FROM",
				"  t",
			),
		},
		{
			name: "function forms",
			in:   "select extract(epoch from t), substring(s from 1 for 2), substring(s similar 'a' escape '#'), position('a' in s), trim(both 'x' from s), trim(leading from s), trim(s), overlay(s placing 'x' from 1 for 2), overlay(s placing 'x' from 1) from t",
			want: join(
				"SELECT",
				"  EXTRACT(epoch FROM t),",
				"  SUBSTRING(s FROM 1 FOR 2),",
				"  SUBSTRING(s SIMILAR 'a' ESCAPE '#'),",
				"  POSITION('a' IN s),",
				"  TRIM(BOTH 'x' FROM s),",
				"  TRIM(LEADING FROM s),",
				"  trim(s),",
				"  OVERLAY(s PLACING 'x' FROM 1 FOR 2),",
				"  OVERLAY(s PLACING 'x' FROM 1)",
				"FROM",
				"  t",
			),
		},
		{
			name: "aggregates",
			in:   `select count(distinct a), string_agg(a, ',' order by a), percentile_cont(0.5) within group (order by a desc), count(*) filter (where a > 0), f(variadic arr), f(a, variadic arr), "count"(a), pg_catalog.count(a), max(a) from t`,
			want: join(
				"SELECT",
				"  COUNT(DISTINCT a),",
				"  string_agg(a, ',' ORDER BY a),",
				"  percentile_cont(0.5) WITHIN GROUP (ORDER BY a DESC),",
				"  COUNT(*) FILTER (WHERE a > 0),",
				"  f(VARIADIC arr),",
				"  f(a, VARIADIC arr),",
				`  "count"(a),`,
				"  pg_catalog.count(a),",
				"  MAX(a)",
				"FROM",
				"  t",
			),
		},
		{
			name: "types",
			in:   `select 1::numeric(10, 2), '1'::timestamp with time zone, 'x'::double precision[], interval '1' day to second, date '2020-01-01', CAST(a AS VARCHAR(3)), 1::"MyType", 1::INTEGER[3]`,
			want: join(
				"SELECT",
				"  1::numeric(10, 2),",
				"  '1'::timestamp with time zone,",
				"  'x'::double precision[],",
				"  interval '1' day to second,",
				"  date '2020-01-01',",
				"  CAST(a AS VARCHAR(3)),",
				`  1::"MyType",`,
				"  1::INTEGER[3]",
			),
		},
		{
			name: "literals as written",
			in:   `select $$a 'b'$$, E'x\'y', B'101', X'1F', 1e10, .5, N'nat', NULL, true from t`,
			want: join(
				"SELECT",
				"  $$a 'b'$$,",
				`  E'x\'y',`,
				"  B'101',",
				"  X'1F',",
				"  1e10,",
				"  .5,",
				"  N'nat',",
				"  NULL,",
				"  TRUE",
				"FROM",
				"  t",
			),
		},
		{
			name: "placeholders and jsonb operators",
			in:   "select a from t where id = ? and data ? 'k' and data ?| array['a'] and data ?& array['b'] and b = $2",
			want: join(
				"SELECT",
				"  a",
				"FROM",
				"  t",
				"WHERE",
				"  id = ?",
				"  AND data ? 'k'",
				"  AND data ?| ARRAY['a']",
				"  AND data ?& ARRAY['b']",
				"  AND b = $2",
			),
		},
		{
			name: "identifiers as written",
			in:   `SELECT Id, "Name", t.* FROM "Users" t WHERE "Users"."Id" = 1 AND LOWER(Name) = 'x' AND s.t.c = 1`,
			want: join(
				"SELECT",
				"  Id,",
				`  "Name",`,
				"  t.*",
				"FROM",
				`  "Users" t`,
				"WHERE",
				`  "Users"."Id" = 1`,
				"  AND LOWER(Name) = 'x'",
				"  AND s.t.c = 1",
			),
		},
		{
			name: "where or chain",
			in:   "select a from t where a = 1 or b = 2 and c = 3 or not d",
			want: join(
				"SELECT",
				"  a",
				"FROM",
				"  t",
				"WHERE",
				"  a = 1",
				"  OR b = 2 AND c = 3",
				"  OR NOT d",
			),
		},
	})
}

func TestFormatPostgreSQL_Insert(t *testing.T) {
	assertPGFormats(t, []pgCase{
		{
			name: "on conflict do update",
			in:   "insert into t as x (a, b) values ($1, $2), (3, default) on conflict (a) where a > 0 do update set b = excluded.b where x.b <> excluded.b returning *",
			want: join(
				"INSERT INTO",
				"  t AS x",
				"(",
				"  a,",
				"  b",
				")",
				"VALUES",
				"  ($1, $2),",
				"  (3, DEFAULT)",
				"ON CONFLICT (a) WHERE a > 0 DO UPDATE SET",
				"  b = excluded.b",
				"WHERE",
				"  x.b <> excluded.b",
				"RETURNING",
				"  *",
			),
		},
		{
			name: "on conflict on constraint do nothing",
			in:   "insert into t default values on conflict on constraint t_pk do nothing returning id",
			want: join(
				"INSERT INTO",
				"  t",
				"DEFAULT VALUES",
				"ON CONFLICT ON CONSTRAINT t_pk DO NOTHING",
				"RETURNING",
				"  id",
			),
		},
		{
			name: "on conflict index elements",
			in:   `insert into t values (1) on conflict (lower(a) collate "C" text_pattern_ops, (b + 1)) do nothing`,
			want: join(
				"INSERT INTO",
				"  t",
				"VALUES",
				"  (1)",
				`ON CONFLICT (lower(a) COLLATE "C" text_pattern_ops, (b + 1)) DO NOTHING`,
			),
		},
		{
			name: "on conflict without target",
			in:   "insert into t values (1) on conflict do nothing",
			want: join(
				"INSERT INTO",
				"  t",
				"VALUES",
				"  (1)",
				"ON CONFLICT DO NOTHING",
			),
		},
		{
			name: "select",
			in:   "insert into t select * from s",
			want: join(
				"INSERT INTO",
				"  t",
				"SELECT",
				"  *",
				"FROM",
				"  s",
			),
		},
		{
			name: "set operation",
			in:   "insert into t select 1 union select 2",
			want: join(
				"INSERT INTO",
				"  t",
				"SELECT",
				"  1",
				"UNION",
				"SELECT",
				"  2",
			),
		},
		{
			name: "column indirection",
			in:   "insert into t (a.b, c[1]) values (1, 2)",
			want: join(
				"INSERT INTO",
				"  t",
				"(",
				"  a.b,",
				"  c[1]",
				")",
				"VALUES",
				"  (1, 2)",
			),
		},
	})
}

func TestFormatPostgreSQL_Update(t *testing.T) {
	assertPGFormats(t, []pgCase{
		{
			name: "from and returning",
			in:   "update only t set a = 1, (b, c) = (select 1, 2) from s where t.id = s.id returning t.*",
			want: join(
				"UPDATE",
				"  ONLY t",
				"SET",
				"  a = 1,",
				"  (b, c) = (",
				"    SELECT",
				"      1,",
				"      2",
				"  )",
				"FROM",
				"  s",
				"WHERE",
				"  t.id = s.id",
				"RETURNING",
				"  t.*",
			),
		},
		{
			name: "target indirection",
			in:   "update t set a.b = 1, c[1] = 2 where current_flag",
			want: join(
				"UPDATE",
				"  t",
				"SET",
				"  a.b = 1,",
				"  c[1] = 2",
				"WHERE",
				"  current_flag",
			),
		},
		{
			name: "alias named set",
			in:   "update t as set " + "set a = 1", // split so the repeated-word linter accepts the intended "set set"
			want: join(
				"UPDATE",
				"  t AS set",
				"SET",
				"  a = 1",
			),
		},
		{
			name: "with",
			in:   "with s as (select 1 as id) update t x set a = default where x.id in (select id from s)",
			want: join(
				"WITH",
				"  s AS (",
				"    SELECT",
				"      1 AS id",
				"  )",
				"UPDATE",
				"  t x",
				"SET",
				"  a = DEFAULT",
				"WHERE",
				"  x.id IN (",
				"    SELECT",
				"      id",
				"    FROM",
				"      s",
				"  )",
			),
		},
	})
}

func TestFormatPostgreSQL_Delete(t *testing.T) {
	assertPGFormats(t, []pgCase{
		{
			name: "using and returning",
			in:   "delete from t using s, u where t.id = s.id returning t.id as deleted_id",
			want: join(
				"DELETE FROM",
				"  t",
				"USING",
				"  s,",
				"  u",
				"WHERE",
				"  t.id = s.id",
				"RETURNING",
				"  t.id AS deleted_id",
			),
		},
		{
			name: "only with alias",
			in:   "delete from only t x where x.a = 1",
			want: join(
				"DELETE FROM",
				"  ONLY t x",
				"WHERE",
				"  x.a = 1",
			),
		},
		{
			name: "alias named set",
			in:   "delete from t as set",
			want: join(
				"DELETE FROM",
				"  t AS set",
			),
		},
	})
}

func TestFormatPostgreSQL_Merge(t *testing.T) {
	assertPGFormats(t, []pgCase{
		{
			name: "all actions",
			in:   "merge into t using s on t.id = s.id when matched and s.d then delete when matched then update set a = s.a, b = s.b when not matched then insert (id, a) values (s.id, s.a) when not matched by source then do nothing returning merge_action(), t.*",
			want: join(
				"MERGE INTO",
				"  t",
				"USING",
				"  s",
				"ON",
				"  t.id = s.id",
				"WHEN MATCHED AND s.d THEN",
				"  DELETE",
				"WHEN MATCHED THEN",
				"  UPDATE SET",
				"    a = s.a,",
				"    b = s.b",
				"WHEN NOT MATCHED THEN",
				"  INSERT (id, a) VALUES (s.id, s.a)",
				"WHEN NOT MATCHED BY SOURCE THEN",
				"  DO NOTHING",
				"RETURNING",
				"  merge_action(),",
				"  t.*",
			),
		},
		{
			name: "derived source and insert forms",
			in:   "merge into t as tt using (select 1 as id) s on tt.id = s.id and tt.x when not matched by target then insert default values when not matched then insert (a) overriding user value values (1) when matched then do nothing",
			want: join(
				"MERGE INTO",
				"  t tt",
				"USING",
				"  (",
				"  SELECT",
				"    1 AS id",
				"  ) s",
				"ON",
				"  tt.id = s.id",
				"  AND tt.x",
				"WHEN NOT MATCHED BY TARGET THEN",
				"  INSERT DEFAULT VALUES",
				"WHEN NOT MATCHED THEN",
				"  INSERT (a) OVERRIDING USER VALUE VALUES (1)",
				"WHEN MATCHED THEN",
				"  DO NOTHING",
			),
		},
		{
			name: "returning with aliases",
			in:   "with v as (select 1 as id) merge into t using v on t.id = v.id when matched then delete returning with (old as o, new as n) o.id",
			want: join(
				"WITH",
				"  v AS (",
				"    SELECT",
				"      1 AS id",
				"  )",
				"MERGE INTO",
				"  t",
				"USING",
				"  v",
				"ON",
				"  t.id = v.id",
				"WHEN MATCHED THEN",
				"  DELETE",
				"RETURNING WITH (OLD AS o, NEW AS n)",
				"  o.id",
			),
		},
	})
}

func TestFormatPostgreSQL_OrderByUsingAndIntervalCast(t *testing.T) {
	assertPGFormats(t, []pgCase{
		{
			name: "order by using",
			in:   "select a::interval day from t order by a using <",
			want: join(
				"SELECT",
				"  a::interval day",
				"FROM",
				"  t",
				"ORDER BY",
				"  a USING <",
			),
		},
	})
}

func TestFormatPostgreSQL_Options(t *testing.T) {
	in := "with a as (select 1), b as (select 2) select x as y, z from a join b on a.id = b.id " +
		"where x is null and y like 'p' or not z in (1, 2) order by x asc, z desc"

	tests := []struct {
		name string
		opts sqlfmt.Options
		want string
	}{
		{
			name: "lower keyword case and leading commas",
			opts: sqlfmt.Options{
				Indent: 2, Dialect: sqlfmt.DialectPostgreSQL,
				KeywordCase: sqlfmt.KeywordCaseLower, CommaStyle: sqlfmt.CommaStyleLeading,
			},
			want: join(
				"WITH",
				"  a as (",
				"    SELECT",
				"      1",
				"  )",
				", b as (",
				"    SELECT",
				"      2",
				"  )",
				"SELECT",
				"  x as y",
				", z",
				"FROM",
				"  a",
				"  JOIN",
				"  b",
				"    on a.id = b.id",
				"WHERE",
				"  x is null",
				"  and y like 'p'",
				"  or not z in (1, 2)",
				"ORDER BY",
				"  x",
				", z desc",
			),
		},
		{
			name: "indent 4",
			opts: sqlfmt.Options{Indent: 4, Dialect: sqlfmt.DialectPostgreSQL},
			want: join(
				"WITH",
				"    a AS (",
				"        SELECT",
				"            1",
				"    ),",
				"    b AS (",
				"        SELECT",
				"            2",
				"    )",
				"SELECT",
				"    x AS y,",
				"    z",
				"FROM",
				"    a",
				"    JOIN",
				"    b",
				"        ON a.id = b.id",
				"WHERE",
				"    x IS NULL",
				"    AND y LIKE 'p'",
				"    OR NOT z IN (1, 2)",
				"ORDER BY",
				"    x,",
				"    z DESC",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sqlfmt.FormatSQLWithOptions(in, tt.opts)
			if !ok {
				t.Fatal("ok = false, want true")
			}

			assertSQL(t, got, tt.want)
		})
	}
}

// TestFormatPostgreSQL_LeftUnchanged covers SQL the PostgreSQL dialect
// returns unchanged with ok == false: SQL with a comment, which the AST
// cannot keep, statements other than the DML pgparser supports, and input
// that does not parse.
func TestFormatPostgreSQL_LeftUnchanged(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"line comment", "select id -- the id\nfrom t"},
		{"block comment", "select /* the id */ id from t"},
		{"DDL", "create table t (id int)"},
		{"session statement", "set search_path = app"},
		{"parse error", "select from where"},
		{"MySQL backtick identifier", "select `id` from t"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := sqlfmt.FormatSQLWithOptions(tt.in, pgOptions())
			if ok {
				t.Fatalf("FormatSQLWithOptions(%q) ok = true, want false", tt.in)
			}

			if got != tt.in {
				t.Errorf("FormatSQLWithOptions(%q) = %q, want input unchanged", tt.in, got)
			}
		})
	}
}

// TestFormatPostgreSQL_CommentMarkersInLiterals covers comment-like text
// inside a string constant or quoted identifier, which is not a comment.
func TestFormatPostgreSQL_CommentMarkersInLiterals(t *testing.T) {
	assertPGFormats(t, []pgCase{
		{
			name: "string and quoted identifier",
			in:   `select '-- x /* y */' as "a--b" from t`,
			want: join(
				"SELECT",
				`  '-- x /* y */' AS "a--b"`,
				"FROM",
				"  t",
			),
		},
	})
}

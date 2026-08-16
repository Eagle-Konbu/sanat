package sqlast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

func TestColIdent(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		c := sqlast.ColIdent("id")
		assertEqual(t, "id", c.String())
	})

	t.Run("IsEmpty", func(t *testing.T) {
		assertEqual(t, true, sqlast.ColIdent("").IsEmpty())
		assertEqual(t, false, sqlast.ColIdent("id").IsEmpty())
	})

	t.Run("String quotes a reserved word", func(t *testing.T) {
		assertEqual(t, "`select`", sqlast.ColIdent("select").String())
	})

	t.Run("String quotes a reserved word case-insensitively", func(t *testing.T) {
		assertEqual(t, "`Select`", sqlast.ColIdent("Select").String())
	})

	t.Run("String quotes an identifier with a space", func(t *testing.T) {
		assertEqual(t, "`order details`", sqlast.ColIdent("order details").String())
	})

	t.Run("String quotes an identifier starting with a digit", func(t *testing.T) {
		assertEqual(t, "`1id`", sqlast.ColIdent("1id").String())
	})

	t.Run("String escapes an embedded backtick", func(t *testing.T) {
		assertEqual(t, "`a``b`", sqlast.ColIdent("a`b").String())
	})

	t.Run("String leaves a plain identifier unquoted", func(t *testing.T) {
		assertEqual(t, "user_id2", sqlast.ColIdent("user_id2").String())
	})

	t.Run("String leaves a non-reserved keyword-adjacent identifier unquoted", func(t *testing.T) {
		assertEqual(t, "selection", sqlast.ColIdent("selection").String())
	})

	t.Run("String quotes an empty identifier", func(t *testing.T) {
		assertEqual(t, "``", sqlast.ColIdent("").String())
	})
}

func TestTableIdent(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		ti := sqlast.TableIdent("users")
		assertEqual(t, "users", ti.String())
	})

	t.Run("IsEmpty", func(t *testing.T) {
		assertEqual(t, true, sqlast.TableIdent("").IsEmpty())
		assertEqual(t, false, sqlast.TableIdent("users").IsEmpty())
	})

	t.Run("String quotes a reserved word", func(t *testing.T) {
		assertEqual(t, "`order`", sqlast.TableIdent("order").String())
	})

	t.Run("String quotes an identifier with a space", func(t *testing.T) {
		assertEqual(t, "`order details`", sqlast.TableIdent("order details").String())
	})
}

func TestTableName(t *testing.T) {
	t.Run("unqualified", func(t *testing.T) {
		tn := sqlast.TableName{Name: "users"}
		assertEqual(t, "users", tn.String())
		assertEqual(t, false, tn.IsEmpty())
	})

	t.Run("qualified", func(t *testing.T) {
		tn := sqlast.TableName{Name: "users", Qualifier: "mydb"}
		assertEqual(t, "mydb.users", tn.String())
	})

	t.Run("empty", func(t *testing.T) {
		tn := sqlast.TableName{}
		assertEqual(t, true, tn.IsEmpty())
	})

	t.Run("qualified with a reserved word", func(t *testing.T) {
		tn := sqlast.TableName{Name: "order", Qualifier: "mydb"}
		assertEqual(t, "mydb.`order`", tn.String())
	})
}

func assertEqual[T comparable](t *testing.T, want, got T) {
	t.Helper()

	if want != got {
		t.Errorf("want %v, got %v", want, got)
	}
}

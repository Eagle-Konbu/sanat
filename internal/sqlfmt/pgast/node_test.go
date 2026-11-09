package pgast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestColIdent(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		c := pgast.ColIdent{Name: "id", Raw: "ID"}
		assertEqual(t, "id", c.String())
	})

	t.Run("IsEmpty", func(t *testing.T) {
		assertEqual(t, true, pgast.ColIdent{Name: ""}.IsEmpty())
		assertEqual(t, false, pgast.ColIdent{Name: "id"}.IsEmpty())
	})
}

func TestTableIdent(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		ti := pgast.TableIdent{Name: "users", Raw: "Users"}
		assertEqual(t, "users", ti.String())
	})

	t.Run("IsEmpty", func(t *testing.T) {
		assertEqual(t, true, pgast.TableIdent{Name: ""}.IsEmpty())
		assertEqual(t, false, pgast.TableIdent{Name: "users"}.IsEmpty())
	})
}

func TestTableName(t *testing.T) {
	t.Run("unqualified", func(t *testing.T) {
		tn := pgast.TableName{Name: pgast.TableIdent{Name: "users"}}
		assertEqual(t, "users", tn.String())
		assertEqual(t, false, tn.IsEmpty())
	})

	t.Run("qualified", func(t *testing.T) {
		tn := pgast.TableName{Name: pgast.TableIdent{Name: "users"}, Qualifier: pgast.TableIdent{Name: "public"}}
		assertEqual(t, "public.users", tn.String())
	})

	t.Run("empty", func(t *testing.T) {
		tn := pgast.TableName{}
		assertEqual(t, true, tn.IsEmpty())
	})
}

func TestColumns(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		assertEqual(t, "", pgast.Columns{}.String())
	})

	t.Run("multiple", func(t *testing.T) {
		c := pgast.Columns{pgast.ColIdent{Name: "id"}, pgast.ColIdent{Name: "name"}}
		assertEqual(t, "id, name", c.String())
	})
}

func assertEqual[T comparable](t *testing.T, want, got T) {
	t.Helper()

	if want != got {
		t.Errorf("want %v, got %v", want, got)
	}
}

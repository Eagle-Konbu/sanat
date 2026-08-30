package pgast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/pgast"
)

func TestColIdent(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		c := pgast.ColIdent("id")
		assertEqual(t, "id", c.String())
	})

	t.Run("IsEmpty", func(t *testing.T) {
		assertEqual(t, true, pgast.ColIdent("").IsEmpty())
		assertEqual(t, false, pgast.ColIdent("id").IsEmpty())
	})
}

func TestTableIdent(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		ti := pgast.TableIdent("users")
		assertEqual(t, "users", ti.String())
	})

	t.Run("IsEmpty", func(t *testing.T) {
		assertEqual(t, true, pgast.TableIdent("").IsEmpty())
		assertEqual(t, false, pgast.TableIdent("users").IsEmpty())
	})
}

func TestTableName(t *testing.T) {
	t.Run("unqualified", func(t *testing.T) {
		tn := pgast.TableName{Name: "users"}
		assertEqual(t, "users", tn.String())
		assertEqual(t, false, tn.IsEmpty())
	})

	t.Run("qualified", func(t *testing.T) {
		tn := pgast.TableName{Name: "users", Qualifier: "public"}
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
		c := pgast.Columns{"id", "name"}
		assertEqual(t, "id, name", c.String())
	})
}

func assertEqual[T comparable](t *testing.T, want, got T) {
	t.Helper()

	if want != got {
		t.Errorf("want %v, got %v", want, got)
	}
}

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
}

func assertEqual[T comparable](t *testing.T, want, got T) {
	t.Helper()

	if want != got {
		t.Errorf("want %v, got %v", want, got)
	}
}

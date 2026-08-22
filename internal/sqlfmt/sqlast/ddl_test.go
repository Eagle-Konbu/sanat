package sqlast_test

import (
	"testing"

	"github.com/Eagle-Konbu/sanat/internal/sqlfmt/sqlast"
)

func TestCreateTable_String(t *testing.T) {
	tests := []struct {
		name string
		c    *sqlast.CreateTable
		want string
	}{
		{
			name: "simple",
			c: &sqlast.CreateTable{
				Table: sqlast.TableName{Name: "users"},
				Elements: []sqlast.TableElement{
					&sqlast.ColumnDef{Name: "id", Type: sqlast.DataType{Name: "INT"}},
				},
			},
			want: "CREATE TABLE users (id INT)",
		},
		{
			name: "if not exists with options",
			c: &sqlast.CreateTable{
				IfNotExists: true,
				Table:       sqlast.TableName{Name: "users"},
				Elements: []sqlast.TableElement{
					&sqlast.ColumnDef{Name: "id", Type: sqlast.DataType{Name: "INT"}},
					&sqlast.PrimaryKeyConstraint{Columns: []sqlast.IndexColumn{{Column: "id"}}},
				},
				Options: []sqlast.TableOption{
					{Name: "ENGINE", Value: lit("InnoDB")},
					{Name: "DEFAULT CHARSET", Value: lit("utf8mb4")},
				},
			},
			want: "CREATE TABLE IF NOT EXISTS users (id INT, PRIMARY KEY (id)) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.String(); got != tt.want {
				t.Errorf("CreateTable.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAlterTable_String(t *testing.T) {
	tests := []struct {
		name string
		a    *sqlast.AlterTable
		want string
	}{
		{
			name: "single action",
			a: &sqlast.AlterTable{
				Table: sqlast.TableName{Name: "users"},
				Actions: []sqlast.AlterAction{
					&sqlast.AddColumnAction{Column: &sqlast.ColumnDef{Name: "age", Type: sqlast.DataType{Name: "INT"}}},
				},
			},
			want: "ALTER TABLE users ADD COLUMN age INT",
		},
		{
			name: "multiple actions",
			a: &sqlast.AlterTable{
				Table: sqlast.TableName{Name: "users"},
				Actions: []sqlast.AlterAction{
					&sqlast.DropColumnAction{Name: "age"},
					&sqlast.RenameTableAction{NewName: sqlast.TableName{Name: "people"}},
				},
			},
			want: "ALTER TABLE users DROP COLUMN age, RENAME TO people",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.String(); got != tt.want {
				t.Errorf("AlterTable.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCreateIndex_String(t *testing.T) {
	tests := []struct {
		name string
		c    *sqlast.CreateIndex
		want string
	}{
		{
			name: "plain",
			c: &sqlast.CreateIndex{
				Name:    "idx_name",
				Table:   sqlast.TableName{Name: "users"},
				Columns: []sqlast.IndexColumn{{Column: "name"}},
			},
			want: "CREATE INDEX idx_name ON users (name)",
		},
		{
			name: "unique",
			c: &sqlast.CreateIndex{
				Unique:  true,
				Name:    "idx_email",
				Table:   sqlast.TableName{Name: "users"},
				Columns: []sqlast.IndexColumn{{Column: "email"}},
			},
			want: "CREATE UNIQUE INDEX idx_email ON users (email)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.String(); got != tt.want {
				t.Errorf("CreateIndex.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDropIndex_String(t *testing.T) {
	d := &sqlast.DropIndex{Name: "idx_name", Table: sqlast.TableName{Name: "users"}}

	want := "DROP INDEX idx_name ON users"
	if got := d.String(); got != want {
		t.Errorf("DropIndex.String() = %q, want %q", got, want)
	}
}

func TestDropTable_String(t *testing.T) {
	tests := []struct {
		name string
		d    *sqlast.DropTable
		want string
	}{
		{
			name: "single",
			d:    &sqlast.DropTable{Tables: []sqlast.TableName{{Name: "users"}}},
			want: "DROP TABLE users",
		},
		{
			name: "if exists multiple",
			d: &sqlast.DropTable{
				IfExists: true,
				Tables:   []sqlast.TableName{{Name: "users"}, {Name: "orders"}},
			},
			want: "DROP TABLE IF EXISTS users, orders",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.String(); got != tt.want {
				t.Errorf("DropTable.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTruncateTable_String(t *testing.T) {
	tr := &sqlast.TruncateTable{Table: sqlast.TableName{Name: "users"}}

	want := "TRUNCATE TABLE users"
	if got := tr.String(); got != want {
		t.Errorf("TruncateTable.String() = %q, want %q", got, want)
	}
}

func TestColumnDef_String(t *testing.T) {
	tests := []struct {
		name string
		c    *sqlast.ColumnDef
		want string
	}{
		{
			name: "bare",
			c:    &sqlast.ColumnDef{Name: "id", Type: sqlast.DataType{Name: "INT"}},
			want: "id INT",
		},
		{
			name: "not null",
			c: &sqlast.ColumnDef{
				Name: "name", Type: sqlast.DataType{Name: "VARCHAR", Params: []string{"255"}},
				Nullability: sqlast.NotNullConstraint,
			},
			want: "name VARCHAR(255) NOT NULL",
		},
		{
			name: "explicit null",
			c:    &sqlast.ColumnDef{Name: "name", Type: sqlast.DataType{Name: "VARCHAR"}, Nullability: sqlast.NullConstraint},
			want: "name VARCHAR NULL",
		},
		{
			name: "default and auto increment",
			c: &sqlast.ColumnDef{
				Name: "id", Type: sqlast.DataType{Name: "INT"},
				Default: lit("0"), AutoIncrement: true, PrimaryKey: true,
			},
			want: "id INT DEFAULT 0 AUTO_INCREMENT PRIMARY KEY",
		},
		{
			name: "unique with comment",
			c: &sqlast.ColumnDef{
				Name: "email", Type: sqlast.DataType{Name: "VARCHAR", Params: []string{"255"}},
				Unique: true, Comment: lit("'the email'"),
			},
			want: "email VARCHAR(255) UNIQUE COMMENT 'the email'",
		},
		{
			name: "on update",
			c: &sqlast.ColumnDef{
				Name: "updated_at", Type: sqlast.DataType{Name: "TIMESTAMP"},
				OnUpdate: lit("CURRENT_TIMESTAMP"),
			},
			want: "updated_at TIMESTAMP ON UPDATE CURRENT_TIMESTAMP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.String(); got != tt.want {
				t.Errorf("ColumnDef.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDataType_String(t *testing.T) {
	tests := []struct {
		name string
		d    sqlast.DataType
		want string
	}{
		{"bare", sqlast.DataType{Name: "INT"}, "INT"},
		{"params", sqlast.DataType{Name: "DECIMAL", Params: []string{"10", "2"}}, "DECIMAL(10, 2)"},
		{"enum params", sqlast.DataType{Name: "ENUM", Params: []string{"'a'", "'b'"}}, "ENUM('a', 'b')"},
		{
			"unsigned zerofill",
			sqlast.DataType{Name: "INT", Unsigned: true, Zerofill: true},
			"INT UNSIGNED ZEROFILL",
		},
		{
			"charset and collate",
			sqlast.DataType{Name: "VARCHAR", Params: []string{"255"}, Charset: "utf8mb4", Collate: "utf8mb4_bin"},
			"VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.String(); got != tt.want {
				t.Errorf("DataType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIndexColumn_String(t *testing.T) {
	tests := []struct {
		name string
		i    sqlast.IndexColumn
		want string
	}{
		{"bare", sqlast.IndexColumn{Column: "name"}, "name"},
		{"prefix length", sqlast.IndexColumn{Column: "name", Length: 20}, "name(20)"},
		{"desc", sqlast.IndexColumn{Column: "name", Desc: true}, "name DESC"},
		{"prefix and desc", sqlast.IndexColumn{Column: "name", Length: 20, Desc: true}, "name(20) DESC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.i.String(); got != tt.want {
				t.Errorf("IndexColumn.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrimaryKeyConstraint_String(t *testing.T) {
	tests := []struct {
		name string
		c    *sqlast.PrimaryKeyConstraint
		want string
	}{
		{
			name: "bare",
			c:    &sqlast.PrimaryKeyConstraint{Columns: []sqlast.IndexColumn{{Column: "id"}}},
			want: "PRIMARY KEY (id)",
		},
		{
			name: "named constraint",
			c: &sqlast.PrimaryKeyConstraint{
				ConstraintName: "pk_users",
				Columns:        []sqlast.IndexColumn{{Column: "id"}, {Column: "tenant_id"}},
			},
			want: "CONSTRAINT pk_users PRIMARY KEY (id, tenant_id)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.String(); got != tt.want {
				t.Errorf("PrimaryKeyConstraint.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestUniqueConstraint_String(t *testing.T) {
	tests := []struct {
		name string
		c    *sqlast.UniqueConstraint
		want string
	}{
		{
			name: "unnamed",
			c:    &sqlast.UniqueConstraint{Columns: []sqlast.IndexColumn{{Column: "email"}}},
			want: "UNIQUE KEY (email)",
		},
		{
			name: "named index and constraint",
			c: &sqlast.UniqueConstraint{
				ConstraintName: "uq_email",
				IndexName:      "idx_email",
				Columns:        []sqlast.IndexColumn{{Column: "email"}},
			},
			want: "CONSTRAINT uq_email UNIQUE KEY idx_email (email)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.String(); got != tt.want {
				t.Errorf("UniqueConstraint.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIndexConstraint_String(t *testing.T) {
	tests := []struct {
		name string
		c    *sqlast.IndexConstraint
		want string
	}{
		{
			name: "unnamed",
			c:    &sqlast.IndexConstraint{Columns: []sqlast.IndexColumn{{Column: "name"}}},
			want: "INDEX (name)",
		},
		{
			name: "named",
			c:    &sqlast.IndexConstraint{IndexName: "idx_name", Columns: []sqlast.IndexColumn{{Column: "name"}}},
			want: "INDEX idx_name (name)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.String(); got != tt.want {
				t.Errorf("IndexConstraint.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestForeignKeyConstraint_String(t *testing.T) {
	tests := []struct {
		name string
		c    *sqlast.ForeignKeyConstraint
		want string
	}{
		{
			name: "bare",
			c: &sqlast.ForeignKeyConstraint{
				Columns:    sqlast.Columns{"user_id"},
				RefTable:   sqlast.TableName{Name: "users"},
				RefColumns: sqlast.Columns{"id"},
			},
			want: "FOREIGN KEY (user_id) REFERENCES users (id)",
		},
		{
			name: "named with index name and actions",
			c: &sqlast.ForeignKeyConstraint{
				ConstraintName: "fk_user",
				IndexName:      "idx_fk_user",
				Columns:        sqlast.Columns{"user_id"},
				RefTable:       sqlast.TableName{Name: "users"},
				RefColumns:     sqlast.Columns{"id"},
				OnDelete:       sqlast.CascadeAction,
				OnUpdate:       sqlast.RestrictAction,
			},
			want: "CONSTRAINT fk_user FOREIGN KEY idx_fk_user (user_id) REFERENCES users (id) ON DELETE CASCADE ON UPDATE RESTRICT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.String(); got != tt.want {
				t.Errorf("ForeignKeyConstraint.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReferenceAction_String(t *testing.T) {
	tests := []struct {
		name string
		r    sqlast.ReferenceAction
		want string
	}{
		{"cascade", sqlast.CascadeAction, "CASCADE"},
		{"set null", sqlast.SetNullAction, "SET NULL"},
		{"restrict", sqlast.RestrictAction, "RESTRICT"},
		{"no action", sqlast.NoActionAction, "NO ACTION"},
		{"set default", sqlast.SetDefaultAction, "SET DEFAULT"},
		{"none", sqlast.NoRefAction, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.String(); got != tt.want {
				t.Errorf("ReferenceAction.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTableOption_String(t *testing.T) {
	o := sqlast.TableOption{Name: "ENGINE", Value: lit("InnoDB")}

	want := "ENGINE=InnoDB"
	if got := o.String(); got != want {
		t.Errorf("TableOption.String() = %q, want %q", got, want)
	}
}

func TestAlterActions_String(t *testing.T) {
	tests := []struct {
		name string
		a    sqlast.AlterAction
		want string
	}{
		{
			name: "add column",
			a:    &sqlast.AddColumnAction{Column: &sqlast.ColumnDef{Name: "age", Type: sqlast.DataType{Name: "INT"}}},
			want: "ADD COLUMN age INT",
		},
		{
			name: "add constraint",
			a: &sqlast.AddConstraintAction{
				Constraint: &sqlast.IndexConstraint{IndexName: "idx_name", Columns: []sqlast.IndexColumn{{Column: "name"}}},
			},
			want: "ADD INDEX idx_name (name)",
		},
		{
			name: "drop column",
			a:    &sqlast.DropColumnAction{Name: "age"},
			want: "DROP COLUMN age",
		},
		{
			name: "drop index",
			a:    &sqlast.DropIndexAction{Name: "idx_name"},
			want: "DROP INDEX idx_name",
		},
		{
			name: "modify column",
			a: &sqlast.ModifyColumnAction{
				Column: &sqlast.ColumnDef{Name: "age", Type: sqlast.DataType{Name: "BIGINT"}},
			},
			want: "MODIFY COLUMN age BIGINT",
		},
		{
			name: "rename to",
			a:    &sqlast.RenameTableAction{NewName: sqlast.TableName{Name: "people"}},
			want: "RENAME TO people",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.String(); got != tt.want {
				t.Errorf("%T.String() = %q, want %q", tt.a, got, tt.want)
			}
		})
	}
}

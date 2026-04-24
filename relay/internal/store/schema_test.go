package store_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/appunvs/appunvs/relay/internal/store"
)

// openSchemaStore boots a fresh SQLite file and inserts a user so the
// foreign-key constraint on app_tables is satisfied.
func openSchemaStore(t *testing.T) (*store.Store, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "schema.db")
	st, err := store.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	u, err := st.Users().Create(context.Background(), "schema@example.com", "hunter22")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return st, u.ID
}

func TestSchemaCreateAndListTable(t *testing.T) {
	st, userID := openSchemaStore(t)
	sc := st.Schema()
	ctx := context.Background()

	tbl, err := sc.CreateTable(ctx, userID, "todos")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if tbl.Name != "todos" {
		t.Fatalf("name = %q", tbl.Name)
	}
	if tbl.CreatedAt.IsZero() {
		t.Fatalf("created_at not set")
	}

	got, err := sc.GetTable(ctx, userID, "todos")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "todos" || len(got.Columns) != 0 {
		t.Fatalf("get = %+v", got)
	}

	tables, err := sc.ListTables(ctx, userID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tables) != 1 || tables[0].Name != "todos" {
		t.Fatalf("list = %+v", tables)
	}
}

func TestSchemaCreateTableDuplicate(t *testing.T) {
	st, userID := openSchemaStore(t)
	sc := st.Schema()
	ctx := context.Background()
	if _, err := sc.CreateTable(ctx, userID, "todos"); err != nil {
		t.Fatalf("first: %v", err)
	}
	_, err := sc.CreateTable(ctx, userID, "todos")
	if !errors.Is(err, store.ErrTableExists) {
		t.Fatalf("want ErrTableExists, got %v", err)
	}
}

func TestSchemaReservedTableName(t *testing.T) {
	st, userID := openSchemaStore(t)
	sc := st.Schema()
	ctx := context.Background()
	for _, name := range []string{"_schema", "_meta", "_foo"} {
		_, err := sc.CreateTable(ctx, userID, name)
		if !errors.Is(err, store.ErrReservedName) {
			t.Fatalf("%q: want ErrReservedName, got %v", name, err)
		}
	}
}

func TestSchemaInvalidTableName(t *testing.T) {
	st, userID := openSchemaStore(t)
	sc := st.Schema()
	ctx := context.Background()
	for _, name := range []string{"", "Todos", "1todos", "to-dos", "todos!"} {
		_, err := sc.CreateTable(ctx, userID, name)
		if err == nil {
			t.Fatalf("%q: expected error", name)
		}
		if !errors.Is(err, store.ErrInvalidName) && !errors.Is(err, store.ErrReservedName) {
			t.Fatalf("%q: want invalid/reserved, got %v", name, err)
		}
	}
}

func TestSchemaAddAndListColumns(t *testing.T) {
	st, userID := openSchemaStore(t)
	sc := st.Schema()
	ctx := context.Background()
	if _, err := sc.CreateTable(ctx, userID, "todos"); err != nil {
		t.Fatalf("create: %v", err)
	}
	cases := []struct {
		name string
		typ  string
		req  bool
	}{
		{"title", store.ColumnTypeText, true},
		{"done", store.ColumnTypeBool, false},
		{"priority", store.ColumnTypeNumber, false},
		{"extra", store.ColumnTypeJSON, false},
	}
	for _, c := range cases {
		if _, err := sc.AddColumn(ctx, userID, "todos", c.name, c.typ, c.req); err != nil {
			t.Fatalf("add %s: %v", c.name, err)
		}
	}
	got, err := sc.GetTable(ctx, userID, "todos")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.Columns) != 4 {
		t.Fatalf("columns = %d", len(got.Columns))
	}
	// Find "title" and confirm Required=true.
	var titleReq bool
	for _, col := range got.Columns {
		if col.Name == "title" {
			titleReq = col.Required
		}
	}
	if !titleReq {
		t.Fatalf("title should be required")
	}
}

func TestSchemaAddColumnRejectedForReserved(t *testing.T) {
	st, userID := openSchemaStore(t)
	sc := st.Schema()
	ctx := context.Background()
	if _, err := sc.CreateTable(ctx, userID, "todos"); err != nil {
		t.Fatalf("create: %v", err)
	}
	for _, n := range []string{"id", "_hidden", "_"} {
		_, err := sc.AddColumn(ctx, userID, "todos", n, store.ColumnTypeText, false)
		if !errors.Is(err, store.ErrReservedName) {
			t.Fatalf("%q: want ErrReservedName, got %v", n, err)
		}
	}
}

func TestSchemaAddColumnUnknownType(t *testing.T) {
	st, userID := openSchemaStore(t)
	sc := st.Schema()
	ctx := context.Background()
	if _, err := sc.CreateTable(ctx, userID, "todos"); err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err := sc.AddColumn(ctx, userID, "todos", "foo", "float", false)
	if !errors.Is(err, store.ErrInvalidType) {
		t.Fatalf("want ErrInvalidType, got %v", err)
	}
}

func TestSchemaAddColumnMissingTable(t *testing.T) {
	st, userID := openSchemaStore(t)
	sc := st.Schema()
	ctx := context.Background()
	_, err := sc.AddColumn(ctx, userID, "ghost", "foo", store.ColumnTypeText, false)
	if !errors.Is(err, store.ErrTableNotFound) {
		t.Fatalf("want ErrTableNotFound, got %v", err)
	}
}

func TestSchemaDuplicateColumn(t *testing.T) {
	st, userID := openSchemaStore(t)
	sc := st.Schema()
	ctx := context.Background()
	if _, err := sc.CreateTable(ctx, userID, "todos"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := sc.AddColumn(ctx, userID, "todos", "title", store.ColumnTypeText, false); err != nil {
		t.Fatalf("first add: %v", err)
	}
	_, err := sc.AddColumn(ctx, userID, "todos", "title", store.ColumnTypeText, false)
	if !errors.Is(err, store.ErrColumnExists) {
		t.Fatalf("want ErrColumnExists, got %v", err)
	}
}

func TestSchemaDeleteColumn(t *testing.T) {
	st, userID := openSchemaStore(t)
	sc := st.Schema()
	ctx := context.Background()
	if _, err := sc.CreateTable(ctx, userID, "todos"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := sc.AddColumn(ctx, userID, "todos", "title", store.ColumnTypeText, false); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := sc.DeleteColumn(ctx, userID, "todos", "title"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// A second delete returns ErrColumnNotFound.
	if err := sc.DeleteColumn(ctx, userID, "todos", "title"); !errors.Is(err, store.ErrColumnNotFound) {
		t.Fatalf("second delete: want ErrColumnNotFound, got %v", err)
	}
}

func TestSchemaDeleteTableMissing(t *testing.T) {
	st, userID := openSchemaStore(t)
	sc := st.Schema()
	ctx := context.Background()
	if err := sc.DeleteTable(ctx, userID, "never"); !errors.Is(err, store.ErrTableNotFound) {
		t.Fatalf("want ErrTableNotFound, got %v", err)
	}
}

// DeleteTable must cascade to app_columns (ON DELETE CASCADE).
func TestSchemaDeleteTableCascade(t *testing.T) {
	st, userID := openSchemaStore(t)
	sc := st.Schema()
	ctx := context.Background()
	if _, err := sc.CreateTable(ctx, userID, "todos"); err != nil {
		t.Fatalf("create: %v", err)
	}
	for _, c := range []string{"title", "done"} {
		if _, err := sc.AddColumn(ctx, userID, "todos", c, store.ColumnTypeText, false); err != nil {
			t.Fatalf("add %s: %v", c, err)
		}
	}
	if err := sc.DeleteTable(ctx, userID, "todos"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	// Drop table of the same name, then confirm columns are gone.
	if _, err := sc.CreateTable(ctx, userID, "todos"); err != nil {
		t.Fatalf("re-create: %v", err)
	}
	got, err := sc.GetTable(ctx, userID, "todos")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.Columns) != 0 {
		t.Fatalf("columns survived cascade: %+v", got.Columns)
	}
}

// Two users with identical table names must not collide.
func TestSchemaPerUserIsolation(t *testing.T) {
	st, aID := openSchemaStore(t)
	sc := st.Schema()
	ctx := context.Background()
	b, err := st.Users().Create(ctx, "other@example.com", "hunter22")
	if err != nil {
		t.Fatalf("create b: %v", err)
	}
	if _, err := sc.CreateTable(ctx, aID, "todos"); err != nil {
		t.Fatalf("a.create: %v", err)
	}
	if _, err := sc.CreateTable(ctx, b.ID, "todos"); err != nil {
		t.Fatalf("b.create: %v", err)
	}
	// A's columns don't leak to B.
	if _, err := sc.AddColumn(ctx, aID, "todos", "title", store.ColumnTypeText, true); err != nil {
		t.Fatalf("a.add: %v", err)
	}
	bt, err := sc.GetTable(ctx, b.ID, "todos")
	if err != nil {
		t.Fatalf("b.get: %v", err)
	}
	if len(bt.Columns) != 0 {
		t.Fatalf("b's table leaked columns: %+v", bt.Columns)
	}
}

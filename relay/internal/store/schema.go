package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ErrTableExists is returned by Schema.CreateTable when the (user_id, name)
// pair is already present.
var ErrTableExists = errors.New("table already exists")

// ErrTableNotFound is returned when a table lookup misses.
var ErrTableNotFound = errors.New("table not found")

// ErrColumnExists is returned by Schema.AddColumn on a duplicate column.
var ErrColumnExists = errors.New("column already exists")

// ErrColumnNotFound is returned when a column lookup misses.
var ErrColumnNotFound = errors.New("column not found")

// ErrReservedName is returned when a caller supplies a name that matches
// one of the relay's reserved identifiers (leading underscore, or special
// values like "_schema", "id", etc).
var ErrReservedName = errors.New("reserved name")

// ErrInvalidName indicates the supplied identifier failed the regex check or
// exceeded the length limit.
var ErrInvalidName = errors.New("invalid name")

// ErrInvalidType is returned by Schema.AddColumn when the type is not one of
// the supported column types.
var ErrInvalidType = errors.New("invalid column type")

// Supported column types. Kept in a small set so clients (browser/desktop)
// don't have to juggle database flavors.
const (
	ColumnTypeText   = "text"
	ColumnTypeNumber = "number"
	ColumnTypeBool   = "bool"
	ColumnTypeJSON   = "json"
)

// nameRE is the canonical identifier rule for user-declared tables and
// columns: [a-z][a-z0-9_]{0,63}.  Underscores are allowed after the first
// character but a leading underscore is always reserved.
var nameRE = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// ValidTableName returns nil if name is a legal, non-reserved user table
// identifier. Reserved forms: leading underscore (including "_schema",
// "_meta"), and any other identifier that fails the regex.
func ValidTableName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty", ErrInvalidName)
	}
	if strings.HasPrefix(name, "_") {
		return fmt.Errorf("%w: %q starts with underscore", ErrReservedName, name)
	}
	if !nameRE.MatchString(name) {
		return fmt.Errorf("%w: %q must match %s", ErrInvalidName, name, nameRE.String())
	}
	return nil
}

// ValidColumnName returns nil if name is a legal, non-reserved column
// identifier. The implicit primary key "id" is reserved, as are any names
// with a leading underscore.
func ValidColumnName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty", ErrInvalidName)
	}
	if name == "id" {
		return fmt.Errorf("%w: %q is the implicit primary key", ErrReservedName, name)
	}
	if strings.HasPrefix(name, "_") {
		return fmt.Errorf("%w: %q starts with underscore", ErrReservedName, name)
	}
	if !nameRE.MatchString(name) {
		return fmt.Errorf("%w: %q must match %s", ErrInvalidName, name, nameRE.String())
	}
	return nil
}

// ValidColumnType returns nil if typ is one of the supported column types.
func ValidColumnType(typ string) error {
	switch typ {
	case ColumnTypeText, ColumnTypeNumber, ColumnTypeBool, ColumnTypeJSON:
		return nil
	default:
		return fmt.Errorf("%w: %q", ErrInvalidType, typ)
	}
}

// Table is a single user-declared table plus its columns.
type Table struct {
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Columns   []Column  `json:"columns"`
}

// Column is one user-declared column attached to a table.
type Column struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	Required  bool      `json:"required"`
	CreatedAt time.Time `json:"created_at"`
}

// Schema is the DAO for user-defined tables and columns.
type Schema struct {
	s *Store
}

// Schema returns the schema DAO bound to this store.
func (s *Store) Schema() *Schema { return &Schema{s: s} }

// CreateTable inserts a new (user_id, name) into app_tables.
// Returns ErrTableExists on duplicate, ErrReservedName/ErrInvalidName on
// illegal input.
func (sc *Schema) CreateTable(ctx context.Context, userID, name string) (*Table, error) {
	if err := ValidTableName(name); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	_, err := sc.s.DB.ExecContext(ctx,
		`INSERT INTO app_tables(user_id, name, created_at) VALUES (?, ?, ?)`,
		userID, name, now.UnixMilli(),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") ||
			strings.Contains(err.Error(), "PRIMARY KEY") {
			return nil, ErrTableExists
		}
		return nil, fmt.Errorf("insert table: %w", err)
	}
	return &Table{Name: name, CreatedAt: now}, nil
}

// DeleteTable removes the table and cascades to its columns.
// Returns ErrTableNotFound if the row wasn't present.
func (sc *Schema) DeleteTable(ctx context.Context, userID, name string) error {
	if err := ValidTableName(name); err != nil {
		return err
	}
	res, err := sc.s.DB.ExecContext(ctx,
		`DELETE FROM app_tables WHERE user_id = ? AND name = ?`,
		userID, name,
	)
	if err != nil {
		return fmt.Errorf("delete table: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete table rows: %w", err)
	}
	if n == 0 {
		return ErrTableNotFound
	}
	return nil
}

// GetTable returns a single table plus its columns.
// Returns ErrTableNotFound if not present.
func (sc *Schema) GetTable(ctx context.Context, userID, name string) (*Table, error) {
	var ts int64
	err := sc.s.DB.QueryRowContext(ctx,
		`SELECT created_at FROM app_tables WHERE user_id = ? AND name = ?`,
		userID, name,
	).Scan(&ts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTableNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get table: %w", err)
	}
	cols, err := sc.listColumns(ctx, userID, name)
	if err != nil {
		return nil, err
	}
	return &Table{
		Name:      name,
		CreatedAt: time.UnixMilli(ts).UTC(),
		Columns:   cols,
	}, nil
}

// ListTables returns every table for userID with its columns populated.
// Order is stable: tables by name ascending, columns by creation order.
func (sc *Schema) ListTables(ctx context.Context, userID string) ([]Table, error) {
	rows, err := sc.s.DB.QueryContext(ctx,
		`SELECT name, created_at FROM app_tables WHERE user_id = ? ORDER BY name`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type tmpRow struct {
		name string
		ts   int64
	}
	var rowsOut []tmpRow
	for rows.Next() {
		var r tmpRow
		if err := rows.Scan(&r.name, &r.ts); err != nil {
			return nil, err
		}
		rowsOut = append(rowsOut, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]Table, 0, len(rowsOut))
	for _, r := range rowsOut {
		cols, err := sc.listColumns(ctx, userID, r.name)
		if err != nil {
			return nil, err
		}
		out = append(out, Table{
			Name:      r.name,
			CreatedAt: time.UnixMilli(r.ts).UTC(),
			Columns:   cols,
		})
	}
	return out, nil
}

// AddColumn inserts a column under (userID, tableName). The table must exist.
// Returns ErrTableNotFound, ErrColumnExists, ErrInvalidName, ErrInvalidType,
// ErrReservedName depending on what went wrong.
func (sc *Schema) AddColumn(ctx context.Context, userID, tableName, colName, typ string, required bool) (*Column, error) {
	if err := ValidTableName(tableName); err != nil {
		return nil, err
	}
	if err := ValidColumnName(colName); err != nil {
		return nil, err
	}
	if err := ValidColumnType(typ); err != nil {
		return nil, err
	}

	// Make sure the parent table exists before we try to insert.  SQLite's
	// foreign-key error would also catch this but it surfaces as a generic
	// "FOREIGN KEY" string; returning ErrTableNotFound directly makes the
	// HTTP handler's error mapping one-for-one.
	var parent int
	err := sc.s.DB.QueryRowContext(ctx,
		`SELECT 1 FROM app_tables WHERE user_id = ? AND name = ?`,
		userID, tableName,
	).Scan(&parent)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTableNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("check parent: %w", err)
	}

	now := time.Now().UTC()
	reqInt := 0
	if required {
		reqInt = 1
	}
	_, err = sc.s.DB.ExecContext(ctx,
		`INSERT INTO app_columns(user_id, table_name, name, type, required, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		userID, tableName, colName, typ, reqInt, now.UnixMilli(),
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") ||
			strings.Contains(err.Error(), "PRIMARY KEY") {
			return nil, ErrColumnExists
		}
		return nil, fmt.Errorf("insert column: %w", err)
	}
	return &Column{
		Name:      colName,
		Type:      typ,
		Required:  required,
		CreatedAt: now,
	}, nil
}

// DeleteColumn removes a column from the table. Returns ErrColumnNotFound if
// the column didn't exist.
func (sc *Schema) DeleteColumn(ctx context.Context, userID, tableName, colName string) error {
	if err := ValidTableName(tableName); err != nil {
		return err
	}
	if err := ValidColumnName(colName); err != nil {
		return err
	}
	res, err := sc.s.DB.ExecContext(ctx,
		`DELETE FROM app_columns WHERE user_id = ? AND table_name = ? AND name = ?`,
		userID, tableName, colName,
	)
	if err != nil {
		return fmt.Errorf("delete column: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete column rows: %w", err)
	}
	if n == 0 {
		return ErrColumnNotFound
	}
	return nil
}

// listColumns returns the columns for (userID, tableName) ordered by
// creation time (stable within a table).
func (sc *Schema) listColumns(ctx context.Context, userID, tableName string) ([]Column, error) {
	rows, err := sc.s.DB.QueryContext(ctx,
		`SELECT name, type, required, created_at FROM app_columns
		 WHERE user_id = ? AND table_name = ?
		 ORDER BY created_at, name`,
		userID, tableName,
	)
	if err != nil {
		return nil, fmt.Errorf("list columns: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Column
	for rows.Next() {
		var (
			name string
			typ  string
			req  int
			ts   int64
		)
		if err := rows.Scan(&name, &typ, &req, &ts); err != nil {
			return nil, err
		}
		out = append(out, Column{
			Name:      name,
			Type:      typ,
			Required:  req != 0,
			CreatedAt: time.UnixMilli(ts).UTC(),
		})
	}
	return out, rows.Err()
}

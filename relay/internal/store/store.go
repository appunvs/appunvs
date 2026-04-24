// Package store is the relay's persistent state — user accounts, devices,
// and (added in later blocks) api keys, plans, subscriptions, app tables.
// Backed by a single SQLite file via modernc.org/sqlite (no CGO).
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // register the SQL driver
)

// Store owns the SQLite connection pool plus schema migrations.
// All domain-specific stores (users, devices, …) take a *Store and build
// queries against its DB handle.
type Store struct {
	DB *sql.DB
}

// Open opens (or creates) the SQLite file at path, configures WAL mode + sane
// pragmas, runs all pending migrations.
func Open(ctx context.Context, path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(on)&_pragma=synchronous(normal)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	// SQLite handles one writer at a time; more connections hurt more than
	// they help. Readers run concurrently in WAL mode regardless.
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)
	db.SetConnMaxIdleTime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	s := &Store{DB: db}
	if err := s.migrate(ctx); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// Close shuts down the connection pool.
func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

// migrate runs every migration statement whose id is greater than
// schema_version.version. Migrations are idempotent; rerunning is safe.
func (s *Store) migrate(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY
);`)
	if err != nil {
		return err
	}
	var current int
	row := s.DB.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_version`)
	if err := row.Scan(&current); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	for _, m := range migrations {
		if m.version <= current {
			continue
		}
		tx, err := s.DB.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, m.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %d: %w", m.version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_version(version) VALUES (?)`, m.version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration %d: record version: %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

type migration struct {
	version int
	sql     string
}

// migrations must be strictly increasing; never renumber or rewrite a shipped
// migration. Blocks 2/3/4 append their own migrations here (or in sibling
// files in this package using a shared migrations list).
var migrations = []migration{
	{
		version: 1,
		sql: `
CREATE TABLE users (
    id            TEXT PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    created_at    INTEGER NOT NULL
);

CREATE TABLE devices (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform   TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    last_seen  INTEGER
);
CREATE INDEX idx_devices_user ON devices(user_id);
`,
	},
	// Block 2: dynamic user-defined schema.  app_tables holds the user's
	// table names; app_columns holds each table's typed columns.  The
	// implicit primary key "id: text" is NOT stored — it's applied in
	// validation code because every record must carry an id.
	{
		version: 2,
		sql: `
CREATE TABLE app_tables (
    user_id    TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    PRIMARY KEY (user_id, name)
);

CREATE TABLE app_columns (
    user_id     TEXT NOT NULL,
    table_name  TEXT NOT NULL,
    name        TEXT NOT NULL,
    type        TEXT NOT NULL CHECK (type IN ('text','number','bool','json')),
    required    INTEGER NOT NULL DEFAULT 0,
    created_at  INTEGER NOT NULL,
    PRIMARY KEY (user_id, table_name, name),
    FOREIGN KEY (user_id, table_name) REFERENCES app_tables(user_id, name) ON DELETE CASCADE
);
CREATE INDEX idx_app_columns_table ON app_columns(user_id, table_name);
`,
	},
	// Block 3: api_keys. Block 2 claimed v2 after all, so this block
	// takes v3. Never renumber a shipped migration.
	{
		version: 3,
		sql: `
CREATE TABLE api_keys (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    prefix       TEXT NOT NULL UNIQUE,
    hash         TEXT NOT NULL,
    created_at   INTEGER NOT NULL,
    last_used_at INTEGER,
    revoked_at   INTEGER
);
CREATE INDEX idx_api_keys_user ON api_keys(user_id);
`,
	},
}

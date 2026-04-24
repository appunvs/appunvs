package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/appunvs/appunvs/relay/internal/pb"
)

// ErrBoxNotFound is returned when a lookup misses.
var ErrBoxNotFound = errors.New("box not found")

// Box is the row shape of app_boxes.  Enum fields are stored as their short
// lowercase protojson form ("published", "rn_bundle"), so `SELECT *` is
// inspectable without a custom dump.
type Box struct {
	ID               string
	Namespace        string
	ProviderDeviceID string
	Title            string
	Runtime          pb.RuntimeKind
	State            pb.PublishState
	CurrentVersion   string
	CreatedAt        int64
	UpdatedAt        int64
}

// Bundle is the row shape of app_bundles.  One row per build attempt,
// keyed (box_id, version).
type Bundle struct {
	BoxID       string
	Version     string
	URI         string
	ContentHash string
	SizeBytes   int64
	BuildState  pb.BuildState
	BuildLog    string
	BuiltAt     int64
	ExpiresAt   int64
}

// Boxes is the app_boxes + app_bundles store.
type Boxes struct {
	db *sql.DB
}

// Boxes returns the singleton boxes sub-store.
func (s *Store) Boxes() *Boxes { return &Boxes{db: s.DB} }

// Create inserts a new draft box with no current bundle.
func (b *Boxes) Create(ctx context.Context, box Box) error {
	now := time.Now().UnixMilli()
	if box.CreatedAt == 0 {
		box.CreatedAt = now
	}
	if box.UpdatedAt == 0 {
		box.UpdatedAt = now
	}
	if box.State == pb.PublishStateUnspecified {
		box.State = pb.PublishStateDraft
	}
	_, err := b.db.ExecContext(ctx, `
INSERT INTO app_boxes (id, namespace, provider_device_id, title, runtime, state, current_version, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		box.ID, box.Namespace, box.ProviderDeviceID, box.Title,
		box.Runtime.String(), box.State.String(), box.CurrentVersion,
		box.CreatedAt, box.UpdatedAt)
	return err
}

// Get looks up a single box. `namespace` is enforced so a caller can't
// probe boxes from other users even if they guess an id.
func (b *Boxes) Get(ctx context.Context, namespace, id string) (Box, error) {
	row := b.db.QueryRowContext(ctx, `
SELECT id, namespace, provider_device_id, title, runtime, state, current_version, created_at, updated_at
FROM app_boxes WHERE namespace = ? AND id = ?`, namespace, id)
	var (
		box              Box
		runtime, stateEv string
	)
	if err := row.Scan(&box.ID, &box.Namespace, &box.ProviderDeviceID, &box.Title,
		&runtime, &stateEv, &box.CurrentVersion, &box.CreatedAt, &box.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Box{}, ErrBoxNotFound
		}
		return Box{}, fmt.Errorf("boxes.get: %w", err)
	}
	box.Runtime = pb.ParseRuntimeKind(runtime)
	box.State = pb.ParsePublishState(stateEv)
	return box, nil
}

// List returns every box owned by the given namespace, most recently
// updated first.
func (b *Boxes) List(ctx context.Context, namespace string) ([]Box, error) {
	rows, err := b.db.QueryContext(ctx, `
SELECT id, namespace, provider_device_id, title, runtime, state, current_version, created_at, updated_at
FROM app_boxes WHERE namespace = ? ORDER BY updated_at DESC`, namespace)
	if err != nil {
		return nil, fmt.Errorf("boxes.list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Box
	for rows.Next() {
		var (
			box              Box
			runtime, stateEv string
		)
		if err := rows.Scan(&box.ID, &box.Namespace, &box.ProviderDeviceID, &box.Title,
			&runtime, &stateEv, &box.CurrentVersion, &box.CreatedAt, &box.UpdatedAt); err != nil {
			return nil, err
		}
		box.Runtime = pb.ParseRuntimeKind(runtime)
		box.State = pb.ParsePublishState(stateEv)
		out = append(out, box)
	}
	return out, rows.Err()
}

// SetState mutates the PublishState of a box (draft→published→archived).
func (b *Boxes) SetState(ctx context.Context, namespace, id string, state pb.PublishState) error {
	res, err := b.db.ExecContext(ctx, `
UPDATE app_boxes SET state = ?, updated_at = ? WHERE namespace = ? AND id = ?`,
		state.String(), time.Now().UnixMilli(), namespace, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrBoxNotFound
	}
	return nil
}

// SetCurrentVersion points the box at a successful build.  Caller is
// responsible for having already inserted the Bundle row with
// build_state=succeeded.
func (b *Boxes) SetCurrentVersion(ctx context.Context, namespace, id, version string) error {
	res, err := b.db.ExecContext(ctx, `
UPDATE app_boxes SET current_version = ?, updated_at = ? WHERE namespace = ? AND id = ?`,
		version, time.Now().UnixMilli(), namespace, id)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return ErrBoxNotFound
	}
	return nil
}

// PutBundle inserts-or-replaces a Bundle row for (box_id, version).
func (b *Boxes) PutBundle(ctx context.Context, bundle Bundle) error {
	_, err := b.db.ExecContext(ctx, `
INSERT INTO app_bundles (box_id, version, uri, content_hash, size_bytes, build_state, build_log, built_at, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(box_id, version) DO UPDATE SET
    uri          = excluded.uri,
    content_hash = excluded.content_hash,
    size_bytes   = excluded.size_bytes,
    build_state  = excluded.build_state,
    build_log    = excluded.build_log,
    built_at     = excluded.built_at,
    expires_at   = excluded.expires_at`,
		bundle.BoxID, bundle.Version, bundle.URI, bundle.ContentHash,
		bundle.SizeBytes, bundle.BuildState.String(), bundle.BuildLog,
		bundle.BuiltAt, bundle.ExpiresAt)
	return err
}

// GetBundle fetches a single Bundle row.  Returns ErrBoxNotFound when the
// (box_id, version) pair doesn't exist.
func (b *Boxes) GetBundle(ctx context.Context, boxID, version string) (Bundle, error) {
	row := b.db.QueryRowContext(ctx, `
SELECT box_id, version, uri, content_hash, size_bytes, build_state, build_log, built_at, expires_at
FROM app_bundles WHERE box_id = ? AND version = ?`, boxID, version)
	var (
		bu     Bundle
		stateV string
	)
	if err := row.Scan(&bu.BoxID, &bu.Version, &bu.URI, &bu.ContentHash,
		&bu.SizeBytes, &stateV, &bu.BuildLog, &bu.BuiltAt, &bu.ExpiresAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Bundle{}, ErrBoxNotFound
		}
		return Bundle{}, err
	}
	bu.BuildState = pb.ParseBuildState(stateV)
	return bu, nil
}

// init registers Block 5: box + bundle tables.
func init() {
	migrations = append(migrations, migration{
		version: 6,
		sql: `
CREATE TABLE app_boxes (
    id                  TEXT PRIMARY KEY,
    namespace           TEXT NOT NULL,
    provider_device_id  TEXT NOT NULL,
    title               TEXT NOT NULL,
    runtime             TEXT NOT NULL,
    state               TEXT NOT NULL,
    current_version     TEXT NOT NULL DEFAULT '',
    created_at          INTEGER NOT NULL,
    updated_at          INTEGER NOT NULL
);
CREATE INDEX idx_app_boxes_ns ON app_boxes(namespace, updated_at DESC);

CREATE TABLE app_bundles (
    box_id        TEXT NOT NULL REFERENCES app_boxes(id) ON DELETE CASCADE,
    version       TEXT NOT NULL,
    uri           TEXT NOT NULL,
    content_hash  TEXT NOT NULL,
    size_bytes    INTEGER NOT NULL,
    build_state   TEXT NOT NULL,
    build_log     TEXT NOT NULL DEFAULT '',
    built_at      INTEGER NOT NULL,
    expires_at    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (box_id, version)
);
`,
	})
}

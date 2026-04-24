package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrDeviceConflict means a device_id already exists but belongs to another
// user (shouldn't happen if clients use UUIDs, but we enforce it).
var ErrDeviceConflict = errors.New("device belongs to another user")

// Device is one registered device.
type Device struct {
	ID        string
	UserID    string
	Platform  string
	CreatedAt time.Time
	LastSeen  *time.Time
}

// Devices is the DAO for the devices table.
type Devices struct {
	s *Store
}

// Devices returns the devices DAO.
func (s *Store) Devices() *Devices { return &Devices{s: s} }

// Register inserts or re-confirms a device for the given user.
// Idempotent: calling with the same (deviceID, userID) is a no-op.
// Calling with (deviceID, different userID) returns ErrDeviceConflict.
func (d *Devices) Register(ctx context.Context, deviceID, userID, platform string) (*Device, error) {
	now := time.Now().UTC()
	tx, err := d.s.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var (
		existingUser string
	)
	row := tx.QueryRowContext(ctx, `SELECT user_id FROM devices WHERE id = ?`, deviceID)
	switch err := row.Scan(&existingUser); {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO devices(id, user_id, platform, created_at, last_seen) VALUES (?, ?, ?, ?, ?)`,
			deviceID, userID, platform, now.UnixMilli(), now.UnixMilli(),
		); err != nil {
			return nil, fmt.Errorf("insert device: %w", err)
		}
	case err != nil:
		return nil, err
	default:
		if existingUser != userID {
			return nil, ErrDeviceConflict
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE devices SET platform = ?, last_seen = ? WHERE id = ?`,
			platform, now.UnixMilli(), deviceID,
		); err != nil {
			return nil, fmt.Errorf("update device: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &Device{
		ID:        deviceID,
		UserID:    userID,
		Platform:  platform,
		CreatedAt: now,
		LastSeen:  &now,
	}, nil
}

// ListByUser returns all devices for a user, newest last-seen first.
func (d *Devices) ListByUser(ctx context.Context, userID string) ([]Device, error) {
	rows, err := d.s.DB.QueryContext(ctx,
		`SELECT id, platform, created_at, last_seen FROM devices WHERE user_id = ? ORDER BY last_seen DESC NULLS LAST`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []Device
	for rows.Next() {
		var (
			id         string
			platform   string
			created    int64
			lastSeenNS sql.NullInt64
		)
		if err := rows.Scan(&id, &platform, &created, &lastSeenNS); err != nil {
			return nil, err
		}
		dev := Device{
			ID:        id,
			UserID:    userID,
			Platform:  platform,
			CreatedAt: time.UnixMilli(created).UTC(),
		}
		if lastSeenNS.Valid {
			t := time.UnixMilli(lastSeenNS.Int64).UTC()
			dev.LastSeen = &t
		}
		out = append(out, dev)
	}
	return out, rows.Err()
}

// Touch updates last_seen for the given device (no-op if device vanished).
func (d *Devices) Touch(ctx context.Context, deviceID string) error {
	_, err := d.s.DB.ExecContext(ctx,
		`UPDATE devices SET last_seen = ? WHERE id = ?`,
		time.Now().UTC().UnixMilli(), deviceID,
	)
	return err
}

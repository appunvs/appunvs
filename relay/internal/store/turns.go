package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrTurnNotFound is returned when a lookup misses.
var ErrTurnNotFound = errors.New("ai turn not found")

// Turn is one AI conversation turn persisted under a Box.  `Messages` is
// the raw JSON-serialized array of Anthropic-format message blocks the
// agent sent/received during this turn — when we need to continue a
// session we replay them into the next /ai/turn call, no RAG required
// while the box fits in the model's context window.
type Turn struct {
	ID         string
	BoxID      string
	UserText   string
	Messages   string // raw JSON; relay does not introspect
	TokensIn   int64
	TokensOut  int64
	StopReason string
	CreatedAt  int64
}

// Turns is the ai_turns store.
type Turns struct {
	db *sql.DB
}

// Turns returns the singleton turns sub-store.
func (s *Store) Turns() *Turns { return &Turns{db: s.DB} }

// Insert persists a completed turn.  `ID` and `CreatedAt` are filled in
// by the caller (the AI engine layer knows the turn_id it emitted on the
// stream).
func (t *Turns) Insert(ctx context.Context, turn Turn) error {
	if turn.CreatedAt == 0 {
		turn.CreatedAt = time.Now().UnixMilli()
	}
	_, err := t.db.ExecContext(ctx, `
INSERT INTO ai_turns (id, box_id, user_text, messages, tokens_in, tokens_out, stop_reason, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		turn.ID, turn.BoxID, turn.UserText, turn.Messages,
		turn.TokensIn, turn.TokensOut, turn.StopReason, turn.CreatedAt)
	return err
}

// Recent returns the most recent `limit` turns for a box, newest-first.
// The AI engine uses this to rebuild conversation context before the next
// turn; long sessions that overrun the context window will feed in through
// a separate compaction step (not yet implemented).
func (t *Turns) Recent(ctx context.Context, boxID string, limit int) ([]Turn, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := t.db.QueryContext(ctx, `
SELECT id, box_id, user_text, messages, tokens_in, tokens_out, stop_reason, created_at
FROM ai_turns WHERE box_id = ? ORDER BY created_at DESC LIMIT ?`, boxID, limit)
	if err != nil {
		return nil, fmt.Errorf("turns.recent: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var out []Turn
	for rows.Next() {
		var tr Turn
		if err := rows.Scan(&tr.ID, &tr.BoxID, &tr.UserText, &tr.Messages,
			&tr.TokensIn, &tr.TokensOut, &tr.StopReason, &tr.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, tr)
	}
	return out, rows.Err()
}

// Get returns a specific turn by id.
func (t *Turns) Get(ctx context.Context, id string) (Turn, error) {
	row := t.db.QueryRowContext(ctx, `
SELECT id, box_id, user_text, messages, tokens_in, tokens_out, stop_reason, created_at
FROM ai_turns WHERE id = ?`, id)
	var tr Turn
	if err := row.Scan(&tr.ID, &tr.BoxID, &tr.UserText, &tr.Messages,
		&tr.TokensIn, &tr.TokensOut, &tr.StopReason, &tr.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Turn{}, ErrTurnNotFound
		}
		return Turn{}, err
	}
	return tr, nil
}

// TokensSpent sums tokens_in + tokens_out over the last `since` ms window
// for a namespace.  Used by the budget gate in /ai/turn.
func (t *Turns) TokensSpent(ctx context.Context, namespace string, sinceMillis int64) (int64, error) {
	// Join through app_boxes to filter by namespace, since ai_turns is
	// box-scoped and boxes are namespace-scoped.
	row := t.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(ai_turns.tokens_in + ai_turns.tokens_out), 0)
FROM ai_turns
JOIN app_boxes ON ai_turns.box_id = app_boxes.id
WHERE app_boxes.namespace = ? AND ai_turns.created_at >= ?`, namespace, sinceMillis)
	var n sql.NullInt64
	if err := row.Scan(&n); err != nil {
		return 0, err
	}
	if !n.Valid {
		return 0, nil
	}
	return n.Int64, nil
}

// init registers Block 6: ai_turns.
func init() {
	migrations = append(migrations, migration{
		version: 7,
		sql: `
CREATE TABLE ai_turns (
    id          TEXT PRIMARY KEY,
    box_id      TEXT NOT NULL REFERENCES app_boxes(id) ON DELETE CASCADE,
    user_text   TEXT NOT NULL,
    messages    TEXT NOT NULL,
    tokens_in   INTEGER NOT NULL DEFAULT 0,
    tokens_out  INTEGER NOT NULL DEFAULT 0,
    stop_reason TEXT NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL
);
CREATE INDEX idx_ai_turns_box ON ai_turns(box_id, created_at DESC);
`,
	})
}

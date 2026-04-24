package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidKey is returned by VerifySecret when the supplied API key is
// malformed, unknown, revoked, or has a mismatched secret. The cases are
// merged deliberately so callers cannot build a probe that distinguishes
// "wrong secret" from "no such key".
var ErrInvalidKey = errors.New("invalid api key")

// ErrKeyNotFound is returned by Revoke when the (user_id, id) pair does
// not match any row.
var ErrKeyNotFound = errors.New("api key not found")

// APIKeyNamespace is the fixed prefix every key starts with. It lets the
// middleware decide which authenticator flavor to route a bearer token to
// without first parsing JWT claims.
const APIKeyNamespace = "apvs_"

// apiKeyPrefixLen is the length of the prefix stored in the `prefix` column:
// "apvs_" (5) + 8 random chars = 13.
const apiKeyPrefixLen = 13

// apiKeyRandChars is the number of base64 url-safe chars after the "apvs_"
// namespace. 22 chars is ~132 bits of entropy which is comfortably beyond
// the brute-force frontier for a sha256-hashed secret.
const apiKeyRandChars = 22

// apiKeyFullLen is the expected total length of a full key string.
const apiKeyFullLen = len(APIKeyNamespace) + apiKeyRandChars

// APIKey is the hydrated row, safe to return to callers (no secret).
type APIKey struct {
	ID         string
	UserID     string
	Name       string
	Prefix     string
	CreatedAt  time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
}

// APIKeys is the DAO for the api_keys table.
type APIKeys struct {
	s *Store
}

// APIKeys returns the api-keys DAO bound to this store.
func (s *Store) APIKeys() *APIKeys { return &APIKeys{s: s} }

// Create mints a new API key for userID and inserts the row. The caller
// receives the hydrated APIKey plus the full secret string; the secret
// is returned exactly once and is not persisted in plaintext.
func (k *APIKeys) Create(ctx context.Context, userID, name string) (*APIKey, string, error) {
	name = strings.TrimSpace(name)
	if userID == "" {
		return nil, "", errors.New("user_id required")
	}
	if name == "" {
		return nil, "", errors.New("name required")
	}
	full, prefix, hash, err := generateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("generate api key: %w", err)
	}

	id := "ak_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	now := time.Now().UTC()

	_, err = k.s.DB.ExecContext(ctx,
		`INSERT INTO api_keys(id, user_id, name, prefix, hash, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		id, userID, name, prefix, hash, now.UnixMilli(),
	)
	if err != nil {
		return nil, "", fmt.Errorf("insert api key: %w", err)
	}
	return &APIKey{
		ID:        id,
		UserID:    userID,
		Name:      name,
		Prefix:    prefix,
		CreatedAt: now,
	}, full, nil
}

// List returns every key for userID, newest-first.
func (k *APIKeys) List(ctx context.Context, userID string) ([]APIKey, error) {
	rows, err := k.s.DB.QueryContext(ctx,
		`SELECT id, name, prefix, created_at, last_used_at, revoked_at
		   FROM api_keys WHERE user_id = ?
		   ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []APIKey
	for rows.Next() {
		var (
			id, name, prefix string
			created          int64
			lastUsed         sql.NullInt64
			revoked          sql.NullInt64
		)
		if err := rows.Scan(&id, &name, &prefix, &created, &lastUsed, &revoked); err != nil {
			return nil, err
		}
		row := APIKey{
			ID:        id,
			UserID:    userID,
			Name:      name,
			Prefix:    prefix,
			CreatedAt: time.UnixMilli(created).UTC(),
		}
		if lastUsed.Valid {
			t := time.UnixMilli(lastUsed.Int64).UTC()
			row.LastUsedAt = &t
		}
		if revoked.Valid {
			t := time.UnixMilli(revoked.Int64).UTC()
			row.RevokedAt = &t
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

// Revoke soft-deletes a key by stamping revoked_at. Scoped to userID so
// one user cannot revoke another's keys by guessing the id. Returns
// ErrKeyNotFound when the (user_id, id) pair does not exist OR when the
// key was already revoked (the caller sees a single "not found" surface).
func (k *APIKeys) Revoke(ctx context.Context, userID, keyID string) error {
	now := time.Now().UTC().UnixMilli()
	res, err := k.s.DB.ExecContext(ctx,
		`UPDATE api_keys SET revoked_at = ? WHERE id = ? AND user_id = ? AND revoked_at IS NULL`,
		now, keyID, userID,
	)
	if err != nil {
		return fmt.Errorf("revoke api key: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrKeyNotFound
	}
	return nil
}

// VerifySecret validates a raw "apvs_..." string against stored rows and
// returns the owning key on success. Returns ErrInvalidKey for any failure
// (malformed, unknown prefix, revoked, or hash mismatch). Uses a
// constant-time compare so callers cannot distinguish "prefix matched but
// secret did not" from "prefix never matched" via timing.
func (k *APIKeys) VerifySecret(ctx context.Context, secret string) (*APIKey, error) {
	if len(secret) != apiKeyFullLen {
		return nil, ErrInvalidKey
	}
	if !strings.HasPrefix(secret, APIKeyNamespace) {
		return nil, ErrInvalidKey
	}
	prefix := secret[:apiKeyPrefixLen]

	var (
		id, userID, name, storedHash string
		created                      int64
		lastUsed                     sql.NullInt64
	)
	err := k.s.DB.QueryRowContext(ctx,
		`SELECT id, user_id, name, hash, created_at, last_used_at
		   FROM api_keys
		  WHERE prefix = ? AND revoked_at IS NULL`,
		prefix,
	).Scan(&id, &userID, &name, &storedHash, &created, &lastUsed)
	if errors.Is(err, sql.ErrNoRows) {
		// Still run a constant-time compare against a dummy value so that
		// a missing prefix looks indistinguishable, timing-wise, from a
		// bad secret.
		dummy := sha256.Sum256([]byte(secret))
		_ = subtle.ConstantTimeCompare(dummy[:], dummy[:])
		return nil, ErrInvalidKey
	}
	if err != nil {
		return nil, fmt.Errorf("lookup api key: %w", err)
	}
	got := sha256.Sum256([]byte(secret))
	gotHex := hex.EncodeToString(got[:])
	if subtle.ConstantTimeCompare([]byte(gotHex), []byte(storedHash)) != 1 {
		return nil, ErrInvalidKey
	}
	row := &APIKey{
		ID:        id,
		UserID:    userID,
		Name:      name,
		Prefix:    prefix,
		CreatedAt: time.UnixMilli(created).UTC(),
	}
	if lastUsed.Valid {
		t := time.UnixMilli(lastUsed.Int64).UTC()
		row.LastUsedAt = &t
	}
	return row, nil
}

// Touch bumps last_used_at to now. Best-effort; a failure does not break
// authentication (callers typically run it in a goroutine).
func (k *APIKeys) Touch(ctx context.Context, keyID string) error {
	_, err := k.s.DB.ExecContext(ctx,
		`UPDATE api_keys SET last_used_at = ? WHERE id = ?`,
		time.Now().UTC().UnixMilli(), keyID,
	)
	return err
}

// generateAPIKey returns (full, prefix, hexSha256Hash). The full string is
// what the user sees ("apvs_<22 chars>"); the prefix is the first 13 chars
// stored plaintext for lookup; the hash is hex(sha256(full)) stored in DB.
func generateAPIKey() (full, prefix, hash string, err error) {
	// base64 url-safe w/o padding gives roughly 6 bits/char; 17 raw bytes
	// encodes to 23 chars, so request 17 and trim to 22.
	raw := make([]byte, 17)
	if _, err := rand.Read(raw); err != nil {
		return "", "", "", err
	}
	enc := base64.RawURLEncoding.EncodeToString(raw)
	if len(enc) < apiKeyRandChars {
		return "", "", "", fmt.Errorf("unexpected encoded length %d", len(enc))
	}
	full = APIKeyNamespace + enc[:apiKeyRandChars]
	prefix = full[:apiKeyPrefixLen]
	sum := sha256.Sum256([]byte(full))
	hash = hex.EncodeToString(sum[:])
	return full, prefix, hash, nil
}

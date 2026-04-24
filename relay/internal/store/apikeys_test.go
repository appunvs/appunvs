package store_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/appunvs/appunvs/relay/internal/store"
)

// openTestStore creates an isolated SQLite store in a temp dir and seeds a
// single user. Returns the store, the user id, and a cancelable context.
func openTestStore(t *testing.T) (*store.Store, string) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "t.db")
	st, err := store.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	u, err := st.Users().Create(context.Background(), "apikey-test@example.com", "hunter22")
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return st, u.ID
}

// TestAPIKeyCreateListVerify covers the happy path: create a key, list it
// (no secret leaked), verify the returned secret resolves to the same row.
func TestAPIKeyCreateListVerify(t *testing.T) {
	st, userID := openTestStore(t)
	ctx := context.Background()

	k, secret, err := st.APIKeys().Create(ctx, userID, "cli")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if k.ID == "" || !strings.HasPrefix(k.ID, "ak_") {
		t.Fatalf("unexpected id %q", k.ID)
	}
	if !strings.HasPrefix(secret, store.APIKeyNamespace) {
		t.Fatalf("secret missing namespace: %q", secret)
	}
	if len(secret) != 27 {
		t.Fatalf("secret length = %d, want 27", len(secret))
	}
	if k.Prefix != secret[:13] {
		t.Fatalf("prefix = %q, want %q", k.Prefix, secret[:13])
	}

	list, err := st.APIKeys().List(ctx, userID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("list returned %d rows, want 1", len(list))
	}
	if list[0].ID != k.ID || list[0].Prefix != k.Prefix {
		t.Fatalf("list row mismatch: %+v vs %+v", list[0], k)
	}
	if list[0].RevokedAt != nil {
		t.Fatalf("fresh key should not be revoked: %+v", list[0].RevokedAt)
	}

	got, err := st.APIKeys().VerifySecret(ctx, secret)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if got.ID != k.ID || got.UserID != userID {
		t.Fatalf("verify returned wrong row: %+v", got)
	}
}

// Touch updates last_used_at.
func TestAPIKeyTouch(t *testing.T) {
	st, userID := openTestStore(t)
	ctx := context.Background()

	k, _, err := st.APIKeys().Create(ctx, userID, "bg")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := st.APIKeys().Touch(ctx, k.ID); err != nil {
		t.Fatalf("touch: %v", err)
	}
	list, _ := st.APIKeys().List(ctx, userID)
	if len(list) != 1 || list[0].LastUsedAt == nil {
		t.Fatalf("last_used_at not set: %+v", list)
	}
	// Basic sanity: recent timestamp.
	if time.Since(*list[0].LastUsedAt) > time.Minute {
		t.Fatalf("last_used_at too old: %v", list[0].LastUsedAt)
	}
}

// Revoke soft-deletes and subsequent VerifySecret rejects the key.
func TestAPIKeyRevoke(t *testing.T) {
	st, userID := openTestStore(t)
	ctx := context.Background()

	k, secret, err := st.APIKeys().Create(ctx, userID, "throwaway")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := st.APIKeys().Revoke(ctx, userID, k.ID); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	// Second revoke on the same key returns ErrKeyNotFound (already revoked).
	if err := st.APIKeys().Revoke(ctx, userID, k.ID); !errors.Is(err, store.ErrKeyNotFound) {
		t.Fatalf("second revoke err = %v, want ErrKeyNotFound", err)
	}

	// The secret no longer verifies.
	if _, err := st.APIKeys().VerifySecret(ctx, secret); !errors.Is(err, store.ErrInvalidKey) {
		t.Fatalf("verify after revoke err = %v, want ErrInvalidKey", err)
	}

	// List still returns the row with revoked_at stamped.
	list, err := st.APIKeys().List(ctx, userID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].RevokedAt == nil {
		t.Fatalf("expected one revoked row, got %+v", list)
	}
}

// Revoke scoped to the owning user — another user's revoke attempt 404s.
func TestAPIKeyRevokeWrongUser(t *testing.T) {
	st, userID := openTestStore(t)
	ctx := context.Background()
	other, err := st.Users().Create(ctx, "other@example.com", "hunter22")
	if err != nil {
		t.Fatalf("seed other: %v", err)
	}

	k, _, err := st.APIKeys().Create(ctx, userID, "mine")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := st.APIKeys().Revoke(ctx, other.ID, k.ID); !errors.Is(err, store.ErrKeyNotFound) {
		t.Fatalf("cross-user revoke err = %v, want ErrKeyNotFound", err)
	}
}

// VerifySecret rejects malformed and unknown keys with ErrInvalidKey; a
// valid-looking string with wrong hash also yields ErrInvalidKey (not a
// different sentinel).
func TestAPIKeyVerifyInvalid(t *testing.T) {
	st, userID := openTestStore(t)
	ctx := context.Background()

	// Seed one real key so the table isn't empty.
	if _, _, err := st.APIKeys().Create(ctx, userID, "x"); err != nil {
		t.Fatalf("seed: %v", err)
	}

	cases := []string{
		"",                             // empty
		"bearer apvs_something",        // wrong namespace
		"apvs_",                        // namespace only
		"apvs_" + strings.Repeat("x", 22), // well-formed but unknown
		"xxxxx",                        // too short
		strings.Repeat("a", 100),       // too long, wrong namespace
	}
	for _, tc := range cases {
		_, err := st.APIKeys().VerifySecret(ctx, tc)
		if !errors.Is(err, store.ErrInvalidKey) {
			t.Errorf("VerifySecret(%q) err = %v, want ErrInvalidKey", tc, err)
		}
	}
}

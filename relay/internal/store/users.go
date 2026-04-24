package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ErrUserExists indicates /auth/signup conflicted with an existing email.
var ErrUserExists = errors.New("user already exists")

// ErrBadCredentials is returned when login fails to match email or password.
// The two cases are deliberately merged so the API surface doesn't hint at
// which accounts exist.
var ErrBadCredentials = errors.New("bad credentials")

// User is the hydrated account row.
type User struct {
	ID        string
	Email     string
	CreatedAt time.Time
}

// Users is the DAO for the users table.
type Users struct {
	s *Store
}

// UsersOf returns the users DAO bound to this store.
func (s *Store) Users() *Users { return &Users{s: s} }

// Create inserts a new user with the given email and plaintext password.
// Returns ErrUserExists on duplicate email.
func (u *Users) Create(ctx context.Context, email, password string) (*User, error) {
	email = normalizeEmail(email)
	if email == "" {
		return nil, errors.New("email required")
	}
	if len(password) < 8 {
		return nil, errors.New("password too short (min 8 chars)")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash: %w", err)
	}
	id := "u_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	now := time.Now().UTC()

	_, err = u.s.DB.ExecContext(ctx,
		`INSERT INTO users(id, email, password_hash, created_at) VALUES (?, ?, ?, ?)`,
		id, email, string(hash), now.UnixMilli(),
	)
	if err != nil {
		// modernc sqlite returns a string with "UNIQUE constraint" on conflict.
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return nil, ErrUserExists
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}
	return &User{ID: id, Email: email, CreatedAt: now}, nil
}

// VerifyLogin checks email + plaintext password against the stored hash.
// Returns ErrBadCredentials on any mismatch, including missing user.
func (u *Users) VerifyLogin(ctx context.Context, email, password string) (*User, error) {
	email = normalizeEmail(email)
	var (
		id   string
		hash string
		ts   int64
	)
	err := u.s.DB.QueryRowContext(ctx,
		`SELECT id, password_hash, created_at FROM users WHERE email = ? COLLATE NOCASE`, email,
	).Scan(&id, &hash, &ts)
	if errors.Is(err, sql.ErrNoRows) {
		// Consume constant time even on miss — avoid timing oracle that
		// reveals which emails are registered.
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$.................................................."), []byte(password))
		return nil, ErrBadCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return nil, ErrBadCredentials
	}
	return &User{ID: id, Email: email, CreatedAt: time.UnixMilli(ts).UTC()}, nil
}

// Get loads a user by id.
func (u *Users) Get(ctx context.Context, id string) (*User, error) {
	var (
		email string
		ts    int64
	)
	err := u.s.DB.QueryRowContext(ctx,
		`SELECT email, created_at FROM users WHERE id = ?`, id,
	).Scan(&email, &ts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrBadCredentials
	}
	if err != nil {
		return nil, err
	}
	return &User{ID: id, Email: email, CreatedAt: time.UnixMilli(ts).UTC()}, nil
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

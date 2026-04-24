// Package artifact stores and serves the immutable build outputs (Metro JS
// bundles plus their assets) referenced by store.Bundle rows.
//
// The contract is intentionally narrow so the local-filesystem implementation
// shipped today can be swapped for an S3-compatible backend (Volcengine TOS,
// AWS S3, Cloudflare R2) without touching callers.  Content is keyed by its
// sha256 hash, making writes idempotent: re-uploading the same bytes is
// cheap and never invalidates a previously signed URL.
package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ErrNotFound is returned when a Get / SignURL targets an absent hash.
var ErrNotFound = errors.New("artifact not found")

// Object describes a stored bundle blob.
type Object struct {
	Hash      string // "sha256:<hex>"
	SizeBytes int64
}

// Store is the storage backend.  Implementations must be safe for
// concurrent use by multiple goroutines.
type Store interface {
	// Put streams reader to durable storage, returning the content hash and
	// size.  Implementations MUST hash the body themselves and key by the
	// hash; do not trust callers to supply it.
	Put(ctx context.Context, r io.Reader) (Object, error)
	// SignURL returns a short-lived URL the runner can fetch over plain
	// HTTPS.  ttl is advisory; backends may clamp it to a configured max.
	SignURL(ctx context.Context, hash string, ttl time.Duration) (url string, expiresAt time.Time, err error)
}

// LocalFS is a development backend that writes artifacts under root and
// serves them via a base URL the relay also exposes (e.g. http://localhost:8080/_artifacts/).
//
// In production swap in an S3-compatible backend; see docs/architecture.md
// for the recommended Volcengine TOS configuration.
type LocalFS struct {
	root    string // filesystem root
	baseURL string // public base URL, no trailing slash
}

// NewLocalFS creates a LocalFS rooted at the given directory.
func NewLocalFS(root, baseURL string) (*LocalFS, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &LocalFS{root: root, baseURL: baseURL}, nil
}

// Put computes sha256 over reader, writes to root/<hex>, returns the Object.
// Existing files with the same hash are kept (their bytes must match by
// definition).
func (l *LocalFS) Put(_ context.Context, r io.Reader) (Object, error) {
	tmp, err := os.CreateTemp(l.root, "in-*.tmp")
	if err != nil {
		return Object{}, err
	}
	tmpPath := tmp.Name()
	defer func() {
		// Best-effort cleanup if rename never happens.
		_ = os.Remove(tmpPath)
	}()

	h := sha256.New()
	w := io.MultiWriter(tmp, h)
	n, err := io.Copy(w, r)
	if err != nil {
		_ = tmp.Close()
		return Object{}, err
	}
	if err := tmp.Close(); err != nil {
		return Object{}, err
	}
	hex := hex.EncodeToString(h.Sum(nil))
	final := filepath.Join(l.root, hex)
	if err := os.Rename(tmpPath, final); err != nil {
		// If rename failed because the target already exists with the same
		// hash, that's fine — content-addressed.
		if _, statErr := os.Stat(final); statErr == nil {
			return Object{Hash: "sha256:" + hex, SizeBytes: n}, nil
		}
		return Object{}, err
	}
	return Object{Hash: "sha256:" + hex, SizeBytes: n}, nil
}

// SignURL returns a public URL under baseURL.  ttl is informational for the
// LocalFS backend — there's no signature.  Callers should treat this URL as
// short-lived and re-request before expiry.
func (l *LocalFS) SignURL(_ context.Context, hash string, ttl time.Duration) (string, time.Time, error) {
	hex, err := hexFromHash(hash)
	if err != nil {
		return "", time.Time{}, err
	}
	if _, err := os.Stat(filepath.Join(l.root, hex)); errors.Is(err, os.ErrNotExist) {
		return "", time.Time{}, ErrNotFound
	} else if err != nil {
		return "", time.Time{}, err
	}
	return fmt.Sprintf("%s/%s", l.baseURL, hex), time.Now().Add(ttl), nil
}

func hexFromHash(hash string) (string, error) {
	const prefix = "sha256:"
	if len(hash) <= len(prefix) || hash[:len(prefix)] != prefix {
		return "", fmt.Errorf("artifact: hash must start with sha256:")
	}
	return hash[len(prefix):], nil
}

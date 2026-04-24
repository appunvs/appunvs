// Package pairing mints short, human-friendly codes that bind a connector
// device to a provider's box.  Codes are stored in Redis with a TTL so they
// expire on their own; the relay never persists them in SQLite.
//
// Code format: 8 uppercase characters from a Crockford-style alphabet that
// drops 0/1/I/O to avoid ambiguity in the QR fallback "type-it-by-hand" flow.
// The keyspace is 32^8 ≈ 1.1e12, ample for short TTLs (<= 15 minutes).
package pairing

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// MaxTTL caps how long a pairing code may live.
const MaxTTL = 15 * time.Minute

// ErrNotFound is returned when Claim sees an unknown / expired code.
var ErrNotFound = errors.New("pairing code not found or expired")

// alphabet is Crockford-base32 with 0/1/I/O removed.
const alphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

// Grant is what gets stored under the short code.  It's intentionally small
// so a single Redis GET resolves the claim.
type Grant struct {
	BoxID            string `json:"box_id"`
	Namespace        string `json:"namespace"`
	ProviderDeviceID string `json:"provider_device_id"`
	IssuedAt         int64  `json:"issued_at"`
}

// Service issues and redeems pairing codes via Redis.
type Service struct {
	rdb *redis.Client
}

// New returns a pairing service backed by the given Redis client.
func New(rdb *redis.Client) *Service { return &Service{rdb: rdb} }

// Issue creates a new short code bound to grant, with an actual TTL of
// min(ttl, MaxTTL).  Returns the code and its absolute expiry time.
func (s *Service) Issue(ctx context.Context, grant Grant, ttl time.Duration) (string, time.Time, error) {
	if ttl <= 0 || ttl > MaxTTL {
		ttl = MaxTTL
	}
	if grant.IssuedAt == 0 {
		grant.IssuedAt = time.Now().UnixMilli()
	}
	body, err := json.Marshal(grant)
	if err != nil {
		return "", time.Time{}, err
	}

	// Up to 5 collision retries — at 8 chars from a 32-symbol alphabet,
	// collisions inside a 15-minute window are negligible in practice.
	for attempt := 0; attempt < 5; attempt++ {
		code, err := newCode(8)
		if err != nil {
			return "", time.Time{}, err
		}
		key := redisKey(code)
		// SET NX: succeeds only if the key didn't exist; collision-safe.
		ok, err := s.rdb.SetNX(ctx, key, body, ttl).Result()
		if err != nil {
			return "", time.Time{}, err
		}
		if ok {
			return code, time.Now().Add(ttl), nil
		}
	}
	return "", time.Time{}, errors.New("pairing: failed to allocate unique code after 5 attempts")
}

// Claim atomically reads-and-deletes a code, returning the bound Grant.
// One-shot semantics: a code can be redeemed exactly once.
func (s *Service) Claim(ctx context.Context, code string) (Grant, error) {
	key := redisKey(code)
	// GETDEL is atomic since Redis 6.2.
	body, err := s.rdb.GetDel(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return Grant{}, ErrNotFound
		}
		return Grant{}, err
	}
	var g Grant
	if err := json.Unmarshal(body, &g); err != nil {
		return Grant{}, fmt.Errorf("pairing: corrupt grant: %w", err)
	}
	return g, nil
}

func redisKey(code string) string { return "pair:" + code }

// newCode draws n alphabet symbols using crypto/rand.  Any "wasted" bits in
// each draw are discarded — the alphabet is exactly 32 symbols so we use
// the bottom 5 bits of each byte.
func newCode(n int) (string, error) {
	out := make([]byte, n)
	// Over-read to avoid bias (we mask 5 bits out of 8).
	src := make([]byte, n*2)
	if _, err := rand.Read(src); err != nil {
		return "", err
	}
	for i := 0; i < n; i++ {
		out[i] = alphabet[src[i]&0x1f]
	}
	return string(out), nil
}

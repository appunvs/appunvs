// Package sequencer assigns monotonic per-namespace sequence numbers via
// Redis INCR.
package sequencer

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Seq is backed by a Redis client.
type Seq struct {
	rdb *redis.Client
}

// New returns a Seq bound to rdb.
func New(rdb *redis.Client) *Seq { return &Seq{rdb: rdb} }

// Next atomically increments seq:{namespace} and returns the new value.
func (s *Seq) Next(ctx context.Context, namespace string) (int64, error) {
	if namespace == "" {
		return 0, fmt.Errorf("sequencer: empty namespace")
	}
	n, err := s.rdb.Incr(ctx, "seq:"+namespace).Result()
	if err != nil {
		return 0, fmt.Errorf("sequencer: incr: %w", err)
	}
	return n, nil
}

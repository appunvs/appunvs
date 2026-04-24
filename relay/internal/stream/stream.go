// Package stream persists provider-origin Messages to a Redis Stream so
// offline clients can catch up on reconnect.
package stream

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/appunvs/appunvs/relay/internal/pb"
)

// Store is a thin wrapper around Redis Streams.
type Store struct {
	rdb    *redis.Client
	maxLen int64
}

// New returns a Store bound to rdb.  maxLen is the approximate cap passed as
// XADD MAXLEN ~; 100000 is a reasonable default.
func New(rdb *redis.Client, maxLen int64) *Store {
	if maxLen <= 0 {
		maxLen = 100000
	}
	return &Store{rdb: rdb, maxLen: maxLen}
}

func key(ns string) string { return "stream:" + ns }

// Append writes msg to stream:{namespace} with approximate MAXLEN trimming.
// The caller must have already assigned msg.Seq.
func (s *Store) Append(ctx context.Context, ns string, msg *pb.Message) error {
	if ns == "" {
		return fmt.Errorf("stream: empty namespace")
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("stream: marshal: %w", err)
	}
	_, err = s.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: key(ns),
		MaxLen: s.maxLen,
		Approx: true,
		Values: map[string]interface{}{
			"seq":     msg.Seq,
			"payload": string(body),
		},
	}).Result()
	if err != nil {
		return fmt.Errorf("stream: xadd: %w", err)
	}
	return nil
}

// Range yields every stored Message with seq > afterSeq, in order.
// Entries without a parseable payload are skipped.
func (s *Store) Range(ctx context.Context, ns string, afterSeq int64) ([]*pb.Message, error) {
	entries, err := s.rdb.XRange(ctx, key(ns), "-", "+").Result()
	if err != nil {
		return nil, fmt.Errorf("stream: xrange: %w", err)
	}
	out := make([]*pb.Message, 0, len(entries))
	for _, e := range entries {
		raw, ok := e.Values["payload"].(string)
		if !ok || raw == "" {
			continue
		}
		var m pb.Message
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			continue
		}
		if m.Seq > afterSeq {
			out = append(out, &m)
		}
	}
	return out, nil
}

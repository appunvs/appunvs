package box

import (
	"testing"
	"time"
)

// TestEventsBasicFanout — a single event published to a namespace
// reaches every subscriber in that namespace and not subscribers in
// other namespaces.
func TestEventsBasicFanout(t *testing.T) {
	e := NewEvents()
	subA := e.Subscribe("ns_alice")
	subB := e.Subscribe("ns_alice")
	subC := e.Subscribe("ns_bob")
	defer e.Unsubscribe(subA)
	defer e.Unsubscribe(subB)
	defer e.Unsubscribe(subC)

	delivered := e.Publish("ns_alice", Event{
		Type:    EventBundleReady,
		BoxID:   "box_1",
		Version: "v1",
	})
	if delivered != 2 {
		t.Fatalf("delivered = %d, want 2", delivered)
	}

	for _, sub := range []*Subscription{subA, subB} {
		select {
		case ev := <-sub.Ch:
			if ev.BoxID != "box_1" {
				t.Errorf("BoxID = %q, want box_1", ev.BoxID)
			}
		case <-time.After(time.Second):
			t.Fatal("subscriber A/B didn't receive")
		}
	}
	select {
	case ev := <-subC.Ch:
		t.Fatalf("ns_bob unexpectedly got event: %+v", ev)
	case <-time.After(50 * time.Millisecond):
		// expected: silent
	}
}

// TestEventsUnsubscribe — after Unsubscribe, the channel is closed and
// subsequent publishes don't see the subscriber.
func TestEventsUnsubscribe(t *testing.T) {
	e := NewEvents()
	sub := e.Subscribe("ns")

	e.Unsubscribe(sub)

	// Channel is closed: receive returns zero value + ok=false.
	select {
	case _, ok := <-sub.Ch:
		if ok {
			t.Fatal("channel should be closed after Unsubscribe")
		}
	case <-time.After(time.Second):
		t.Fatal("read from closed channel should return immediately")
	}

	// Publish to namespace finds zero subscribers.
	if delivered := e.Publish("ns", Event{Type: EventBundleReady}); delivered != 0 {
		t.Fatalf("delivered after Unsubscribe = %d, want 0", delivered)
	}

	// Idempotent.
	e.Unsubscribe(sub)
}

// TestEventsSlowSubscriberDrops — if a subscriber doesn't drain its
// channel, publishes past the buffer are dropped (Publish returns a
// lower delivered count) but the publisher itself never blocks.
// Other subscribers still receive (no head-of-line blocking).
func TestEventsSlowSubscriberDrops(t *testing.T) {
	e := NewEvents()
	slow := e.Subscribe("ns")
	fast := e.Subscribe("ns")
	defer e.Unsubscribe(slow)
	defer e.Unsubscribe(fast)

	// Interleave publish + fast-drain so we exercise the
	// "fast-doesn't-block-on-slow" property deterministically (no
	// goroutine timing involved).
	const total = subBufferSize * 2
	var fastCount int
	for i := 0; i < total; i++ {
		e.Publish("ns", Event{Type: EventBundleReady, BoxID: "box"})
		// Drain one event from fast — keeps it from saturating.
		select {
		case <-fast.Ch:
			fastCount++
		default:
		}
	}

	// Slow subscriber saturated at subBufferSize.
	if got := len(slow.Ch); got != subBufferSize {
		t.Fatalf("slow buffered %d, want %d (saturation)", got, subBufferSize)
	}
	// Fast subscriber received all events: each publish enqueued one,
	// the immediate drain dequeued one.
	if fastCount != total {
		t.Fatalf("fast received %d, want %d", fastCount, total)
	}
}

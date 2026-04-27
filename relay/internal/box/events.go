// Box events — in-process pub/sub for "a bundle is ready" notifications.
//
// Single-process fanout: when BuildAndPublish succeeds, it calls
// Events.Publish; every Subscribe() that matches the namespace receives
// the event on its channel.  Used by the SSE handler at GET /box/events
// so connected hosts can refresh + hot-reload Stage when a publish
// completes (whether the trigger was THIS device's chat-driven
// publish_box, another device's manual POST /box/:id/publish, or a
// future collaborator action).
//
// Cross-process scaling (multi-relay-instance behind a load balancer)
// is NOT handled here — that needs Redis pub/sub or a streams-backed
// fanout layered on top.  For v1 single-instance the channel registry
// is sufficient; switching backends only requires reimplementing
// Events without changing the call sites.
//
// Bounded channels (subBufferSize): a slow subscriber drops events
// rather than blocking the publisher.  On the host side a
// missed-event window is recoverable — the host can call boxes.list
// on (re)connect to catch up.
package box

import (
	"sync"
)

// subBufferSize caps how many pending events a single subscriber can
// queue before further publishes to it are dropped.  16 covers the
// typical case where a host is briefly stalled (GC, view transition)
// without making the publisher block on a wedged subscriber.
const subBufferSize = 16

// EventType is the discriminator carried in `Event.Type` and used as the
// SSE `event:` line by the handler.  More types may be added as
// product features land (bundle_failed, box_archived, etc.).
type EventType string

const (
	// EventBundleReady fires when a publish completes successfully and
	// the box now has a new current_version pointing at a downloadable
	// bundle.
	EventBundleReady EventType = "bundle_ready"
)

// Event is the payload broadcast to subscribers.  Fields mirror
// store.Bundle's wire shape; the host doesn't need to round-trip the
// box list just to know the new URI.
type Event struct {
	Type        EventType `json:"type"`
	BoxID       string    `json:"box_id"`
	Version     string    `json:"version"`
	URI         string    `json:"uri"`
	ContentHash string    `json:"content_hash"`
	SizeBytes   int64     `json:"size_bytes"`
}

// Subscription is the handle a caller holds to receive events.  It is
// safe to range over Ch until the subscription is removed via
// Events.Unsubscribe — at that point Ch is closed.
type Subscription struct {
	Ch        chan Event
	namespace string
}

// Events is the per-namespace fanout registry.  Zero value is unusable;
// construct via NewEvents.
type Events struct {
	mu   sync.Mutex
	subs map[string]map[*Subscription]struct{}
}

// NewEvents returns a ready-to-use registry.
func NewEvents() *Events {
	return &Events{subs: make(map[string]map[*Subscription]struct{})}
}

// Subscribe registers a new subscriber for namespace.  Caller MUST
// pair this with Unsubscribe (typically via defer) — leaking a
// subscription leaks its goroutine and channel.
func (e *Events) Subscribe(namespace string) *Subscription {
	sub := &Subscription{
		Ch:        make(chan Event, subBufferSize),
		namespace: namespace,
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	subs, ok := e.subs[namespace]
	if !ok {
		subs = make(map[*Subscription]struct{})
		e.subs[namespace] = subs
	}
	subs[sub] = struct{}{}
	return sub
}

// Unsubscribe removes sub and closes its channel.  Idempotent — calling
// twice is safe; the second call is a no-op.
func (e *Events) Unsubscribe(sub *Subscription) {
	e.mu.Lock()
	defer e.mu.Unlock()
	subs, ok := e.subs[sub.namespace]
	if !ok {
		return
	}
	if _, present := subs[sub]; !present {
		return
	}
	delete(subs, sub)
	if len(subs) == 0 {
		delete(e.subs, sub.namespace)
	}
	close(sub.Ch)
}

// Publish fans event out to every subscriber of namespace.  Slow
// subscribers (full channel) silently drop the event — see package
// comment on recovery semantics.  Returns the number of subscribers
// that successfully enqueued; useful for tests / metrics.
func (e *Events) Publish(namespace string, event Event) int {
	e.mu.Lock()
	subs := e.subs[namespace]
	// Snapshot the subscriber set so we can release the lock before
	// pushing — pushes do non-blocking sends but we still don't want
	// to hold the registry lock across them.
	targets := make([]*Subscription, 0, len(subs))
	for sub := range subs {
		targets = append(targets, sub)
	}
	e.mu.Unlock()

	delivered := 0
	for _, sub := range targets {
		select {
		case sub.Ch <- event:
			delivered++
		default:
			// Channel full — drop.  See comment at top.
		}
	}
	return delivered
}

// box_events — GET /box/events SSE stream of bundle_ready notifications
// for the authenticated user's boxes.
//
// Hosts open one long-lived connection at sign-in.  When the AI agent
// (or any other actor) calls publish_box for any of the user's boxes,
// box.Service.BuildAndPublish emits an Event into box.Events; this
// handler is subscribed and forwards it as an SSE record.  The host
// reacts by refreshing its box list and letting the Stage view auto-
// reload its RuntimeView (Stage already binds reactively to the
// current bundle URL).
//
// We send a `heartbeat` event every 15s so intermediate proxies and
// load balancers don't kill an idle TCP connection — there's no
// inactivity timer in EventSource itself, but reverse proxies routinely
// drop streams idle past 30-60s.
//
// On the wire, each record is:
//
//   event: bundle_ready
//   data: {"type":"bundle_ready","box_id":"box_…","version":"v1234-…","uri":"https://…","content_hash":"sha256:…","size_bytes":12345}
//
// or:
//
//   event: heartbeat
//   data: {}
//
// followed by the standard blank-line terminator.
package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/box"
)

// BoxEventsDeps groups what GET /box/events needs.
type BoxEventsDeps struct {
	Signer *auth.Signer
	Events *box.Events
	Log    *zap.Logger
}

// heartbeatInterval is how often we emit an `event: heartbeat` record
// to keep idle connections alive through reverse proxies.  15s is well
// inside common 30-60s idle limits while staying invisible-cheap on
// bandwidth (one ~30B record per interval per connected host).
const heartbeatInterval = 15 * time.Second

// RegisterBoxEventsRoutes wires GET /box/events.
func RegisterBoxEventsRoutes(r gin.IRouter, d BoxEventsDeps) {
	r.GET("/box/events", boxEvents(d))
}

func boxEvents(d BoxEventsDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := requireDevice(c, d.Signer)
		if !ok {
			return
		}

		// SSE headers BEFORE WriteHeader runs (gin lazy-writes on first
		// flush).  Disable proxy buffering so events surface immediately.
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("X-Accel-Buffering", "no")
		c.Writer.WriteHeader(http.StatusOK)
		flusher, _ := c.Writer.(http.Flusher)
		if flusher == nil {
			d.Log.Error("box.events: writer is not an http.Flusher; SSE will buffer")
			return
		}
		flusher.Flush()

		sub := d.Events.Subscribe(claims.UserID)
		defer d.Events.Unsubscribe(sub)

		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()

		ctx := c.Request.Context()
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-sub.Ch:
				if !ok {
					// Subscription closed (Unsubscribe ran somewhere else).
					return
				}
				if !writeBoxEvent(c, flusher, string(ev.Type), ev) {
					return
				}
			case <-ticker.C:
				if !writeBoxEvent(c, flusher, "heartbeat", struct{}{}) {
					return
				}
			}
		}
	}
}

// writeBoxEvent emits one SSE record.  Returns false if the write
// fails (broken connection) — caller should exit the stream loop.
func writeBoxEvent(c *gin.Context, flusher http.Flusher, event string, payload any) bool {
	data, err := json.Marshal(payload)
	if err != nil {
		// Should never happen — the payloads are simple structs.
		// If it does, drop the record rather than killing the stream.
		return true
	}
	if _, err := c.Writer.Write([]byte("event: " + event + "\ndata: ")); err != nil {
		return false
	}
	if _, err := c.Writer.Write(data); err != nil {
		return false
	}
	if _, err := c.Writer.Write([]byte("\n\n")); err != nil {
		return false
	}
	flusher.Flush()
	return true
}

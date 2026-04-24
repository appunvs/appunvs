package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/ai"
	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/box"
	"github.com/appunvs/appunvs/relay/internal/store"
)

// AIDeps groups everything /ai/turn depends on.
type AIDeps struct {
	Signer *auth.Signer
	Engine ai.Engine
	Box    *box.Service
	Log    *zap.Logger
}

// RegisterAIRoutes wires POST /ai/turn.  The route requires a device
// JWT; the request body names the Box whose workspace the turn operates
// on (ownership is verified against the caller's namespace).
func RegisterAIRoutes(r gin.IRouter, d AIDeps) {
	r.POST("/ai/turn", aiTurn(d))
}

// aiTurnBody is the inbound JSON shape.
type aiTurnBody struct {
	BoxID string `json:"box_id"`
	Text  string `json:"text"`
}

// aiTurn streams the AI engine's frames as Server-Sent Events.  Each
// frame is one `data:` line with a JSON payload and an `event:` tag
// matching the frame kind.  The stream terminates on `finished` or
// `error`; the client MUST NOT reconnect to resume (per-turn; for
// cross-turn resume, open a new /ai/turn with fresh text).
func aiTurn(d AIDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := requireDevice(c, d.Signer)
		if !ok {
			return
		}
		var body aiTurnBody
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		body.BoxID = strings.TrimSpace(body.BoxID)
		body.Text = strings.TrimSpace(body.Text)
		if body.BoxID == "" || body.Text == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "box_id and text required"})
			return
		}

		// Ownership check — the caller must own the box (same namespace).
		// The engine will also refuse cross-namespace access via the box
		// service, but bailing early saves an SDK round trip.
		if _, _, err := d.Box.Get(c.Request.Context(), claims.UserID, body.BoxID); err != nil {
			if errors.Is(err, store.ErrBoxNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": "box not found"})
				return
			}
			d.Log.Error("ai.turn: box lookup", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}

		// Switch the response into SSE mode BEFORE starting the engine;
		// otherwise a slow model-response prelude can stall behind
		// gin's default buffered writer.
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("X-Accel-Buffering", "no") // nginx: disable proxy buffering
		c.Writer.WriteHeaderNow()

		// Use the request context so client disconnects propagate into
		// the engine goroutine and cancel in-flight SDK calls.
		ctx, cancel := context.WithCancel(c.Request.Context())
		defer cancel()

		frames, err := d.Engine.Run(ctx, ai.Request{
			BoxID:  body.BoxID,
			Text:   body.Text,
			UserID: claims.UserID,
		})
		if err != nil {
			writeSSE(c, "error", gin.H{"error": err.Error()})
			return
		}

		flusher, _ := c.Writer.(http.Flusher)
		for frame := range frames {
			event, payload := frameToSSE(frame)
			writeSSE(c, event, payload)
			if flusher != nil {
				flusher.Flush()
			}
			// Stop draining if the client hung up.  Engine goroutine
			// cleans up via ctx cancellation above.
			if c.Request.Context().Err() != nil {
				return
			}
		}
	}
}

// frameToSSE maps an internal ai.Frame to an (event, payload) pair.
// Exactly one of the optional fields on Frame is non-nil; we pick the
// matching event tag and send the payload as-is.
func frameToSSE(f ai.Frame) (string, any) {
	switch {
	case f.Token != nil:
		return "token", gin.H{"turn_id": f.TurnID, "text": f.Token.Text}
	case f.ToolCall != nil:
		return "tool_call", gin.H{
			"turn_id":   f.TurnID,
			"call_id":   f.ToolCall.CallID,
			"name":      f.ToolCall.Name,
			"args_json": f.ToolCall.ArgsJSON,
		}
	case f.ToolRes != nil:
		return "tool_result", gin.H{
			"turn_id":     f.TurnID,
			"call_id":     f.ToolRes.CallID,
			"result_json": f.ToolRes.ResultJSON,
			"is_error":    f.ToolRes.IsError,
		}
	case f.Finished != nil:
		return "finished", gin.H{
			"turn_id":     f.TurnID,
			"stop_reason": f.Finished.StopReason,
			"tokens_in":   f.Finished.TokensIn,
			"tokens_out":  f.Finished.TokensOut,
		}
	case f.Err != nil:
		return "error", gin.H{"turn_id": f.TurnID, "error": f.Err.Error}
	default:
		return "unknown", gin.H{"turn_id": f.TurnID}
	}
}

// writeSSE emits one SSE record.  Payload is JSON-encoded onto a single
// `data:` line; multi-line JSON is avoided because some proxies split
// on newlines.
func writeSSE(c *gin.Context, event string, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		body = []byte(`{"error":"marshal"}`)
	}
	_, _ = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, body)
}

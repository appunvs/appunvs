// Package ai is the server-side AI agent that turns a Chat turn into a
// stream of token / tool-call / tool-result frames.
//
// The transport-facing contract (Engine) is intentionally minimal so the
// HTTP handler stays unaware of which model provider is in use.  The
// initial implementation shipped here is StubEngine: it echoes the prompt
// back in fixed-size chunks so the WebSocket framing and client-side UI
// can be wired before the Anthropic SDK is dropped in.
//
// Production targets — see docs/architecture.md:
//   - Direct: Anthropic Messages API (Claude Opus 4.7 / Sonnet 4.6) with
//     tool_use loop driving sandbox.Builder + box.Service
//   - Routed: small router fronting Claude / OpenAI / a fast-apply model
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Frame is a single message in the agent → client event stream.  Exactly
// one of the typed fields is non-nil per frame.
type Frame struct {
	TurnID   string        `json:"turn_id"`
	Token    *TokenDelta   `json:"token,omitempty"`
	ToolCall *ToolCall     `json:"tool_call,omitempty"`
	ToolRes  *ToolResult   `json:"tool_res,omitempty"`
	Finished *TurnFinished `json:"finished,omitempty"`
	Err      *TurnError    `json:"error,omitempty"`
}

// TokenDelta is a streamed slice of model output text.
type TokenDelta struct {
	Text string `json:"text"`
}

// ToolCall is a model-issued invocation of a server-side tool.  args_json is
// validated by the tool implementation; the agent does not introspect it.
type ToolCall struct {
	CallID   string `json:"call_id"`
	Name     string `json:"name"`
	ArgsJSON string `json:"args_json"`
}

// ToolResult is the response a tool returns for one ToolCall.
type ToolResult struct {
	CallID     string `json:"call_id"`
	ResultJSON string `json:"result_json"`
	IsError    bool   `json:"is_error,omitempty"`
}

// TurnFinished marks the end of a turn.  Token counts are best-effort —
// providers that don't expose them set both to 0.
type TurnFinished struct {
	StopReason string `json:"stop_reason"`
	TokensIn   int64  `json:"tokens_in,omitempty"`
	TokensOut  int64  `json:"tokens_out,omitempty"`
}

// TurnError carries a fatal error that aborted the turn.  Frames after a
// TurnError MUST NOT be emitted.
type TurnError struct {
	Error string `json:"error"`
}

// Request is the typed input to a turn.
type Request struct {
	BoxID  string
	Text   string
	UserID string
}

// Engine produces a stream of Frames for one Request.  Implementations
// MUST close the returned channel after emitting either Finished or Err.
// The context cancels both the model call and any in-flight tool calls.
type Engine interface {
	Run(ctx context.Context, req Request) (<-chan Frame, error)
}

// StubEngine echoes the prompt back as ~32-byte token chunks with a small
// delay so the client UI can exercise streaming.  Useful for early UI work
// and for offline tests; does NOT hit any external API.
type StubEngine struct {
	ChunkSize int
	Delay     time.Duration
}

// NewStub returns a StubEngine with sane defaults.
func NewStub() *StubEngine {
	return &StubEngine{ChunkSize: 32, Delay: 30 * time.Millisecond}
}

// Run emits TokenDelta frames sized to ChunkSize, followed by a Finished
// frame.  No tool calls are issued; this engine exists only to validate
// the framing.
func (s *StubEngine) Run(ctx context.Context, req Request) (<-chan Frame, error) {
	out := make(chan Frame, 8)
	turnID := uuid.NewString()
	go func() {
		defer close(out)
		text := echoTemplate(req)
		for i := 0; i < len(text); i += s.ChunkSize {
			end := i + s.ChunkSize
			if end > len(text) {
				end = len(text)
			}
			select {
			case <-ctx.Done():
				select {
				case out <- Frame{TurnID: turnID, Err: &TurnError{Error: ctx.Err().Error()}}:
				default:
				}
				return
			case out <- Frame{TurnID: turnID, Token: &TokenDelta{Text: text[i:end]}}:
			}
			if s.Delay > 0 {
				time.Sleep(s.Delay)
			}
		}
		out <- Frame{TurnID: turnID, Finished: &TurnFinished{StopReason: "end_turn"}}
	}()
	return out, nil
}

// echoTemplate is the deterministic stand-in response.  It quotes the user
// turn back so the client can verify round-tripping while real model
// integration is still being wired.
func echoTemplate(req Request) string {
	var b strings.Builder
	fmt.Fprintf(&b, "[stub-engine reply for box=%s]\n", req.BoxID)
	b.WriteString("you said: ")
	b.WriteString(req.Text)
	b.WriteString("\n(replace ai.StubEngine with the Anthropic-backed agent before shipping)\n")
	return b.String()
}

// MarshalArgs is a tiny helper for tool implementations that need to emit a
// JSON-stringified arguments map.
func MarshalArgs(args map[string]any) string {
	if args == nil {
		return "{}"
	}
	b, err := json.Marshal(args)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// Package ai — Anthropic Messages API agent loop.
//
// Mirrors openai_engine.go's contract — same Engine.Run signature, same
// Frame stream — but talks the Messages API instead of OpenAI's chat
// completions.  Differences callers should know:
//
//   - One provider (Anthropic).  No Provider/Registry indirection;
//     just a Model id.
//   - System prompt is a top-level field, not a message.  We attach
//     cache_control: ephemeral so the {tools, system} prefix is
//     server-side cached across turns (90% discount on those tokens).
//   - Tool results are content blocks inside a user message, not a
//     dedicated role.
//
// History persistence: we serialize []anthropic.MessageParam to JSON
// and store it in the same store.Turns table the OpenAI engine writes
// to.  The blob is opaque to relay; only this engine knows how to
// replay it.  Switching engines mid-Box is unsupported — old turns
// stay in DB for audit but are skipped if their format doesn't
// unmarshal cleanly.
package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/box"
	"github.com/appunvs/appunvs/relay/internal/store"
	"github.com/appunvs/appunvs/relay/internal/workspace"
)

// AnthropicConfig controls AnthropicEngine.  APIKey is required;
// everything else has sane defaults.
type AnthropicConfig struct {
	APIKey    string
	Model     string        // default: claude-sonnet-4-6
	BaseURL   string        // default: official Anthropic endpoint
	System    string        // default: shared defaultSystemPrompt
	MaxIters  int           // default: 10
	MaxTokens int           // default: 8000
	Timeout   time.Duration // default: 10m
}

// AnthropicEngine implements Engine against the Anthropic Messages API.
type AnthropicEngine struct {
	client    anthropic.Client
	workspace *workspace.Store
	box       *box.Service
	turns     *store.Turns
	cfg       AnthropicConfig
	log       *zap.Logger
}

// NewAnthropicEngine constructs the engine.
func NewAnthropicEngine(cfg AnthropicConfig, ws *workspace.Store, boxSvc *box.Service, turns *store.Turns, log *zap.Logger) (*AnthropicEngine, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("ai: AnthropicConfig.APIKey required")
	}
	if cfg.Model == "" {
		// Default to the most capable available model.  Hard-coded; if you
		// need to pick a different model per deploy, set Config.Model.
		cfg.Model = string(anthropic.ModelClaudeSonnet4_6)
	}
	if cfg.MaxIters == 0 {
		cfg.MaxIters = 10
	}
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 8000
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Minute
	}
	if cfg.System == "" {
		cfg.System = defaultSystemPrompt
	}

	opts := []option.RequestOption{option.WithAPIKey(cfg.APIKey)}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	return &AnthropicEngine{
		client:    anthropic.NewClient(opts...),
		workspace: ws,
		box:       boxSvc,
		turns:     turns,
		cfg:       cfg,
		log:       log,
	}, nil
}

// Run implements Engine.Run.
func (e *AnthropicEngine) Run(ctx context.Context, req Request) (<-chan Frame, error) {
	if req.BoxID == "" {
		return nil, errors.New("ai: Request.BoxID required")
	}
	turnID := uuid.NewString()
	out := make(chan Frame, 16)

	go func() {
		defer close(out)

		runCtx, cancel := context.WithTimeout(ctx, e.cfg.Timeout)
		defer cancel()

		history, err := e.loadHistory(runCtx, req.BoxID)
		if err != nil {
			e.emitErr(out, turnID, err)
			return
		}
		messages := append([]anthropic.MessageParam{}, history...)
		messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(req.Text)))

		deps := ToolDeps{
			BoxID:     req.BoxID,
			Namespace: req.UserID,
			Workspace: e.workspace,
			Box:       e.box,
		}

		var lastFinish string
		var tokensIn, tokensOut int64

		for iter := 0; iter < e.cfg.MaxIters; iter++ {
			asst, used, finish, err := e.runOne(runCtx, out, turnID, messages)
			if err != nil {
				e.emitErr(out, turnID, err)
				return
			}
			tokensIn += used.in
			tokensOut += used.out
			lastFinish = finish
			messages = append(messages, asst)

			toolUses := extractToolUses(asst)
			if len(toolUses) == 0 {
				break
			}

			// One user message bundles every tool_result for this round —
			// matches Anthropic's expected shape.
			var resultBlocks []anthropic.ContentBlockParamUnion
			for _, tu := range toolUses {
				select {
				case out <- Frame{TurnID: turnID, ToolCall: &ToolCall{
					CallID: tu.id, Name: tu.name, ArgsJSON: tu.argsJSON,
				}}:
				case <-runCtx.Done():
					return
				}
				result, isErr := RunTool(runCtx, deps, tu.name, tu.argsJSON)
				select {
				case out <- Frame{TurnID: turnID, ToolRes: &ToolResult{
					CallID: tu.id, ResultJSON: result, IsError: isErr,
				}}:
				case <-runCtx.Done():
					return
				}
				resultBlocks = append(resultBlocks, anthropic.NewToolResultBlock(tu.id, result, isErr))
			}
			messages = append(messages, anthropic.NewUserMessage(resultBlocks...))
		}

		// Persist the completed turn.  Stored as a JSON array of
		// MessageParam — only this engine knows the schema.
		rowMessages, _ := json.Marshal(messages)
		if err := e.turns.Insert(runCtx, store.Turn{
			ID:         turnID,
			BoxID:      req.BoxID,
			UserText:   req.Text,
			Messages:   string(rowMessages),
			TokensIn:   tokensIn,
			TokensOut:  tokensOut,
			StopReason: lastFinish,
			CreatedAt:  time.Now().UnixMilli(),
		}); err != nil {
			e.log.Warn("ai: persist turn failed", zap.String("turn_id", turnID), zap.Error(err))
		}

		out <- Frame{TurnID: turnID, Finished: &TurnFinished{
			StopReason: lastFinish,
			TokensIn:   tokensIn,
			TokensOut:  tokensOut,
		}}
	}()

	return out, nil
}

// usage holds the just-the-counts subset of anthropic.Usage we care
// about for billing / display.
type usage struct {
	in  int64
	out int64
}

// extractedToolUse pulls a flat (id, name, argsJSON) view out of an
// accumulated assistant message's tool_use content blocks.
type extractedToolUse struct {
	id       string
	name     string
	argsJSON string
}

func extractToolUses(asst anthropic.MessageParam) []extractedToolUse {
	var out []extractedToolUse
	for _, block := range asst.Content {
		if block.OfToolUse == nil {
			continue
		}
		tu := block.OfToolUse
		args, _ := json.Marshal(tu.Input)
		out = append(out, extractedToolUse{
			id:       tu.ID,
			name:     tu.Name,
			argsJSON: string(args),
		})
	}
	return out
}

// runOne issues one streaming Messages call, forwards text deltas as
// Token frames, accumulates the full response into a MessageParam ready
// to append to history, and returns it + the final stop_reason + usage.
func (e *AnthropicEngine) runOne(ctx context.Context, out chan<- Frame, turnID string, messages []anthropic.MessageParam) (anthropic.MessageParam, usage, string, error) {
	tools, err := buildAnthropicTools()
	if err != nil {
		return anthropic.MessageParam{}, usage{}, "", err
	}
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(e.cfg.Model),
		MaxTokens: int64(e.cfg.MaxTokens),
		Messages:  messages,
		System: []anthropic.TextBlockParam{
			{
				Text: e.cfg.System,
				// Cache the system prompt + tool prefix for ~90% cost cut on
				// repeat turns.  Anthropic hashes the prefix; any byte
				// difference invalidates.
				CacheControl: anthropic.CacheControlEphemeralParam{},
			},
		},
		Tools: tools,
	}

	stream := e.client.Messages.NewStreaming(ctx, params)
	defer func() { _ = stream.Close() }()

	acc := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		if err := acc.Accumulate(event); err != nil {
			return anthropic.MessageParam{}, usage{}, "", fmt.Errorf("ai: stream accumulate: %w", err)
		}
		// Forward text deltas as Token frames — the only event we surface
		// to the client mid-stream.  tool_use is announced via a ToolCall
		// frame AFTER the stream completes (Anthropic's input_json_delta
		// arrives partial; reading after stop is far simpler than buffering
		// per-block).
		switch ev := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			if td := ev.Delta.AsTextDelta(); td.Text != "" {
				select {
				case out <- Frame{TurnID: turnID, Token: &TokenDelta{Text: td.Text}}:
				case <-ctx.Done():
					return anthropic.MessageParam{}, usage{}, "", ctx.Err()
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		return anthropic.MessageParam{}, usage{}, "", err
	}

	// Convert the accumulated Message (response shape) into a
	// MessageParam (history shape).  They have parallel content but
	// different Go types because one is server→client and the other is
	// client→server.
	asst := anthropic.MessageParam{
		Role:    anthropic.MessageParamRoleAssistant,
		Content: contentBlocksToParams(acc.Content),
	}
	used := usage{
		in:  acc.Usage.InputTokens,
		out: acc.Usage.OutputTokens,
	}
	return asst, used, string(acc.StopReason), nil
}

// contentBlocksToParams maps response content blocks (text / tool_use)
// onto their input-form counterparts so we can append to history.
func contentBlocksToParams(blocks []anthropic.ContentBlockUnion) []anthropic.ContentBlockParamUnion {
	var out []anthropic.ContentBlockParamUnion
	for _, b := range blocks {
		switch b.Type {
		case "text":
			out = append(out, anthropic.NewTextBlock(b.Text))
		case "tool_use":
			// Input on the response side is raw JSON bytes; pass through
			// to the assistant-message content block as a json.RawMessage
			// so it's re-encoded verbatim.
			var input any = json.RawMessage(b.Input)
			if len(b.Input) == 0 {
				input = json.RawMessage(`{}`)
			}
			out = append(out, anthropic.NewToolUseBlock(b.ID, input, b.Name))
		}
		// Other block types (thinking, server tool use, etc.) are
		// dropped — we don't issue them and the model won't either with
		// our current toolset.
	}
	return out
}

// loadHistory rebuilds Anthropic-shaped conversation state from prior
// turns.  Skips rows whose Messages JSON doesn't unmarshal as a
// []MessageParam (e.g. turns persisted by a different engine).
func (e *AnthropicEngine) loadHistory(ctx context.Context, boxID string) ([]anthropic.MessageParam, error) {
	rows, err := e.turns.Recent(ctx, boxID, 20)
	if err != nil {
		return nil, err
	}
	var out []anthropic.MessageParam
	for i := len(rows) - 1; i >= 0; i-- {
		var msgs []anthropic.MessageParam
		if err := json.Unmarshal([]byte(rows[i].Messages), &msgs); err != nil {
			e.log.Warn("ai: skipping unrecognized turn history (different engine?)",
				zap.String("turn_id", rows[i].ID))
			continue
		}
		out = append(out, msgs...)
	}
	return out, nil
}

func (e *AnthropicEngine) emitErr(out chan<- Frame, turnID string, err error) {
	e.log.Warn("ai: turn aborted", zap.String("turn_id", turnID), zap.Error(err))
	select {
	case out <- Frame{TurnID: turnID, Err: &TurnError{Error: err.Error()}}:
	default:
	}
	if e.turns != nil {
		_ = e.turns.Insert(context.Background(), store.Turn{
			ID:         turnID,
			UserText:   "",
			StopReason: "error:" + err.Error(),
			CreatedAt:  time.Now().UnixMilli(),
		})
	}
}

// buildAnthropicTools translates our shared tools.Tools() (OpenAI shape)
// into Anthropic's ToolUnionParam list.  Done once per turn; cheap.
func buildAnthropicTools() ([]anthropic.ToolUnionParam, error) {
	var out []anthropic.ToolUnionParam
	for _, t := range Tools() {
		if t.Function == nil {
			continue
		}
		// Marshal the OpenAI-style jsonschema.Definition + roundtrip into
		// an `any` map so the SDK's Properties field accepts it.  Anthropic
		// expects properties as a generic map, not a typed struct.
		var props any
		if t.Function.Parameters != nil {
			raw, err := json.Marshal(t.Function.Parameters)
			if err != nil {
				return nil, fmt.Errorf("ai: marshal tool %q schema: %w", t.Function.Name, err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(raw, &parsed); err != nil {
				return nil, fmt.Errorf("ai: parse tool %q schema: %w", t.Function.Name, err)
			}
			props = parsed["properties"]
		}
		var required []string
		if t.Function.Parameters != nil {
			raw, _ := json.Marshal(t.Function.Parameters)
			var parsed struct {
				Required []string `json:"required"`
			}
			_ = json.Unmarshal(raw, &parsed)
			required = parsed.Required
		}
		tool := anthropic.ToolParam{
			Name:        t.Function.Name,
			Description: anthropic.String(t.Function.Description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: props,
				Required:   required,
			},
		}
		out = append(out, anthropic.ToolUnionParam{OfTool: &tool})
	}
	return out, nil
}

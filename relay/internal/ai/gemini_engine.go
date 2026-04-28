// Package ai — native Gemini API agent loop.
//
// Mirrors anthropic_engine.go's contract — same Engine.Run signature,
// same Frame stream — but talks the native Google Gemini API
// (generativelanguage.googleapis.com) via google.golang.org/genai.
// We deliberately pick the native path over Gemini's OpenAI-compat
// endpoint because tool calling on the OpenAI-compat layer is partial
// and the appunvs agent is tool-heavy (fs_write / publish_box).
//
// Differences callers should know:
//
//   - Conversation roles are "user" and "model" (vs Anthropic's
//     user/assistant); function results go in a user message as
//     FunctionResponse parts.
//   - System prompt is a separate Content on GenerateContentConfig.
//     SystemInstruction, not a role in messages.
//   - Streaming: each chunk is a *GenerateContentResponse; we forward
//     text parts as Token frames and collect FunctionCall parts to
//     dispatch after the stream closes (Gemini API backend doesn't
//     support streamed function-call args).
//
// History persistence: we serialize []*genai.Content as JSON into the
// same store.Turns.Messages column the OpenAI/Anthropic engines write
// to. The blob is opaque to relay; only this engine knows how to
// replay it. Switching engines mid-Box is unsupported.
package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/genai"

	"github.com/appunvs/appunvs/relay/internal/box"
	"github.com/appunvs/appunvs/relay/internal/store"
	"github.com/appunvs/appunvs/relay/internal/workspace"
)

// GeminiConfig controls GeminiEngine.  APIKey is required; everything
// else has sane defaults.
type GeminiConfig struct {
	APIKey    string
	Model     string        // default: gemini-2.5-pro
	System    string        // default: shared defaultSystemPrompt
	MaxIters  int           // default: 10
	MaxTokens int           // default: 8000
	Timeout   time.Duration // default: 10m
}

// GeminiEngine implements Engine against the native Gemini API.
type GeminiEngine struct {
	client    *genai.Client
	workspace *workspace.Store
	box       *box.Service
	turns     *store.Turns
	cfg       GeminiConfig
	log       *zap.Logger
}

// NewGeminiEngine constructs the engine.
func NewGeminiEngine(cfg GeminiConfig, ws *workspace.Store, boxSvc *box.Service, turns *store.Turns, log *zap.Logger) (*GeminiEngine, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("ai: GeminiConfig.APIKey required")
	}
	if cfg.Model == "" {
		cfg.Model = "gemini-2.5-pro"
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

	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("ai: gemini client: %w", err)
	}
	return &GeminiEngine{
		client:    client,
		workspace: ws,
		box:       boxSvc,
		turns:     turns,
		cfg:       cfg,
		log:       log,
	}, nil
}

// Run implements Engine.Run.
func (e *GeminiEngine) Run(ctx context.Context, req Request) (<-chan Frame, error) {
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
		messages := append([]*genai.Content{}, history...)
		messages = append(messages, &genai.Content{
			Role:  "user",
			Parts: []*genai.Part{{Text: req.Text}},
		})

		deps := ToolDeps{
			BoxID:     req.BoxID,
			Namespace: req.UserID,
			Workspace: e.workspace,
			Box:       e.box,
		}

		var lastFinish string
		var tokensIn, tokensOut int64

		// Per-turn model override; same semantics as the other engines.
		effectiveModel := e.cfg.Model
		if req.Model != "" {
			effectiveModel = req.Model
		}

		for iter := 0; iter < e.cfg.MaxIters; iter++ {
			modelMsg, calls, used, finish, err := e.runOne(runCtx, out, turnID, messages, effectiveModel)
			if err != nil {
				e.emitErr(out, turnID, err)
				return
			}
			tokensIn += used.in
			tokensOut += used.out
			lastFinish = finish
			messages = append(messages, modelMsg)

			if len(calls) == 0 {
				break
			}

			// Bundle every tool result into a single user message —
			// matches Gemini's expected shape.
			var resultParts []*genai.Part
			for _, c := range calls {
				argsJSON, _ := json.Marshal(c.Args)
				select {
				case out <- Frame{TurnID: turnID, ToolCall: &ToolCall{
					CallID: c.ID, Name: c.Name, ArgsJSON: string(argsJSON),
				}}:
				case <-runCtx.Done():
					return
				}
				result, isErr := RunTool(runCtx, deps, c.Name, string(argsJSON))
				select {
				case out <- Frame{TurnID: turnID, ToolRes: &ToolResult{
					CallID: c.ID, ResultJSON: result, IsError: isErr,
				}}:
				case <-runCtx.Done():
					return
				}
				// Wrap the tool's string output into a JSON object — the
				// Gemini API expects FunctionResponse.Response as a
				// map[string]any.  IsError is signalled in-band.
				resp := map[string]any{"result": result}
				if isErr {
					resp = map[string]any{"error": result}
				}
				resultParts = append(resultParts, &genai.Part{
					FunctionResponse: &genai.FunctionResponse{
						Name:     c.Name,
						Response: resp,
					},
				})
			}
			messages = append(messages, &genai.Content{
				Role:  "user",
				Parts: resultParts,
			})
		}

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

// extractedGeminiCall flattens a FunctionCall part into the shape our
// agent loop deals in.  IDs are synthesized — Gemini doesn't issue
// per-call ids the way Anthropic does.
type extractedGeminiCall struct {
	ID   string
	Name string
	Args map[string]any
}

// runOne issues one streaming GenerateContent call.  Forwards text
// deltas as Token frames; returns the accumulated model message, any
// function calls it emitted, usage counts, and the finish reason.
func (e *GeminiEngine) runOne(ctx context.Context, out chan<- Frame, turnID string, messages []*genai.Content, model string) (*genai.Content, []extractedGeminiCall, usage, string, error) {
	tools, err := buildGeminiTools()
	if err != nil {
		return nil, nil, usage{}, "", err
	}

	maxOut := int32(e.cfg.MaxTokens)
	cfg := &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{
			Parts: []*genai.Part{{Text: e.cfg.System}},
		},
		Tools:           tools,
		MaxOutputTokens: maxOut,
	}

	var assembledParts []*genai.Part
	var calls []extractedGeminiCall
	var used usage
	var finish string

	for resp, streamErr := range e.client.Models.GenerateContentStream(ctx, model, messages, cfg) {
		if streamErr != nil {
			return nil, nil, usage{}, "", streamErr
		}
		if resp == nil || len(resp.Candidates) == 0 {
			continue
		}
		cand := resp.Candidates[0]
		if cand.FinishReason != "" {
			finish = string(cand.FinishReason)
		}
		if cand.Content == nil {
			continue
		}
		for _, part := range cand.Content.Parts {
			switch {
			case part.Text != "" && !part.Thought:
				select {
				case out <- Frame{TurnID: turnID, Token: &TokenDelta{Text: part.Text}}:
				case <-ctx.Done():
					return nil, nil, usage{}, "", ctx.Err()
				}
				assembledParts = append(assembledParts, &genai.Part{Text: part.Text})
			case part.FunctionCall != nil:
				fc := part.FunctionCall
				id := fc.ID
				if id == "" {
					id = uuid.NewString()
				}
				calls = append(calls, extractedGeminiCall{
					ID:   id,
					Name: fc.Name,
					Args: fc.Args,
				})
				assembledParts = append(assembledParts, &genai.Part{FunctionCall: fc})
			}
		}
		if u := resp.UsageMetadata; u != nil {
			used.in = int64(u.PromptTokenCount)
			used.out = int64(u.CandidatesTokenCount)
		}
	}

	modelMsg := &genai.Content{
		Role:  "model",
		Parts: assembledParts,
	}
	if finish == "" {
		finish = "STOP"
	}
	return modelMsg, calls, used, finish, nil
}

// loadHistory rebuilds Gemini-shaped conversation state from prior
// turns.  Skips rows whose Messages JSON doesn't unmarshal cleanly
// (turns persisted by a different engine).
func (e *GeminiEngine) loadHistory(ctx context.Context, boxID string) ([]*genai.Content, error) {
	rows, err := e.turns.Recent(ctx, boxID, 20)
	if err != nil {
		return nil, err
	}
	var out []*genai.Content
	for i := len(rows) - 1; i >= 0; i-- {
		var msgs []*genai.Content
		if err := json.Unmarshal([]byte(rows[i].Messages), &msgs); err != nil {
			e.log.Warn("ai: skipping unrecognized turn history (different engine?)",
				zap.String("turn_id", rows[i].ID))
			continue
		}
		out = append(out, msgs...)
	}
	return out, nil
}

func (e *GeminiEngine) emitErr(out chan<- Frame, turnID string, err error) {
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

// buildGeminiTools translates our shared tools.Tools() (OpenAI shape)
// into Gemini's []*Tool with FunctionDeclarations.  We pass the
// JSON-Schema map through ParametersJsonSchema rather than translating
// to genai.Schema so we don't have to maintain a parallel schema
// builder.
func buildGeminiTools() ([]*genai.Tool, error) {
	var decls []*genai.FunctionDeclaration
	for _, t := range Tools() {
		if t.Function == nil {
			continue
		}
		var schema any
		if t.Function.Parameters != nil {
			raw, err := json.Marshal(t.Function.Parameters)
			if err != nil {
				return nil, fmt.Errorf("ai: marshal tool %q schema: %w", t.Function.Name, err)
			}
			var parsed map[string]any
			if err := json.Unmarshal(raw, &parsed); err != nil {
				return nil, fmt.Errorf("ai: parse tool %q schema: %w", t.Function.Name, err)
			}
			schema = parsed
		}
		decls = append(decls, &genai.FunctionDeclaration{
			Name:                 t.Function.Name,
			Description:          t.Function.Description,
			ParametersJsonSchema: schema,
		})
	}
	return []*genai.Tool{{FunctionDeclarations: decls}}, nil
}

// Package ai — OpenAI-compatible agent loop (DeepSeek by default).
//
// The engine drives one Chat turn end-to-end:
//
//	for {
//	    stream = CreateChatCompletionStream(...)        // one model call
//	    while stream.Recv() { forward text deltas, accumulate tool_calls }
//	    append assistant turn to history
//	    if no tool_calls: break
//	    for each tool_call: run handler, append tool_result as role=tool
//	}
//
// The client sees a single SSE stream of token / tool_call / tool_result
// frames terminated by a finished frame; the N model round-trips inside
// one turn are invisible from the outside.
//
// Why OpenAI-compatible and not provider-native SDKs: DeepSeek, 火山 Ark,
// 阿里百炼, 智谱 GLM, Moonshot all expose the same wire shape.  One client
// pointed at a different `BaseURL` + `Model` swaps provider without
// touching the agent.
package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	openai "github.com/sashabaranov/go-openai"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/box"
	"github.com/appunvs/appunvs/relay/internal/store"
	"github.com/appunvs/appunvs/relay/internal/workspace"
)

// Config controls the DeepSeekEngine.  Zero values fall back to sane
// defaults except `APIKey`, which must be set explicitly.
type Config struct {
	BaseURL   string        // default: https://api.deepseek.com/v1
	APIKey    string        // required
	Model     string        // default: deepseek-chat
	System    string        // default: built-in agent system prompt
	MaxIters  int           // hard cap on tool-use iterations, default 10
	MaxTokens int           // per-turn max_tokens, default 8000
	Timeout   time.Duration // per-turn wall clock, default 10m
}

// Common model strings.  Pass a literal if your provider uses a different
// id (e.g. `glm-4.6`, `qwen3-coder-plus`, `doubao-seed-coder`).
const (
	ModelDeepSeekChat      = "deepseek-chat"
	ModelDeepSeekReasoner  = "deepseek-reasoner"
	DefaultDeepSeekBaseURL = "https://api.deepseek.com/v1"
)

// DeepSeekEngine implements Engine against any OpenAI-compatible provider.
// Named for the default target; works against DeepSeek / 火山 Ark / 阿里
// 百炼 / 智谱 GLM / Moonshot / OpenAI with only `BaseURL` and `Model`
// differing.
type DeepSeekEngine struct {
	client    *openai.Client
	workspace *workspace.Store
	box       *box.Service
	turns     *store.Turns
	cfg       Config
	log       *zap.Logger
}

// NewDeepSeekEngine wires everything up.
func NewDeepSeekEngine(cfg Config, ws *workspace.Store, boxSvc *box.Service, turns *store.Turns, log *zap.Logger) (*DeepSeekEngine, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("ai: APIKey required")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultDeepSeekBaseURL
	}
	if cfg.Model == "" {
		cfg.Model = ModelDeepSeekChat
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
	ocfg := openai.DefaultConfig(cfg.APIKey)
	ocfg.BaseURL = cfg.BaseURL
	return &DeepSeekEngine{
		client:    openai.NewClientWithConfig(ocfg),
		workspace: ws,
		box:       boxSvc,
		turns:     turns,
		cfg:       cfg,
		log:       log,
	}, nil
}

// defaultSystemPrompt is byte-stable across turns — the prefix
// `tools + system` gets cached server-side, so keep it deterministic
// (no timestamps / per-user state / varying tool order).
const defaultSystemPrompt = `You are appunvs's coding agent. You edit a React Native (Expo + TypeScript) project stored as a git workspace on the relay. Your goal each turn is to advance the project toward the user's intent and, when the change set is coherent, publish a new bundle.

Tools available:
- fs_read(path): read one file at HEAD.
- fs_write(path, content): overwrite or create a file — each call is one commit.
- list_files(): enumerate tracked paths at HEAD.
- publish_box(entry_point?): build the current HEAD into an immutable bundle and mark the Box PUBLISHED. Call this exactly once at the end of a turn when the code compiles and implements the requested change. Skip it if the turn is purely exploratory.

Rules:
- Every source file must be valid TSX/TS that Metro can bundle. Entry point defaults to index.tsx.
- Prefer many small files over one large file; keep imports clean.
- Never write secrets, API keys, or user tokens into the workspace.
- If a user request is ambiguous, ask a clarifying question instead of guessing.
- When you publish, include a one-sentence summary of what changed.`

// Run implements Engine.Run.  The returned channel emits Token / ToolCall
// / ToolResult frames as the agent progresses, terminating with a
// Finished or Err frame.
func (e *DeepSeekEngine) Run(ctx context.Context, req Request) (<-chan Frame, error) {
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
		// System prompt first so the provider can cache {tools, system}
		// together; history (varying) lives after that; current user
		// message is the only truly-fresh bytes per turn.
		messages := []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: e.cfg.System},
		}
		messages = append(messages, history...)
		messages = append(messages, openai.ChatCompletionMessage{
			Role: openai.ChatMessageRoleUser, Content: req.Text,
		})

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
			tokensIn += int64(used.PromptTokens)
			tokensOut += int64(used.CompletionTokens)
			lastFinish = finish
			messages = append(messages, asst)

			// Stop if the model stopped asking for tools.  DeepSeek returns
			// finish_reason="tool_calls" when more tools are wanted and
			// "stop" once the assistant is done.
			if len(asst.ToolCalls) == 0 {
				break
			}

			for _, tc := range asst.ToolCalls {
				// Announce call before running so UI can render "running fs_write..."
				select {
				case out <- Frame{TurnID: turnID, ToolCall: &ToolCall{
					CallID: tc.ID, Name: tc.Function.Name, ArgsJSON: tc.Function.Arguments,
				}}:
				case <-runCtx.Done():
					return
				}
				result, isErr := RunTool(runCtx, deps, tc.Function.Name, tc.Function.Arguments)
				select {
				case out <- Frame{TurnID: turnID, ToolRes: &ToolResult{
					CallID: tc.ID, ResultJSON: result, IsError: isErr,
				}}:
				case <-runCtx.Done():
					return
				}
				// OpenAI protocol: each tool_result is a separate message
				// with role=tool and tool_call_id matching the assistant's call.
				messages = append(messages, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    result,
					ToolCallID: tc.ID,
				})
			}
		}

		// Persist the completed turn.  Store the full messages chain so
		// the next turn can replay it verbatim.
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

// runOne issues one streaming chat completion, forwards text deltas to
// out as they arrive, accumulates tool_calls (which stream as partial
// JSON across many deltas in the OpenAI protocol), and returns the
// assembled assistant message + finish reason + usage.
func (e *DeepSeekEngine) runOne(ctx context.Context, out chan<- Frame, turnID string, messages []openai.ChatCompletionMessage) (openai.ChatCompletionMessage, openai.Usage, string, error) {
	req := openai.ChatCompletionRequest{
		Model:     e.cfg.Model,
		Messages:  messages,
		Tools:     Tools(),
		MaxTokens: e.cfg.MaxTokens,
		Stream:    true,
		StreamOptions: &openai.StreamOptions{
			IncludeUsage: true, // DeepSeek honors this, returns final usage block.
		},
	}
	stream, err := e.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		return openai.ChatCompletionMessage{}, openai.Usage{}, "", err
	}
	defer func() { _ = stream.Close() }()

	var content string
	var finish string
	var usage openai.Usage
	// tool_calls arrive as incremental deltas indexed by position; we
	// accumulate name + arguments strings per index, then compact into a
	// ToolCall slice once the stream ends.
	type pending struct {
		id   string
		name string
		args string
	}
	byIdx := map[int]*pending{}

	for {
		resp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return openai.ChatCompletionMessage{}, openai.Usage{}, "", err
		}
		if resp.Usage != nil {
			usage = *resp.Usage
		}
		for _, choice := range resp.Choices {
			if choice.FinishReason != "" {
				finish = string(choice.FinishReason)
			}
			if choice.Delta.Content != "" {
				content += choice.Delta.Content
				select {
				case out <- Frame{TurnID: turnID, Token: &TokenDelta{Text: choice.Delta.Content}}:
				case <-ctx.Done():
					return openai.ChatCompletionMessage{}, openai.Usage{}, "", ctx.Err()
				}
			}
			for _, d := range choice.Delta.ToolCalls {
				idx := 0
				if d.Index != nil {
					idx = *d.Index
				}
				p, ok := byIdx[idx]
				if !ok {
					p = &pending{}
					byIdx[idx] = p
				}
				if d.ID != "" {
					p.id = d.ID
				}
				if d.Function.Name != "" {
					p.name = d.Function.Name
				}
				if d.Function.Arguments != "" {
					p.args += d.Function.Arguments
				}
			}
		}
	}

	// Assemble tool_calls in index order.
	var toolCalls []openai.ToolCall
	for i := 0; i < len(byIdx); i++ {
		p, ok := byIdx[i]
		if !ok {
			continue
		}
		toolCalls = append(toolCalls, openai.ToolCall{
			ID:   p.id,
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionCall{
				Name:      p.name,
				Arguments: p.args,
			},
		})
	}

	asst := openai.ChatCompletionMessage{
		Role:      openai.ChatMessageRoleAssistant,
		Content:   content,
		ToolCalls: toolCalls,
	}
	return asst, usage, finish, nil
}

// loadHistory rebuilds conversation state from the last N turns.  Capped
// by count in v1; compaction lands when real users start overrunning
// 128k context.
func (e *DeepSeekEngine) loadHistory(ctx context.Context, boxID string) ([]openai.ChatCompletionMessage, error) {
	rows, err := e.turns.Recent(ctx, boxID, 20)
	if err != nil {
		return nil, err
	}
	// Recent() returns newest-first; chat order needs oldest-first.
	var out []openai.ChatCompletionMessage
	for i := len(rows) - 1; i >= 0; i-- {
		var msgs []openai.ChatCompletionMessage
		if err := json.Unmarshal([]byte(rows[i].Messages), &msgs); err != nil {
			e.log.Warn("ai: skipping corrupt turn history",
				zap.String("turn_id", rows[i].ID), zap.Error(err))
			continue
		}
		// Drop the stored system prompt from each replayed turn; we add
		// one fresh copy at the start of the current turn.  Leaving it
		// in duplicates system prompts across turns.
		for _, m := range msgs {
			if m.Role == openai.ChatMessageRoleSystem {
				continue
			}
			out = append(out, m)
		}
	}
	return out, nil
}

func (e *DeepSeekEngine) emitErr(out chan<- Frame, turnID string, err error) {
	e.log.Warn("ai: turn aborted", zap.String("turn_id", turnID), zap.Error(err))
	// Best-effort send; if the consumer gave up we don't want to block.
	select {
	case out <- Frame{TurnID: turnID, Err: &TurnError{Error: err.Error()}}:
	default:
	}
	// Also record that this turn bailed so billing / debugging has a trail.
	if e.turns != nil {
		_ = e.turns.Insert(context.Background(), store.Turn{
			ID:         turnID,
			UserText:   "",
			StopReason: "error:" + err.Error(),
			CreatedAt:  time.Now().UnixMilli(),
		})
	}
}

// Ensure the fmt import is kept even if the file only uses it through Errorf
// in tool replies (the linter occasionally prunes it otherwise).
var _ = fmt.Sprintf

package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"

	"github.com/appunvs/appunvs/relay/internal/box"
	"github.com/appunvs/appunvs/relay/internal/sandbox"
	"github.com/appunvs/appunvs/relay/internal/workspace"
)

// ToolDeps is everything a tool handler may need.  Handlers receive it
// frozen at turn start so a tool can't bump into a swapped dependency
// mid-turn.
type ToolDeps struct {
	BoxID     string
	Namespace string
	Workspace *workspace.Store
	Box       *box.Service
}

// Tools returns the tool schema list the engine hands to DeepSeek.  Keep
// the list stable across turns for prompt-cache friendliness — DeepSeek,
// like Anthropic, hashes `tools` + `system` together; any byte difference
// invalidates the cache.
func Tools() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "fs_read",
				Description: "Read a single file from the Box workspace at HEAD. Returns the file's contents as text, or an error if the path does not exist.",
				Parameters: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"path": {
							Type:        jsonschema.String,
							Description: "POSIX-style path relative to the workspace root, e.g. \"index.tsx\" or \"src/lib/api.ts\".",
						},
					},
					Required: []string{"path"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "fs_write",
				Description: "Write (create or overwrite) a single file in the Box workspace. Each call produces one git commit. Use this for every source edit; do not batch unrelated edits into one path.",
				Parameters: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"path": {
							Type:        jsonschema.String,
							Description: "POSIX-style path relative to the workspace root.",
						},
						"content": {
							Type:        jsonschema.String,
							Description: "Full file contents. The previous contents at this path (if any) are replaced.",
						},
					},
					Required: []string{"path", "content"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "list_files",
				Description: "List every tracked file in the Box workspace at HEAD. Returns an array of POSIX paths.",
				Parameters: &jsonschema.Definition{
					Type:       jsonschema.Object,
					Properties: map[string]jsonschema.Definition{},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "publish_box",
				Description: "Build the current workspace HEAD into an immutable bundle, upload it, and mark the Box as PUBLISHED. Call this only when the code is complete for this turn; subsequent edits will require another publish_box call to become visible to connectors.",
				Parameters: &jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"entry_point": {
							Type:        jsonschema.String,
							Description: "Optional entrypoint module, defaults to \"index.tsx\".",
						},
					},
				},
			},
		},
	}
}

// RunTool dispatches a tool call to its implementation and returns the
// string content of the tool_result.  Returns (result, isError).
func RunTool(ctx context.Context, d ToolDeps, name string, rawArgs string) (string, bool) {
	switch name {
	case "fs_read":
		var in struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(rawArgs), &in); err != nil {
			return fmt.Sprintf("invalid input: %v", err), true
		}
		if in.Path == "" {
			return "missing path", true
		}
		repo, err := d.Workspace.Open(d.BoxID)
		if err != nil {
			return err.Error(), true
		}
		body, err := repo.ReadFile(ctx, in.Path)
		if err != nil {
			if errors.Is(err, workspace.ErrFileNotFound) {
				return fmt.Sprintf("file not found: %s", in.Path), true
			}
			return err.Error(), true
		}
		return string(body), false

	case "fs_write":
		var in struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(rawArgs), &in); err != nil {
			return fmt.Sprintf("invalid input: %v", err), true
		}
		if in.Path == "" {
			return "missing path", true
		}
		repo, err := d.Workspace.Open(d.BoxID)
		if err != nil {
			return err.Error(), true
		}
		sha, err := repo.Commit(ctx, []workspace.WriteOp{
			{Path: in.Path, Content: []byte(in.Content)},
		}, fmt.Sprintf("ai: write %s", in.Path), "appunvs", "agent@appunvs")
		if err != nil {
			return err.Error(), true
		}
		return fmt.Sprintf("ok: commit %s", sha[:8]), false

	case "list_files":
		repo, err := d.Workspace.Open(d.BoxID)
		if err != nil {
			return err.Error(), true
		}
		files, err := repo.ListFiles(ctx)
		if err != nil {
			return err.Error(), true
		}
		if len(files) == 0 {
			return "(workspace is empty)", false
		}
		encoded, _ := json.Marshal(files)
		return string(encoded), false

	case "publish_box":
		var in struct {
			EntryPoint string `json:"entry_point"`
		}
		_ = json.Unmarshal([]byte(rawArgs), &in)
		if in.EntryPoint == "" {
			in.EntryPoint = "index.tsx"
		}
		bundle, err := d.Box.BuildAndPublish(ctx, d.Namespace, sandbox.Source{
			BoxID:      d.BoxID,
			EntryPoint: in.EntryPoint,
		})
		if err != nil {
			return err.Error(), true
		}
		out := map[string]any{
			"version":      bundle.Version,
			"content_hash": bundle.ContentHash,
			"size_bytes":   bundle.SizeBytes,
			"uri":          bundle.URI,
		}
		encoded, _ := json.Marshal(out)
		return string(encoded), false
	}

	return fmt.Sprintf("unknown tool: %s", name), true
}

package ai_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/appunvs/appunvs/relay/internal/ai"
	"github.com/appunvs/appunvs/relay/internal/artifact"
	"github.com/appunvs/appunvs/relay/internal/box"
	"github.com/appunvs/appunvs/relay/internal/pb"
	"github.com/appunvs/appunvs/relay/internal/sandbox"
	"github.com/appunvs/appunvs/relay/internal/store"
	"github.com/appunvs/appunvs/relay/internal/workspace"
)

// TestRunToolEndToEnd exercises every tool handler against real
// workspace / box / artifact collaborators.  This is the regression
// guard for the AI tool surface — any rename, protocol tweak, or
// argument shape change will break a case here before it breaks live
// traffic.
func TestRunToolEndToEnd(t *testing.T) {
	ctx := context.Background()
	tmp := t.TempDir()

	st, err := store.Open(ctx, tmp+"/relay.db")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer func() { _ = st.Close() }()
	if _, err := st.DB.ExecContext(ctx,
		`INSERT INTO users(id, email, password_hash, created_at) VALUES(?,?,?,?)`,
		"u_t", "t@e.com", "x", int64(1)); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	ws, _ := workspace.NewStore(workspace.Config{Root: tmp + "/ws"})
	art, _ := artifact.NewLocalFS(tmp+"/art", "http://localhost:8080/_artifacts")
	svc := box.New(st.Boxes(), sandbox.NewLocalStub(), art, ws, nil)

	b, err := svc.Create(ctx, "u_t", "dev_a", "demo", pb.RuntimeKindRNBundle)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	deps := ai.ToolDeps{BoxID: b.ID, Namespace: "u_t", Workspace: ws, Box: svc}

	// fs_write creates a new commit.
	out, isErr := ai.RunTool(ctx, deps, "fs_write", `{"path":"index.tsx","content":"export default function App(){return null}\n"}`)
	if isErr {
		t.Fatalf("fs_write errored: %s", out)
	}
	if !strings.HasPrefix(out, "ok: commit ") {
		t.Fatalf("fs_write result = %q, want commit confirmation", out)
	}

	// fs_read returns what we just wrote.
	out, isErr = ai.RunTool(ctx, deps, "fs_read", `{"path":"index.tsx"}`)
	if isErr {
		t.Fatalf("fs_read errored: %s", out)
	}
	if !strings.Contains(out, "export default function App") {
		t.Fatalf("fs_read body mismatch: %q", out)
	}

	// fs_read on a missing file produces is_error=true with a clean message.
	out, isErr = ai.RunTool(ctx, deps, "fs_read", `{"path":"missing.ts"}`)
	if !isErr {
		t.Fatalf("fs_read on missing path should error")
	}
	if !strings.Contains(out, "not found") {
		t.Fatalf("fs_read missing msg = %q, want 'not found'", out)
	}

	// list_files reflects the one committed file.
	out, isErr = ai.RunTool(ctx, deps, "list_files", `{}`)
	if isErr {
		t.Fatalf("list_files errored: %s", out)
	}
	var files []string
	if err := json.Unmarshal([]byte(out), &files); err != nil {
		t.Fatalf("list_files result not JSON array: %s", out)
	}
	if len(files) != 1 || files[0] != "index.tsx" {
		t.Fatalf("list_files = %v, want [index.tsx]", files)
	}

	// publish_box builds and flips state=published.
	out, isErr = ai.RunTool(ctx, deps, "publish_box", `{}`)
	if isErr {
		t.Fatalf("publish_box errored: %s", out)
	}
	var published struct {
		Version     string `json:"version"`
		ContentHash string `json:"content_hash"`
	}
	if err := json.Unmarshal([]byte(out), &published); err != nil {
		t.Fatalf("publish_box result not JSON: %s", out)
	}
	if !strings.HasPrefix(published.ContentHash, "sha256:") {
		t.Fatalf("publish_box content_hash = %q", published.ContentHash)
	}
	// Box should now be published.
	got, _, err := svc.Get(ctx, "u_t", b.ID)
	if err != nil {
		t.Fatalf("get after publish: %v", err)
	}
	if got.State != pb.PublishStatePublished {
		t.Fatalf("state after publish = %s", got.State)
	}

	// Unknown tool is surfaced as an error, not a panic.
	out, isErr = ai.RunTool(ctx, deps, "nope", `{}`)
	if !isErr {
		t.Fatalf("unknown tool should error")
	}
	if !strings.Contains(out, "unknown tool") {
		t.Fatalf("unknown-tool msg = %q", out)
	}
}

// TestToolsListContract locks the tool surface Anthropic/DeepSeek sees.
// Add a case here when you add a new tool; do not rename without bumping
// everything that matches on tool names (handler, tests, client docs).
func TestToolsListContract(t *testing.T) {
	tools := ai.Tools()
	want := map[string]bool{"fs_read": false, "fs_write": false, "list_files": false, "publish_box": false}
	for _, tl := range tools {
		if tl.Function == nil {
			t.Fatalf("tool %q has nil Function", tl.Type)
		}
		if _, ok := want[tl.Function.Name]; !ok {
			t.Fatalf("unexpected tool %q in Tools() list", tl.Function.Name)
		}
		want[tl.Function.Name] = true
	}
	for name, seen := range want {
		if !seen {
			t.Fatalf("tool %q missing from Tools() list", name)
		}
	}
}

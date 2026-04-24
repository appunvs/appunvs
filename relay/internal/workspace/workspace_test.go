package workspace_test

import (
	"context"
	"errors"
	"testing"

	"github.com/appunvs/appunvs/relay/internal/workspace"
)

func TestCommitAndSnapshot(t *testing.T) {
	ctx := context.Background()
	s, err := workspace.NewStore(workspace.Config{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	repo, err := s.Open("box_abc")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	// First commit — two files, one nested.
	sha1, err := repo.Commit(ctx, []workspace.WriteOp{
		{Path: "index.tsx", Content: []byte("hello\n")},
		{Path: "src/lib/api.ts", Content: []byte("export const x = 1\n")},
	}, "ai: initial", "agent", "agent@appunvs")
	if err != nil {
		t.Fatalf("commit 1: %v", err)
	}
	if sha1 == "" {
		t.Fatalf("sha1 empty")
	}

	snap, err := repo.Snapshot(ctx)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if string(snap["index.tsx"]) != "hello\n" || string(snap["src/lib/api.ts"]) != "export const x = 1\n" {
		t.Fatalf("snapshot mismatch: %#v", snap)
	}

	// Second commit — modify one, delete the other, add a third.
	sha2, err := repo.Commit(ctx, []workspace.WriteOp{
		{Path: "index.tsx", Content: []byte("hi\n")},
		{Path: "src/lib/api.ts", Delete: true},
		{Path: "package.json", Content: []byte(`{"name":"demo"}`)},
	}, "ai: tweak", "", "")
	if err != nil {
		t.Fatalf("commit 2: %v", err)
	}
	if sha2 == "" || sha2 == sha1 {
		t.Fatalf("sha2 same as sha1 or empty")
	}

	// ReadFile for an existing path.
	body, err := repo.ReadFile(ctx, "index.tsx")
	if err != nil {
		t.Fatalf("read index.tsx: %v", err)
	}
	if string(body) != "hi\n" {
		t.Fatalf("read body = %q, want %q", body, "hi\n")
	}

	// ReadFile for a deleted path returns ErrFileNotFound.
	if _, err := repo.ReadFile(ctx, "src/lib/api.ts"); !errors.Is(err, workspace.ErrFileNotFound) {
		t.Fatalf("deleted file read: got %v, want ErrFileNotFound", err)
	}

	// ListFiles reflects the latest state.
	files, err := repo.ListFiles(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	want := []string{"index.tsx", "package.json"}
	if len(files) != len(want) {
		t.Fatalf("list = %v, want %v", files, want)
	}
	for i := range files {
		if files[i] != want[i] {
			t.Fatalf("list[%d] = %s, want %s", i, files[i], want[i])
		}
	}

	// Log returns both our commits; no seed commit in this design.
	log, err := repo.Log(ctx, 10)
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	if len(log) != 2 {
		t.Fatalf("log len = %d, want 2", len(log))
	}
	if log[0].SHA != sha2 {
		t.Fatalf("log[0].SHA = %s, want %s", log[0].SHA, sha2)
	}
}

// TestReopenPreservesHistory confirms the on-disk storage is durable: a
// second Store pointed at the same root sees commits written by the first.
func TestReopenPreservesHistory(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()

	s1, err := workspace.NewStore(workspace.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	r1, err := s1.Open("box_persist")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r1.Commit(ctx, []workspace.WriteOp{
		{Path: "a.txt", Content: []byte("one")},
	}, "first", "", ""); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// Re-open via a fresh Store.
	s2, err := workspace.NewStore(workspace.Config{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := s2.Open("box_persist")
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	body, err := r2.ReadFile(ctx, "a.txt")
	if err != nil {
		t.Fatalf("read after reopen: %v", err)
	}
	if string(body) != "one" {
		t.Fatalf("reopen body = %q, want %q", body, "one")
	}
}

// TestIllegalPath rejects absolute and escaping paths.
func TestIllegalPath(t *testing.T) {
	s, err := workspace.NewStore(workspace.Config{Root: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	r, err := s.Open("box_sanity")
	if err != nil {
		t.Fatal(err)
	}
	bad := []string{"/etc/passwd", "../outside", ""}
	for _, p := range bad {
		if _, err := r.Commit(context.Background(), []workspace.WriteOp{
			{Path: p, Content: []byte("x")},
		}, "bad", "", ""); err == nil {
			t.Fatalf("path %q should have failed", p)
		}
	}
}

package sandbox

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRunner is a stand-in for the real `docker run` invocation.  It
// records the args it was called with and runs the supplied callback —
// the callback can write the bundle / log into the bind-mounted dir to
// simulate what build-bundle.sh would do.
type fakeRunner struct {
	args     []string
	callback func(workDir string) error
	output   []byte
	err      error
}

func (f *fakeRunner) Run(ctx context.Context, args []string) ([]byte, error) {
	f.args = args
	if f.callback != nil {
		// args contains "-v <tempdir>:/work" — extract <tempdir> so the
		// callback can simulate the in-image script's output writes.
		work := extractWorkDir(args)
		if work != "" {
			if err := f.callback(work); err != nil {
				return f.output, err
			}
		}
	}
	return f.output, f.err
}

func extractWorkDir(args []string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "-v" {
			parts := strings.SplitN(args[i+1], ":", 2)
			if len(parts) == 2 && parts[1] == "/work" {
				return parts[0]
			}
		}
	}
	return ""
}

// TestDockerBuilderHappyPath: source files reach the bind dir, docker
// args are well-formed, the simulated bundle output round-trips back
// into Result.
func TestDockerBuilderHappyPath(t *testing.T) {
	runner := &fakeRunner{
		callback: func(work string) error {
			// Verify our source landed where build-bundle.sh would
			// look for it: /work/src/index.tsx.
			seen, err := os.ReadFile(filepath.Join(work, "src", "index.tsx"))
			if err != nil {
				return err
			}
			if string(seen) != "export const x = 1\n" {
				return errors.New("source content mismatch")
			}
			// Pretend we ran metro: write the bundle + log.
			if err := os.WriteFile(filepath.Join(work, "index.bundle"),
				[]byte("// fake bundle\n"), 0o644); err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(work, "build.log"),
				[]byte("metro ok\n"), 0o644); err != nil {
				return err
			}
			return nil
		},
	}

	b := &DockerBuilder{Image: "appunvs/sandbox:test", runner: runner}
	res, err := b.Build(context.Background(), Source{
		BoxID:      "box_42",
		EntryPoint: "index.tsx",
		Files:      map[string][]byte{"index.tsx": []byte("export const x = 1\n")},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if string(res.Bytes) != "// fake bundle\n" {
		t.Errorf("bundle = %q, want fake-bundle marker", string(res.Bytes))
	}
	if res.Log != "metro ok\n" {
		t.Errorf("log = %q, want metro-ok marker", res.Log)
	}

	// Args sanity: image is positional last, --rm + bind mount + image present.
	gotArgs := strings.Join(runner.args, " ")
	if !strings.Contains(gotArgs, "appunvs/sandbox:test") {
		t.Errorf("args missing image: %q", gotArgs)
	}
	if !strings.Contains(gotArgs, "--rm") {
		t.Errorf("args missing --rm: %q", gotArgs)
	}
	if !strings.Contains(gotArgs, ":/work") {
		t.Errorf("args missing :/work bind mount: %q", gotArgs)
	}
	if !strings.Contains(gotArgs, "no-new-privileges") {
		t.Errorf("args missing no-new-privileges: %q", gotArgs)
	}
}

// TestDockerBuilderRunFailureKeepsLog: when docker exits non-zero, we
// should still surface whatever build.log got written so the caller
// can see metro's error output.  Result.Log non-empty even on error.
func TestDockerBuilderRunFailureKeepsLog(t *testing.T) {
	runner := &fakeRunner{
		callback: func(work string) error {
			_ = os.WriteFile(filepath.Join(work, "build.log"),
				[]byte("metro error: blah\n"), 0o644)
			return nil
		},
		err: errors.New("exit status 1"),
	}
	b := &DockerBuilder{Image: "appunvs/sandbox:test", runner: runner}
	res, err := b.Build(context.Background(), Source{
		BoxID: "box_x",
		Files: map[string][]byte{"index.tsx": []byte("syntax error")},
	})
	if err == nil {
		t.Fatal("expected build error")
	}
	if !strings.Contains(res.Log, "metro error: blah") {
		t.Errorf("log = %q, want metro error captured", res.Log)
	}
}

// TestDockerBuilderRejectsParentTraversal: AI source files with ".."
// components must be rejected before any filesystem writes.
func TestDockerBuilderRejectsParentTraversal(t *testing.T) {
	runner := &fakeRunner{}
	b := &DockerBuilder{Image: "x", runner: runner}
	_, err := b.Build(context.Background(), Source{
		BoxID: "box",
		Files: map[string][]byte{"../etc/passwd": []byte("root:x:0:0\n")},
	})
	if err == nil {
		t.Fatal("expected rejection of parent-traversal path")
	}
	if runner.args != nil {
		t.Fatalf("docker should NOT have been invoked; args=%v", runner.args)
	}
}

// TestDockerBuilderEntryShim: when caller passes an EntryPoint other
// than the default, an entry.tsx shim lands at /work/entry.tsx so
// build-bundle.sh picks it up.
func TestDockerBuilderEntryShim(t *testing.T) {
	runner := &fakeRunner{
		callback: func(work string) error {
			data, err := os.ReadFile(filepath.Join(work, "entry.tsx"))
			if err != nil {
				return err
			}
			if !strings.Contains(string(data), "./src/main") {
				return errors.New("shim missing ./src/main re-export: " + string(data))
			}
			_ = os.WriteFile(filepath.Join(work, "index.bundle"), []byte("ok"), 0o644)
			return nil
		},
	}
	b := &DockerBuilder{Image: "x", runner: runner}
	_, err := b.Build(context.Background(), Source{
		BoxID:      "box",
		EntryPoint: "main.tsx",
		Files:      map[string][]byte{"main.tsx": []byte("export const x = 1\n")},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
}

// TestDockerBuilderEmptyBundleFails: docker exited 0 but no bundle
// file written.  The builder should surface a clear error rather than
// returning a successful but empty Result.
func TestDockerBuilderEmptyBundleFails(t *testing.T) {
	runner := &fakeRunner{
		callback: func(work string) error {
			_ = os.WriteFile(filepath.Join(work, "build.log"), []byte("???"), 0o644)
			return nil
		},
	}
	b := &DockerBuilder{Image: "x", runner: runner}
	_, err := b.Build(context.Background(), Source{
		BoxID: "box",
		Files: map[string][]byte{"index.tsx": []byte("ok")},
	})
	if err == nil {
		t.Fatal("expected error when bundle missing")
	}
}

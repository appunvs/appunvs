// Package sandbox is the build orchestrator for AI-generated RN projects.
//
// The Builder interface deliberately hides whether a build runs in-process,
// in a local container, or in a managed Firecracker microVM.  The
// LocalStub implementation shipped today just zips the source tree into a
// fake bundle so the rest of the pipeline (artifact upload + Box.SetCurrentVersion
// + connector fanout) can be exercised end-to-end without a Metro install.
//
// Production targets — see docs/architecture.md:
//   - Self-hosted: Firecracker microVM pool managed by relay
//   - Managed:     Modal / E2B / Vercel Sandbox adapter
package sandbox

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

// Source is the input to a build.  Files is a flat map from POSIX path
// (relative to project root) to file bytes.  EntryPoint is the bundle's
// main module ("index.tsx" by default).
type Source struct {
	BoxID      string
	Version    string
	EntryPoint string
	Files      map[string][]byte
}

// Result is what a Builder returns on success.  Bytes is the JS bundle
// payload the runner will load; the caller pushes it through artifact.Store
// to obtain a content-addressed URL.
type Result struct {
	Bytes []byte
	Log   string
}

// Builder produces a runnable RN bundle from a Source.
type Builder interface {
	Build(ctx context.Context, src Source) (Result, error)
}

// LocalStub is a placeholder that concatenates source files into a single
// "bundle" with a clearly-marked header.  It's enough for the connector
// path — pair, fetch, version-update — to be exercised before the real
// Metro pipeline is wired in.
type LocalStub struct{}

// NewLocalStub returns a stub Builder.
func NewLocalStub() *LocalStub { return &LocalStub{} }

// Build writes a concatenated text artifact whose contents document that
// this is a stub.  Real runners will refuse to load it — that's intentional;
// the stub exists so backend tests can flow without standing up Metro.
func (s *LocalStub) Build(ctx context.Context, src Source) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	var buf bytes.Buffer
	header := fmt.Sprintf(
		"// appunvs sandbox.LocalStub bundle\n// box=%s version=%s entry=%s built=%s\n// THIS IS A PLACEHOLDER. Wire a real Metro build before shipping.\n\n",
		src.BoxID, src.Version, src.EntryPoint, time.Now().UTC().Format(time.RFC3339))
	if _, err := io.WriteString(&buf, header); err != nil {
		return Result{}, err
	}
	// Deterministic ordering so equivalent inputs produce identical bytes
	// (and thus the same content hash).
	keys := sortedKeys(src.Files)
	for _, k := range keys {
		fmt.Fprintf(&buf, "// ---- %s ----\n", k)
		buf.Write(src.Files[k])
		if !bytes.HasSuffix(src.Files[k], []byte("\n")) {
			buf.WriteByte('\n')
		}
		buf.WriteByte('\n')
	}
	return Result{
		Bytes: buf.Bytes(),
		Log:   fmt.Sprintf("LocalStub: bundled %d files (%d bytes)", len(src.Files), buf.Len()),
	}, nil
}

func sortedKeys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// tiny insertion sort — files maps are O(few hundred) at most
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && strings.Compare(out[j-1], out[j]) > 0; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

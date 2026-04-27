// docker.go — DockerBuilder backs the sandbox.Builder interface with a
// `docker run` against the appunvs/sandbox image (built from
// runtime/sandbox/Dockerfile).
//
// The sandbox image ships preinstalled with runtime/package.json's
// dependencies (metro, react-native, all Tier 1 native modules), so
// AI-edited TypeScript/TSX trees can be bundled without touching the
// host's filesystem.  See runtime/sandbox/Dockerfile and
// runtime/sandbox/build-bundle.sh for the in-image entry point.
//
// Lifecycle of one Build call:
//
//   1. Create a tempdir and write Source.Files into <tempdir>/src/.
//   2. If Source.EntryPoint is non-default, also write a small
//      <tempdir>/entry.tsx that re-exports it (build-bundle.sh looks
//      at /work/entry.tsx first, then /work/src/index.tsx).
//   3. `docker run --rm -v <tempdir>:/work appunvs/sandbox:<tag>`
//   4. Read <tempdir>/index.bundle as Result.Bytes.
//   5. Read <tempdir>/build.log as Result.Log.
//   6. Cleanup tempdir.
//
// Future backends (Modal, Vercel Sandbox, Aliyun ECI, Firecracker
// pool) implement the same Builder interface — call sites in
// box.Service stay unchanged.
package sandbox

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// DockerBuilder runs metro inside the appunvs/sandbox docker image.
// Construct via NewDockerBuilder (validates docker is on PATH).
type DockerBuilder struct {
	// Image is the sandbox image reference, e.g. "appunvs/sandbox:latest".
	Image string
	// runner is the indirection that lets tests stub out the actual
	// `docker run` invocation.  Production wires it to an exec.Command
	// runner; tests pass a fake.
	runner dockerRunner
}

// dockerRunner abstracts the one host-side primitive the builder
// needs: "go run this command, capture stdout+stderr, give me the
// exit error if any".  Real impl uses os/exec; tests mock.
type dockerRunner interface {
	Run(ctx context.Context, args []string) (combinedOutput []byte, err error)
}

// NewDockerBuilder verifies docker is callable and returns a builder
// configured to use the given image tag.  Use this from production
// code paths; tests call newDockerBuilderForTest.
func NewDockerBuilder(image string) (*DockerBuilder, error) {
	if image == "" {
		return nil, errors.New("sandbox.DockerBuilder: image is required")
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return nil, fmt.Errorf("sandbox.DockerBuilder: docker not on PATH: %w", err)
	}
	return &DockerBuilder{Image: image, runner: execRunner{}}, nil
}

// Build implements Builder.  See package-level comment for the flow.
func (d *DockerBuilder) Build(ctx context.Context, src Source) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	tmp, err := os.MkdirTemp("", "appunvs-sandbox-*")
	if err != nil {
		return Result{}, fmt.Errorf("sandbox: tempdir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	// Materialize Source.Files under <tmp>/src/.  build-bundle.sh
	// looks at /work/src/index.tsx by default.
	srcDir := filepath.Join(tmp, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return Result{}, fmt.Errorf("sandbox: mkdir src: %w", err)
	}
	for relPath, content := range src.Files {
		// Reject paths trying to escape via "../" — the sandbox
		// image is bind-mounted, so a malicious AI bundle could
		// otherwise scribble outside /work.
		if !isCleanRelPath(relPath) {
			return Result{}, fmt.Errorf("sandbox: invalid path %q in source files", relPath)
		}
		full := filepath.Join(srcDir, relPath)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return Result{}, fmt.Errorf("sandbox: mkdir %s: %w", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, content, 0o644); err != nil {
			return Result{}, fmt.Errorf("sandbox: write %s: %w", relPath, err)
		}
	}

	// EntryPoint override: build-bundle.sh prefers /work/entry.tsx if
	// it exists.  When the caller picked a non-default entry, write a
	// re-export shim there so the sandbox doesn't need to know our
	// path conventions.
	if ep := src.EntryPoint; ep != "" && ep != "index.tsx" && ep != "src/index.tsx" {
		if !isCleanRelPath(ep) {
			return Result{}, fmt.Errorf("sandbox: invalid entry %q", ep)
		}
		shim := []byte(fmt.Sprintf("export * from './src/%s';\n", trimTSExt(ep)))
		if err := os.WriteFile(filepath.Join(tmp, "entry.tsx"), shim, 0o644); err != nil {
			return Result{}, fmt.Errorf("sandbox: write entry shim: %w", err)
		}
	}

	args := []string{
		"run", "--rm",
		// Drop privileges and capabilities.  AI source ran through
		// metro shouldn't need anything beyond filesystem r/w on
		// /work; --read-only is too aggressive (npm/metro write
		// scratch files inside the image), but no-new-privileges
		// blocks privilege escalation, and dropping all caps closes
		// the obvious holes.
		"--security-opt", "no-new-privileges",
		"--cap-drop", "ALL",
		// Keep the workdir bind read-write because build-bundle.sh
		// writes the bundle / build log there.
		"-v", tmp + ":/work",
		d.Image,
	}
	output, runErr := d.runner.Run(ctx, args)

	// Read whatever build.log got written, even on docker error.
	logBytes, _ := os.ReadFile(filepath.Join(tmp, "build.log"))
	combinedLog := string(logBytes)
	if len(combinedLog) == 0 {
		// Docker itself failed before the script could run, or the
		// script failed before writing build.log.  Fall back to the
		// docker output as a poor-man's diagnostic.
		combinedLog = string(output)
	}

	if runErr != nil {
		return Result{Log: combinedLog}, fmt.Errorf("sandbox: docker run: %w", runErr)
	}

	bundle, err := os.ReadFile(filepath.Join(tmp, "index.bundle"))
	if err != nil {
		return Result{Log: combinedLog}, fmt.Errorf("sandbox: read bundle: %w", err)
	}
	if len(bundle) == 0 {
		return Result{Log: combinedLog}, errors.New("sandbox: empty bundle")
	}
	return Result{Bytes: bundle, Log: combinedLog}, nil
}

// execRunner is the production runner — `docker <args>` via os/exec.
type execRunner struct{}

func (execRunner) Run(ctx context.Context, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	return buf.Bytes(), err
}

// isCleanRelPath rejects paths containing ".." components or absolute
// paths.  Used to refuse AI-source files that would write outside the
// sandbox tempdir's src/ tree.
func isCleanRelPath(p string) bool {
	if p == "" {
		return false
	}
	if filepath.IsAbs(p) {
		return false
	}
	// filepath.Clean collapses "./foo/../bar" but doesn't reject
	// "../bar" — check explicitly.
	cleaned := filepath.Clean(p)
	if cleaned == ".." || cleaned == "." {
		return false
	}
	if hasParentTraversal(cleaned) {
		return false
	}
	return true
}

func hasParentTraversal(p string) bool {
	// Use forward slash semantics — filepath.Clean uses OS separator
	// but Source.Files paths come from JSON / wire so they're
	// forward-slash by convention.
	for {
		dir, last := filepath.Split(p)
		if last == ".." {
			return true
		}
		if dir == "" || dir == p {
			return false
		}
		p = filepath.Clean(filepath.Dir(p))
	}
}

// trimTSExt strips a trailing .ts / .tsx / .js / .jsx so the re-export
// shim's `from './src/foo'` resolves through metro's extension search.
func trimTSExt(p string) string {
	for _, ext := range []string{".tsx", ".ts", ".jsx", ".js"} {
		if hasSuffix(p, ext) {
			return p[:len(p)-len(ext)]
		}
	}
	return p
}

func hasSuffix(s, suf string) bool {
	if len(suf) > len(s) {
		return false
	}
	return s[len(s)-len(suf):] == suf
}

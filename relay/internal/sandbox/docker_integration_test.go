//go:build docker_integration

// Real docker integration test for DockerBuilder.  Skipped by default
// (build tag `docker_integration` not set).  Enabled by:
//
//   APPUNVS_SANDBOX_IMAGE=appunvs/sandbox:test \
//   go test -tags docker_integration ./internal/sandbox/... -timeout 5m
//
// Requires:
//   - docker on PATH
//   - the sandbox image already built (see runtime/packaging/build-sandbox.sh)
//   - APPUNVS_SANDBOX_IMAGE env var pointing at the local tag
//
// CI runs this as a dedicated job in .github/workflows/ci.yml.

package sandbox

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDockerBuilderRealImage(t *testing.T) {
	image := os.Getenv("APPUNVS_SANDBOX_IMAGE")
	if image == "" {
		t.Skip("APPUNVS_SANDBOX_IMAGE not set; skipping docker integration")
	}

	b, err := NewDockerBuilder(image)
	if err != nil {
		t.Fatalf("NewDockerBuilder: %v", err)
	}

	// Tiny AI source that exercises the metro resolver against a Tier 1
	// module (`react-native`) — proves the image's preinstalled deps
	// can satisfy a real bundle, and that the sandbox metro config's
	// allowlist passes a legitimate import.
	src := []byte(`
import { Text } from 'react-native';
import { AppRegistry } from 'react-native';

function App() {
  return null as unknown as ReturnType<typeof Text>;
}

AppRegistry.registerComponent('TestApp', () => App);
`)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	res, err := b.Build(ctx, Source{
		BoxID: "box_int",
		Files: map[string][]byte{"index.tsx": src},
	})
	if err != nil {
		t.Fatalf("Build: %v\n=== build.log ===\n%s", err, res.Log)
	}
	if len(res.Bytes) == 0 {
		t.Fatal("bundle is empty")
	}
	// Sanity: the bundle must contain SOMETHING from react-native — the
	// AppRegistry.registerComponent call goes through the RN bridge so
	// the marker name shows up in the minified output.
	if !strings.Contains(string(res.Bytes), "TestApp") {
		t.Errorf("bundle missing TestApp marker; head: %q", head(res.Bytes, 256))
	}
}

// TestDockerBuilderRealImageRejectsForbiddenImport: the sandbox metro
// config's allowlist is the SECURITY boundary — verify a non-allowlisted
// import fails the build rather than silently bundling.
func TestDockerBuilderRealImageRejectsForbiddenImport(t *testing.T) {
	image := os.Getenv("APPUNVS_SANDBOX_IMAGE")
	if image == "" {
		t.Skip("APPUNVS_SANDBOX_IMAGE not set; skipping docker integration")
	}

	b, err := NewDockerBuilder(image)
	if err != nil {
		t.Fatalf("NewDockerBuilder: %v", err)
	}

	// `lodash` is intentionally NOT in the allowlist (see
	// runtime/sandbox/metro.config.js ALLOWED_MODULES).  The bundle
	// must fail with a metro resolver error.
	src := []byte(`
import _ from 'lodash';
export default _;
`)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	res, err := b.Build(ctx, Source{
		BoxID: "box_block",
		Files: map[string][]byte{"index.tsx": src},
	})
	if err == nil {
		t.Fatalf("expected allowlist rejection; instead got bundle of %d bytes", len(res.Bytes))
	}
	if !strings.Contains(res.Log, "allowlist") && !strings.Contains(res.Log, "lodash") {
		t.Errorf("error log doesn't mention allowlist/lodash:\n%s", res.Log)
	}
}

func head(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "...(truncated)"
}

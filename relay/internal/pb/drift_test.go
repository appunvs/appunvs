package pb_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/appunvs/appunvs/relay/internal/pb"
)

// TestWireDriftAgainstGolden loads shared/proto/testdata/messages.json and
// asserts every language parses and re-serializes each case identically.  The
// same fixture is used by the TS, Rust, and Dart drift tests; any language
// diverging from canonical protojson fails its own test.
func TestWireDriftAgainstGolden(t *testing.T) {
	path := filepath.Join("..", "..", "..", "shared", "proto", "testdata", "messages.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	var cases []struct {
		Name    string                 `json:"name"`
		Message map[string]interface{} `json:"message"`
	}
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("parse golden: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("golden fixture is empty")
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			// Re-serialize the golden "message" so we have a canonical byte
			// sequence to diff against.  map-based JSON encoding with sorted
			// keys (Go's default) gives a stable ordering.
			golden, err := canonical(c.Message)
			if err != nil {
				t.Fatalf("canonical(golden): %v", err)
			}

			// Parse into the relay's Message type and back.
			var m pb.Message
			if err := json.Unmarshal(golden, &m); err != nil {
				t.Fatalf("pb.Message unmarshal: %v", err)
			}
			encoded, err := json.Marshal(m)
			if err != nil {
				t.Fatalf("pb.Message marshal: %v", err)
			}
			produced, err := canonicalBytes(encoded)
			if err != nil {
				t.Fatalf("canonical(produced): %v", err)
			}

			var a, b map[string]interface{}
			_ = json.Unmarshal(golden, &a)
			_ = json.Unmarshal(produced, &b)
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("drift:\n  want %s\n  got  %s", golden, produced)
			}
		})
	}
}

func canonical(m map[string]interface{}) ([]byte, error) {
	return json.Marshal(m)
}
func canonicalBytes(b []byte) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

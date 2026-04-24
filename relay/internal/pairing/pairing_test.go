package pairing

import (
	"strings"
	"testing"
)

// TestNewCodeShapeAndAlphabet asserts every emitted code uses the documented
// Crockford-without-0/1/I/O alphabet.  This is a regression guard: relaxing
// the alphabet without updating clients (which paint the code into a QR /
// "type me" UI) is a wire break.
func TestNewCodeShapeAndAlphabet(t *testing.T) {
	for i := 0; i < 200; i++ {
		c, err := newCode(8)
		if err != nil {
			t.Fatalf("newCode: %v", err)
		}
		if len(c) != 8 {
			t.Fatalf("len = %d, want 8", len(c))
		}
		for _, r := range c {
			if !strings.ContainsRune(alphabet, r) {
				t.Fatalf("char %q not in alphabet", r)
			}
		}
	}
}

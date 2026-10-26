package stubs

import (
	"io"
	"strings"
	"testing"
)

// Tests must inspect the distributed resources, not generate replacements first.
func TestBundledExtractors(t *testing.T) {
	for _, target := range Targets {
		t.Run(target, func(t *testing.T) {
			parts := strings.Split(target, "/")
			r, err := Open(parts[0], parts[1])
			if err != nil {
				t.Fatal(err)
			}
			defer r.Close()
			// Reading to EOF checks gzip integrity as well as resource presence.
			n, err := io.Copy(io.Discard, r)
			if err != nil {
				t.Fatal(err)
			}
			if n == 0 {
				t.Fatal("empty extractor")
			}
		})
	}
}

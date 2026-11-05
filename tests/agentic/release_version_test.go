//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// The actual executable must advertise the release consistently without any
// API call. Only the external peer is a fixture; parsing and output are real.
func TestReleaseVersionCLI(t *testing.T) {
	var calls atomic.Int32
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "version must be offline", 500)
	}))
	t.Cleanup(peer.Close)
	t.Run("legacy", func(t *testing.T) {
		out, stderr, err := runCLI(t, peer.URL, false, "version")
		// Preserve Cobra's historical legacy stderr stream, even for a success.
		if err != nil || len(out) != 0 || string(stderr) != "gate-cli version v0.9.2\n" {
			t.Fatalf("legacy version: %v stdout=%s stderr=%s", err, out, stderr)
		}
	})
	t.Run("root version flag", func(t *testing.T) {
		out, stderr, err := runCLI(t, peer.URL, false, "--version")
		if err != nil || len(stderr) != 0 || string(out) != "gate-cli version v0.9.2\n" {
			t.Fatalf("root version: %v stdout=%s stderr=%s", err, out, stderr)
		}
	})
	for _, format := range []string{"json", "pretty"} {
		t.Run(format, func(t *testing.T) {
			out, stderr, err := runCLI(t, peer.URL, false, "version", "--details", "--format", format)
			var details struct {
				Version string `json:"version"`
				Schema  int    `json:"schema_version"`
			}
			if err != nil || len(stderr) != 0 || json.Unmarshal(out, &details) != nil || details.Version != "v0.9.2" || details.Schema != 1 {
				t.Fatalf("details: %v stdout=%s stderr=%s", err, out, stderr)
			}
		})
	}
	t.Run("unsupported table", func(t *testing.T) {
		out, stderr, err := runCLI(t, peer.URL, false, "version", "--details", "--format", "table")
		if err == nil || len(out) != 0 || !bytes.Contains(stderr, []byte("UNSUPPORTED_FORMAT")) {
			t.Fatalf("invalid format: %v stdout=%s stderr=%s", err, out, stderr)
		}
	})
	if calls.Load() != 0 {
		t.Fatalf("version made %d API calls", calls.Load())
	}
}

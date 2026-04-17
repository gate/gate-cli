package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

func TestDefaultMaxOutputBytes(t *testing.T) {
	t.Run("empty env uses zero", func(t *testing.T) {
		t.Setenv("GATE_MAX_OUTPUT_BYTES", "")
		if got := defaultMaxOutputBytes(); got != 0 {
			t.Fatalf("expected 0, got %d", got)
		}
	})

	t.Run("valid env parsed", func(t *testing.T) {
		t.Setenv("GATE_MAX_OUTPUT_BYTES", "2048")
		if got := defaultMaxOutputBytes(); got != 2048 {
			t.Fatalf("expected 2048, got %d", got)
		}
	})

	t.Run("invalid env falls back to zero", func(t *testing.T) {
		t.Setenv("GATE_MAX_OUTPUT_BYTES", "bad")
		if got := defaultMaxOutputBytes(); got != 0 {
			t.Fatalf("expected 0, got %d", got)
		}
	})
}

func TestEmitFormatCompatNotice(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	defer func() { os.Stderr = oldStderr }()

	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", "pretty", "")
	emitFormatCompatNotice(root)
	_ = w.Close()

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if buf.Len() == 0 {
		t.Fatal("expected format compatibility notice")
	}
}

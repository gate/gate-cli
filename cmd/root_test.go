package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
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

func TestNormalizeMaxOutputBytesFlagNegativeClampsToZero(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().Int64("max-output-bytes", 100, "")
	require.NoError(t, root.PersistentFlags().Set("max-output-bytes", "-5"))
	normalizeMaxOutputBytesFlag(root)
	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "negative") {
		t.Fatalf("expected stderr warning about negative max-output-bytes, got %q", buf.String())
	}
	v, err := root.PersistentFlags().GetInt64("max-output-bytes")
	require.NoError(t, err)
	if v != 0 {
		t.Fatalf("expected 0 after clamp, got %d", v)
	}
}

func TestEmitFormatCompatNotice(t *testing.T) {
	t.Run("prints when format not explicitly set and force env set (non-TTY safe)", func(t *testing.T) {
		t.Setenv("GATE_CLI_FORMAT_NOTICE_FORCE", "1")
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
	})

	t.Run("silent when format explicitly set", func(t *testing.T) {
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w
		defer func() { os.Stderr = oldStderr }()

		root := &cobra.Command{Use: "gate-cli"}
		root.PersistentFlags().String("format", "pretty", "")
		require.NoError(t, root.PersistentFlags().Set("format", "json"))
		emitFormatCompatNotice(root)
		_ = w.Close()

		var buf bytes.Buffer
		_, _ = buf.ReadFrom(r)
		if buf.Len() != 0 {
			t.Fatalf("expected no notice when format set, got %q", buf.String())
		}
	})
}

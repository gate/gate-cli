package cmd

import (
	"bytes"
	"errors"
	"github.com/gate/gate-cli/internal/output"
	"io"
	"testing"
)

type failedDetailsWriter struct{}

func (failedDetailsWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

// The new branch respects format/cap and writer errors; the legacy Run is not
// routed through this renderer and retains its original stream and semantics.
func TestCEXCompatVersionRenderer(t *testing.T) {
	for _, format := range []output.Format{output.FormatJSON, output.FormatPretty} {
		var out bytes.Buffer
		if err := printVersionDetails(output.New(&out, format)); err != nil || !bytes.Contains(out.Bytes(), []byte("schema_version")) {
			t.Fatalf("render %v %s", err, out.Bytes())
		}
	}
	if err := printVersionDetails(output.New(io.Discard, output.FormatTable)); err == nil {
		t.Fatal("table accepted")
	}
	if err := printVersionDetails(output.New(failedDetailsWriter{}, output.FormatJSON)); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal("writer failure lost", err)
	}
}

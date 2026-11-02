package configcmd

import (
	"bytes"
	"errors"
	"github.com/gate/gate-cli/internal/config"
	"github.com/gate/gate-cli/internal/output"
	"io"
	"testing"
)

// Report output must precede the failure summary, and output failure takes
// priority so consumers cannot mistake a missing report for a complete check.
func TestCEXCompatCheckRenderer(t *testing.T) {
	for _, status := range []string{"pass", "warn", "fail"} {
		var out bytes.Buffer
		err := PrintFileCheck(output.New(&out, output.FormatJSON), config.FileCheck{Status: status, Scope: "file"})
		if (err != nil) != (status != "pass") || out.Len() == 0 {
			t.Fatalf("status=%s err=%v", status, err)
		}
	}
	if err := PrintFileCheck(output.New(failedWriter{}, output.FormatJSON), config.FileCheck{Status: "fail"}); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatal("writer error lost", err)
	}
}

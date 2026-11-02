//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gate/gate-cli/internal/output"
)

// TestAPIErrorFormats 真实CLI到HTTP边界验证：失败退出、stdout为空，三种格式均保留诊断。
func TestAPIErrorFormats(t *testing.T) {
	for _, format := range []string{"json", "pretty", "table"} {
		for _, status := range []int{401, 403, 503} {
			t.Run(fmt.Sprintf("%s/%d", format, status), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Gate-Trace-Id", "synthetic-trace")
					w.WriteHeader(status)
					if _, err := w.Write([]byte(`{"label":"SYNTHETIC_FAILURE","message":"synthetic API failure"}`)); err != nil {
						t.Error(err)
					}
				}))
				defer server.Close()
				stdout, stderr, err := runCLI(t, server.URL, true, "--format", format, "cex", "stock", "account", "assets")
				if err == nil || len(bytes.TrimSpace(stdout)) != 0 {
					t.Fatalf("error must fail without stdout: err=%v stdout=%s", err, stdout)
				}
				for _, expected := range []string{fmt.Sprint(status), "SYNTHETIC_FAILURE", "synthetic-trace"} {
					if !strings.Contains(string(stderr), expected) {
						t.Errorf("missing %q in stderr %q", expected, stderr)
					}
				}
			})
		}
	}
}

// TestLostWriteResponseFormats 结果未知必须展示核实建议，且包括table在内任何格式都不得补发。
func TestLostWriteResponseFormats(t *testing.T) {
	for _, format := range []string{"json", "pretty", "table"} {
		t.Run(format, func(t *testing.T) {
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				conn, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Error(err)
					return
				}
				if err := conn.Close(); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			stdout, stderr, err := runCLI(t, server.URL, true, "--format", format, "cex", "stock", "account", "transfer", "--change", "1", "--type", "deposit", "--ref-id", "synthetic-ref")
			if err == nil || len(bytes.TrimSpace(stdout)) != 0 || !strings.Contains(string(stderr), "outcome unknown") || !strings.Contains(string(stderr), "reconcile") {
				t.Fatalf("lost recovery advice: err=%v stdout=%s stderr=%s", err, stdout, stderr)
			}
			if calls.Load() != 1 {
				t.Fatalf("write attempts=%d", calls.Load())
			}
			if format == "json" {
				var envelope struct{ Error output.GateError }
				if err := json.Unmarshal(stderr, &envelope); err != nil || envelope.Error.Retryable {
					t.Fatalf("lost write must not become retryable: decode=%v stderr=%s", err, stderr)
				}
			}
		})
	}
}

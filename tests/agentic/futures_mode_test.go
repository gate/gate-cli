//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// TestFuturesModeFailureCLI 真实命令必须向Agent报告模式查询错误，禁止继续查询仓位或下单。
func TestFuturesModeFailureCLI(t *testing.T) {
	commands := [][]string{
		{"order", "close", "--contract", "SYNTH_USDT", "--side", "short"},
		{"order", "add", "--contract", "SYNTH_USDT", "--size", "1"},
		{"order", "remove", "--contract", "SYNTH_USDT", "--size", "1"},
		{"position", "update-dual-margin", "--contract", "SYNTH_USDT", "--change", "1"},
	}
	for _, status := range []int{401, 403, 503} {
		for _, args := range commands {
			t.Run(fmt.Sprintf("%d/%s", status, strings.Join(args[:2], "/")), func(t *testing.T) {
				var downstream atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if r.URL.Path == "/api/v4/futures/usdt/accounts" {
						w.Header().Set("X-Gate-Trace-Id", "synthetic-mode-trace")
						w.WriteHeader(status)
						if _, err := w.Write([]byte(`{"label":"MODE_UNAVAILABLE","message":"cannot determine position mode"}`)); err != nil {
							t.Error(err)
						}
						return
					}
					downstream.Add(1)
					if _, err := w.Write([]byte(`{"id":123,"size":"0"}`)); err != nil {
						t.Error(err)
					}
				}))
				defer server.Close()
				stdout, stderr, err := runCLI(t, server.URL, true, append([]string{"--format", "json", "cex", "futures"}, args...)...)
				if err == nil || len(bytes.TrimSpace(stdout)) != 0 || downstream.Load() != 0 {
					t.Fatalf("continued after mode failure: err=%v downstream=%d stdout=%s stderr=%s", err, downstream.Load(), stdout, stderr)
				}
				for _, value := range []string{"MODE_UNAVAILABLE", "synthetic-mode-trace", "/accounts"} {
					if !strings.Contains(string(stderr), value) {
						t.Errorf("missing %q in %s", value, stderr)
					}
				}
			})
		}
	}
}

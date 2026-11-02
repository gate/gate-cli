//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
)

// TestExistingCEXAuthorizationFailure 授权接入存量命令后，401/403/503不能只打印错误再成功退出。
func TestExistingCEXAuthorizationFailure(t *testing.T) {
	cases := [][]string{{"spot", "account", "list"}, {"futures", "account", "get"}, {"wallet", "balance", "total"}, {"earn", "uni", "lends"}, {"unified", "account", "get"}, {"wallet", "deposit", "saved", "--currency", "USDT"}, {"cross-ex", "position", "adl-rank"}, {"futures", "price-trigger", "update", "--id", "pending-id", "--body-order-id", "9007199254740993", "--price", "10"}}
	for _, args := range cases {
		for _, status := range []int{401, 403, 503} {
			t.Run(strings.Join(args, " ")+http.StatusText(status), func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(status)
					if _, err := w.Write([]byte(`{"label":"SYNTHETIC_REJECTED","message":"synthetic rejection"}`)); err != nil {
						t.Error(err)
					}
				}))
				defer server.Close()
				argv := append([]string{"--format", "json", "cex"}, args...)
				stdout, stderr, err := runCLI(t, server.URL, true, argv...)
				if err == nil {
					t.Fatalf("HTTP%d exited0: %s", status, stderr)
				}
				if args[0] == "cross-ex" {
					if calls.Load() != 0 || !strings.Contains(string(stderr), "unknown command") {
						t.Fatalf("retired command reached network: %s", stderr)
					}
					return
				}
				var processErr *exec.ExitError
				if status != 503 && (!errors.As(err, &processErr) || processErr.ExitCode() != 4) {
					t.Errorf("auth exit=%v want4", err)
				}
				if len(bytes.TrimSpace(stdout)) != 0 || calls.Load() != 1 {
					t.Fatalf("failure stdout or retry: calls=%d", calls.Load())
				}
				if strings.Contains(string(stderr), "unknown command") {
					t.Fatal("missing command is not a protocol error test")
				}
			})
		}
	}
}

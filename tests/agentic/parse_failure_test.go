//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

// The shared flag hook covers Common/CEX as well as Intel. Check complete
// executable failures, not merely that one expected substring exists.
func TestParseFailureSingleDiagnostic(t *testing.T) {
	endpoint, calls := intelPeer(t, "info_coin_get_coin_info", func(http.ResponseWriter, *http.Request, intelRPCRequest) {
		t.Error("parse failure reached backend")
	})
	home := intelContractHome(t, endpoint, "5s")
	for _, path := range []string{"config check", "version", "cex spot market ticker", "info coin get-coin-info", "news feed search-news"} {
		for _, format := range []string{"json", "pretty", "table"} {
			t.Run(path+"/"+format, func(t *testing.T) {
				for _, order := range []string{"format-first", "format-last"} {
					args := append([]string{"--format", format}, strings.Fields(path)...)
					args = append(args, "--test-single-error")
					if order == "format-last" {
						args = append(strings.Fields(path), "--test-single-error", "--format", format)
					}
					out, stderr, err := intelCLI(t, home, nil, args...)
					// Agent metadata repeats the original message by design. Count
					// actual error frames, not occurrences inside diagnostic JSON.
					frame := `{"error":`
					if format == "pretty" {
						frame = "Error [400 "
					}
					if format == "table" {
						frame = "Error: unknown flag: --test-single-error"
					}
					if !intelFailedExit(err) || len(out) != 0 || calls.RPC.Load() != 0 || bytes.Count(stderr, []byte(frame)) != 1 || !bytes.Contains(stderr, []byte("unknown flag: --test-single-error")) {
						t.Fatalf("parse diagnostic lost/duplicated (%s): %v RPC=%d stdout=%s stderr=%s", order, err, calls.RPC.Load(), out, stderr)
					}
					if format == "json" {
						intelParseError(t, stderr, path, "test-single-error")
					}
				}
			})
		}
	}
}

// Tool results deliberately reject table format; tool indexes still support
// it. This distinction must fail before the first remote RPC.
func TestIntelUnsupportedTable(t *testing.T) {
	for _, tc := range []struct{ path, tool, domain string }{
		{"info coin get-coin-info", "info_coin_get_coin_info", "info"},
		{"news feed search-news", "news_feed_search_news", "news"},
	} {
		t.Run(tc.domain, func(t *testing.T) {
			endpoint, calls := intelPeer(t, tc.tool, func(http.ResponseWriter, *http.Request, intelRPCRequest) {
				t.Error("unsupported format reached tool")
			})
			home := intelContractHome(t, endpoint, "5s")
			args := append([]string{"--format", "table"}, strings.Fields(tc.path)...)
			out, stderr, err := intelCLI(t, home, nil, append(args, "--query", "BTC")...)
			if !intelFailedExit(err) || len(out) != 0 || calls.RPC.Load() != 0 || !bytes.Contains(stderr, []byte(tc.domain+" list --format table")) || !bytes.Contains(stderr, []byte("--format json")) {
				t.Fatalf("unsupported format contract changed: %v stdout=%s stderr=%s", err, out, stderr)
			}
		})
	}
}

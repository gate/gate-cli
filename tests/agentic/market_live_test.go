//go:build agentic_e2e && integration

package agentic_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"testing"
	"time"
)

var marketLiveBase = flag.String("market-live-base", "", "Explicit APIv4 test environment origin for public read-only market regression")

// Explicit integration entry: real CLI/SDK/HTTP, isolated HOME, no credentials or writes.
// Missing environment input fails rather than masquerading as a passed live test.
func TestMarketPublicLive(t *testing.T) {
	if *marketLiveBase == "" {
		t.Fatal("pass -market-live-base with the test environment origin")
	}
	now := time.Now().Unix()
	for _, tc := range []struct {
		name, shape string
		args        []string
	}{
		{"kline-default", "klines", []string{"tradfi", "market", "kline", "--symbol", "XAUUSD", "--limit", "3"}},
		{"kline-legacy", "klines", []string{"tradfi", "market", "kline", "--symbol", "XAUUSD", "--type", "M1", "--limit", "3"}},
		{"kline-native", "klines", []string{"tradfi", "market", "kline", "--symbol", "XAUUSD", "--type", "1m", "--limit", "3"}},
		{"futures-insurance", "insurance", []string{"futures", "market", "insurance", "--settle", "usdt", "--limit", "3"}},
		{"delivery-insurance", "insurance", []string{"delivery", "market", "insurance", "--limit", "3"}},
		{"currencies", "currencies", []string{"spot", "market", "currencies", "--max-output-bytes", "0"}},
		{"spot-insurance", "spot-insurance", []string{"spot", "market", "insurance", "--currency", "BTC", "--limit", "3", "--from", fmt.Sprint(now - 48*60*60), "--to", fmt.Sprint(now)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"--format", "json", "cex"}, tc.args...)
			stdout, stderr, err := runCLI(t, *marketLiveBase, false, args...)
			if err != nil {
				if len(stderr) > 1024 {
					stderr = stderr[:1024]
				}
				t.Fatalf("public CLI query failed: %v stderr=%s", err, stderr)
			}
			decoder := json.NewDecoder(bytes.NewReader(stdout))
			decoder.UseNumber()
			var data interface{}
			if err := decoder.Decode(&data); err != nil {
				t.Fatalf("invalid output JSON: %v", err)
			}
			if tc.shape == "klines" {
				root, ok := data.(map[string]interface{})
				if !ok {
					t.Fatal("kline envelope is not an object")
				}
				payload, ok := root["data"].(map[string]interface{})
				if !ok {
					t.Fatal("missing kline data")
				}
				rows, ok := payload["list"].([]interface{})
				if !ok || len(rows) != 3 {
					t.Fatalf("expected 3 klines, got %d", len(rows))
				}
				for _, row := range rows {
					candle, ok := row.(map[string]interface{})
					if !ok {
						t.Fatal("invalid kline record")
					}
					stamp, ok := candle["t"].(json.Number)
					if !ok {
						t.Fatal("missing kline timestamp")
					}
					seconds, err := stamp.Int64()
					if err != nil || seconds <= 0 {
						t.Fatal("invalid kline timestamp")
					}
					for _, key := range []string{"o", "c", "h", "l"} {
						if value, ok := candle[key].(string); !ok || value == "" {
							t.Fatalf("missing kline field %s", key)
						}
					}
				}
				t.Logf("real public API: klines=%d output_bytes=%d", len(rows), len(stdout))
				return
			}
			rows, ok := data.([]interface{})
			if !ok {
				t.Fatal("expected array output")
			}
			if tc.shape == "currencies" {
				if len(rows) == 0 {
					t.Fatal("empty currency catalog")
				}
			} else if len(rows) > 3 {
				t.Fatalf("limit ignored: rows=%d", len(rows))
			}
			if tc.shape == "insurance" {
				for _, row := range rows {
					record, ok := row.(map[string]interface{})
					if !ok {
						t.Fatal("invalid insurance record")
					}
					balance, ok := record["b"].(string)
					if !ok || balance == "" {
						t.Fatal("insurance balance was not preserved as a string")
					}
				}
			}
			t.Logf("real public API: records=%d output_bytes=%d", len(rows), len(stdout))
		})
	}
}

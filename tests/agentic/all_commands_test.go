//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gate/gate-cli/internal/agentfeature"
)

type publicCommandContract struct {
	Path    string   `json:"path"`
	Use     string   `json:"use"`
	Aliases []string `json:"aliases"`
	Flags   []struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Default  string `json:"default"`
		Required bool   `json:"required"`
	} `json:"flags"`
}

func publicCommandContracts(t *testing.T) []publicCommandContract {
	t.Helper()
	raw, err := os.ReadFile("fixtures/command_contracts.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []publicCommandContract
	if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 411 {
		t.Fatalf("public command contracts=%d, want 411", len(cases))
	}
	if agentfeature.Commands {
		raw, err := os.ReadFile("fixtures/agent_command_contracts.json")
		if err != nil {
			t.Fatal(err)
		}
		var extra []publicCommandContract
		if err := json.Unmarshal(raw, &extra); err != nil {
			t.Fatal(err)
		}
		if len(extra) != 3 {
			t.Fatal("expected three CEX-capable agent common command contracts")
		}
		cases = append(cases, extra...)
	}
	return cases
}

// TestAllCommandsOfflineContract runs the frozen Common/CEX leaf inventory in
// isolated processes. Intel leaves have their own independent wire contracts.
// Help and parse failures must never authenticate or call any backend. These
// checks supplement, and do not stand in for, the business scenario/wire suites.
func TestAllCommandsOfflineContract(t *testing.T) {
	var calls atomic.Int32
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "offline command unexpectedly reached backend", http.StatusInternalServerError)
	}))
	t.Cleanup(peer.Close)
	for _, tc := range publicCommandContracts(t) {
		t.Run(tc.Path, func(t *testing.T) {
			path := strings.Fields(tc.Path)
			for _, help := range []string{"--help", "-h"} {
				out, stderr, err := runCLI(t, peer.URL, false, append(append([]string{}, path...), help)...)
				if err != nil || !bytes.Contains(out, []byte(tc.Use)) || len(stderr) != 0 {
					t.Fatalf("offline help %s: %v stdout=%s stderr=%s", help, err, out, stderr)
				}
			}
			for _, alias := range tc.Aliases {
				args := append([]string{}, path...)
				args[len(args)-1] = alias
				out, stderr, err := runCLI(t, peer.URL, false, append(args, "--help")...)
				if err != nil || !bytes.Contains(out, []byte(tc.Use)) || len(stderr) != 0 {
					t.Fatalf("alias %s lost canonical help: %v %s %s", alias, err, out, stderr)
				}
			}
			out, stderr, err := runCLI(t, peer.URL, false, append(append([]string{}, path...), "--test-unknown-flag")...)
			if err == nil || len(out) != 0 || !bytes.Contains(stderr, []byte("test-unknown-flag")) {
				t.Fatalf("unknown flag must fail: %v stdout=%s stderr=%s", err, out, stderr)
			}
			for _, f := range tc.Flags {
				// Native flag parsing rejects invalid scalars before config or HTTP.
				if f.Type == "string" || f.Type == "stringSlice" || f.Type == "stringArray" {
					continue
				}
				out, stderr, err := runCLI(t, peer.URL, false, append(append([]string{}, path...), "--"+f.Name+"=not-a-"+f.Type)...)
				if err == nil || len(out) != 0 || !bytes.Contains(stderr, []byte(f.Name)) {
					t.Fatalf("invalid --%s must fail before execution: %v %s %s", f.Name, err, out, stderr)
				}
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatalf("offline tests made %d network requests", calls.Load())
	}
}

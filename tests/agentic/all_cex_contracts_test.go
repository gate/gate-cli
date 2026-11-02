//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gate/gate-cli/tests/contracts"
)

// Each broad CEX fixture has the six declared permissions. These are synthetic
// preissued credentials, not evidence of live gateway permission enforcement.
func cexContractHome(t *testing.T, base, auth string) string {
	t.Helper()
	home := t.TempDir()
	if auth != "oauth" {
		return home
	}
	dir := filepath.Join(home, ".gate-cli")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"oauth2": map[string]any{"token": map[string]any{"client_id": "fixture-client", "access_token": "fixture-at", "token_type": "Bearer", "resource": base + "/api/v4", "expires_at": "2099-01-01T00:00:00Z", "scopes": "base spot_trade futures_trade earn cfd stock"}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	return home
}

// Every published CEX business command crosses the actual executable, Cobra,
// config/auth transport and released SDK. Only the remote API/credentials are
// fixtures. Stateful business semantics are checked by lifecycle tests as well.
func TestAllCEXCommandContracts(t *testing.T) {
	for _, tc := range contracts.Cases(t) {
		for _, auth := range []string{"hmac", "oauth"} {
			for _, format := range []string{"json", "pretty"} {
				t.Run(tc.Name+"/"+auth+"/"+format, func(t *testing.T) {
					// Each child owns its HOME, process and HTTP peer. Go's -parallel=2
					// bounds active subprocesses without sharing Cobra/stdout globals.
					t.Parallel()
					wireAuth := auth
					if tc.Public {
						wireAuth = "anonymous"
					}
					peer := contracts.NewPeer(t, tc, wireAuth, "success")
					home := cexContractHome(t, peer.URL, auth)
					out, stderr, err := futuresConfirmedInvoke(t, cliBinary, home, peer.URL, auth, append([]string{"--format", format}, tc.Args...)...)
					peer.AssertCalls(len(tc.Exchanges))
					valid := contracts.EqualJSON(out, tc.Want)
					if format == "pretty" {
						valid = string(out) == tc.WantPretty
					}
					if err != nil || len(stderr) != 0 || !valid {
						t.Fatalf("%v stdout=%s stderr=%s", err, out, stderr)
					}
				})
			}
		}
	}
}

func TestAllCEXCommandInputFailures(t *testing.T) {
	byPath := map[string]publicCommandContract{}
	for _, c := range publicCommandContracts(t) {
		byPath[c.Path] = c
	}
	for _, original := range contracts.Cases(t) {
		for _, f := range byPath[original.Name].Flags {
			if !f.Required {
				continue
			}
			t.Run(original.Name+"/missing-"+f.Name, func(t *testing.T) {
				t.Parallel()
				args := []string{}
				removed := false
				for i := 0; i < len(original.Args); i++ {
					arg := original.Args[i]
					if arg == "--"+f.Name {
						i++
						removed = true
						continue
					}
					if strings.HasPrefix(arg, "--"+f.Name+"=") {
						removed = true
						continue
					}
					args = append(args, arg)
				}
				if !removed {
					t.Fatal("required flag missing from valid input fixture")
				}
				peer := contracts.NewPeer(t, original, "hmac", "success")
				out, stderr, err := runCLI(t, peer.URL, true, append([]string{"--format", "json"}, args...)...)
				if err == nil || len(out) != 0 || !bytes.Contains(stderr, []byte(f.Name)) {
					t.Fatalf("missing flag accepted: %v %s %s", err, out, stderr)
				}
				peer.AssertCalls(0)
			})
		}
		for i, arg := range original.Args {
			if arg != "--json" && arg != "--orders-json" && arg != "--input" {
				continue
			}
			t.Run(original.Name+"/invalid-json", func(t *testing.T) {
				t.Parallel()
				args := append([]string{}, original.Args...)
				args[i+1] = "{invalid-private-input-canary"
				peer := contracts.NewPeer(t, original, "hmac", "success")
				out, stderr, err := runCLI(t, peer.URL, true, append([]string{"--format", "json"}, args...)...)
				if err == nil || len(out) != 0 || bytes.Contains(stderr, []byte("private-input-canary")) {
					t.Fatalf("invalid JSON accepted/exposed: %v %s %s", err, out, stderr)
				}
				peer.AssertCalls(0)
			})
		}
	}
}

func TestAllCEXCommandFailures(t *testing.T) {
	for _, tc := range contracts.Cases(t) {
		faults := []string{"401", "403", "429", "503"}
		if !tc.Void {
			faults = append(faults, "malformed")
		}
		// An accepted write followed by EOF must never be submitted a second time.
		if tc.Exchanges[len(tc.Exchanges)-1].Method != "GET" {
			faults = append(faults, "disconnect")
		}
		for _, fault := range faults {
			t.Run(tc.Name+"/"+fault, func(t *testing.T) {
				t.Parallel()
				auth := "hmac"
				if tc.Public {
					auth = "anonymous"
				}
				peer := contracts.NewPeer(t, tc, auth, fault)
				out, stderr, err := runCLI(t, peer.URL, true, append([]string{"--format", "json"}, tc.Args...)...)
				peer.AssertCalls(len(tc.Exchanges))
				if err == nil || len(bytes.TrimSpace(out)) != 0 {
					t.Fatalf("dependency failure accepted: %v %s %s", err, out, stderr)
				}
				var envelope struct {
					Error json.RawMessage `json:"error"`
				}
				if json.Unmarshal(stderr, &envelope) != nil || len(envelope.Error) == 0 {
					t.Fatalf("missing structured failure: %s", stderr)
				}
				if fault != "malformed" && fault != "disconnect" && !bytes.Contains(stderr, []byte("contract-trace")) {
					t.Errorf("trace lost: %s", stderr)
				}
				for _, secret := range []string{"synthetic-secret", "synthetic-key", "fixture-at"} {
					if bytes.Contains(stderr, []byte(secret)) {
						t.Errorf("credential leaked: %s", secret)
					}
				}
			})
		}
	}
}

func TestAllCEXAnonymousAccess(t *testing.T) {
	for _, tc := range contracts.Cases(t) {
		t.Run(tc.Name, func(t *testing.T) {
			t.Parallel()
			peer := contracts.NewPeer(t, tc, "anonymous", "success")
			out, stderr, err := runCLI(t, peer.URL, false, append([]string{"--format", "json"}, tc.Args...)...)
			if tc.Anonymous {
				peer.AssertCalls(len(tc.Exchanges))
				if err != nil || !contracts.EqualJSON(out, tc.Want) {
					t.Fatalf("public call: %v %s %s", err, out, stderr)
				}
			} else {
				peer.AssertCalls(0)
				if err == nil || len(out) != 0 {
					t.Fatalf("private command accepted anonymously: %v %s %s", err, out, stderr)
				}
				if !bytes.Contains(stderr, []byte("auth")) {
					t.Errorf("missing authentication diagnostic: %s", stderr)
				}
			}
		})
	}
}

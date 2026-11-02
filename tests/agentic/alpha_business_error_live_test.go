//go:build agentic_e2e && integration

package agentic_test

import (
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/gate/gate-cli/internal/output"
)

// Three public, bounded GETs verify business-error handling against an explicitly selected
// test origin. runCLI isolates HOME and removes credentials; no API peer is mocked.
func TestAlphaBusinessErrorLive(t *testing.T) {
	if *marketLiveBase == "" {
		t.Fatal("pass -market-live-base with the test environment origin")
	}
	for _, tc := range []struct {
		command, flag, value, label, message string
	}{
		{"tickers", "--currency", "USDS", "BAD_REQUEST", "meme token not found"},
		{"currencies", "--currency", "USDS", "BAD_REQUEST", "meme token not found"},
		{"tokens", "--chain", "ETH", "INVALID_ARGUMENT", "json: cannot unmarshal object into Go value of type []*types.AlphaV4CurrencyItem"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			args := []string{"--format", "json", "cex", "alpha", "market", tc.command, tc.flag, tc.value, "--page", "1"}
			if tc.command != "tokens" {
				args = append(args, "--limit", "1")
			}
			stdout, stderr, err := runCLI(t, *marketLiveBase, false, args...)
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 1 || len(stdout) != 0 {
				t.Fatalf("exit=%v stdout=%s stderr=%s", err, stdout, stderr)
			}
			var envelope struct{ Error output.GateError }
			if err := json.Unmarshal(stderr, &envelope); err != nil {
				t.Fatalf("error envelope: %v stderr=%s", err, stderr)
			}
			ge := envelope.Error
			if ge.Status != 200 || ge.Label != tc.label || ge.Message != tc.message || ge.ErrorType != "INVALID_ARGS" || !ge.Retryable || !strings.Contains(ge.SuggestedNextAction, "corrected args") {
				t.Fatalf("live business error=%+v", ge)
			}
			t.Logf("HTTP=%d label=%s message=%q type=%s exit=1", ge.Status, ge.Label, ge.Message, ge.ErrorType)
		})
	}
}

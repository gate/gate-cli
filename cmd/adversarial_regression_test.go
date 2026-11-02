package cmd

import (
	"strings"
	"testing"

	"github.com/gate/gate-cli/tests/contracts"
)

// Regressions enter the same startup boundary as the real CLI. They are ordinary
// unit tests: repaired business defects must never rely on a manual-only tag.
func TestAdversarialCommandsRejectUnexpectedArguments(t *testing.T) {
	setupRootForExecute()
	for _, tc := range loadCommandContracts(t) {
		t.Run(tc.Path, func(t *testing.T) {
			command, rest, err := rootCmd.Find(strings.Fields(tc.Path))
			if err != nil || len(rest) != 0 {
				t.Fatalf("resolve command: %v %v", err, rest)
			}
			args := []string{"__unexpected_position__"}
			// Preserve each documented positional argument before adding the
			// unexpected word, including config set's key and value.
			positionals := 0
			switch tc.Path {
			case "config set":
				positionals = 2
			case "cex assetswap order get", "cex earn dual refund-preview", "cex futures market risk-limit-table":
				positionals = 1
			}
			for range positionals {
				args = append([]string{"synthetic-positional-value"}, args...)
			}
			if err := command.ValidateArgs(args); err == nil {
				t.Error("undocumented positional argument accepted; must reject before any business call")
			}
		})
	}
}

func TestAdversarialTradingIntentRejected(t *testing.T) {
	for _, tc := range contracts.UnsafeIntents(t) {
		t.Run(tc.ID, func(t *testing.T) {
			peer := contracts.NewPeer(t, tc.Case, "hmac", "success")
			out, err := invokeContractHandler(t, tc.Case, peer.URL, t.Context())
			terminalReached := peer.Calls() >= len(tc.Case.Exchanges)
			if err == nil || terminalReached || len(out) != 0 {
				t.Errorf("unsafe intent not rejected: err=%v terminal_write=%t calls=%d; forbidden_body=%s", err, terminalReached, peer.Calls(), tc.Case.Exchanges[len(tc.Case.Exchanges)-1].Body)
			}
		})
	}
}

package cmdhint

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSuggestUnknownFlagPairOnSpotTicker(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "gate-cli"}
	cex := &cobra.Command{Use: "cex"}
	spot := &cobra.Command{Use: "spot"}
	market := &cobra.Command{Use: "market"}
	ticker := &cobra.Command{Use: "ticker", Run: func(*cobra.Command, []string) {}}
	ticker.Flags().String("pair", "", "pair")
	root.AddCommand(cex)
	cex.AddCommand(spot)
	spot.AddCommand(market)
	market.AddCommand(ticker)

	argv := []string{"gate-cli", "cex", "spot", "market", "ticker", "--pairs", "BTC_USDT"}
	got := suggestUnknownFlag(root, argv, `unknown flag: --pairs`)
	if got == "" || !strings.Contains(got, "--pair") {
		t.Fatalf("got %q", got)
	}
}

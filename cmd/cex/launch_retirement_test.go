package cex

import (
	"strings"
	"testing"

	"github.com/gate/gate-cli/internal/operations"
)

// Keep the old inventory independent of registration: removing a command must
// not remove its negative test or the transport's release-level deny rule.
func TestLaunchRetirementInventory(t *testing.T) {
	cases := []struct{ command, method, path string }{
		{"projects", "GET", "/project-list"},
		{"pledge", "POST", "/create-order"},
		{"redeem", "POST", "/redeem"},
		{"pledge-records", "GET", "/user-pledge-records"},
		{"reward-records", "GET", "/get-user-reward-records"},
		{"hodler projects", "GET", "/hodler-airdrop/project-list"},
		{"hodler order", "POST", "/hodler-airdrop/order"},
		{"hodler order-records", "GET", "/hodler-airdrop/user-order-records"},
		{"hodler airdrop-records", "GET", "/hodler-airdrop/user-airdrop-records"},
		{"candy-drop activities", "GET", "/candydrop/activity-list"},
		{"candy-drop rules", "GET", "/candydrop/activity-rules"},
		{"candy-drop register", "POST", "/candydrop/register"},
		{"candy-drop progress", "GET", "/candydrop/task-progress"},
		{"candy-drop participations", "GET", "/candydrop/participation-records"},
		{"candy-drop airdrops", "GET", "/candydrop/airdrop-records"},
	}
	for _, tc := range cases {
		t.Run(tc.command, func(t *testing.T) {
			if command, rest, err := Cmd.Find(strings.Fields("launch " + tc.command)); err == nil && len(rest) == 0 && command != Cmd {
				t.Fatal("retired command still resolves")
			}
			disabled, err := operations.IsDisabled(tc.method, "/api/v4/launch"+tc.path)
			if err != nil || !disabled {
				t.Fatalf("old SDK route can send: disabled=%v err=%v", disabled, err)
			}
		})
	}
}

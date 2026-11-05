package cex

import "testing"

// Cached command paths must no longer resolve to financial handlers. Retained
// wallet reads guard against removing their parent groups with the write leaves.
func TestWithdrawalRetirementRegistration(t *testing.T) {
	for _, leaf := range []string{"create", "cancel"} {
		command, rest, err := Cmd.Find([]string{"withdrawal", leaf})
		if err == nil && len(rest) == 0 && command.Runnable() {
			t.Errorf("removed withdrawal leaf still registered: %s", leaf)
		}
	}
	for _, path := range [][]string{
		{"wallet", "deposit", "withdrawals"},
		{"wallet", "market", "withdraw-status"},
		{"wallet", "deposit", "address"},
		{"wallet", "transfer", "create"},
	} {
		command, rest, err := Cmd.Find(path)
		if err != nil || len(rest) != 0 || !command.Runnable() {
			t.Errorf("retained asset command missing: %v rest=%v err=%v", path, rest, err)
		}
	}
}

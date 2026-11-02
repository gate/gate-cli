package cex

import (
	"io"
	"testing"
)

// TestRetiredCommandsUnavailable checks actual Cobra resolution and execution,
// including cached command paths. It never invokes any financial handler.
func TestRetiredCommandsUnavailable(t *testing.T) {
	Cmd.SetOut(io.Discard)
	Cmd.SetErr(io.Discard)
	defer func() { Cmd.SetOut(nil); Cmd.SetErr(nil); Cmd.SetArgs(nil) }()
	for _, name := range []string{"sub-account", "launch", "p2p", "cross-ex", "welfare", "activity", "coupon", "ann", "announcement"} {
		for _, command := range Cmd.Commands() {
			if command.Name() == name || command.HasAlias(name) {
				t.Fatalf("retired command registered: %s", name)
			}
		}
		Cmd.SetArgs([]string{name})
		if err := Cmd.Execute(); err == nil {
			t.Errorf("retired command succeeded: %s", name)
		}
	}
	for _, path := range [][]string{
		{"alpha", "market", "currencies"}, {"alpha", "market", "tickers"}, {"alpha", "market", "tokens"},
		{"alpha", "account", "balances"}, {"alpha", "account", "book"},
		{"alpha", "order", "quote"}, {"alpha", "order", "place"}, {"alpha", "order", "get"}, {"alpha", "order", "list"},
		{"wallet", "balance", "sub"}, {"wallet", "transfer", "to-sub"}, {"wallet", "transfer", "sub-to-sub"},
		{"earn", "fixed", "products"}, {"earn", "uni", "currencies"},
		{"spot", "market", "ticker"}, {"futures", "market", "contracts"},
	} {
		command, rest, err := Cmd.Find(path)
		if err != nil || len(rest) != 0 || command.RunE == nil {
			t.Errorf("retained command missing: %v err=%v", path, err)
		}
	}
}

// Cached leaf paths must fail before invoking any handler, not only disappear
// from the top-level menu. Wallet subaccount routes remain covered above.
func TestSubaccountRetiredLeaves(t *testing.T) {
	Cmd.SetOut(io.Discard)
	Cmd.SetErr(io.Discard)
	defer func() { Cmd.SetOut(nil); Cmd.SetErr(nil); Cmd.SetArgs(nil) }()
	for _, path := range [][]string{
		{"create"}, {"get"}, {"list"}, {"lock"}, {"unlock"}, {"unified-mode"},
		{"key", "create"}, {"key", "delete"}, {"key", "get"}, {"key", "list"}, {"key", "update"},
	} {
		args := append([]string{"sub-account"}, path...)
		Cmd.SetArgs(args)
		if err := Cmd.Execute(); err == nil {
			t.Errorf("retired leaf succeeded: %v", args)
		}
	}
}

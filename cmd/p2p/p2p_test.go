package p2p

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestP2pCommandStructure(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"user-info", "counterparty", "payment", "ads", "transaction", "chat"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing subcommand: %s", name)
	}
}

func TestAdsSubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range adsCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"list", "my-list", "detail", "update-status"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing ads subcommand: %s", name)
	}
}

func TestTransactionSubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range transactionCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"pending", "completed", "detail", "confirm-payment", "confirm-receipt", "cancel", "push-order"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing transaction subcommand: %s", name)
	}
}

func TestChatSubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range chatCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"list", "send", "upload"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing chat subcommand: %s", name)
	}
}

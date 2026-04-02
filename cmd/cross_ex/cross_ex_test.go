package crossex

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCrossExCommandStructure(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"market", "account", "transfer", "order", "position", "convert"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing subcommand group: %s", name)
	}
}

func TestMarketSubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range marketCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"symbols", "risk-limits", "transfer-coins", "fee", "interest-rate", "discount-rate"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing market subcommand: %s", name)
	}
}

func TestAccountSubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range accountCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"get", "update", "book"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing account subcommand: %s", name)
	}
}

func TestTransferSubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range transferCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"list", "create"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing transfer subcommand: %s", name)
	}
}

func TestOrderSubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range orderCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"list", "get", "create", "update", "cancel", "history", "trades"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing order subcommand: %s", name)
	}
}

func TestPositionSubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range positionCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"list", "margin-list", "close", "leverage", "set-leverage",
		"margin-leverage", "set-margin-leverage", "adl-rank", "history", "margin-history", "margin-interests"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing position subcommand: %s", name)
	}
}

func TestConvertSubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range convertCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"quote", "create"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing convert subcommand: %s", name)
	}
}

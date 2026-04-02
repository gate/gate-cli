package rebate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRebateCommandStructure(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range Cmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"user-info", "sub-relation", "partner", "broker", "agency"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing subcommand: %s", name)
	}
}

func TestPartnerSubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range partnerCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"transactions", "commissions", "sub-list", "eligibility", "application", "agent-data"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing partner subcommand: %s", name)
	}
}

func TestBrokerSubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range brokerCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"commissions", "transactions"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing broker subcommand: %s", name)
	}
}

func TestAgencySubcommands(t *testing.T) {
	subCmds := make(map[string]bool)
	for _, c := range agencyCmd.Commands() {
		subCmds[c.Name()] = true
	}

	expected := []string{"commissions", "transactions"}
	for _, name := range expected {
		assert.True(t, subCmds[name], "missing agency subcommand: %s", name)
	}
}

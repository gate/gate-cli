//go:build !agent

package cmdhint

import "io"

// AgentModeEnabled is false when agent support is not compiled in (-tags agent).
func AgentModeEnabled() bool { return false }

// AgentResolveHint is a no-op without -tags agent.
func AgentResolveHint(string) string { return "" }

// AgentPreflightNextAction is unused when AgentModeEnabled is false.
func AgentPreflightNextAction(string) string { return "" }

// AgentMigrateNextAction is unused when AgentModeEnabled is false.
func AgentMigrateNextAction(string) string { return "" }

// AgentDoctorNextAction is unused when AgentModeEnabled is false.
func AgentDoctorNextAction(string) string { return "" }

// EnrichDiagnosticWithAgentLeaf is a no-op without -tags agent.
func EnrichDiagnosticWithAgentLeaf(d *Diagnostic, argv []string) {}

// ShouldBlockParentHelp is a no-op without -tags agent.
func ShouldBlockParentHelp(argv []string) bool { return false }

func printAgentResolveHint(w io.Writer, argv []string) {}

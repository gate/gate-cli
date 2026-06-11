//go:build agent

package agentfeature

// Commands enables GateAI agent discovery and GATE_CLI_AGENT runtime behaviors.
// Wire agent commands: go build -tags agent .  (see cmd/root_agent_wire.go).
const Commands = true

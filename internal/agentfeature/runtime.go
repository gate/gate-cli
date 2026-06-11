//go:build agent

package agentfeature

import (
	"os"
	"strings"
)

// DefaultMaxOutputBytes is the agent-mode stdout cap when GATE_MAX_OUTPUT_BYTES is unset.
const DefaultMaxOutputBytes int64 = 65536

// RuntimeEnvEnabled reports GATE_CLI_AGENT / GATE_AI_AGENT truthy values (ignores build tag).
func RuntimeEnvEnabled() bool {
	switch strings.TrimSpace(os.Getenv("GATE_CLI_AGENT")) {
	case "1", "true", "yes":
		return true
	}
	switch strings.TrimSpace(os.Getenv("GATE_AI_AGENT")) {
	case "1", "true", "yes":
		return true
	}
	return false
}

// RuntimeActive is true when agent commands are compiled in and agent env is set.
func RuntimeActive() bool {
	return Commands && RuntimeEnvEnabled()
}

// DefaultMaxOutputWhenUnset returns the --max-output-bytes default after env is absent.
func DefaultMaxOutputWhenUnset() int64 {
	if !RuntimeActive() {
		return 0
	}
	return DefaultMaxOutputBytes
}

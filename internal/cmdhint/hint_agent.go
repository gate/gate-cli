//go:build agent

package cmdhint

import (
	"fmt"
	"io"
	"strings"
)

func printAgentResolveHint(w io.Writer, argv []string) {
	q := strings.TrimSpace(QueryFromArgv(argv))
	if q == "" || w == nil {
		return
	}
	_, _ = fmt.Fprintf(w, "gate_cli_agent_resolve_hint=%s\n", AgentResolveHint(q))
}

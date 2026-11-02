package version

import (
	"github.com/gate/gate-cli/internal/agentfeature"
	"runtime"
	"runtime/debug"
)

// Detail is an explicit diagnostic whitelist; paths, dependencies and environment
// values must never be added by dumping BuildInfo or configuration wholesale.
type Detail struct {
	SchemaVersion int    `json:"schema_version"`
	Version       string `json:"version"`
	Revision      string `json:"revision"`
	Go            string `json:"go"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	AgentCommands bool   `json:"agent_commands"`
}

// Details describes this compiled artifact, independently of agent runtime env.
func Details() Detail {
	d := Detail{1, Version, "unknown", runtime.Version(), runtime.GOOS, runtime.GOARCH, agentfeature.Commands}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				d.Revision = s.Value
				break
			}
		}
	}
	return d
}

package intelcmd

import "github.com/spf13/cobra"

// AddFallbackArgFlags registers --params / --args-json / --args-file for intel-style leaf commands.
func AddFallbackArgFlags(cmd *cobra.Command) {
	cmd.Flags().String("params", "", "Fallback JSON object arguments (use flat flags by default)")
	cmd.Flags().String("args-json", "", "Alias of --params (fallback JSON object)")
	cmd.Flags().String("args-file", "", "Path to JSON file containing fallback arguments object")
}

// rootCLIOnlyFlagNames are gate-cli root PersistentFlags that must never be forwarded as
// MCP tools/call arguments for info/news. Cobra marks changed persistent flags as Changed on
// the leaf FlagSet, so MergeFromCommand would otherwise leak them (e.g. format=json) into
// backends that reject additionalProperties.
//
// Keep this list aligned with cmd/root.go PersistentFlags for Intel-facing globals only.
var rootCLIOnlyFlagNames = []string{
	"format",
	"profile",
	"debug",
	"verbose",
	"max-output-bytes",
	"api-key",
	"api-secret",
	"help",
}

func cloneStringSet(src map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(src)+8)
	for k := range src {
		out[k] = struct{}{}
	}
	return out
}

// ReservedCLIGlobalFlags returns root CLI-only flags excluded from MCP argument merge.
func ReservedCLIGlobalFlags() map[string]struct{} {
	out := make(map[string]struct{}, len(rootCLIOnlyFlagNames))
	for _, name := range rootCLIOnlyFlagNames {
		out[name] = struct{}{}
	}
	return out
}

// ReservedMCPJSONFallbackFlags is passed to MergeFromCommand for info/news leaf aliases.
// It excludes JSON fallback flags and root CLI globals; it does not strip keys that the user
// explicitly places inside --params / --args-json / --args-file.
func ReservedMCPJSONFallbackFlags() map[string]struct{} {
	m := ReservedCLIGlobalFlags()
	m["params"] = struct{}{}
	m["args-json"] = struct{}{}
	m["args-file"] = struct{}{}
	return m
}

// ReservedInvokeFlags is ReservedMCPJSONFallbackFlags plus the hidden invoke --name flag.
func ReservedInvokeFlags() map[string]struct{} {
	m := cloneStringSet(ReservedMCPJSONFallbackFlags())
	m["name"] = struct{}{}
	return m
}

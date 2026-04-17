package intelcmd

import "github.com/spf13/cobra"

// AddFallbackArgFlags registers --params / --args-json / --args-file for intel-style leaf commands.
func AddFallbackArgFlags(cmd *cobra.Command) {
	cmd.Flags().String("params", "", "Fallback JSON object arguments (use flat flags by default)")
	cmd.Flags().String("args-json", "", "Alias of --params (fallback JSON object)")
	cmd.Flags().String("args-file", "", "Path to JSON file containing fallback arguments object")
}

// ReservedMCPJSONFallbackFlags is the flag set passed to MergeFromCommand for JSON fallback paths.
func ReservedMCPJSONFallbackFlags() map[string]struct{} {
	return map[string]struct{}{
		"params":    {},
		"args-json": {},
		"args-file": {},
	}
}

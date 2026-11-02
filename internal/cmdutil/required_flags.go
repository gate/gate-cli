package cmdutil

import "github.com/spf13/cobra"

// MustRequireFlags marks statically declared flags during command construction.
// Missing declarations are programmer errors: fail at startup rather than silently
// publishing a write command without its required arguments. Do not use on user input.
func MustRequireFlags(cmd *cobra.Command, names ...string) {
	for _, name := range names {
		if err := cmd.MarkFlagRequired(name); err != nil {
			panic(err)
		}
	}
}

package cmdutil

import (
	"fmt"
	"github.com/spf13/cobra"
	"strings"
)

// OrderText validates only the new optional ordinary-order field. Missing flags
// preserve old direct handler calls; existing POV/chase validation stays intact.
func OrderText(cmd *cobra.Command) (string, error) {
	if cmd.Flags().Lookup("text") == nil {
		return "", nil
	}
	value, err := cmd.Flags().GetString("text")
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", nil
	}
	if !strings.HasPrefix(value, "t-") || len(value) > 30 {
		return "", fmt.Errorf("--text must start with t- and have at most 28 suffix bytes")
	}
	// Validate the whole token so suffixes '.' and '..' remain valid SDK text.
	if err := TradingToken("--text", value); err != nil {
		return "", err
	}
	return value, nil
}

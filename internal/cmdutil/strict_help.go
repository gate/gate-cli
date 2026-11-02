package cmdutil

import (
	"fmt"

	"github.com/gate/gate-cli/internal/output"
	"github.com/spf13/cobra"
)

// InstallStrictHelpCommand preserves Cobra's help metadata and completion while
// returning lookup failures to the application's normal error/exit boundary.
// Installation is startup-only and can be repeated without adding registrations.
func InstallStrictHelpCommand(root *cobra.Command) {
	if root == nil {
		return
	}
	root.InitDefaultHelpCmd()
	help := rootHelpCommand(root)
	if help == nil {
		return
	}
	help.Run = nil
	help.RunE = func(c *cobra.Command, args []string) error {
		target, err := resolveHelpTopic(c.Root(), args)
		if err != nil {
			return err
		}
		// Match Cobra's rendering/context behavior without executing business
		// Args, required flags, hooks or handlers on the help target.
		if target.Context() == nil {
			target.SetContext(c.Context())
		}
		target.InitDefaultHelpFlag()
		target.InitDefaultVersionFlag()
		return target.Help()
	}
	root.SetHelpCommand(help)
}

func rootHelpCommand(root *cobra.Command) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() == "help" {
			return command
		}
	}
	return nil
}

// resolveHelpTopic requires the entire topic to be a command path. Find alone
// also accepts a known prefix plus leftover arguments, which is not valid help.
func resolveHelpTopic(root *cobra.Command, args []string) (*cobra.Command, error) {
	if len(args) == 0 {
		return root, nil
	}
	target, remaining, err := root.Find(args)
	if err != nil {
		return nil, output.InvalidArgsError(err.Error())
	}
	// The existing structured-error boundary renders these failures in every
	// output format, including table where plain execution errors show only hints.
	if target == nil {
		return nil, output.InvalidArgsError(fmt.Sprintf("unknown command %q for %q", args[0], root.CommandPath()))
	}
	if len(remaining) != 0 {
		return nil, output.InvalidArgsError(fmt.Sprintf("unknown command %q for %q", remaining[0], target.CommandPath()))
	}
	return target, nil
}

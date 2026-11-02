package cmdutil

import (
	"io"

	"github.com/gate/gate-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// helpFlagValue consumes flag syntax without mutating the command's real values.
// Only arity matters here; Cobra still performs the authoritative flag validation.
type helpFlagValue struct{ kind string }

func (v *helpFlagValue) String() string   { return "" }
func (v *helpFlagValue) Set(string) error { return nil }
func (v *helpFlagValue) Type() string     { return v.kind }

// ValidateCommandPath rejects unresolved group arguments before Cobra's help
// early return, including non-runnable groups without a help flag. Runnable
// commands retain ownership of their business positional arguments.
func ValidateCommandPath(root *cobra.Command, args []string) error {
	if root == nil {
		return nil
	}
	// Cobra normally registers this group inside Execute, after our preflight.
	// Its public initializer is idempotent and respects custom/disabled completion.
	root.InitDefaultCompletionCmd(args...)
	command, remaining, err := root.Find(args)
	if err != nil {
		// Execute initializes dynamic commands such as __complete before its
		// own lookup. Defer lookup errors so preflight cannot reject
		// a valid completion request before Cobra has registered that command.
		return nil
	}
	if command == nil || command.DisableFlagParsing {
		return nil
	}
	explicitHelp := command == rootHelpCommand(root)
	if !explicitHelp && !command.HasSubCommands() {
		return nil
	}
	command.InitDefaultHelpFlag()
	parser := pflag.NewFlagSet(command.Name(), pflag.ContinueOnError)
	parser.SetOutput(io.Discard)
	parser.SetNormalizeFunc(command.Flags().GetNormalizeFunc())
	var help bool
	helpValue := pflag.NewFlagSet("help-value", pflag.ContinueOnError)
	helpValue.BoolVar(&help, "help", false, "")
	copyFlag := func(flag *pflag.Flag) {
		if parser.Lookup(flag.Name) != nil {
			return
		}
		cloned := *flag
		cloned.Changed = false
		if flag.Name == "help" {
			cloned.Value = helpValue.Lookup("help").Value
		} else {
			cloned.Value = &helpFlagValue{kind: flag.Value.Type()}
		}
		parser.AddFlag(&cloned)
	}
	command.Flags().VisitAll(copyFlag)
	command.InheritedFlags().VisitAll(copyFlag)
	// Real parsing remains authoritative for flag errors. This clone only
	// separates positional words from values without changing live flags.
	if err := parser.Parse(remaining); err != nil || len(parser.Args()) == 0 {
		return nil
	}
	// Cobra handles help flags before RunE. Use the same topic resolver here
	// so "help missing --help" cannot bypass strict command-path validation.
	if explicitHelp {
		if !help {
			return nil // The strict help RunE owns ordinary explicit help topics.
		}
		_, err := resolveHelpTopic(root, parser.Args())
		return err
	}
	if !command.Runnable() {
		// Cobra checks Runnable before Args and otherwise returns successful
		// help. Pure groups accept only registered children, regardless of Args.
		if err := cobra.NoArgs(command, parser.Args()); err != nil {
			return output.InvalidArgsError(err.Error())
		}
	}
	if !help {
		return nil // Avoid executing a runnable parent's custom Args twice.
	}
	if command.Args != nil {
		return command.ValidateArgs(parser.Args())
	}
	return nil
}

package info

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/toolschema"
)

var verifySchemaCmd = &cobra.Command{
	Use:   "verify-schema",
	Short: "Verify cached Info tool schemas and print warnings",
	Long:  "Offline check for cached Info schemas. Prints warning list with actionable suggestions.",
	Example: "  gate-cli info verify-schema --format json\n" +
		"  gate-cli info verify-schema --strict",
	RunE:          runInfoVerifySchema,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	verifySchemaCmd.Flags().Bool("strict", false, "Treat schema warnings as command error")
	Cmd.AddCommand(verifySchemaCmd)
}

func runInfoVerifySchema(cmd *cobra.Command, args []string) error {
	p := getPrinter(cmd)
	if p.IsTable() {
		return intelcmd.FailUnsupportedTable(p)
	}
	items, fresh, err := toolschema.LoadCache("info")
	if err != nil {
		return intelcmd.FailAfterPrintError(p, &output.GateError{Status: 500, Label: "SCHEMA_CACHE_READ_FAILED", Message: err.Error()})
	}
	if len(items) == 0 {
		return intelcmd.FailAfterPrintError(p, &output.GateError{Status: 404, Label: "SCHEMA_CACHE_EMPTY", Message: "no cached schema found; run an info command with intel backend configured first"})
	}
	report := toolschema.ValidateTools("info", items, fresh)
	strict, _ := cmd.Flags().GetBool("strict")
	report.StrictMode = strict
	report.StrictFailed = strict && report.WarningCount > 0
	if err := p.Print(report); err != nil {
		return err
	}
	if strict && report.WarningCount > 0 {
		return intelcmd.FailAfterPrintError(p, &output.GateError{
			Status:  422,
			Label:   "SCHEMA_VERIFY_FAILED",
			Message: "schema warnings found in strict mode",
		})
	}
	return nil
}

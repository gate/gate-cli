package news

import (
	"errors"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/toolschema"
)

var verifySchemaCmd = &cobra.Command{
	Use:   "verify-schema",
	Short: "Verify cached News tool schemas and print warnings",
	Long:  "Offline check for cached News schemas. Prints warning list with actionable suggestions.",
	Example: "  gate-cli news verify-schema --format json\n" +
		"  gate-cli news verify-schema --strict",
	RunE:  runNewsVerifySchema,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	verifySchemaCmd.Flags().Bool("strict", false, "Treat schema warnings as command error")
	Cmd.AddCommand(verifySchemaCmd)
}

func runNewsVerifySchema(cmd *cobra.Command, args []string) error {
	p := getPrinter(cmd)
	items, fresh, err := toolschema.LoadCache("news")
	if err != nil {
		p.PrintError(&output.GateError{Status: 500, Label: "SCHEMA_CACHE_READ_FAILED", Message: err.Error()})
		return errors.New("schema cache read failed")
	}
	if len(items) == 0 {
		p.PrintError(&output.GateError{Status: 404, Label: "SCHEMA_CACHE_EMPTY", Message: "no cached schema found; run a news command with intel backend configured first"})
		return errors.New("schema cache empty")
	}
	report := toolschema.ValidateTools("news", items, fresh)
	strict, _ := cmd.Flags().GetBool("strict")
	report.StrictMode = strict
	report.StrictFailed = strict && report.WarningCount > 0
	if p.IsJSON() {
		_ = p.Print(report)
	} else {
		_ = p.Table(
			[]string{"Backend", "Status", "ToolCount", "CacheFresh", "WarningCount"},
			[][]string{{report.Backend, report.Status, itoa(report.ToolCount), boolString(report.CacheFresh), itoa(report.WarningCount)}},
		)
		rows := make([][]string, 0, len(report.Warnings))
		for _, w := range report.Warnings {
			rows = append(rows, []string{w.Tool, w.Field, w.Code, w.Message})
		}
		if len(rows) == 0 {
			rows = append(rows, []string{"-", "-", "ok", "no schema warnings"})
		}
		_ = p.Table([]string{"Tool", "Field", "Code", "Message"}, rows)
	}
	if strict && report.WarningCount > 0 {
		p.PrintError(&output.GateError{
			Status:  422,
			Label:   "SCHEMA_VERIFY_FAILED",
			Message: "schema warnings found in strict mode",
		})
		return errors.New("schema verify failed in strict mode")
	}
	return nil
}

func itoa(v int) string {
	return strconv.Itoa(v)
}

func boolString(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

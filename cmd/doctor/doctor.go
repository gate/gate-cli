package doctor

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdutil"
	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/migration"
	"github.com/gate/gate-cli/internal/output"
)

// Cmd is the doctor command.
var Cmd = &cobra.Command{
	Use:           "doctor",
	Short:         "Diagnose local CLI and legacy MCP state",
	RunE:          runDoctor,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	Cmd.Flags().String("check", "all", "Comma-separated checks: all,cli,version,config,connectivity,legacy-mcp")
	Cmd.Flags().Bool("strict", false, "Treat warnings as command failure")
}

func runDoctor(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	checkRaw, _ := cmd.Flags().GetString("check")
	strict, _ := cmd.Flags().GetBool("strict")
	profile, _ := cmd.Root().PersistentFlags().GetString("profile")

	report := migration.BuildDoctorReport(migration.DoctorOptions{
		Profile: profile,
		Checks:  migration.ParseCheckList(checkRaw),
		Strict:  strict,
		InfoURL: os.Getenv("GATE_INTEL_INFO_MCP_URL"),
		NewsURL: os.Getenv("GATE_INTEL_NEWS_MCP_URL"),
	})

	if p.IsJSON() {
		if err := p.Print(report); err != nil {
			return exitcode.New(30, err)
		}
	} else {
		if err := p.Table(
			[]string{"Status", "CLIInstalled", "CLIVersion", "MinVersion", "LegacyMCPDetected"},
			[][]string{{
				report.Status,
				boolString(report.Summary.CLIInstalled),
				report.Summary.CLIVersion,
				report.Summary.MinimumRequiredVersion,
				boolString(report.Summary.LegacyMCPDetected),
			}},
		); err != nil {
			return exitcode.New(30, err)
		}
		checkRows := make([][]string, 0, len(report.Checks))
		for _, c := range report.Checks {
			checkRows = append(checkRows, []string{c.ID, c.Status, boolString(c.Blocking), c.Message})
		}
		if err := p.Table([]string{"ID", "Status", "Blocking", "Message"}, checkRows); err != nil {
			return exitcode.New(30, err)
		}
	}

	if report.Status == "fail" {
		p.PrintError(&output.GateError{Status: 422, Label: "DOCTOR_FAILED", Message: "doctor checks failed"})
		return exitcode.New(migration.DoctorExitCode(report), errors.New("doctor failed"))
	}
	if report.Status == "warn" {
		return exitcode.New(migration.DoctorExitCode(report), nil)
	}
	return nil
}

func boolString(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

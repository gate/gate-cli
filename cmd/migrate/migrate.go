package migrate

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdutil"
	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/migration"
	"github.com/gate/gate-cli/internal/output"
)

// Cmd is the migrate command.
var Cmd = &cobra.Command{
	Use:           "migrate",
	Short:         "Migrate local provider configs away from legacy Gate MCP entries",
	RunE:          runMigrate,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func init() {
	Cmd.Flags().Bool("dry-run", false, "Preview migration without changing files")
	Cmd.Flags().Bool("apply", false, "Apply migration changes")
	Cmd.Flags().Bool("yes", false, "Confirm apply without prompt")
	Cmd.Flags().String("provider", "", "Comma-separated providers: codex,cursor,claude_desktop")
	Cmd.Flags().String("backup-dir", "", "Backup directory for modified config files")
}

func runMigrate(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	apply, _ := cmd.Flags().GetBool("apply")
	yes, _ := cmd.Flags().GetBool("yes")
	providerRaw, _ := cmd.Flags().GetString("provider")
	backupDir, _ := cmd.Flags().GetString("backup-dir")

	if err := migration.ValidateMode(apply, dryRun); err != nil {
		p.PrintError(&output.GateError{Status: 400, Label: "INVALID_ARGUMENTS", Message: err.Error()})
		return exitcode.New(30, err)
	}
	if apply && !yes {
		p.PrintError(&output.GateError{Status: 400, Label: "CONFIRMATION_REQUIRED", Message: "use --yes with --apply to run non-interactive migration"})
		return exitcode.New(30, errors.New("confirmation required"))
	}

	report, err := migration.RunMigrate(migration.MigrateOptions{
		Apply:       apply,
		ProviderIDs: migration.ParseProviders(providerRaw),
		BackupDir:   backupDir,
	})
	if err != nil {
		p.PrintError(&output.GateError{Status: 500, Label: "MIGRATE_FAILED", Message: err.Error()})
		return exitcode.New(30, errors.New("migrate failed"))
	}

	if p.IsJSON() {
		if err := p.Print(report); err != nil {
			return exitcode.New(30, err)
		}
	} else {
		if err := p.Table([]string{"Mode", "Status", "RecommendedNextStep"}, [][]string{{report.Mode, report.Status, report.RecommendedNextStep}}); err != nil {
			return exitcode.New(30, err)
		}
		rows := make([][]string, 0, len(report.Providers))
		for _, pr := range report.Providers {
			rows = append(rows, []string{
				pr.ProviderID,
				pr.Status,
				pr.Action,
				pr.FilePath,
				pr.BackupFile,
			})
		}
		if len(rows) == 0 {
			rows = append(rows, []string{"-", "pass", "none", "-", "-"})
		}
		if err := p.Table([]string{"Provider", "Status", "Action", "FilePath", "BackupFile"}, rows); err != nil {
			return exitcode.New(30, err)
		}
	}

	if report.Status == "fail" {
		p.PrintError(&output.GateError{Status: 422, Label: "MIGRATE_FAILED", Message: "migrate completed with failures"})
		return exitcode.New(migration.MigrateExitCode(report), errors.New("migrate report failed"))
	}
	if report.Status == "warn" {
		return exitcode.New(migration.MigrateExitCode(report), nil)
	}
	return nil
}

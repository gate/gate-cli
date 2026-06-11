package migrate

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/cmdhint"
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
	if p.IsTable() {
		p.PrintError(output.UnsupportedTableFormatError())
		return exitcode.New(exitcode.RenderOrInternal, errors.New("unsupported format"))
	}
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	apply, _ := cmd.Flags().GetBool("apply")
	yes, _ := cmd.Flags().GetBool("yes")
	providerRaw, _ := cmd.Flags().GetString("provider")
	backupDir, _ := cmd.Flags().GetString("backup-dir")

	if err := migration.ValidateMode(apply, dryRun); err != nil {
		p.PrintError(output.InvalidArgsError(err.Error()))
		return exitcode.New(exitcode.RenderOrInternal, err)
	}
	if apply && !yes {
		p.PrintError(&output.GateError{Status: 400, Label: "CONFIRMATION_REQUIRED", Message: "use --yes with --apply to run non-interactive migration"})
		return exitcode.New(exitcode.RenderOrInternal, errors.New("confirmation required"))
	}

	report, err := migration.RunMigrate(migration.MigrateOptions{
		Apply:       apply,
		ProviderIDs: migration.ParseProviders(providerRaw),
		BackupDir:   backupDir,
	})
	if err != nil {
		ge := &output.GateError{Status: 500, Label: "MIGRATE_FAILED", Message: err.Error()}
		output.FillAgentErrorConvergence(ge)
		p.PrintError(ge)
		return exitcode.New(exitcode.RenderOrInternal, errors.New("migrate failed"))
	}

	payload := interface{}(report)
	if p.IsJSON() && cmdhint.AgentModeEnabled() {
		payload = map[string]interface{}{
			"mode":                  report.Mode,
			"status":                report.Status,
			"providers":             report.Providers,
			"recommended_next_step": report.RecommendedNextStep,
			"suggested_next_action": cmdhint.AgentMigrateNextAction(report.Status),
			"agent_resolve_hint":    cmdhint.AgentResolveHint("intel migrate"),
		}
	}

	if report.Status == "fail" {
		ge := &output.GateError{Status: 422, Label: "MIGRATE_FAILED", Message: migrateFailMessage(report)}
		output.FillAgentErrorConvergence(ge)
		if cmdhint.AgentModeEnabled() {
			ge.SuggestedNextAction = cmdhint.AgentMigrateNextAction(report.Status)
		} else if err := p.Print(payload); err != nil {
			return exitcode.New(exitcode.RenderOrInternal, err)
		}
		p.PrintError(ge)
		return exitcode.New(migration.MigrateExitCode(report), errors.New("migrate report failed"))
	}

	if err := p.Print(payload); err != nil {
		return exitcode.New(exitcode.RenderOrInternal, err)
	}
	if report.Status == "warn" {
		return exitcode.New(migration.MigrateExitCode(report), nil)
	}
	return nil
}

func migrateFailMessage(report migration.MigrateReport) string {
	for _, pr := range report.Providers {
		if strings.TrimSpace(pr.Status) != "fail" {
			continue
		}
		if msg := strings.TrimSpace(pr.ManualPatch); msg != "" {
			return "migrate failed: " + msg
		}
		if id := strings.TrimSpace(pr.ProviderID); id != "" {
			return "migrate failed: provider " + id
		}
	}
	return "migrate completed with failures"
}

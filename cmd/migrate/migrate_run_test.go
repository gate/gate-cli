package migrate

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/exitcode"
)

func newMigrateTestCmd(format string) *cobra.Command {
	root := &cobra.Command{Use: "gate-cli"}
	root.PersistentFlags().String("format", format, "Output format")
	root.PersistentFlags().String("profile", "default", "Profile")
	root.PersistentFlags().Bool("debug", false, "Debug")
	root.PersistentFlags().Int64("max-output-bytes", 0, "Limit")

	cmd := &cobra.Command{Use: "migrate"}
	cmd.SetContext(root.Context())
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Bool("apply", false, "")
	cmd.Flags().Bool("yes", false, "")
	cmd.Flags().String("provider", "", "")
	cmd.Flags().String("backup-dir", "", "")
	root.AddCommand(cmd)
	return cmd
}

func TestRunMigrateReturnsExitCodeOnInvalidMode(t *testing.T) {
	cmd := newMigrateTestCmd("json")
	_ = cmd.Flags().Set("dry-run", "true")
	_ = cmd.Flags().Set("apply", "true")

	err := runMigrate(cmd, nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	var coded *exitcode.Error
	if !errors.As(err, &coded) {
		t.Fatalf("expected exitcode.Error, got %T", err)
	}
	if coded.Code != 30 {
		t.Fatalf("expected code 30, got %d", coded.Code)
	}
}

func TestRunMigrateReturnsExitCodeWhenApplyWithoutYes(t *testing.T) {
	cmd := newMigrateTestCmd("json")
	_ = cmd.Flags().Set("apply", "true")

	err := runMigrate(cmd, nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	var coded *exitcode.Error
	if !errors.As(err, &coded) {
		t.Fatalf("expected exitcode.Error, got %T", err)
	}
	if coded.Code != 30 {
		t.Fatalf("expected code 30, got %d", coded.Code)
	}
}

package configcmd

import (
	"github.com/gate/gate-cli/internal/config"
	shared "github.com/gate/gate-cli/internal/configcmd"
	"github.com/spf13/cobra"
)

// Cmd 保留原有命令结构，读写与脱敏统一委托共享实现。
var Cmd = &cobra.Command{Use: "config", Short: "Manage gate-cli configuration"}
var initCmd = &cobra.Command{Use: "init", Short: "Interactively create or merge config", RunE: runInit}
var listCmd = &cobra.Command{Use: "list", Short: "List configuration with credentials masked", RunE: runList}
var setCmd = &cobra.Command{Use: "set <key> <value>", Short: "Set api-key, api-secret or base-url", Args: cobra.ExactArgs(2), RunE: runSet}

func init() {
	initCmd.Flags().Bool("hide-input", false, "Hide all fields on a real terminal, including profile; pipes keep existing input behavior")
	Cmd.AddCommand(&cobra.Command{Use: "check", Short: "Read-only local config file permissions/readability check (no content or auth validation)", Args: cobra.NoArgs, RunE: runCheck})
	setCmd.Flags().String("profile", "default", "Profile to update")
	listCmd.Flags().Bool("show-secrets", false, "Show credential values explicitly")
	Cmd.AddCommand(initCmd, listCmd, setCmd)
}

type profileEntry struct {
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
	BaseURL   string `yaml:"base_url,omitempty"`
}

type fileLayout struct {
	DefaultProfile string                  `yaml:"default_profile"`
	DefaultSettle  string                  `yaml:"default_settle"`
	Intel          config.IntelFile        `yaml:"intel,omitempty"`
	Profiles       map[string]profileEntry `yaml:"profiles"`
}

func runInit(cmd *cobra.Command, _ []string) error   { return shared.RunInit(cmd) }
func runSet(cmd *cobra.Command, args []string) error { return shared.RunSet(cmd, args) }
func runList(cmd *cobra.Command, _ []string) error   { return shared.RunList(cmd) }
func maskSecrets(content string) string              { return shared.MaskSecrets(content) }

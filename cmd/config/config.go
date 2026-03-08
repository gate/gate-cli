package configcmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/gate/gate-cli/internal/config"
)

// Cmd is the root config subcommand.
var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Manage gate-cli configuration",
}

func init() {
	setCmd.Flags().String("profile", "default", "Profile to update")
	Cmd.AddCommand(initCmd, listCmd, setCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactively create config file",
	RunE:  runInit,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured profiles",
	RunE:  runList,
}

var setCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value (api-key, api-secret, base-url)",
	Args:  cobra.ExactArgs(2),
	RunE:  runSet,
}

type profileEntry struct {
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
	BaseURL   string `yaml:"base_url,omitempty"`
}

type fileLayout struct {
	DefaultProfile string                  `yaml:"default_profile"`
	DefaultSettle  string                  `yaml:"default_settle"`
	Profiles       map[string]profileEntry `yaml:"profiles"`
}

func runInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Profile name [default]: ")
	profileName, _ := reader.ReadString('\n')
	profileName = strings.TrimSpace(profileName)
	if profileName == "" {
		profileName = "default"
	}

	fmt.Print("API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)

	fmt.Print("API Secret: ")
	apiSecret, _ := reader.ReadString('\n')
	apiSecret = strings.TrimSpace(apiSecret)

	cfgPath := config.DefaultConfigPath()
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	fc := fileLayout{
		DefaultProfile: profileName,
		DefaultSettle:  "usdt",
		Profiles: map[string]profileEntry{
			profileName: {APIKey: apiKey, APISecret: apiSecret},
		},
	}

	data, err := yaml.Marshal(fc)
	if err != nil {
		return err
	}
	if err := os.WriteFile(cfgPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	fmt.Printf("Config written to %s\n", cfgPath)
	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(config.DefaultConfigPath())
	if err != nil {
		return fmt.Errorf("no config file found — run: gate-cli config init")
	}
	fmt.Print(string(data))
	return nil
}

func runSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]
	profileName, _ := cmd.Flags().GetString("profile")

	cfgPath := config.DefaultConfigPath()
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("no config file found — run: gate-cli config init")
	}

	var fc fileLayout
	if err := yaml.Unmarshal(data, &fc); err != nil {
		return fmt.Errorf("invalid config file: %w", err)
	}
	if fc.Profiles == nil {
		fc.Profiles = make(map[string]profileEntry)
	}

	p := fc.Profiles[profileName]
	switch key {
	case "api-key":
		p.APIKey = value
	case "api-secret":
		p.APISecret = value
	case "base-url":
		p.BaseURL = value
	default:
		return fmt.Errorf("unknown key %q — valid keys: api-key, api-secret, base-url", key)
	}
	fc.Profiles[profileName] = p

	out, err := yaml.Marshal(fc)
	if err != nil {
		return err
	}
	if err := os.WriteFile(cfgPath, out, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	fmt.Printf("Set %s for profile %q\n", key, profileName)
	return nil
}

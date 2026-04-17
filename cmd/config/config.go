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
	listCmd.Flags().Bool("show-secrets", false, "Show api_secret in plain text (hidden by default)")
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
	Intel          config.IntelFile        `yaml:"intel,omitempty"`
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
		Intel: config.IntelFile{
			InfoMCPURL: config.DefaultIntelInfoMCPURL,
			NewsMCPURL: config.DefaultIntelNewsMCPURL,
		},
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

func maskSecrets(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "api_key:") || strings.HasPrefix(trimmed, "api_secret:") ||
			strings.HasPrefix(trimmed, "bearer_token:") || strings.HasPrefix(trimmed, "news_bearer_token:") ||
			strings.HasPrefix(trimmed, "info_bearer_token:") {
			indent := line[:len(line)-len(trimmed)]
			field := trimmed[:strings.Index(trimmed, ":")]
			lines[i] = indent + field + ": ****"
		}
	}
	return strings.Join(lines, "\n")
}

func runList(cmd *cobra.Command, args []string) error {
	showSecrets, _ := cmd.Flags().GetBool("show-secrets")
	data, err := os.ReadFile(config.DefaultConfigPath())
	if err != nil {
		return fmt.Errorf("no config file found — run: gate-cli config init")
	}
	output := string(data)
	if !showSecrets {
		output = maskSecrets(output)
	}
	fmt.Print(output)
	return nil
}

func runSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]

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

	// Resolve profile name: explicit --profile flag > default_profile in file > "default".
	profileName, _ := cmd.Flags().GetString("profile")
	if !cmd.Flags().Changed("profile") && fc.DefaultProfile != "" {
		profileName = fc.DefaultProfile
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

package subaccount

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var keyCmd = &cobra.Command{
	Use:   "key",
	Short: "Sub-account API key management",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all API keys of a sub-account",
		RunE:  runKeyList,
	}
	listCmd.Flags().Int32("user-id", 0, "Sub-account user ID (required)")
	listCmd.MarkFlagRequired("user-id")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new API key for a sub-account",
		RunE:  runKeyCreate,
	}
	createCmd.Flags().Int64("user-id", 0, "Sub-account user ID (required)")
	createCmd.MarkFlagRequired("user-id")
	createCmd.Flags().String("name", "", "API key name")
	createCmd.Flags().StringSlice("perms", nil, "Permissions, e.g. spot,futures,wallet")
	createCmd.Flags().Int32("mode", 1, "Mode: 1=classic, 2=portfolio")
	createCmd.Flags().StringSlice("ip-whitelist", nil, "IP whitelist")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get a specific API key of a sub-account",
		RunE:  runKeyGet,
	}
	getCmd.Flags().Int32("user-id", 0, "Sub-account user ID (required)")
	getCmd.MarkFlagRequired("user-id")
	getCmd.Flags().String("api-key", "", "API key (required)")
	getCmd.MarkFlagRequired("api-key")

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a sub-account API key",
		RunE:  runKeyUpdate,
	}
	updateCmd.Flags().Int32("user-id", 0, "Sub-account user ID (required)")
	updateCmd.MarkFlagRequired("user-id")
	updateCmd.Flags().String("api-key", "", "API key (required)")
	updateCmd.MarkFlagRequired("api-key")
	updateCmd.Flags().String("name", "", "API key name")
	updateCmd.Flags().StringSlice("perms", nil, "Permissions, e.g. spot,futures,wallet")
	updateCmd.Flags().StringSlice("ip-whitelist", nil, "IP whitelist")

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a sub-account API key",
		RunE:  runKeyDelete,
	}
	deleteCmd.Flags().Int32("user-id", 0, "Sub-account user ID (required)")
	deleteCmd.MarkFlagRequired("user-id")
	deleteCmd.Flags().String("api-key", "", "API key (required)")
	deleteCmd.MarkFlagRequired("api-key")

	keyCmd.AddCommand(listCmd, createCmd, getCmd, updateCmd, deleteCmd)
	Cmd.AddCommand(keyCmd)
}

func runKeyList(cmd *cobra.Command, args []string) error {
	userID, _ := cmd.Flags().GetInt32("user-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.SubAccountAPI.ListSubAccountKeys(c.Context(), userID)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", fmt.Sprintf("/api/v4/sub_accounts/%d/keys", userID), ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, k := range result {
		rows[i] = []string{
			k.Key,
			k.Name,
			fmt.Sprintf("%d", k.State),
			fmt.Sprintf("%d", k.Mode),
			formatPerms(k.Perms),
		}
	}
	return p.Table([]string{"API Key", "Name", "State", "Mode", "Perms"}, rows)
}

func runKeyCreate(cmd *cobra.Command, args []string) error {
	userID, _ := cmd.Flags().GetInt64("user-id")
	name, _ := cmd.Flags().GetString("name")
	perms, _ := cmd.Flags().GetStringSlice("perms")
	mode, _ := cmd.Flags().GetInt32("mode")
	ipWhitelist, _ := cmd.Flags().GetStringSlice("ip-whitelist")

	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.SubAccountKey{
		Name: name,
		Mode: mode,
	}
	if len(perms) > 0 {
		body.Perms = buildPerms(perms)
	}
	if len(ipWhitelist) > 0 {
		body.IpWhitelist = ipWhitelist
	}

	result, httpResp, err := c.SubAccountAPI.CreateSubAccountKeys(c.Context(), userID, body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", fmt.Sprintf("/api/v4/sub_accounts/%d/keys", userID), ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"API Key", "Name", "State", "Mode", "Perms"},
		[][]string{{
			result.Key,
			result.Name,
			fmt.Sprintf("%d", result.State),
			fmt.Sprintf("%d", result.Mode),
			formatPerms(result.Perms),
		}},
	)
}

func runKeyGet(cmd *cobra.Command, args []string) error {
	userID, _ := cmd.Flags().GetInt32("user-id")
	apiKey, _ := cmd.Flags().GetString("api-key")

	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.SubAccountAPI.GetSubAccountKey(c.Context(), userID, apiKey)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", fmt.Sprintf("/api/v4/sub_accounts/%d/keys/%s", userID, apiKey), ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"API Key", "Name", "State", "Mode", "Perms"},
		[][]string{{
			result.Key,
			result.Name,
			fmt.Sprintf("%d", result.State),
			fmt.Sprintf("%d", result.Mode),
			formatPerms(result.Perms),
		}},
	)
}

func runKeyUpdate(cmd *cobra.Command, args []string) error {
	userID, _ := cmd.Flags().GetInt32("user-id")
	apiKey, _ := cmd.Flags().GetString("api-key")
	name, _ := cmd.Flags().GetString("name")
	perms, _ := cmd.Flags().GetStringSlice("perms")
	ipWhitelist, _ := cmd.Flags().GetStringSlice("ip-whitelist")

	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.SubAccountKey{}
	if name != "" {
		body.Name = name
	}
	if len(perms) > 0 {
		body.Perms = buildPerms(perms)
	}
	if len(ipWhitelist) > 0 {
		body.IpWhitelist = ipWhitelist
	}

	httpResp, err := c.SubAccountAPI.UpdateSubAccountKeys(c.Context(), userID, apiKey, body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "PUT", fmt.Sprintf("/api/v4/sub_accounts/%d/keys/%s", userID, apiKey), ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(map[string]interface{}{"user_id": userID, "api_key": apiKey, "action": "updated"})
	}
	return p.Table([]string{"User ID", "API Key", "Action"}, [][]string{{fmt.Sprintf("%d", userID), apiKey, "updated"}})
}

func runKeyDelete(cmd *cobra.Command, args []string) error {
	userID, _ := cmd.Flags().GetInt32("user-id")
	apiKey, _ := cmd.Flags().GetString("api-key")

	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	httpResp, err := c.SubAccountAPI.DeleteSubAccountKeys(c.Context(), userID, apiKey)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", fmt.Sprintf("/api/v4/sub_accounts/%d/keys/%s", userID, apiKey), ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(map[string]interface{}{"user_id": userID, "api_key": apiKey, "action": "deleted"})
	}
	return p.Table([]string{"User ID", "API Key", "Action"}, [][]string{{fmt.Sprintf("%d", userID), apiKey, "deleted"}})
}

// buildPerms converts a string slice like ["spot", "futures"] to SubAccountKeyPerms.
func buildPerms(names []string) []gateapi.SubAccountKeyPerms {
	perms := make([]gateapi.SubAccountKeyPerms, len(names))
	for i, n := range names {
		perms[i] = gateapi.SubAccountKeyPerms{Name: strings.TrimSpace(n)}
	}
	return perms
}

// formatPerms joins permission names for table display.
func formatPerms(perms []gateapi.SubAccountKeyPerms) string {
	names := make([]string, len(perms))
	for i, p := range perms {
		names[i] = p.Name
	}
	return strings.Join(names, ",")
}

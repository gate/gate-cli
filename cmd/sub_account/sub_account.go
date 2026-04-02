package subaccount

import (
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// Cmd is the root command for the sub-account module.
var Cmd = &cobra.Command{
	Use:   "sub-account",
	Short: "Sub-account management commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List sub-accounts",
		RunE:  runList,
	}
	listCmd.Flags().String("type", "", "Sub-account type: 0=all, 1=regular (default: regular only)")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new sub-account",
		RunE:  runCreate,
	}
	createCmd.Flags().String("login-name", "", "Login name (required)")
	createCmd.MarkFlagRequired("login-name")
	createCmd.Flags().String("password", "", "Password (default: same as main account)")
	createCmd.Flags().String("email", "", "Email (default: same as main account)")
	createCmd.Flags().String("remark", "", "Remark")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get sub-account details",
		RunE:  runGet,
	}
	getCmd.Flags().Int64("user-id", 0, "Sub-account user ID (required)")
	getCmd.MarkFlagRequired("user-id")

	lockCmd := &cobra.Command{
		Use:   "lock",
		Short: "Lock a sub-account",
		RunE:  runLock,
	}
	lockCmd.Flags().Int64("user-id", 0, "Sub-account user ID (required)")
	lockCmd.MarkFlagRequired("user-id")

	unlockCmd := &cobra.Command{
		Use:   "unlock",
		Short: "Unlock a sub-account",
		RunE:  runUnlock,
	}
	unlockCmd.Flags().Int64("user-id", 0, "Sub-account user ID (required)")
	unlockCmd.MarkFlagRequired("user-id")

	unifiedModeCmd := &cobra.Command{
		Use:   "unified-mode",
		Short: "List sub-account unified mode",
		RunE:  runUnifiedMode,
	}

	Cmd.AddCommand(listCmd, createCmd, getCmd, lockCmd, unlockCmd, unifiedModeCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var opts *gateapi.ListSubAccountsOpts
	if t, _ := cmd.Flags().GetString("type"); t != "" {
		opts = &gateapi.ListSubAccountsOpts{Type_: optional.NewString(t)}
	}

	result, httpResp, err := c.SubAccountAPI.ListSubAccounts(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/sub_accounts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, a := range result {
		rows[i] = []string{
			fmt.Sprintf("%d", a.UserId),
			a.LoginName,
			fmt.Sprintf("%d", a.State),
			fmt.Sprintf("%d", a.Type),
			a.Remark,
		}
	}
	return p.Table([]string{"User ID", "Login Name", "State", "Type", "Remark"}, rows)
}

func runCreate(cmd *cobra.Command, args []string) error {
	loginName, _ := cmd.Flags().GetString("login-name")
	password, _ := cmd.Flags().GetString("password")
	email, _ := cmd.Flags().GetString("email")
	remark, _ := cmd.Flags().GetString("remark")

	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.SubAccount{
		LoginName: loginName,
	}
	if password != "" {
		body.Password = password
	}
	if email != "" {
		body.Email = email
	}
	if remark != "" {
		body.Remark = remark
	}

	result, httpResp, err := c.SubAccountAPI.CreateSubAccounts(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/sub_accounts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"User ID", "Login Name", "State", "Type", "Remark"},
		[][]string{{
			fmt.Sprintf("%d", result.UserId),
			result.LoginName,
			fmt.Sprintf("%d", result.State),
			fmt.Sprintf("%d", result.Type),
			result.Remark,
		}},
	)
}

func runGet(cmd *cobra.Command, args []string) error {
	userID, _ := cmd.Flags().GetInt64("user-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.SubAccountAPI.GetSubAccount(c.Context(), userID)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", fmt.Sprintf("/api/v4/sub_accounts/%d", userID), ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"User ID", "Login Name", "State", "Type", "Email", "Remark"},
		[][]string{{
			fmt.Sprintf("%d", result.UserId),
			result.LoginName,
			fmt.Sprintf("%d", result.State),
			fmt.Sprintf("%d", result.Type),
			result.Email,
			result.Remark,
		}},
	)
}

func runLock(cmd *cobra.Command, args []string) error {
	userID, _ := cmd.Flags().GetInt64("user-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	httpResp, err := c.SubAccountAPI.LockSubAccount(c.Context(), userID)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", fmt.Sprintf("/api/v4/sub_accounts/%d/lock", userID), ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(map[string]interface{}{"user_id": userID, "action": "locked"})
	}
	return p.Table([]string{"User ID", "Action"}, [][]string{{fmt.Sprintf("%d", userID), "locked"}})
}

func runUnlock(cmd *cobra.Command, args []string) error {
	userID, _ := cmd.Flags().GetInt64("user-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	httpResp, err := c.SubAccountAPI.UnlockSubAccount(c.Context(), userID)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", fmt.Sprintf("/api/v4/sub_accounts/%d/unlock", userID), ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(map[string]interface{}{"user_id": userID, "action": "unlocked"})
	}
	return p.Table([]string{"User ID", "Action"}, [][]string{{fmt.Sprintf("%d", userID), "unlocked"}})
}

func runUnifiedMode(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.SubAccountAPI.ListUnifiedMode(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/sub_accounts/unified_mode", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, m := range result {
		rows[i] = []string{
			fmt.Sprintf("%d", m.UserId),
			fmt.Sprintf("%v", m.IsUnified),
			m.Mode,
		}
	}
	return p.Table([]string{"User ID", "Is Unified", "Mode"}, rows)
}

package rebate

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

// Cmd is the root command for the rebate module.
var Cmd = &cobra.Command{
	Use:   "rebate",
	Short: "Rebate & commission commands",
}

func init() {
	userInfoCmd := &cobra.Command{
		Use:   "user-info",
		Short: "Get rebate user info",
		RunE:  runRebateUserInfo,
	}

	subRelationCmd := &cobra.Command{
		Use:   "sub-relation",
		Short: "Query user subordinate relation",
		RunE:  runSubRelation,
	}
	subRelationCmd.Flags().String("user-id-list", "", "Comma-separated user ID list (required)")
	subRelationCmd.MarkFlagRequired("user-id-list")

	Cmd.AddCommand(userInfoCmd, subRelationCmd)
}

func runRebateUserInfo(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.RebateAPI.RebateUserInfo(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rebate/user_info", ""))
		return nil
	}
	return p.Print(result)
}

func runSubRelation(cmd *cobra.Command, args []string) error {
	userIDList, _ := cmd.Flags().GetString("user-id-list")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.RebateAPI.UserSubRelation(c.Context(), userIDList)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rebate/user/sub_relation", ""))
		return nil
	}
	return p.Print(result)
}

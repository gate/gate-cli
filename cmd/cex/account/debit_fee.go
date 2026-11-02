package account

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var debitFeeCmd = &cobra.Command{
	Use:   "debit-fee",
	Short: "Debit fee settings",
}

func init() {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get debit fee setting",
		RunE:  runDebitFeeGet,
	}

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Enable or disable debit fee",
		RunE:  runDebitFeeSet,
	}
	setCmd.Flags().String("enabled", "", "Enable or disable debit fee: true or false (required)")
	setCmd.MarkFlagRequired("enabled")

	debitFeeCmd.AddCommand(getCmd, setCmd)
	Cmd.AddCommand(debitFeeCmd)
}

func runDebitFeeGet(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.AccountAPI.GetDebitFee(c.Context())
	if err != nil {
		// 服务端拒绝必须传播为命令失败，避免Agentic鉴权错误被自动化当作执行成功。
		return client.CommandError(err, httpResp, "GET", "/api/v4/account/debit_fee", "")
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	enabled := "false"
	if result.Enabled {
		enabled = "true"
	}
	return p.Table([]string{"Enabled"}, [][]string{{enabled}})
}

func runDebitFeeSet(cmd *cobra.Command, args []string) error {
	enabledStr, _ := cmd.Flags().GetString("enabled")
	p := cmdutil.GetPrinter(cmd)

	var enabled bool
	switch enabledStr {
	case "true":
		enabled = true
	case "false":
		enabled = false
	default:
		return fmt.Errorf("--enabled must be true or false, got %q", enabledStr)
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.DebitFee{Enabled: enabled}
	body, _ := json.Marshal(req)
	httpResp, err := c.AccountAPI.SetDebitFee(c.Context(), req)
	if err != nil {
		return client.CommandError(err, httpResp, "POST", "/api/v4/account/debit_fee", string(body))
	}
	return p.Table([]string{"Enabled"}, [][]string{{enabledStr}})
}

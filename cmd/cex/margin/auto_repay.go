package margin

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

var autoRepayCmd = &cobra.Command{
	Use:   "auto-repay",
	Short: "Margin auto-repay settings",
}

func init() {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get current auto-repay status",
		RunE:  runAutoRepayGet,
	}

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Enable or disable auto-repay",
		RunE:  runAutoRepaySet,
	}
	setCmd.Flags().String("status", "", "Auto-repay status: on or off (required)")
	setCmd.MarkFlagRequired("status")

	autoRepayCmd.AddCommand(getCmd, setCmd)
	Cmd.AddCommand(autoRepayCmd)
}

func runAutoRepayGet(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.MarginAPI.GetAutoRepayStatus(c.Context())
	if err != nil {
		// 服务端拒绝必须传播为命令失败，避免Agentic鉴权错误被自动化当作执行成功。
		return client.CommandError(err, httpResp, "GET", "/api/v4/margin/auto_repay", "")
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Status"},
		[][]string{{result.Status}},
	)
}

func runAutoRepaySet(cmd *cobra.Command, args []string) error {
	status, _ := cmd.Flags().GetString("status")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.MarginAPI.SetAutoRepay(c.Context(), status)
	if err != nil {
		return client.CommandError(err, httpResp, "POST", "/api/v4/margin/auto_repay", status)
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Status"},
		[][]string{{result.Status}},
	)
}

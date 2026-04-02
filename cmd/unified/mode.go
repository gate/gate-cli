package unified

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var modeCmd = &cobra.Command{
	Use:   "mode",
	Short: "Unified account mode commands",
}

func init() {
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get current unified account mode",
		RunE:  runModeGet,
	}

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set unified account mode",
		RunE:  runModeSet,
	}
	setCmd.Flags().String("mode", "", "Account mode: classic, multi_currency, portfolio, single_currency (required)")
	setCmd.MarkFlagRequired("mode")

	modeCmd.AddCommand(getCmd, setCmd)
	Cmd.AddCommand(modeCmd)
}

func runModeGet(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.UnifiedAPI.GetUnifiedMode(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/unified/unified_mode", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Mode", "USDT Futures", "Spot Hedge", "Use Funding", "Options"},
		[][]string{{
			result.Mode,
			fmt.Sprintf("%v", result.Settings.UsdtFutures),
			fmt.Sprintf("%v", result.Settings.SpotHedge),
			fmt.Sprintf("%v", result.Settings.UseFunding),
			fmt.Sprintf("%v", result.Settings.Options),
		}},
	)
}

func runModeSet(cmd *cobra.Command, args []string) error {
	mode, _ := cmd.Flags().GetString("mode")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.UnifiedModeSet{
		Mode: mode,
	}

	httpResp, err := c.UnifiedAPI.SetUnifiedMode(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "PUT", "/api/v4/unified/unified_mode", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(map[string]string{"status": "ok", "mode": mode})
	}
	return p.Table(
		[]string{"Status", "Mode"},
		[][]string{{"ok", mode}},
	)
}

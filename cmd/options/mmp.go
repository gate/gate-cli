package options

import (
	"encoding/json"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

var mmpCmd = &cobra.Command{
	Use:   "mmp",
	Short: "Market maker protection commands",
}

func init() {
	getMmpCmd := &cobra.Command{
		Use:   "get",
		Short: "Query MMP settings",
		RunE:  runOptionsMmpGet,
	}
	getMmpCmd.Flags().String("underlying", "", "Filter by underlying")

	setMmpCmd := &cobra.Command{
		Use:   "set",
		Short: "Set MMP configuration",
		RunE:  runOptionsMmpSet,
	}
	setMmpCmd.Flags().String("underlying", "", "Underlying name (required)")
	setMmpCmd.Flags().Int32("window", 0, "Time window in milliseconds (required)")
	setMmpCmd.Flags().Int32("frozen-period", 0, "Freeze duration in milliseconds (required)")
	setMmpCmd.Flags().String("qty-limit", "", "Trading volume upper limit (required)")
	setMmpCmd.Flags().String("delta-limit", "", "Net delta upper limit (required)")
	setMmpCmd.MarkFlagRequired("underlying")
	setMmpCmd.MarkFlagRequired("window")
	setMmpCmd.MarkFlagRequired("frozen-period")
	setMmpCmd.MarkFlagRequired("qty-limit")
	setMmpCmd.MarkFlagRequired("delta-limit")

	resetMmpCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset MMP (unfreeze)",
		RunE:  runOptionsMmpReset,
	}
	resetMmpCmd.Flags().String("underlying", "", "Underlying name (required)")
	resetMmpCmd.MarkFlagRequired("underlying")

	mmpCmd.AddCommand(getMmpCmd, setMmpCmd, resetMmpCmd)
	Cmd.AddCommand(mmpCmd)
}

func runOptionsMmpGet(cmd *cobra.Command, args []string) error {
	underlying, _ := cmd.Flags().GetString("underlying")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetOptionsMMPOpts{}
	if underlying != "" {
		opts.Underlying = optional.NewString(underlying)
	}

	result, httpResp, err := c.OptionsAPI.GetOptionsMMP(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/options/mmp", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, m := range result {
		rows[i] = []string{m.Underlying, m.QtyLimit, m.DeltaLimit}
	}
	return p.Table([]string{"Underlying", "Qty Limit", "Delta Limit"}, rows)
}

func runOptionsMmpSet(cmd *cobra.Command, args []string) error {
	underlying, _ := cmd.Flags().GetString("underlying")
	window, _ := cmd.Flags().GetInt32("window")
	frozenPeriod, _ := cmd.Flags().GetInt32("frozen-period")
	qtyLimit, _ := cmd.Flags().GetString("qty-limit")
	deltaLimit, _ := cmd.Flags().GetString("delta-limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.OptionsMmp{
		Underlying:   underlying,
		Window:       window,
		FrozenPeriod: frozenPeriod,
		QtyLimit:     qtyLimit,
		DeltaLimit:   deltaLimit,
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.OptionsAPI.SetOptionsMMP(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/options/mmp", string(body)))
		return nil
	}
	return p.Print(result)
}

func runOptionsMmpReset(cmd *cobra.Command, args []string) error {
	underlying, _ := cmd.Flags().GetString("underlying")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.OptionsMmpReset{Underlying: underlying}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.OptionsAPI.ResetOptionsMMP(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/options/mmp/reset", string(body)))
		return nil
	}
	return p.Print(result)
}

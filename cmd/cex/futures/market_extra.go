package futures

import (
	"fmt"

	"github.com/antihax/optional"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/spf13/cobra"
)

// SDK-backed commands share the existing transport and authentication boundary.
func init() {
	futuresContractsAllCmd := &cobra.Command{Use: "contracts-all", Short: "List contracts including delisted contracts", Args: cobra.NoArgs, RunE: runFuturesContractsAll}
	futuresContractsAllCmd.Flags().Int32("limit", 100, "Records per page, 1 to 1000")
	futuresContractsAllCmd.Flags().Int32("offset", 0, "Record offset, 0 to 100000")
	addSettleFlag(futuresContractsAllCmd)
	marketCmd.AddCommand(futuresContractsAllCmd)
	futuresRPIOrderBookCmd := &cobra.Command{Use: "rpi-orderbook", Short: "Get the futures RPI order book", Args: cobra.NoArgs, RunE: runFuturesRPIOrderBook}
	futuresRPIOrderBookCmd.Flags().String("contract", "", "Contract name, e.g. BTC_USDT")
	cmdutil.MustRequireFlags(futuresRPIOrderBookCmd, "contract")
	futuresRPIOrderBookCmd.Flags().String("interval", "", "Price aggregation interval")
	futuresRPIOrderBookCmd.Flags().Int32("limit", 20, "Depth limit, 1 to 50")
	futuresRPIOrderBookCmd.Flags().Bool("with-id", false, "Include order book ID")
	addSettleFlag(futuresRPIOrderBookCmd)
	marketCmd.AddCommand(futuresRPIOrderBookCmd)
	futuresADLRiskStatesCmd := &cobra.Command{Use: "adl-risk-states", Short: "Get futures ADL risk states", Args: cobra.NoArgs, RunE: runFuturesADLRiskStates}
	addSettleFlag(futuresADLRiskStatesCmd)
	marketCmd.AddCommand(futuresADLRiskStatesCmd)
}

func runFuturesContractsAll(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt32("limit")
	offset, _ := cmd.Flags().GetInt32("offset")
	settle := cmdutil.GetSettle(cmd)
	if err := cmdutil.TradingToken("--settle", settle); err != nil {
		return err
	}
	if limit < 1 || limit > 1000 {
		return fmt.Errorf("--limit must be between 1 and 1000")
	}
	if offset < 0 || offset > 100000 {
		return fmt.Errorf("--offset must be between 0 and 100000")
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	result, resp, err := c.FuturesAPI.ListFuturesContractsAll(c.Context(), settle, &gateapi.ListFuturesContractsAllOpts{Limit: optional.NewInt32(limit), Offset: optional.NewInt32(offset)})
	if err != nil {
		return client.CommandError(err, resp, "GET", "/api/v4/futures/"+settle+"/contracts_all", "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runFuturesRPIOrderBook(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	interval, _ := cmd.Flags().GetString("interval")
	limit, _ := cmd.Flags().GetInt32("limit")
	withId, _ := cmd.Flags().GetBool("with-id")
	settle := cmdutil.GetSettle(cmd)
	if err := cmdutil.TradingToken("--settle", settle); err != nil {
		return err
	}
	if err := cmdutil.TradingToken("--contract", contract); err != nil {
		return err
	}
	if limit < 1 || limit > 50 {
		return fmt.Errorf("--limit must be between 1 and 50")
	}
	opts := &gateapi.ListFuturesRPIOrderBookOpts{Limit: optional.NewInt32(limit)}
	if interval != "" {
		opts.Interval = optional.NewString(interval)
	}
	if cmd.Flags().Changed("with-id") {
		opts.WithId = optional.NewBool(withId)
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	result, resp, err := c.FuturesAPI.ListFuturesRPIOrderBook(c.Context(), settle, contract, opts)
	if err != nil {
		return client.CommandError(err, resp, "GET", "/api/v4/futures/"+settle+"/rpi_order_book", "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runFuturesADLRiskStates(cmd *cobra.Command, args []string) error {
	settle := cmdutil.GetSettle(cmd)
	if err := cmdutil.TradingToken("--settle", settle); err != nil {
		return err
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	result, resp, err := c.FuturesAPI.ListFuturesADLRiskStates(c.Context(), settle)
	if err != nil {
		return client.CommandError(err, resp, "GET", "/api/v4/futures/"+settle+"/adl_risk_states", "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

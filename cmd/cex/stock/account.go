package stock

import (
	"github.com/antihax/optional"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/spf13/cobra"
)

func init() {
	group := &cobra.Command{Use: "account", Short: "Stock account assets and transfers", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() }}
	assets := &cobra.Command{Use: "assets", Short: "Query stock account assets", RunE: runAssets}
	addPnL(assets)
	transactions := &cobra.Command{Use: "transactions", Short: "Query stock transaction records", RunE: runTransactions}
	transactions.Flags().Int64("begin", 0, "Begin Unix seconds")
	transactions.Flags().Int64("end", 0, "End Unix seconds")
	transactions.Flags().String("ref-id", "", "Query an exact business reference")
	transactions.Flags().String("type", "", "Transaction type")
	addPagination(transactions)
	transfer := &cobra.Command{Use: "transfer", Short: "Transfer once using a caller-supplied idempotency reference", RunE: runTransfer}
	transfer.Flags().String("asset", "USDT", "Asset: USDT")
	for _, key := range []string{"change", "type", "ref-id"} {
		transfer.Flags().String(key, "", key+" (required)")
		if err := transfer.MarkFlagRequired(key); err != nil {
			panic(err)
		} // 该flag必须先声明，失败是注册错误。
	}
	group.AddCommand(assets, transactions, transfer)
	Cmd.AddCommand(group)
}
func runAssets(cmd *cobra.Command, _ []string) error {
	cost, price, err := pnl(cmd)
	if err != nil {
		return err
	}
	c, err := sdkClient(cmd, true)
	if err != nil {
		return err
	}
	result, response, err := c.StockAPI.QueryStockUserAssets(c.Context(), &gateapi.QueryStockUserAssetsOpts{PnlCalcType: cost, PnlCalcPrice: price})
	return finish(cmd, result, response, err, "GET", "/stock/users/assets")
}
func runTransactions(cmd *cobra.Command, _ []string) error {
	ref := flagString(cmd, "ref-id")
	if err := text(ref, "ref-id", false); err != nil {
		return err
	}
	begin, _ := cmd.Flags().GetInt64("begin")
	end, _ := cmd.Flags().GetInt64("end")
	if err := validateTimes(begin, end); err != nil {
		return err
	}
	if err := pagination(cmd); err != nil {
		return err
	}
	opts := &gateapi.QueryStockTransactionsOpts{}
	if ref != "" {
		// SDK声明ref_id查询忽略其他参数；显式冲突在本地拒绝，防止用户以为过滤条件生效。
		for _, key := range []string{"begin", "end", "type", "page", "page-size"} {
			if cmd.Flags().Changed(key) {
				return invalid("ref-id cannot be combined with transaction filters")
			}
		}
		opts.RefId = optional.NewString(ref)
	} else {
		opts.Page = optional.NewInt32(flagInt32(cmd, "page"))
		opts.PageSize = optional.NewInt32(flagInt32(cmd, "page-size"))
		opts.Type_ = optionalString(cmd, "type")
		if cmd.Flags().Changed("begin") {
			opts.BeginTime = optional.NewInt64(begin)
		}
		if cmd.Flags().Changed("end") {
			opts.EndTime = optional.NewInt64(end)
		}
	}
	c, err := sdkClient(cmd, true)
	if err != nil {
		return err
	}
	result, response, err := c.StockAPI.QueryStockTransactions(c.Context(), opts)
	return finish(cmd, result, response, err, "GET", "/stock/transactions")
}

// runTransfer 业务幂等标识由调用方提供；失败不生成新ID、不自动补发。
func runTransfer(cmd *cobra.Command, _ []string) error {
	amount := flagString(cmd, "change")
	if err := positiveDecimal(amount); err != nil {
		return err
	}
	kind := flagString(cmd, "type")
	if kind != "deposit" && kind != "withdraw" {
		return invalid("type must be deposit or withdraw")
	}
	if flagString(cmd, "asset") != "USDT" {
		return invalid("asset must be USDT")
	}
	ref := flagString(cmd, "ref-id")
	if err := text(ref, "ref-id", true); err != nil {
		return err
	}
	c, err := sdkClient(cmd, true)
	if err != nil {
		return err
	}
	result, response, err := c.StockAPI.CreateStockTransaction(c.Context(), gateapi.TradFiSpotTransactionRequest{Asset: "USDT", Change: amount, Type: kind, RefId: ref})
	return finish(cmd, result, response, err, "POST", "/stock/transactions")
}

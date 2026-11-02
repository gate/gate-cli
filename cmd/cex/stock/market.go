package stock

import (
	"github.com/antihax/optional"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/spf13/cobra"
)

func init() {
	market := &cobra.Command{Use: "market", Short: "Public stock market data", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() }}
	market.AddCommand(&cobra.Command{Use: "exchanges", Short: "List exchanges", RunE: runExchanges}, &cobra.Command{Use: "fee-rate", Short: "Query fee rates", RunE: runFeeRate})
	book := &cobra.Command{Use: "orderbook", Short: "Query symbol order book", RunE: runOrderbook}
	book.Flags().String("symbol", "", "Stock symbol (required)")
	if err := book.MarkFlagRequired("symbol"); err != nil {
		panic(err)
	} // 该flag必须先声明，失败是注册错误。
	market.AddCommand(book)
	for _, name := range []string{"symbols", "symbol"} {
		command := &cobra.Command{Use: name, Short: "Query stock symbols", RunE: runSymbols}
		command.Flags().String("symbols", "", "Comma-separated symbols")
		command.Flags().String("exchange", "", "Exchange: us,hk,kr,jp")
		addPagination(command)
		if name == "symbols" {
			command.Flags().Bool("with-desc-i18n", false, "Include translated descriptions")
		} else {
			if err := command.MarkFlagRequired("symbols"); err != nil {
				panic(err)
			} // 该flag必须先声明，失败是注册错误。
		}
		market.AddCommand(command)
	}
	Cmd.AddCommand(market)
}
func runExchanges(cmd *cobra.Command, _ []string) error {
	c, err := sdkClient(cmd, false)
	if err != nil {
		return err
	}
	result, response, err := c.StockAPI.QueryStockExchanges(c.Context())
	return finish(cmd, result, response, err, "GET", "/stock/exchanges")
}
func runFeeRate(cmd *cobra.Command, _ []string) error {
	c, err := sdkClient(cmd, false)
	if err != nil {
		return err
	}
	result, response, err := c.StockAPI.QueryStockFeeRate(c.Context())
	return finish(cmd, result, response, err, "GET", "/stock/fee-rate")
}
func runOrderbook(cmd *cobra.Command, _ []string) error {
	value := flagString(cmd, "symbol")
	if err := symbol(value, true); err != nil {
		return err
	}
	c, err := sdkClient(cmd, false)
	if err != nil {
		return err
	}
	result, response, err := c.StockAPI.QueryStockOrderBook(c.Context(), value)
	return finish(cmd, result, response, err, "GET", "/stock/market/"+value+"/orderbook")
}

// runSymbols 两个叶子共享参数校验，分别调用公共SDK明确的列表/详情方法。
func runSymbols(cmd *cobra.Command, _ []string) error {
	if err := pagination(cmd); err != nil {
		return err
	}
	if err := exchange(cmd); err != nil {
		return err
	}
	if err := text(flagString(cmd, "symbols"), "symbols", cmd.Name() == "symbol"); err != nil {
		return err
	}
	c, err := sdkClient(cmd, false)
	if err != nil {
		return err
	}
	if cmd.Name() == "symbol" {
		opts := &gateapi.QueryStockSymbolDetailOpts{Symbols: optionalString(cmd, "symbols"), Exchange: optionalString(cmd, "exchange"), Page: optional.NewInt32(flagInt32(cmd, "page")), PageSize: optional.NewInt32(flagInt32(cmd, "page-size"))}
		result, response, err := c.StockAPI.QueryStockSymbolDetail(c.Context(), opts)
		return finish(cmd, result, response, err, "GET", "/stock/symbols/detail")
	}
	opts := &gateapi.QueryStockSymbolsOpts{Symbols: optionalString(cmd, "symbols"), Exchange: optionalString(cmd, "exchange"), Page: optional.NewInt32(flagInt32(cmd, "page")), PageSize: optional.NewInt32(flagInt32(cmd, "page-size"))}
	if cmd.Flags().Changed("with-desc-i18n") {
		value, _ := cmd.Flags().GetBool("with-desc-i18n")
		opts.WithDescI18n = optional.NewBool(value)
	}
	result, response, err := c.StockAPI.QueryStockSymbols(c.Context(), opts)
	return finish(cmd, result, response, err, "GET", "/stock/symbols")
}

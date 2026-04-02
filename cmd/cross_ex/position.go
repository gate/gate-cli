package crossex

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var positionCmd = &cobra.Command{
	Use:   "position",
	Short: "Cross-exchange position commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Query contract positions",
		RunE:  runPositionList,
	}
	listCmd.Flags().String("symbol", "", "Filter by symbol")
	listCmd.Flags().String("exchange-type", "", "Filter by exchange")

	marginListCmd := &cobra.Command{
		Use:   "margin-list",
		Short: "Query leveraged positions",
		RunE:  runMarginPositionList,
	}
	marginListCmd.Flags().String("symbol", "", "Filter by symbol")
	marginListCmd.Flags().String("exchange-type", "", "Filter by exchange")

	closeCmd := &cobra.Command{
		Use:   "close",
		Short: "Full close position",
		RunE:  runPositionClose,
	}
	closeCmd.Flags().String("json", "", "JSON body for close position request (required)")
	closeCmd.MarkFlagRequired("json")

	leverageCmd := &cobra.Command{
		Use:   "leverage",
		Short: "Query contract leverage multiplier",
		RunE:  runPositionLeverage,
	}
	leverageCmd.Flags().String("symbols", "", "Symbol list, comma-separated")

	setLeverageCmd := &cobra.Command{
		Use:   "set-leverage",
		Short: "Modify contract leverage multiplier",
		RunE:  runPositionSetLeverage,
	}
	setLeverageCmd.Flags().String("json", "", "JSON body for leverage request (required)")
	setLeverageCmd.MarkFlagRequired("json")

	marginLeverageCmd := &cobra.Command{
		Use:   "margin-leverage",
		Short: "Query margin leverage multiplier",
		RunE:  runMarginPositionLeverage,
	}
	marginLeverageCmd.Flags().String("symbols", "", "Symbol list, comma-separated")

	setMarginLeverageCmd := &cobra.Command{
		Use:   "set-margin-leverage",
		Short: "Modify margin leverage multiplier",
		RunE:  runMarginPositionSetLeverage,
	}
	setMarginLeverageCmd.Flags().String("json", "", "JSON body for leverage request (required)")
	setMarginLeverageCmd.MarkFlagRequired("json")

	adlRankCmd := &cobra.Command{
		Use:   "adl-rank",
		Short: "Query ADL rank for a symbol",
		RunE:  runAdlRank,
	}
	adlRankCmd.Flags().String("symbol", "", "Symbol (required)")
	adlRankCmd.MarkFlagRequired("symbol")

	historyCmd := &cobra.Command{
		Use:   "history",
		Short: "Query contract position history",
		RunE:  runPositionHistory,
	}
	historyCmd.Flags().Int32("page", 0, "Page number")
	historyCmd.Flags().Int32("limit", 0, "Max records")
	historyCmd.Flags().String("symbol", "", "Filter by symbol")
	historyCmd.Flags().Int32("from", 0, "Start millisecond timestamp")
	historyCmd.Flags().Int32("to", 0, "End millisecond timestamp")

	marginHistoryCmd := &cobra.Command{
		Use:   "margin-history",
		Short: "Query leveraged position history",
		RunE:  runMarginPositionHistory,
	}
	marginHistoryCmd.Flags().Int32("page", 0, "Page number")
	marginHistoryCmd.Flags().Int32("limit", 0, "Max records")
	marginHistoryCmd.Flags().String("symbol", "", "Filter by symbol")
	marginHistoryCmd.Flags().Int32("from", 0, "Start millisecond timestamp")
	marginHistoryCmd.Flags().Int32("to", 0, "End millisecond timestamp")

	marginInterestsCmd := &cobra.Command{
		Use:   "margin-interests",
		Short: "Query leveraged interest deduction history",
		RunE:  runMarginInterests,
	}
	marginInterestsCmd.Flags().String("symbol", "", "Filter by symbol")
	marginInterestsCmd.Flags().Int32("from", 0, "Start millisecond timestamp")
	marginInterestsCmd.Flags().Int32("to", 0, "End millisecond timestamp")
	marginInterestsCmd.Flags().Int32("page", 0, "Page number")
	marginInterestsCmd.Flags().Int32("limit", 0, "Max records")
	marginInterestsCmd.Flags().String("exchange-type", "", "Filter by exchange")

	positionCmd.AddCommand(listCmd, marginListCmd, closeCmd, leverageCmd, setLeverageCmd,
		marginLeverageCmd, setMarginLeverageCmd, adlRankCmd, historyCmd, marginHistoryCmd, marginInterestsCmd)
	Cmd.AddCommand(positionCmd)
}

func runPositionList(cmd *cobra.Command, args []string) error {
	symbol, _ := cmd.Flags().GetString("symbol")
	exchangeType, _ := cmd.Flags().GetString("exchange-type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListCrossexPositionsOpts{}
	if symbol != "" {
		opts.Symbol = optional.NewString(symbol)
	}
	if exchangeType != "" {
		opts.ExchangeType = optional.NewString(exchangeType)
	}

	result, httpResp, err := c.CrossExAPI.ListCrossexPositions(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/positions", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Symbol, r.PositionSide, r.PositionQty, r.EntryPrice, r.MarkPrice, r.Upnl, r.Leverage}
	}
	return p.Table([]string{"Symbol", "Side", "Qty", "Entry Price", "Mark Price", "UPNL", "Leverage"}, rows)
}

func runMarginPositionList(cmd *cobra.Command, args []string) error {
	symbol, _ := cmd.Flags().GetString("symbol")
	exchangeType, _ := cmd.Flags().GetString("exchange-type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListCrossexMarginPositionsOpts{}
	if symbol != "" {
		opts.Symbol = optional.NewString(symbol)
	}
	if exchangeType != "" {
		opts.ExchangeType = optional.NewString(exchangeType)
	}

	result, httpResp, err := c.CrossExAPI.ListCrossexMarginPositions(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/margin_positions", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Symbol, r.PositionSide, r.AssetQty, r.EntryPrice, r.IndexPrice, r.Upnl, r.Leverage}
	}
	return p.Table([]string{"Symbol", "Side", "Asset Qty", "Entry Price", "Index Price", "UPNL", "Leverage"}, rows)
}

func runPositionClose(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.CrossexClosePositionRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	opts := &gateapi.CloseCrossexPositionOpts{
		CrossexClosePositionRequest: optional.NewInterface(body),
	}
	result, httpResp, err := c.CrossExAPI.CloseCrossexPosition(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/crossex/close_position", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runPositionLeverage(cmd *cobra.Command, args []string) error {
	symbols, _ := cmd.Flags().GetString("symbols")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var opts *gateapi.GetCrossexPositionsLeverageOpts
	if symbols != "" {
		opts = &gateapi.GetCrossexPositionsLeverageOpts{
			Symbols: optional.NewString(symbols),
		}
	}

	result, httpResp, err := c.CrossExAPI.GetCrossexPositionsLeverage(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/positions/leverage", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, 0, len(result))
	for k, v := range result {
		rows = append(rows, []string{k, v})
	}
	return p.Table([]string{"Symbol", "Leverage"}, rows)
}

func runPositionSetLeverage(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.CrossexLeverageRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	opts := &gateapi.UpdateCrossexPositionsLeverageOpts{
		CrossexLeverageRequest: optional.NewInterface(body),
	}
	result, httpResp, err := c.CrossExAPI.UpdateCrossexPositionsLeverage(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/crossex/positions/leverage", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runMarginPositionLeverage(cmd *cobra.Command, args []string) error {
	symbols, _ := cmd.Flags().GetString("symbols")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var opts *gateapi.GetCrossexMarginPositionsLeverageOpts
	if symbols != "" {
		opts = &gateapi.GetCrossexMarginPositionsLeverageOpts{
			Symbols: optional.NewString(symbols),
		}
	}

	result, httpResp, err := c.CrossExAPI.GetCrossexMarginPositionsLeverage(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/margin_positions/leverage", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, 0, len(result))
	for k, v := range result {
		rows = append(rows, []string{k, v})
	}
	return p.Table([]string{"Symbol", "Leverage"}, rows)
}

func runMarginPositionSetLeverage(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.CrossexLeverageRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	opts := &gateapi.UpdateCrossexMarginPositionsLeverageOpts{
		CrossexLeverageRequest: optional.NewInterface(body),
	}
	result, httpResp, err := c.CrossExAPI.UpdateCrossexMarginPositionsLeverage(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/crossex/margin_positions/leverage", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runAdlRank(cmd *cobra.Command, args []string) error {
	symbol, _ := cmd.Flags().GetString("symbol")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.CrossExAPI.ListCrossexAdlRank(c.Context(), symbol)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/adl_rank", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Symbol, r.CrossexAdlRank, r.ExchangeAdlRank}
	}
	return p.Table([]string{"Symbol", "CrossEx ADL Rank", "Exchange ADL Rank"}, rows)
}

func runPositionHistory(cmd *cobra.Command, args []string) error {
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	symbol, _ := cmd.Flags().GetString("symbol")
	from, _ := cmd.Flags().GetInt32("from")
	to, _ := cmd.Flags().GetInt32("to")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListCrossexHistoryPositionsOpts{}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if symbol != "" {
		opts.Symbol = optional.NewString(symbol)
	}
	if from != 0 {
		opts.From = optional.NewInt32(from)
	}
	if to != 0 {
		opts.To = optional.NewInt32(to)
	}

	result, httpResp, err := c.CrossExAPI.ListCrossexHistoryPositions(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/history_positions", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.PositionId, r.Symbol, r.ClosedType, r.ClosedPnl, r.OpenAvgPrice, r.ClosedAvgPrice, r.ClosedQty}
	}
	return p.Table([]string{"Position ID", "Symbol", "Close Type", "Closed PnL", "Open Avg Price", "Close Avg Price", "Closed Qty"}, rows)
}

func runMarginPositionHistory(cmd *cobra.Command, args []string) error {
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	symbol, _ := cmd.Flags().GetString("symbol")
	from, _ := cmd.Flags().GetInt32("from")
	to, _ := cmd.Flags().GetInt32("to")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListCrossexHistoryMarginPositionsOpts{}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if symbol != "" {
		opts.Symbol = optional.NewString(symbol)
	}
	if from != 0 {
		opts.From = optional.NewInt32(from)
	}
	if to != 0 {
		opts.To = optional.NewInt32(to)
	}

	result, httpResp, err := c.CrossExAPI.ListCrossexHistoryMarginPositions(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/history_margin_positions", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.PositionId, r.Symbol, r.ClosedType, r.ClosedPnl, r.OpenAvgPrice, r.ClosedAvgPrice, r.Interest}
	}
	return p.Table([]string{"Position ID", "Symbol", "Close Type", "Closed PnL", "Open Avg Price", "Close Avg Price", "Interest"}, rows)
}

func runMarginInterests(cmd *cobra.Command, args []string) error {
	symbol, _ := cmd.Flags().GetString("symbol")
	from, _ := cmd.Flags().GetInt32("from")
	to, _ := cmd.Flags().GetInt32("to")
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	exchangeType, _ := cmd.Flags().GetString("exchange-type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListCrossexHistoryMarginInterestsOpts{}
	if symbol != "" {
		opts.Symbol = optional.NewString(symbol)
	}
	if from != 0 {
		opts.From = optional.NewInt32(from)
	}
	if to != 0 {
		opts.To = optional.NewInt32(to)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if exchangeType != "" {
		opts.ExchangeType = optional.NewString(exchangeType)
	}

	result, httpResp, err := c.CrossExAPI.ListCrossexHistoryMarginInterests(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/history_margin_interests", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.InterestId, r.Symbol, r.LiabilityCoin, r.Liability, r.Interest, r.InterestRate, r.InterestType, r.CreateTime}
	}
	return p.Table([]string{"Interest ID", "Symbol", "Liability Coin", "Liability", "Interest", "Rate", "Type", "Created"}, rows)
}

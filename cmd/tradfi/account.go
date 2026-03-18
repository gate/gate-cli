package tradfi

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "TradFi account commands",
}

func init() {
	infoCmd := &cobra.Command{
		Use:   "info",
		Short: "Get MT5 account information",
		RunE:  runTradfiAccountInfo,
	}

	assetsCmd := &cobra.Command{
		Use:   "assets",
		Short: "Get TradFi account asset balances",
		RunE:  runTradfiAccountAssets,
	}

	transactionsCmd := &cobra.Command{
		Use:   "transactions",
		Short: "List TradFi account transactions",
		RunE:  runTradfiTransactions,
	}
	transactionsCmd.Flags().Int64("begin", 0, "Begin timestamp (Unix seconds)")
	transactionsCmd.Flags().Int64("end", 0, "End timestamp (Unix seconds)")
	transactionsCmd.Flags().String("type", "", "Transaction type filter")
	transactionsCmd.Flags().Int32("page", 1, "Page number")
	transactionsCmd.Flags().Int32("page-size", 20, "Number of records per page")

	createTransactionCmd := &cobra.Command{
		Use:   "transfer",
		Short: "Transfer funds into or out of the TradFi account",
		RunE:  runTradfiCreateTransaction,
	}
	createTransactionCmd.Flags().String("asset", "USDT", "Asset type, e.g. USDT")
	createTransactionCmd.Flags().String("change", "", "Amount to transfer (required)")
	createTransactionCmd.Flags().String("type", "", "Transaction type: deposit (transfer in) or withdraw (transfer out) (required)")
	createTransactionCmd.MarkFlagRequired("change")
	createTransactionCmd.MarkFlagRequired("type")

	accountCmd.AddCommand(infoCmd, assetsCmd, transactionsCmd, createTransactionCmd)
	Cmd.AddCommand(accountCmd)
}

func runTradfiAccountInfo(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.TradFiAPI.QueryMt5AccountInfo(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/tradfi/users/mt5-account", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	d := result.Data
	return p.Table(
		[]string{"MT5 UID", "Leverage", "Stop Out Level", "Status"},
		[][]string{{fmt.Sprintf("%d", d.Mt5Uid), fmt.Sprintf("%d", d.Leverage), d.StopOutLevel, fmt.Sprintf("%d", d.Status)}},
	)
}

func runTradfiAccountAssets(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.TradFiAPI.QueryUserAssets(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/tradfi/users/assets", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	d := result.Data
	return p.Table(
		[]string{"Equity", "Balance", "Margin Free", "Margin", "Margin Level", "Unrealized PNL"},
		[][]string{{d.Equity, d.Balance, d.MarginFree, d.Margin, d.MarginLevel, d.UnrealizedPnl}},
	)
}

func runTradfiTransactions(cmd *cobra.Command, args []string) error {
	begin, _ := cmd.Flags().GetInt64("begin")
	end, _ := cmd.Flags().GetInt64("end")
	txType, _ := cmd.Flags().GetString("type")
	page, _ := cmd.Flags().GetInt32("page")
	pageSize, _ := cmd.Flags().GetInt32("page-size")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.QueryTransactionOpts{
		Page:     optional.NewInt32(page),
		PageSize: optional.NewInt32(pageSize),
	}
	if begin != 0 {
		opts.BeginTime = optional.NewInt64(begin)
	}
	if end != 0 {
		opts.EndTime = optional.NewInt64(end)
	}
	if txType != "" {
		opts.Type_ = optional.NewString(txType)
	}

	result, httpResp, err := c.TradFiAPI.QueryTransaction(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/tradfi/users/transactions", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	if result.Data.List == nil {
		return p.Table([]string{"Time", "Type", "Asset", "Change", "Balance"}, nil)
	}
	rows := make([][]string, len(result.Data.List))
	for i, tx := range result.Data.List {
		rows[i] = []string{fmt.Sprintf("%d", tx.Time), tx.TypeDesc, tx.Asset, tx.Change, tx.Balance}
	}
	return p.Table([]string{"Time", "Type", "Asset", "Change", "Balance"}, rows)
}

func runTradfiCreateTransaction(cmd *cobra.Command, args []string) error {
	asset, _ := cmd.Flags().GetString("asset")
	change, _ := cmd.Flags().GetString("change")
	txType, _ := cmd.Flags().GetString("type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.TradFiTransactionRequest{Asset: asset, Change: change, Type: txType}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.TradFiAPI.CreateTransaction(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/tradfi/users/transactions", string(body)))
		return nil
	}
	return p.Print(result)
}

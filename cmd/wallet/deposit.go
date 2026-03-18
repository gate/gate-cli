package wallet

import (
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

func init() {
	depositAddressCmd := &cobra.Command{
		Use:   "deposit-address",
		Short: "Get deposit address for a currency",
		RunE:  runWalletDepositAddress,
	}
	depositAddressCmd.Flags().String("currency", "", "Currency name (required)")
	depositAddressCmd.MarkFlagRequired("currency")

	depositsCmd := &cobra.Command{
		Use:   "deposits",
		Short: "List deposit records",
		RunE:  runWalletDeposits,
	}
	depositsCmd.Flags().String("currency", "", "Filter by currency name")
	depositsCmd.Flags().Int64("from", 0, "Start Unix timestamp")
	depositsCmd.Flags().Int64("to", 0, "End Unix timestamp")
	depositsCmd.Flags().Int32("limit", 0, "Number of records to return")
	depositsCmd.Flags().Int32("offset", 0, "Number of records to skip")

	withdrawalsCmd := &cobra.Command{
		Use:   "withdrawals",
		Short: "List withdrawal records",
		RunE:  runWalletWithdrawals,
	}
	withdrawalsCmd.Flags().String("currency", "", "Filter by currency name")
	withdrawalsCmd.Flags().Int64("from", 0, "Start Unix timestamp")
	withdrawalsCmd.Flags().Int64("to", 0, "End Unix timestamp")
	withdrawalsCmd.Flags().Int32("limit", 0, "Number of records to return")
	withdrawalsCmd.Flags().Int32("offset", 0, "Number of records to skip")

	savedAddressCmd := &cobra.Command{
		Use:   "saved-address",
		Short: "List saved withdrawal address whitelist",
		RunE:  runWalletSavedAddress,
	}
	savedAddressCmd.Flags().String("currency", "", "Currency name (required)")
	savedAddressCmd.Flags().String("chain", "", "Filter by chain name")
	savedAddressCmd.MarkFlagRequired("currency")

	pushOrdersCmd := &cobra.Command{
		Use:   "push-orders",
		Short: "List UID push orders",
		RunE:  runWalletPushOrders,
	}
	pushOrdersCmd.Flags().Int32("from", 0, "Start Unix timestamp")
	pushOrdersCmd.Flags().Int32("to", 0, "End Unix timestamp")
	pushOrdersCmd.Flags().Int32("limit", 0, "Number of records to return")
	pushOrdersCmd.Flags().Int32("offset", 0, "Number of records to skip")

	Cmd.AddCommand(depositAddressCmd, depositsCmd, withdrawalsCmd, savedAddressCmd, pushOrdersCmd)
}

func runWalletDepositAddress(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.WalletAPI.GetDepositAddress(c.Context(), currency)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/deposit_address", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result.MultichainAddresses))
	for i, a := range result.MultichainAddresses {
		rows[i] = []string{result.Currency, a.Chain, a.Address, a.PaymentId, a.PaymentName}
	}
	if len(rows) == 0 {
		rows = [][]string{{result.Currency, "", result.Address, "", ""}}
	}
	return p.Table([]string{"Currency", "Chain", "Address", "Payment ID", "Memo Type"}, rows)
}

func runWalletDeposits(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	limit, _ := cmd.Flags().GetInt32("limit")
	offset, _ := cmd.Flags().GetInt32("offset")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListDepositsOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}

	result, httpResp, err := c.WalletAPI.ListDeposits(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/deposits", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, d := range result {
		rows[i] = []string{d.Id, d.Currency, d.Amount, d.Address, d.Status, d.Timestamp}
	}
	return p.Table([]string{"ID", "Currency", "Amount", "Address", "Status", "Time"}, rows)
}

func runWalletWithdrawals(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	limit, _ := cmd.Flags().GetInt32("limit")
	offset, _ := cmd.Flags().GetInt32("offset")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListWithdrawalsOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}

	result, httpResp, err := c.WalletAPI.ListWithdrawals(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/withdrawals", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, w := range result {
		rows[i] = []string{w.Id, w.Currency, w.Amount, w.Fee, w.Address, w.Status, w.Timestamp}
	}
	return p.Table([]string{"ID", "Currency", "Amount", "Fee", "Address", "Status", "Time"}, rows)
}

func runWalletSavedAddress(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	chain, _ := cmd.Flags().GetString("chain")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListSavedAddressOpts{}
	if chain != "" {
		opts.Chain = optional.NewString(chain)
	}

	result, httpResp, err := c.WalletAPI.ListSavedAddress(c.Context(), currency, opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/saved_address", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, a := range result {
		rows[i] = []string{a.Currency, a.Chain, a.Address, a.Name, a.Tag, a.Verified}
	}
	return p.Table([]string{"Currency", "Chain", "Address", "Name", "Tag", "Verified"}, rows)
}

func runWalletPushOrders(cmd *cobra.Command, args []string) error {
	from, _ := cmd.Flags().GetInt32("from")
	to, _ := cmd.Flags().GetInt32("to")
	limit, _ := cmd.Flags().GetInt32("limit")
	offset, _ := cmd.Flags().GetInt32("offset")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListPushOrdersOpts{}
	if from != 0 {
		opts.From = optional.NewInt32(from)
	}
	if to != 0 {
		opts.To = optional.NewInt32(to)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}

	result, httpResp, err := c.WalletAPI.ListPushOrders(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/push", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, o := range result {
		rows[i] = []string{
			fmt.Sprintf("%d", o.Id),
			o.Currency,
			o.Amount,
			o.Status,
			fmt.Sprintf("%d", o.PushUid),
			fmt.Sprintf("%d", o.ReceiveUid),
		}
	}
	return p.Table([]string{"ID", "Currency", "Amount", "Status", "Push UID", "Receive UID"}, rows)
}

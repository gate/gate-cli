package wallet

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// futuresSettle maps a currency to its futures settlement currency.
// Only USDT and BTC are supported for spot↔futures transfers.
func futuresSettle(currency string) (string, error) {
	switch strings.ToUpper(currency) {
	case "USDT":
		return "usdt", nil
	case "BTC":
		return "btc", nil
	default:
		return "", fmt.Errorf("currency %s is not supported for futures transfers: only USDT and BTC are accepted", currency)
	}
}

// isContractAccount returns true for account types that require a settle parameter.
func isContractAccount(accountType string) bool {
	return accountType == "futures" || accountType == "delivery"
}

var transferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Wallet transfer commands",
}

func init() {
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Transfer between accounts",
		RunE:  runWalletTransfer,
	}
	createCmd.Flags().String("currency", "", "Currency name (required)")
	createCmd.Flags().String("from", "", "Source account type (required)")
	createCmd.Flags().String("to", "", "Destination account type (required)")
	createCmd.Flags().String("amount", "", "Transfer amount (required)")
	createCmd.Flags().String("currency-pair", "", "Margin trading pair (required for margin accounts)")
	createCmd.MarkFlagRequired("currency")
	createCmd.MarkFlagRequired("from")
	createCmd.MarkFlagRequired("to")
	createCmd.MarkFlagRequired("amount")

	toSubCmd := &cobra.Command{
		Use:   "to-sub",
		Short: "Transfer between main and sub accounts",
		RunE:  runWalletSubTransfer,
	}
	toSubCmd.Flags().String("currency", "", "Currency name (required)")
	toSubCmd.Flags().String("sub-account", "", "Sub-account user ID (required)")
	toSubCmd.Flags().String("direction", "", "Transfer direction: to or from (required)")
	toSubCmd.Flags().String("amount", "", "Transfer amount (required)")
	toSubCmd.Flags().String("sub-account-type", "", "Sub-account trading account type")
	toSubCmd.MarkFlagRequired("currency")
	toSubCmd.MarkFlagRequired("sub-account")
	toSubCmd.MarkFlagRequired("direction")
	toSubCmd.MarkFlagRequired("amount")

	subToSubCmd := &cobra.Command{
		Use:   "sub-to-sub",
		Short: "Transfer between sub accounts",
		RunE:  runWalletSubToSub,
	}
	subToSubCmd.Flags().String("currency", "", "Currency name (required)")
	subToSubCmd.Flags().String("from-uid", "", "Source sub-account user ID (required)")
	subToSubCmd.Flags().String("from-type", "", "Source sub-account type (required)")
	subToSubCmd.Flags().String("to-uid", "", "Destination sub-account user ID (required)")
	subToSubCmd.Flags().String("to-type", "", "Destination sub-account type (required)")
	subToSubCmd.Flags().String("amount", "", "Transfer amount (required)")
	subToSubCmd.MarkFlagRequired("currency")
	subToSubCmd.MarkFlagRequired("from-uid")
	subToSubCmd.MarkFlagRequired("from-type")
	subToSubCmd.MarkFlagRequired("to-uid")
	subToSubCmd.MarkFlagRequired("to-type")
	subToSubCmd.MarkFlagRequired("amount")

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Query main-sub account transfer status",
		RunE:  runWalletTransferStatus,
	}
	statusCmd.Flags().String("client-order-id", "", "Client-defined transfer order ID")
	statusCmd.Flags().String("tx-id", "", "Transaction ID returned from transfer")

	subListCmd := &cobra.Command{
		Use:   "sub-list",
		Short: "List transfer records between main and sub accounts",
		RunE:  runWalletSubTransfers,
	}
	subListCmd.Flags().String("sub-uid", "", "Filter by sub-account user ID")
	subListCmd.Flags().Int64("from", 0, "Start Unix timestamp")
	subListCmd.Flags().Int64("to", 0, "End Unix timestamp")
	subListCmd.Flags().Int32("limit", 0, "Number of records to return")
	subListCmd.Flags().Int32("offset", 0, "Number of records to skip")

	transferCmd.AddCommand(createCmd, toSubCmd, subToSubCmd, statusCmd, subListCmd)
	Cmd.AddCommand(transferCmd)
}

func runWalletTransfer(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	amount, _ := cmd.Flags().GetString("amount")
	currencyPair, _ := cmd.Flags().GetString("currency-pair")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var settle string
	if isContractAccount(from) || isContractAccount(to) {
		settle, err = futuresSettle(currency)
		if err != nil {
			return err
		}
	}

	req := gateapi.Transfer{
		Currency:     currency,
		From:         from,
		To:           to,
		Amount:       amount,
		CurrencyPair: currencyPair,
		Settle:       settle,
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.WalletAPI.Transfer(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/wallet/transfers", string(body)))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table([]string{"TX ID"}, [][]string{{fmt.Sprintf("%d", result.TxId)}})
}

func runWalletSubTransfer(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	subAccount, _ := cmd.Flags().GetString("sub-account")
	direction, _ := cmd.Flags().GetString("direction")
	amount, _ := cmd.Flags().GetString("amount")
	subAccountType, _ := cmd.Flags().GetString("sub-account-type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	if isContractAccount(subAccountType) {
		if _, err := futuresSettle(currency); err != nil {
			return err
		}
	}

	req := gateapi.SubAccountTransfer{
		Currency:       currency,
		SubAccount:     subAccount,
		Direction:      direction,
		Amount:         amount,
		SubAccountType: subAccountType,
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.WalletAPI.TransferWithSubAccount(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/wallet/sub_account_transfers", string(body)))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table([]string{"TX ID"}, [][]string{{fmt.Sprintf("%d", result.TxId)}})
}

func runWalletSubToSub(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	fromUID, _ := cmd.Flags().GetString("from-uid")
	fromType, _ := cmd.Flags().GetString("from-type")
	toUID, _ := cmd.Flags().GetString("to-uid")
	toType, _ := cmd.Flags().GetString("to-type")
	amount, _ := cmd.Flags().GetString("amount")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	if isContractAccount(fromType) || isContractAccount(toType) {
		if _, err := futuresSettle(currency); err != nil {
			return err
		}
	}

	req := gateapi.SubAccountToSubAccount{
		Currency:           currency,
		SubAccountFrom:     fromUID,
		SubAccountFromType: fromType,
		SubAccountTo:       toUID,
		SubAccountToType:   toType,
		Amount:             amount,
	}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.WalletAPI.SubAccountToSubAccount(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/wallet/sub_account_to_sub_account", string(body)))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table([]string{"TX ID"}, [][]string{{fmt.Sprintf("%d", result.TxId)}})
}

func runWalletTransferStatus(cmd *cobra.Command, args []string) error {
	clientOrderID, _ := cmd.Flags().GetString("client-order-id")
	txID, _ := cmd.Flags().GetString("tx-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetTransferOrderStatusOpts{}
	if clientOrderID != "" {
		opts.ClientOrderId = optional.NewString(clientOrderID)
	}
	if txID != "" {
		opts.TxId = optional.NewString(txID)
	}

	result, httpResp, err := c.WalletAPI.GetTransferOrderStatus(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/order_status", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"TX ID", "Status"},
		[][]string{{result.TxId, result.Status}},
	)
}

func runWalletSubTransfers(cmd *cobra.Command, args []string) error {
	subUID, _ := cmd.Flags().GetString("sub-uid")
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

	opts := &gateapi.ListSubAccountTransfersOpts{}
	if subUID != "" {
		opts.SubUid = optional.NewString(subUID)
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

	result, httpResp, err := c.WalletAPI.ListSubAccountTransfers(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/wallet/sub_account_transfers", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{t.Currency, t.Amount, t.SubAccount, t.Direction, t.SubAccountType, t.Timest}
	}
	return p.Table([]string{"Currency", "Amount", "Sub Account", "Direction", "Account Type", "Time"}, rows)
}

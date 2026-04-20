package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var transactionCmd = &cobra.Command{
	Use:   "transaction",
	Short: "P2P transaction commands",
}

func init() {
	pendingCmd := &cobra.Command{
		Use:   "pending",
		Short: "List pending transactions",
		RunE:  runTransactionPending,
	}
	pendingCmd.Flags().String("json", "", "JSON body for pending list request (required)")
	pendingCmd.MarkFlagRequired("json")

	completedCmd := &cobra.Command{
		Use:   "completed",
		Short: "List completed transactions",
		RunE:  runTransactionCompleted,
	}
	completedCmd.Flags().String("json", "", "JSON body for completed list request (required)")
	completedCmd.MarkFlagRequired("json")

	detailCmd := &cobra.Command{
		Use:   "detail",
		Short: "Get transaction detail",
		RunE:  runTransactionDetail,
	}
	detailCmd.Flags().Int32("txid", 0, "Transaction/order ID (required)")
	detailCmd.MarkFlagRequired("txid")
	detailCmd.Flags().String("channel", "", "Channel (empty or web3)")

	confirmPaymentCmd := &cobra.Command{
		Use:   "confirm-payment",
		Short: "Confirm payment for a transaction",
		RunE:  runConfirmPayment,
	}
	confirmPaymentCmd.Flags().String("trade-id", "", "Trade ID (required)")
	confirmPaymentCmd.MarkFlagRequired("trade-id")
	confirmPaymentCmd.Flags().String("payment-method", "", "Payment method (required)")
	confirmPaymentCmd.MarkFlagRequired("payment-method")

	confirmReceiptCmd := &cobra.Command{
		Use:   "confirm-receipt",
		Short: "Confirm receipt for a transaction",
		RunE:  runConfirmReceipt,
	}
	confirmReceiptCmd.Flags().String("trade-id", "", "Trade ID (required)")
	confirmReceiptCmd.MarkFlagRequired("trade-id")

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a transaction",
		RunE:  runTransactionCancel,
	}
	cancelCmd.Flags().String("trade-id", "", "Trade ID (required)")
	cancelCmd.MarkFlagRequired("trade-id")
	cancelCmd.Flags().String("reason-id", "", "Reason ID")
	cancelCmd.Flags().String("reason-memo", "", "Reason memo")

	pushOrderCmd := &cobra.Command{
		Use:   "push-order",
		Short: "Place a push order",
		RunE:  runPushOrder,
	}
	pushOrderCmd.Flags().String("json", "", "JSON body for push order request (required)")
	pushOrderCmd.MarkFlagRequired("json")

	transactionCmd.AddCommand(pendingCmd, completedCmd, detailCmd, confirmPaymentCmd, confirmReceiptCmd, cancelCmd, pushOrderCmd)
	Cmd.AddCommand(transactionCmd)
}

func runTransactionPending(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.GetPendingTransactionListRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantTransactionGetPendingTransactionList(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/transaction/get_pending_transaction_list", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runTransactionCompleted(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.GetCompletedTransactionListRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantTransactionGetCompletedTransactionList(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/transaction/get_completed_transaction_list", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runTransactionDetail(cmd *cobra.Command, args []string) error {
	txid, _ := cmd.Flags().GetInt32("txid")
	channel, _ := cmd.Flags().GetString("channel")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.GetTransactionDetailsRequest{
		Txid: txid,
	}
	if channel != "" {
		body.Channel = channel
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantTransactionGetTransactionDetails(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/transaction/get_transaction_details", ""))
		return nil
	}
	return p.Print(result)
}

func runConfirmPayment(cmd *cobra.Command, args []string) error {
	tradeID, _ := cmd.Flags().GetString("trade-id")
	paymentMethod, _ := cmd.Flags().GetString("payment-method")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.ConfirmPayment{
		TradeId:       tradeID,
		PaymentMethod: paymentMethod,
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantTransactionConfirmPayment(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/transaction/confirm_payment", ""))
		return nil
	}
	return p.Print(result)
}

func runConfirmReceipt(cmd *cobra.Command, args []string) error {
	tradeID, _ := cmd.Flags().GetString("trade-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.ConfirmReceipt{
		TradeId: tradeID,
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantTransactionConfirmReceipt(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/transaction/confirm_receipt", ""))
		return nil
	}
	return p.Print(result)
}

func runTransactionCancel(cmd *cobra.Command, args []string) error {
	tradeID, _ := cmd.Flags().GetString("trade-id")
	reasonID, _ := cmd.Flags().GetString("reason-id")
	reasonMemo, _ := cmd.Flags().GetString("reason-memo")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.CancelOrder{
		TradeId: tradeID,
	}
	if reasonID != "" {
		body.ReasonId = reasonID
	}
	if reasonMemo != "" {
		body.ReasonMemo = reasonMemo
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantTransactionCancel(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/transaction/cancel", ""))
		return nil
	}
	return p.Print(result)
}

func runPushOrder(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.PlaceBizPushOrder
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantBooksPlaceBizPushOrder(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/books/place_biz_push_order", jsonStr))
		return nil
	}
	return p.Print(result)
}

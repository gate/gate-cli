package withdrawal

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// Cmd is the root command for the withdrawal module.
var Cmd = &cobra.Command{
	Use:   "withdrawal",
	Short: "Withdrawal commands",
}

func init() {
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new withdrawal",
		RunE:  runCreate,
	}
	createCmd.Flags().String("currency", "", "Currency name (required)")
	createCmd.MarkFlagRequired("currency")
	createCmd.Flags().String("amount", "", "Withdrawal amount (required)")
	createCmd.MarkFlagRequired("amount")
	createCmd.Flags().String("address", "", "Withdrawal address (required)")
	createCmd.MarkFlagRequired("address")
	createCmd.Flags().String("chain", "", "Chain name")
	createCmd.Flags().String("memo", "", "Memo/tag for the withdrawal")

	pushOrderCmd := &cobra.Command{
		Use:   "push-order",
		Short: "UID transfer between main spot accounts",
		RunE:  runPushOrder,
	}
	pushOrderCmd.Flags().Int64("receive-uid", 0, "Recipient UID (required)")
	pushOrderCmd.MarkFlagRequired("receive-uid")
	pushOrderCmd.Flags().String("currency", "", "Currency name (required)")
	pushOrderCmd.MarkFlagRequired("currency")
	pushOrderCmd.Flags().String("amount", "", "Transfer amount (required)")
	pushOrderCmd.MarkFlagRequired("amount")

	cancelCmd := &cobra.Command{
		Use:   "cancel",
		Short: "Cancel a pending withdrawal",
		RunE:  runCancel,
	}
	cancelCmd.Flags().String("id", "", "Withdrawal ID (required)")
	cancelCmd.MarkFlagRequired("id")

	Cmd.AddCommand(createCmd, pushOrderCmd, cancelCmd)
}

func runCreate(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	amount, _ := cmd.Flags().GetString("amount")
	address, _ := cmd.Flags().GetString("address")
	chain, _ := cmd.Flags().GetString("chain")
	memo, _ := cmd.Flags().GetString("memo")

	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.LedgerRecord{
		Currency: currency,
		Amount:   amount,
		Address:  address,
		Chain:    chain,
	}
	if memo != "" {
		body.Memo = memo
	}

	result, httpResp, err := c.WithdrawalAPI.Withdraw(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/withdrawals", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"ID", "Currency", "Amount", "Address", "Chain", "Status"},
		[][]string{{result.Id, result.Currency, result.Amount, result.Address, result.Chain, result.Status}},
	)
}

func runPushOrder(cmd *cobra.Command, args []string) error {
	receiveUID, _ := cmd.Flags().GetInt64("receive-uid")
	currency, _ := cmd.Flags().GetString("currency")
	amount, _ := cmd.Flags().GetString("amount")

	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.UidPushWithdrawal{
		ReceiveUid: receiveUID,
		Currency:   currency,
		Amount:     amount,
	}

	result, httpResp, err := c.WithdrawalAPI.WithdrawPushOrder(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/withdrawals/push", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Order ID"},
		[][]string{{result.Id}},
	)
}

func runCancel(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")

	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.WithdrawalAPI.CancelWithdrawal(c.Context(), id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", fmt.Sprintf("/api/v4/withdrawals/%s", id), ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"ID", "Currency", "Amount", "Address", "Status"},
		[][]string{{result.Id, result.Currency, result.Amount, result.Address, result.Status}},
	)
}

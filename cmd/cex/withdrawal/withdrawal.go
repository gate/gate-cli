package withdrawal

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// Cmd retains UID transfers under their existing path for compatibility.
var Cmd = &cobra.Command{
	Use:   "withdrawal",
	Short: "UID transfer commands",
}

func init() {
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

	Cmd.AddCommand(pushOrderCmd)
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
		return client.CommandError(err, httpResp, "POST", "/api/v4/withdrawals/push", "")
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Order ID"},
		[][]string{{result.Id}},
	)
}

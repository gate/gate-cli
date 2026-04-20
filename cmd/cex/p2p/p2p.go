package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// Cmd is the root command for the P2P module.
var Cmd = &cobra.Command{
	Use:   "p2p",
	Short: "P2P trading commands",
}

func init() {
	userInfoCmd := &cobra.Command{
		Use:   "user-info",
		Short: "Get P2P merchant account information",
		RunE:  runUserInfo,
	}

	counterpartyCmd := &cobra.Command{
		Use:   "counterparty",
		Short: "Get counterparty user information",
		RunE:  runCounterparty,
	}
	counterpartyCmd.Flags().String("biz-uid", "", "Counterparty encrypted UID (required)")
	counterpartyCmd.MarkFlagRequired("biz-uid")

	paymentCmd := &cobra.Command{
		Use:   "payment",
		Short: "Get payment method list",
		RunE:  runPayment,
	}
	paymentCmd.Flags().String("json", "", "JSON body for payment request (optional)")

	Cmd.AddCommand(userInfoCmd, counterpartyCmd, paymentCmd)
}

func runUserInfo(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantAccountGetUserInfo(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/account/get_user_info", ""))
		return nil
	}
	return p.Print(result)
}

func runCounterparty(cmd *cobra.Command, args []string) error {
	bizUID, _ := cmd.Flags().GetString("biz-uid")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.GetCounterpartyUserInfoRequest{
		BizUid: bizUID,
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantAccountGetCounterpartyUserInfo(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/account/get_counterparty_user_info", ""))
		return nil
	}
	return p.Print(result)
}

func runPayment(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var opts *gateapi.P2pMerchantAccountGetMyselfPaymentOpts
	if jsonStr != "" {
		var body gateapi.GetMyselfPaymentRequest
		if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
			return fmt.Errorf("invalid --json: %w", err)
		}
		opts = &gateapi.P2pMerchantAccountGetMyselfPaymentOpts{
			GetMyselfPaymentRequest: optional.NewInterface(body),
		}
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantAccountGetMyselfPayment(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/account/get_myself_payment", jsonStr))
		return nil
	}
	return p.Print(result)
}

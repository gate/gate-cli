package rebate

import (
	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var agencyCmd = &cobra.Command{
	Use:   "agency",
	Short: "Agency rebate commands",
}

func init() {
	commissionsCmd := &cobra.Command{
		Use:   "commissions",
		Short: "Agency obtains rebate history of recommended users",
		RunE:  runAgencyCommissions,
	}
	commissionsCmd.Flags().String("currency", "", "Currency filter")
	commissionsCmd.Flags().Int32("commission-type", 0, "Type: 1=Direct, 2=Indirect, 3=Self")
	commissionsCmd.Flags().Int64("user-id", 0, "User ID filter")
	commissionsCmd.Flags().Int64("from", 0, "Start timestamp")
	commissionsCmd.Flags().Int64("to", 0, "End timestamp")
	commissionsCmd.Flags().Int32("limit", 0, "Max records")
	commissionsCmd.Flags().Int32("offset", 0, "List offset")

	transactionsCmd := &cobra.Command{
		Use:   "transactions",
		Short: "Agency obtains transaction history of recommended users",
		RunE:  runAgencyTransactions,
	}
	transactionsCmd.Flags().String("pair", "", "Currency pair filter")
	transactionsCmd.Flags().Int64("user-id", 0, "User ID filter")
	transactionsCmd.Flags().Int64("from", 0, "Start timestamp")
	transactionsCmd.Flags().Int64("to", 0, "End timestamp")
	transactionsCmd.Flags().Int32("limit", 0, "Max records")
	transactionsCmd.Flags().Int32("offset", 0, "List offset")

	agencyCmd.AddCommand(commissionsCmd, transactionsCmd)
	Cmd.AddCommand(agencyCmd)
}

func runAgencyCommissions(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	commissionType, _ := cmd.Flags().GetInt32("commission-type")
	userID, _ := cmd.Flags().GetInt64("user-id")
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

	opts := &gateapi.AgencyCommissionsHistoryOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if commissionType != 0 {
		opts.CommissionType = optional.NewInt32(commissionType)
	}
	if userID != 0 {
		opts.UserId = optional.NewInt64(userID)
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

	result, httpResp, err := c.RebateAPI.AgencyCommissionsHistory(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rebate/agency/commission_history", ""))
		return nil
	}
	return p.Print(result)
}

func runAgencyTransactions(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	userID, _ := cmd.Flags().GetInt64("user-id")
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

	opts := &gateapi.AgencyTransactionHistoryOpts{}
	if pair != "" {
		opts.CurrencyPair = optional.NewString(pair)
	}
	if userID != 0 {
		opts.UserId = optional.NewInt64(userID)
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

	result, httpResp, err := c.RebateAPI.AgencyTransactionHistory(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rebate/agency/transaction_history", ""))
		return nil
	}
	return p.Print(result)
}

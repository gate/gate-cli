package rebate

import (
	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var brokerCmd = &cobra.Command{
	Use:   "broker",
	Short: "Broker rebate commands",
}

func init() {
	commissionsCmd := &cobra.Command{
		Use:   "commissions",
		Short: "Broker obtains user rebate records",
		RunE:  runBrokerCommissions,
	}
	commissionsCmd.Flags().Int32("limit", 0, "Max records")
	commissionsCmd.Flags().Int32("offset", 0, "List offset")
	commissionsCmd.Flags().Int64("user-id", 0, "User ID filter")
	commissionsCmd.Flags().Int64("from", 0, "Start timestamp")
	commissionsCmd.Flags().Int64("to", 0, "End timestamp")

	transactionsCmd := &cobra.Command{
		Use:   "transactions",
		Short: "Broker obtains user trading history",
		RunE:  runBrokerTransactions,
	}
	transactionsCmd.Flags().Int32("limit", 0, "Max records")
	transactionsCmd.Flags().Int32("offset", 0, "List offset")
	transactionsCmd.Flags().Int64("user-id", 0, "User ID filter")
	transactionsCmd.Flags().Int64("from", 0, "Start timestamp")
	transactionsCmd.Flags().Int64("to", 0, "End timestamp")

	brokerCmd.AddCommand(commissionsCmd, transactionsCmd)
	Cmd.AddCommand(brokerCmd)
}

func runBrokerCommissions(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt32("limit")
	offset, _ := cmd.Flags().GetInt32("offset")
	userID, _ := cmd.Flags().GetInt64("user-id")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.RebateBrokerCommissionHistoryOpts{}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
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

	result, httpResp, err := c.RebateAPI.RebateBrokerCommissionHistory(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rebate/broker/commission_history", ""))
		return nil
	}
	return p.Print(result)
}

func runBrokerTransactions(cmd *cobra.Command, args []string) error {
	limit, _ := cmd.Flags().GetInt32("limit")
	offset, _ := cmd.Flags().GetInt32("offset")
	userID, _ := cmd.Flags().GetInt64("user-id")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.RebateBrokerTransactionHistoryOpts{}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
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

	result, httpResp, err := c.RebateAPI.RebateBrokerTransactionHistory(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rebate/broker/transaction_history", ""))
		return nil
	}
	return p.Print(result)
}

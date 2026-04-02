package rebate

import (
	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var partnerCmd = &cobra.Command{
	Use:   "partner",
	Short: "Partner rebate commands",
}

func init() {
	transactionsCmd := &cobra.Command{
		Use:   "transactions",
		Short: "Partner obtains transaction history of recommended users",
		RunE:  runPartnerTransactions,
	}
	transactionsCmd.Flags().String("pair", "", "Currency pair filter")
	transactionsCmd.Flags().Int64("user-id", 0, "User ID filter")
	transactionsCmd.Flags().Int64("from", 0, "Start timestamp")
	transactionsCmd.Flags().Int64("to", 0, "End timestamp")
	transactionsCmd.Flags().Int32("limit", 0, "Max records")
	transactionsCmd.Flags().Int32("offset", 0, "List offset")

	commissionsCmd := &cobra.Command{
		Use:   "commissions",
		Short: "Partner obtains rebate records of recommended users",
		RunE:  runPartnerCommissions,
	}
	commissionsCmd.Flags().String("currency", "", "Currency filter")
	commissionsCmd.Flags().Int64("user-id", 0, "User ID filter")
	commissionsCmd.Flags().Int64("from", 0, "Start timestamp")
	commissionsCmd.Flags().Int64("to", 0, "End timestamp")
	commissionsCmd.Flags().Int32("limit", 0, "Max records")
	commissionsCmd.Flags().Int32("offset", 0, "List offset")

	subListCmd := &cobra.Command{
		Use:   "sub-list",
		Short: "Partner subordinate list",
		RunE:  runPartnerSubList,
	}
	subListCmd.Flags().Int64("user-id", 0, "User ID filter")
	subListCmd.Flags().Int32("limit", 0, "Max records")
	subListCmd.Flags().Int32("offset", 0, "List offset")

	eligibilityCmd := &cobra.Command{
		Use:   "eligibility",
		Short: "Check partner eligibility",
		RunE:  runPartnerEligibility,
	}

	applicationCmd := &cobra.Command{
		Use:   "application",
		Short: "Get recent partner application",
		RunE:  runPartnerApplication,
	}

	agentDataCmd := &cobra.Command{
		Use:   "agent-data",
		Short: "Aggregated partner agent statistics",
		RunE:  runPartnerAgentData,
	}
	agentDataCmd.Flags().String("start-date", "", "Start date (yyyy-mm-dd hh:ii:ss, UTC+8)")
	agentDataCmd.Flags().String("end-date", "", "End date (yyyy-mm-dd hh:ii:ss, UTC+8)")
	agentDataCmd.Flags().Int32("business-type", 0, "Business type: 0=All, 1=Spot, 2=Futures, 3=Alpha, etc.")

	partnerCmd.AddCommand(transactionsCmd, commissionsCmd, subListCmd, eligibilityCmd, applicationCmd, agentDataCmd)
	Cmd.AddCommand(partnerCmd)
}

func runPartnerTransactions(cmd *cobra.Command, args []string) error {
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

	opts := &gateapi.PartnerTransactionHistoryOpts{}
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

	result, httpResp, err := c.RebateAPI.PartnerTransactionHistory(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rebate/partner/transaction_history", ""))
		return nil
	}
	return p.Print(result)
}

func runPartnerCommissions(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
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

	opts := &gateapi.PartnerCommissionsHistoryOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
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

	result, httpResp, err := c.RebateAPI.PartnerCommissionsHistory(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rebate/partner/commission_history", ""))
		return nil
	}
	return p.Print(result)
}

func runPartnerSubList(cmd *cobra.Command, args []string) error {
	userID, _ := cmd.Flags().GetInt64("user-id")
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

	opts := &gateapi.PartnerSubListOpts{}
	if userID != 0 {
		opts.UserId = optional.NewInt64(userID)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if offset != 0 {
		opts.Offset = optional.NewInt32(offset)
	}

	result, httpResp, err := c.RebateAPI.PartnerSubList(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rebate/partner/sub_list", ""))
		return nil
	}
	return p.Print(result)
}

func runPartnerEligibility(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.RebateAPI.GetPartnerEligibility(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rebate/partner/eligibility", ""))
		return nil
	}
	return p.Print(result)
}

func runPartnerApplication(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.RebateAPI.GetPartnerApplicationRecent(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rebate/partner/application_recent", ""))
		return nil
	}
	return p.Print(result)
}

func runPartnerAgentData(cmd *cobra.Command, args []string) error {
	startDate, _ := cmd.Flags().GetString("start-date")
	endDate, _ := cmd.Flags().GetString("end-date")
	businessType, _ := cmd.Flags().GetInt32("business-type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetPartnerAgentDataAggregatedOpts{}
	if startDate != "" {
		opts.StartDate = optional.NewString(startDate)
	}
	if endDate != "" {
		opts.EndDate = optional.NewString(endDate)
	}
	if businessType != 0 {
		opts.BusinessType = optional.NewInt32(businessType)
	}

	result, httpResp, err := c.RebateAPI.GetPartnerAgentDataAggregated(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rebate/partner/agent_data_aggregated", ""))
		return nil
	}
	return p.Print(result)
}

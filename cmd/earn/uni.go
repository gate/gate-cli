package earn

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var uniCmd = &cobra.Command{
	Use:   "uni",
	Short: "Simple earn (Uni) commands",
}

func init() {
	currenciesCmd := &cobra.Command{
		Use:   "currencies",
		Short: "List Uni currencies (public, no auth required)",
		RunE:  runUniCurrencies,
	}

	currencyCmd := &cobra.Command{
		Use:   "currency",
		Short: "Get Uni currency detail (public, no auth required)",
		RunE:  runUniCurrency,
	}
	currencyCmd.Flags().String("currency", "", "Currency name (required)")
	currencyCmd.MarkFlagRequired("currency")

	lendsCmd := &cobra.Command{
		Use:   "lends",
		Short: "List user lending orders",
		RunE:  runUniLends,
	}
	lendsCmd.Flags().String("currency", "", "Filter by currency")
	lendsCmd.Flags().Int32("page", 0, "Page number")
	lendsCmd.Flags().Int32("limit", 0, "Maximum items returned (max 100)")

	lendCmd := &cobra.Command{
		Use:   "lend",
		Short: "Create a Uni lending order",
		RunE:  runUniLend,
	}
	lendCmd.Flags().String("json", "", "JSON body for lend request (required)")
	lendCmd.MarkFlagRequired("json")

	changeCmd := &cobra.Command{
		Use:   "change",
		Short: "Amend lending information",
		RunE:  runUniChange,
	}
	changeCmd.Flags().String("json", "", "JSON body for change request (required)")
	changeCmd.MarkFlagRequired("json")

	recordsCmd := &cobra.Command{
		Use:   "records",
		Short: "List lending transaction records",
		RunE:  runUniRecords,
	}
	recordsCmd.Flags().String("currency", "", "Filter by currency")
	recordsCmd.Flags().Int32("page", 0, "Page number")
	recordsCmd.Flags().Int32("limit", 0, "Maximum items returned (max 100)")
	recordsCmd.Flags().Int64("from", 0, "Start timestamp")
	recordsCmd.Flags().Int64("to", 0, "End timestamp")
	recordsCmd.Flags().String("type", "", "Operation type: lend or redeem")

	interestCmd := &cobra.Command{
		Use:   "interest",
		Short: "Get Uni lending interest for a currency",
		RunE:  runUniInterest,
	}
	interestCmd.Flags().String("currency", "", "Currency name (required)")
	interestCmd.MarkFlagRequired("currency")

	interestRecordsCmd := &cobra.Command{
		Use:   "interest-records",
		Short: "List Uni interest/dividend records",
		RunE:  runUniInterestRecords,
	}
	interestRecordsCmd.Flags().String("currency", "", "Filter by currency")
	interestRecordsCmd.Flags().Int32("page", 0, "Page number")
	interestRecordsCmd.Flags().Int32("limit", 0, "Maximum items returned (max 100)")
	interestRecordsCmd.Flags().Int64("from", 0, "Start timestamp")
	interestRecordsCmd.Flags().Int64("to", 0, "End timestamp")

	interestStatusCmd := &cobra.Command{
		Use:   "interest-status",
		Short: "Get Uni interest status for a currency",
		RunE:  runUniInterestStatus,
	}
	interestStatusCmd.Flags().String("currency", "", "Currency name (required)")
	interestStatusCmd.MarkFlagRequired("currency")

	chartCmd := &cobra.Command{
		Use:   "chart",
		Short: "List Uni chart data (public, no auth required)",
		RunE:  runUniChart,
	}
	chartCmd.Flags().String("currency", "", "Currency name (required)")
	chartCmd.Flags().Int64("from", 0, "Start timestamp (required)")
	chartCmd.Flags().Int64("to", 0, "End timestamp (required)")
	chartCmd.MarkFlagRequired("currency")
	chartCmd.MarkFlagRequired("from")
	chartCmd.MarkFlagRequired("to")

	rateCmd := &cobra.Command{
		Use:   "rate",
		Short: "List estimated Uni lending rates (public, no auth required)",
		RunE:  runUniRate,
	}

	uniCmd.AddCommand(currenciesCmd, currencyCmd, lendsCmd, lendCmd, changeCmd,
		recordsCmd, interestCmd, interestRecordsCmd, interestStatusCmd, chartCmd, rateCmd)
}

func runUniCurrencies(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.EarnUniAPI.ListUniCurrencies(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/uni/currencies", ""))
		return nil
	}
	return p.Print(result)
}

func runUniCurrency(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.EarnUniAPI.GetUniCurrency(c.Context(), currency)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/uni/currencies/"+currency, ""))
		return nil
	}
	return p.Print(result)
}

func runUniLends(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListUserUniLendsOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.EarnUniAPI.ListUserUniLends(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/uni/lends", ""))
		return nil
	}
	return p.Print(result)
}

func runUniLend(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.CreateUniLend
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}

	bodyJSON, _ := json.Marshal(body)
	httpResp, err := c.EarnUniAPI.CreateUniLend(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/earn/uni/lends", string(bodyJSON)))
		return nil
	}
	return p.Print(map[string]string{"status": "ok"})
}

func runUniChange(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.PatchUniLend
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid JSON body: %w", err)
	}

	bodyJSON, _ := json.Marshal(body)
	httpResp, err := c.EarnUniAPI.ChangeUniLend(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "PATCH", "/api/v4/earn/uni/lends", string(bodyJSON)))
		return nil
	}
	return p.Print(map[string]string{"status": "ok"})
}

func runUniRecords(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	typ, _ := cmd.Flags().GetString("type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListUniLendRecordsOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}
	if typ != "" {
		opts.Type_ = optional.NewString(typ)
	}

	result, httpResp, err := c.EarnUniAPI.ListUniLendRecords(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/uni/lend_records", ""))
		return nil
	}
	return p.Print(result)
}

func runUniInterest(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.EarnUniAPI.GetUniInterest(c.Context(), currency)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/uni/interests/"+currency, ""))
		return nil
	}
	return p.Print(result)
}

func runUniInterestRecords(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	page, _ := cmd.Flags().GetInt32("page")
	limit, _ := cmd.Flags().GetInt32("limit")
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

	opts := &gateapi.ListUniInterestRecordsOpts{}
	if currency != "" {
		opts.Currency = optional.NewString(currency)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}
	if from != 0 {
		opts.From = optional.NewInt64(from)
	}
	if to != 0 {
		opts.To = optional.NewInt64(to)
	}

	result, httpResp, err := c.EarnUniAPI.ListUniInterestRecords(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/uni/interest_records", ""))
		return nil
	}
	return p.Print(result)
}

func runUniInterestStatus(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.EarnUniAPI.GetUniInterestStatus(c.Context(), currency)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/uni/interest_status/"+currency, ""))
		return nil
	}
	return p.Print(result)
}

func runUniChart(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	from, _ := cmd.Flags().GetInt64("from")
	to, _ := cmd.Flags().GetInt64("to")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.EarnUniAPI.ListUniChart(c.Context(), from, to, currency)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/uni/chart", ""))
		return nil
	}
	return p.Print(result)
}

func runUniRate(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.EarnUniAPI.ListUniRate(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/earn/uni/estimate_rate", ""))
		return nil
	}
	return p.Print(result)
}

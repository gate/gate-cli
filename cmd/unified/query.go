package unified

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

var queryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query borrowable, transferable, and estimate rate",
}

func init() {
	borrowableCmd := &cobra.Command{
		Use:   "borrowable",
		Short: "Get max borrowable amount for a currency",
		RunE:  runQueryBorrowable,
	}
	borrowableCmd.Flags().String("currency", "", "Currency name (required)")
	borrowableCmd.MarkFlagRequired("currency")

	borrowableListCmd := &cobra.Command{
		Use:   "borrowable-list",
		Short: "Batch query max borrowable amounts",
		RunE:  runQueryBorrowableList,
	}
	borrowableListCmd.Flags().String("currencies", "", "Comma-separated currency names, max 10 (required)")
	borrowableListCmd.MarkFlagRequired("currencies")

	transferableCmd := &cobra.Command{
		Use:   "transferable",
		Short: "Get max transferable amount for a currency",
		RunE:  runQueryTransferable,
	}
	transferableCmd.Flags().String("currency", "", "Currency name (required)")
	transferableCmd.MarkFlagRequired("currency")

	transferablesCmd := &cobra.Command{
		Use:   "transferables",
		Short: "Batch query max transferable amounts",
		RunE:  runQueryTransferables,
	}
	transferablesCmd.Flags().String("currencies", "", "Comma-separated currency names, max 100 (required)")
	transferablesCmd.MarkFlagRequired("currencies")

	estimateRateCmd := &cobra.Command{
		Use:   "estimate-rate",
		Short: "Query estimated interest rate for currencies",
		RunE:  runQueryEstimateRate,
	}
	estimateRateCmd.Flags().String("currencies", "", "Comma-separated currency names, max 10 (required)")
	estimateRateCmd.MarkFlagRequired("currencies")

	queryCmd.AddCommand(borrowableCmd, borrowableListCmd, transferableCmd, transferablesCmd, estimateRateCmd)
	Cmd.AddCommand(queryCmd)
}

func runQueryBorrowable(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.UnifiedAPI.GetUnifiedBorrowable(c.Context(), currency)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/unified/borrowable", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Currency", "Max Borrowable"},
		[][]string{{result.Currency, result.Amount}},
	)
}

func runQueryBorrowableList(cmd *cobra.Command, args []string) error {
	currencies, _ := cmd.Flags().GetString("currencies")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	list := strings.Split(currencies, ",")
	result, httpResp, err := c.UnifiedAPI.GetUnifiedBorrowableList(c.Context(), list)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/unified/borrowable_list", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, b := range result {
		rows[i] = []string{b.Currency, b.Amount}
	}
	return p.Table([]string{"Currency", "Max Borrowable"}, rows)
}

func runQueryTransferable(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.UnifiedAPI.GetUnifiedTransferable(c.Context(), currency)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/unified/transferable", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"Currency", "Max Transferable"},
		[][]string{{result.Currency, result.Amount}},
	)
}

func runQueryTransferables(cmd *cobra.Command, args []string) error {
	currencies, _ := cmd.Flags().GetString("currencies")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.UnifiedAPI.GetUnifiedTransferables(c.Context(), currencies)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/unified/transferables", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, t := range result {
		rows[i] = []string{t.Currency, t.Amount}
	}
	return p.Table([]string{"Currency", "Max Transferable"}, rows)
}

func runQueryEstimateRate(cmd *cobra.Command, args []string) error {
	currencies, _ := cmd.Flags().GetString("currencies")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	list := strings.Split(currencies, ",")
	result, httpResp, err := c.UnifiedAPI.GetUnifiedEstimateRate(c.Context(), list)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/unified/estimate_rate", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, 0, len(result))
	for cur, rate := range result {
		rows = append(rows, []string{cur, rate})
	}
	return p.Table([]string{"Currency", "Estimated Rate"}, rows)
}

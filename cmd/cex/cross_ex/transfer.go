package crossex

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var transferCmd = &cobra.Command{
	Use:   "transfer",
	Short: "Cross-exchange fund transfer commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Query fund transfer history",
		RunE:  runTransferList,
	}
	listCmd.Flags().String("coin", "", "Filter by currency")
	listCmd.Flags().String("order-id", "", "Filter by order ID or custom text ID")
	listCmd.Flags().Int32("from", 0, "Start timestamp")
	listCmd.Flags().Int32("to", 0, "End timestamp")
	listCmd.Flags().Int32("page", 0, "Page number")
	listCmd.Flags().Int32("limit", 0, "Max records (max 1000)")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a fund transfer",
		RunE:  runTransferCreate,
	}
	createCmd.Flags().String("json", "", "JSON body for transfer request (required)")
	createCmd.MarkFlagRequired("json")

	transferCmd.AddCommand(listCmd, createCmd)
	Cmd.AddCommand(transferCmd)
}

func runTransferList(cmd *cobra.Command, args []string) error {
	coin, _ := cmd.Flags().GetString("coin")
	orderID, _ := cmd.Flags().GetString("order-id")
	from, _ := cmd.Flags().GetInt32("from")
	to, _ := cmd.Flags().GetInt32("to")
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

	opts := &gateapi.ListCrossexTransfersOpts{}
	if coin != "" {
		opts.Coin = optional.NewString(coin)
	}
	if orderID != "" {
		opts.OrderId = optional.NewString(orderID)
	}
	if from != 0 {
		opts.From = optional.NewInt32(from)
	}
	if to != 0 {
		opts.To = optional.NewInt32(to)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if limit != 0 {
		opts.Limit = optional.NewInt32(limit)
	}

	result, httpResp, err := c.CrossExAPI.ListCrossexTransfers(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/crossex/transfers", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Id, r.Coin, r.Amount, r.FromAccountType, r.ToAccountType, r.Status, fmt.Sprintf("%d", r.CreateTime)}
	}
	return p.Table([]string{"ID", "Coin", "Amount", "From", "To", "Status", "Created"}, rows)
}

func runTransferCreate(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.CrossexTransferRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	opts := &gateapi.CreateCrossexTransferOpts{
		CrossexTransferRequest: optional.NewInterface(body),
	}
	result, httpResp, err := c.CrossExAPI.CreateCrossexTransfer(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/crossex/transfers", jsonStr))
		return nil
	}
	return p.Print(result)
}

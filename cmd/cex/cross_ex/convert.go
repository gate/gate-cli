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

var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "Cross-exchange flash swap / convert commands",
}

func init() {
	quoteCmd := &cobra.Command{
		Use:   "quote",
		Short: "Get a flash swap quote",
		RunE:  runConvertQuote,
	}
	quoteCmd.Flags().String("json", "", "JSON body for convert quote request (required)")
	quoteCmd.MarkFlagRequired("json")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Execute a flash swap order",
		RunE:  runConvertCreate,
	}
	createCmd.Flags().String("json", "", "JSON body for convert order request (required)")
	createCmd.MarkFlagRequired("json")

	convertCmd.AddCommand(quoteCmd, createCmd)
	Cmd.AddCommand(convertCmd)
}

func runConvertQuote(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.CrossexConvertQuoteRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	opts := &gateapi.CreateCrossexConvertQuoteOpts{
		CrossexConvertQuoteRequest: optional.NewInterface(body),
	}
	result, httpResp, err := c.CrossExAPI.CreateCrossexConvertQuote(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/crossex/convert/quote", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runConvertCreate(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.CrossexConvertOrderRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	opts := &gateapi.CreateCrossexConvertOrderOpts{
		CrossexConvertOrderRequest: optional.NewInterface(body),
	}
	result, httpResp, err := c.CrossExAPI.CreateCrossexConvertOrder(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/crossex/convert/order", jsonStr))
		return nil
	}
	return p.Print(result)
}

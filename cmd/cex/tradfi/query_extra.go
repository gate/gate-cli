package tradfi

import (
	"fmt"

	"github.com/antihax/optional"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/spf13/cobra"
)

// SDK-backed commands share the existing transport and authentication boundary.
func init() {
	tradfiCommissionsCmd := &cobra.Command{Use: "commissions", Short: "Query authenticated TradFi symbol commissions", Args: cobra.NoArgs, RunE: runTradfiCommissions}
	tradfiCommissionsCmd.Flags().String("symbols", "", "Comma-separated symbols; omitted to use server defaults")
	tradfiCommissionsCmd.Flags().String("category-code", "", "Optional category code")
	marketCmd.AddCommand(tradfiCommissionsCmd)
	tradfiOrderLogCmd := &cobra.Command{Use: "log", Short: "Get a TradFi order log entry", Args: cobra.NoArgs, RunE: runTradfiOrderLog}
	tradfiOrderLogCmd.Flags().Int64("log-id", 0, "Positive log ID")
	cmdutil.MustRequireFlags(tradfiOrderLogCmd, "log-id")
	orderCmd.AddCommand(tradfiOrderLogCmd)
}

func runTradfiCommissions(cmd *cobra.Command, args []string) error {
	symbols, _ := cmd.Flags().GetString("symbols")
	categoryCode, _ := cmd.Flags().GetString("category-code")
	opts := &gateapi.QuerySymbolCommissionsOpts{}
	if symbols != "" {
		opts.Symbols = optional.NewString(symbols)
	}
	if categoryCode != "" {
		opts.CategoryCode = optional.NewString(categoryCode)
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.TradFiAPI.QuerySymbolCommissions(c.Context(), opts)
	if err != nil {
		return client.CommandError(err, resp, "GET", "/api/v4/tradfi/symbols/commissions", "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runTradfiOrderLog(cmd *cobra.Command, args []string) error {
	logId, _ := cmd.Flags().GetInt64("log-id")
	if logId <= 0 {
		return fmt.Errorf("--log-id must be positive")
	}

	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	result, resp, err := c.TradFiAPI.QueryOrderLog(c.Context(), logId)
	if err != nil {
		return client.CommandError(err, resp, "GET", fmt.Sprintf("/api/v4/tradfi/orders/log/%d", logId), "")
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

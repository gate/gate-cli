package account

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

func init() {
	detailCmd := &cobra.Command{
		Use:   "detail",
		Short: "Get account detail",
		RunE:  runAccountDetail,
	}

	keysCmd := &cobra.Command{
		Use:   "keys",
		Short: "Get API key information for the current key",
		RunE:  runAccountKeys,
	}

	rateLimitCmd := &cobra.Command{
		Use:   "rate-limit",
		Short: "Get account rate limit tiers",
		RunE:  runAccountRateLimit,
	}

	Cmd.AddCommand(detailCmd, keysCmd, rateLimitCmd)
}

func runAccountDetail(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.AccountAPI.GetAccountDetail(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/account/detail", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"User ID", "Tier", "Account Mode", "Copy Trading Role"},
		[][]string{{
			fmt.Sprintf("%d", result.UserId),
			fmt.Sprintf("%d", result.Tier),
			fmt.Sprintf("%d", result.Key.Mode),
			fmt.Sprintf("%d", result.CopyTradingRole),
		}},
	)
}

func runAccountKeys(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.AccountAPI.GetAccountMainKeys(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/account/keys", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	perms := make([]string, 0, len(result.Perms))
	for _, perm := range result.Perms {
		ro := ""
		if perm.ReadOnly {
			ro = "(ro)"
		}
		perms = append(perms, perm.Name+ro)
	}
	return p.Table(
		[]string{"User ID", "State", "Mode", "Permissions", "Last Access", "Created At"},
		[][]string{{
			fmt.Sprintf("%d", result.UserId),
			fmt.Sprintf("%d", result.State),
			fmt.Sprintf("%d", result.Mode),
			strings.Join(perms, " "),
			result.LastAccess,
			result.CreatedAt,
		}},
	)
}

func runAccountRateLimit(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.AccountAPI.GetAccountRateLimit(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/account/rate_limit", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, r := range result {
		rows[i] = []string{r.Tier, r.Ratio, r.MainRatio, r.UpdatedAt}
	}
	return p.Table([]string{"Tier", "Ratio", "Main Ratio", "Updated At"}, rows)
}

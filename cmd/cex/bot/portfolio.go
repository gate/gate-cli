package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/spf13/cobra"
)

// decodeBotJSON rejects misspelled fields instead of silently dropping trading intent.
func decodeBotJSON(raw string, target any) error {
	if len(raw) > 64<<10 {
		return fmt.Errorf("bot JSON exceeds 64 KiB")
	}
	if !strings.HasPrefix(strings.TrimSpace(raw), "{") {
		return fmt.Errorf("bot JSON must be an object, not null or an array")
	}
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid Bot JSON: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("bot JSON must contain exactly one object")
	}
	return nil
}

func init() {
	for _, spec := range []struct {
		name, description string
		run               func(*cobra.Command, []string) error
	}{
		{"smart-position", "Create smart_position: USDT amount, 2–10 asset/rate entries; exactly one period (1m–28d) or threshold; optional close_flag (0/1)", runBotSmartPosition},
		{"futures-combo", "Create futures_combo: USDT amount, 2–10 market/side/rate entries, leverage; exactly one period or threshold; optional stop_profit/stop_loss", runBotFuturesCombo},
	} {
		group := &cobra.Command{Use: spec.name, Short: spec.description}
		create := &cobra.Command{Use: "create", Short: spec.description, RunE: spec.run}
		create.Flags().String("json", "", "Typed JSON request with strategy_type and create_params; amounts and rates are strings")
		mustRequireBotFlag(create, "json")
		group.AddCommand(create)
		Cmd.AddCommand(group)
	}
}

func runBotSmartPosition(cmd *cobra.Command, _ []string) error {
	raw, _ := cmd.Flags().GetString("json")
	var body gateapi.AiHubRebalanceCreateRequest
	if err := decodeBotJSON(raw, &body); err != nil {
		return err
	}
	if body.StrategyType != "smart_position" {
		return fmt.Errorf("strategy_type must be smart_position")
	}
	if err := validateSchedule(body.CreateParams.Period, body.CreateParams.Threshold); err != nil {
		return err
	}
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err = c.RequireAuth(); err != nil {
		return err
	}
	// The SDK owns HTTP and credentials; a write is sent once, without automatic retries.
	result, response, callErr := c.BotAPI.PostAIHubSmartPositionCreate(c.Context(), body, nil)
	if err = client.BotResultError(result, callErr, response, "POST", "/api/v4/bot/smart-position/create", raw); err != nil {
		return err
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

func runBotFuturesCombo(cmd *cobra.Command, _ []string) error {
	raw, _ := cmd.Flags().GetString("json")
	var body gateapi.AiHubFuturesComboCreateRequest
	if err := decodeBotJSON(raw, &body); err != nil {
		return err
	}
	if body.StrategyType != "futures_combo" {
		return fmt.Errorf("strategy_type must be futures_combo")
	}
	if err := validateSchedule(body.CreateParams.Period, body.CreateParams.Threshold); err != nil {
		return err
	}
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err = c.RequireAuth(); err != nil {
		return err
	}
	result, response, callErr := c.BotAPI.PostAIHubFuturesComboCreate(c.Context(), body, nil)
	if err = client.BotResultError(result, callErr, response, "POST", "/api/v4/bot/futures-combo/create", raw); err != nil {
		return err
	}
	return cmdutil.GetPrinter(cmd).Print(result)
}

// These checks prevent ambiguous instructions; financial limits and ownership remain upstream-owned.
func validateSchedule(period, threshold string) error {
	if (period == "") == (threshold == "") {
		return fmt.Errorf("provide exactly one of period or threshold")
	}
	return nil
}
func validateBotContractStop(body gateapi.ContractMartingaleCreateParams) error {
	if (body.StopLossPrice == "") != (body.StopLossPriceType == "") {
		return fmt.Errorf("stop_loss_price and stop_loss_price_type must be supplied together")
	}
	switch body.StopLossPriceType {
	case "", "last", "index", "mark":
		return nil
	default:
		return fmt.Errorf("stop_loss_price_type must be last, index or mark")
	}
}
func validateBotStop(body gateapi.AiHubPortfolioStopRequest) error {
	if body.StrategyType != gateapi.StrategyType("smart_position") {
		if body.StopMode != "" || body.StopOptions != nil {
			return fmt.Errorf("stop_mode/stop_options are only valid for smart_position")
		}
		return nil
	}
	switch body.StopMode {
	case "market", "no_close":
		if body.StopOptions != nil {
			return fmt.Errorf("stop_options requires limit mode")
		}
	case "limit":
		if body.StopOptions == nil || body.StopOptions.Price == "" || body.StopOptions.Expiration <= 0 {
			return fmt.Errorf("limit mode requires stop_options.price and positive expiration")
		}
	default:
		return fmt.Errorf("smart_position requires stop-mode market, limit or no_close")
	}
	return nil
}

// Missing a declared required flag is a programmer error: fail during command
// registration rather than silently making a trading parameter optional.
func mustRequireBotFlag(cmd *cobra.Command, name string) {
	if err := cmd.MarkFlagRequired(name); err != nil {
		panic(err)
	}
}

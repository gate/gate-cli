//go:build integration

package integration_test

import (
	"net/http"
	"testing"

	"github.com/antihax/optional"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/integration"
	gateapi "github.com/gate/gateapi-go/v7"
)

// skipOnError skips the test if the API call fails (e.g. endpoint not available, permission denied).
// Use for endpoints that may not be accessible with the test key.
func skipOnError(t *testing.T, err error, httpResp *http.Response) {
	t.Helper()
	if err != nil {
		status := 0
		if httpResp != nil {
			status = httpResp.StatusCode
		}
		t.Skipf("API not available (status=%d): %v", status, err)
	}
}

// ============================================================
// Public API Tests (should always pass, no auth needed)
// ============================================================

func TestMarginUniPairs(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.MarginUniAPI.ListUniCurrencyPairs(c.Context())
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	assert.NotEmpty(t, result)
	t.Logf("uni currency pairs: %d", len(result))
}

func TestUnifiedDiscountTiers(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.UnifiedAPI.ListCurrencyDiscountTiers(c.Context())
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	t.Logf("discount tiers: %d", len(result))
}

func TestUnifiedLoanTiers(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.UnifiedAPI.ListLoanMarginTiers(c.Context())
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	t.Logf("loan margin tiers: %d", len(result))
}

func TestEarnDualPlans(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.EarnAPI.ListDualInvestmentPlans(c.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	t.Logf("dual investment plans: %d", len(result))
}

func TestEarnUniCurrencies(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.EarnUniAPI.ListUniCurrencies(c.Context())
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	assert.NotEmpty(t, result)
	t.Logf("uni earn currencies: %d", len(result))
}

func TestEarnUniRate(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.EarnUniAPI.ListUniRate(c.Context())
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	t.Logf("uni rates: %d", len(result))
}

func TestFlashSwapPairs(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.FlashSwapAPI.ListFlashSwapCurrencyPair(c.Context(), nil)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	assert.NotEmpty(t, result)
	t.Logf("flash swap pairs: %d", len(result))
}

func TestSpotTickerStillWorks(t *testing.T) {
	c := integration.LoadClient(t)
	tickers, httpResp, err := c.SpotAPI.ListTickers(c.Context(), &gateapi.ListTickersOpts{
		CurrencyPair: optional.NewString("BTC_USDT"),
	})
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	assert.NotEmpty(t, tickers)
	t.Logf("spot BTC_USDT last: %s", tickers[0].Last)
}

func TestFuturesContractsStillWork(t *testing.T) {
	c := integration.LoadClient(t)
	contracts, httpResp, err := c.FuturesAPI.ListFuturesContracts(c.Context(), "usdt", nil)
	require.NoError(t, err)
	assert.Equal(t, 200, httpResp.StatusCode)
	assert.NotEmpty(t, contracts)
	t.Logf("futures contracts: %d", len(contracts))
}

// ============================================================
// Auth-Required API Tests (skip if key lacks permission)
// ============================================================

func TestMarginListAccounts(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.MarginAPI.ListMarginAccounts(c.Context(), nil)
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	t.Logf("margin accounts: %d", len(result))
}

func TestMarginAutoRepayStatus(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.MarginAPI.GetAutoRepayStatus(c.Context())
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	t.Logf("auto repay status: %s", result.Status)
}

func TestMarginFundingAccounts(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.MarginAPI.ListFundingAccounts(c.Context(), nil)
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	t.Logf("funding accounts: %d", len(result))
}

func TestUnifiedAccounts(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.UnifiedAPI.ListUnifiedAccounts(c.Context(), nil)
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	t.Logf("unified total: %s", result.Total)
}

func TestUnifiedMode(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.UnifiedAPI.GetUnifiedMode(c.Context())
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	t.Logf("unified mode: %s", result.Mode)
}

func TestUnifiedCurrencies(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.UnifiedAPI.ListUnifiedCurrencies(c.Context(), nil)
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	t.Logf("unified currencies: %d", len(result))
}

func TestSubAccountList(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.SubAccountAPI.ListSubAccounts(c.Context(), nil)
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	t.Logf("sub accounts: %d", len(result))
}

func TestEarnAutoInvestCoins(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.EarnAPI.ListAutoInvestCoins(c.Context(), nil)
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	t.Logf("auto invest coins: %d", len(result))
}

func TestEarnFixedTermProducts(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.EarnAPI.ListEarnFixedTermProducts(c.Context(), 1, 10, nil)
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	_ = result
	t.Logf("fixed term products returned")
}

func TestMclCurrencies(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.MultiCollateralLoanAPI.ListMultiCollateralCurrencies(c.Context())
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	_ = result
	t.Logf("mcl currencies returned")
}

func TestMclLtv(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.MultiCollateralLoanAPI.GetMultiCollateralLtv(c.Context())
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	_ = result
	t.Logf("mcl LTV returned")
}

func TestCrossExSymbols(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.CrossExAPI.ListCrossexRuleSymbols(c.Context(), nil)
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	t.Logf("cross-ex symbols: %d", len(result))
}

func TestCrossExAccount(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.CrossExAPI.GetCrossexAccount(c.Context(), nil)
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	_ = result
	t.Logf("cross-ex account returned")
}

func TestP2pAdsList(t *testing.T) {
	c := integration.LoadClient(t)
	body := gateapi.AdsListRequest{
		TradeType: "buy",
		Asset:     "USDT",
		FiatUnit:  "CNY",
	}
	result, httpResp, err := c.P2pAPI.P2pMerchantBooksAdsList(c.Context(), body)
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	_ = result
	t.Logf("p2p ads returned")
}

func TestRebateUserInfo(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.RebateAPI.RebateUserInfo(c.Context())
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	_ = result
	t.Logf("rebate user info returned")
}

func TestAccountDetail(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.AccountAPI.GetAccountDetail(c.Context())
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	t.Logf("account user_id: %d", result.UserId)
}

func TestActivityTypes(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.ActivityAPI.ListActivityTypes(c.Context())
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	_ = result
	t.Logf("activity types returned")
}

func TestCouponList(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.CouponAPI.ListUserCoupons(c.Context(), nil)
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	_ = result
	t.Logf("coupons returned")
}

func TestLaunchProjects(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.LaunchAPI.ListLaunchPoolProjects(c.Context(), nil)
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	_ = result
	t.Logf("launch projects returned")
}

func TestSquareAiSearch(t *testing.T) {
	c := integration.LoadClient(t)
	opts := &gateapi.ListSquareAiSearchOpts{
		Keyword: optional.NewString("BTC"),
	}
	result, httpResp, err := c.SquareAPI.ListSquareAiSearch(c.Context(), opts)
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	_ = result
	t.Logf("square ai search returned")
}

func TestWelfareIdentity(t *testing.T) {
	c := integration.LoadClient(t)
	result, httpResp, err := c.WelfareAPI.GetUserIdentity(c.Context())
	skipOnError(t, err, httpResp)
	assert.Equal(t, 200, httpResp.StatusCode)
	_ = result
	t.Logf("welfare identity returned")
}

// ============================================================
// Withdrawal: READ-ONLY check, never execute real withdrawal
// ============================================================

func TestWithdrawalClientReady(t *testing.T) {
	c := integration.LoadClient(t)
	require.NotNil(t, c.WithdrawalAPI, "WithdrawalAPI should be initialized")
	require.True(t, c.IsAuthenticated(), "client should be authenticated")
}

// ============================================================
// Cross-Module: All 17 new API fields initialized
// ============================================================

func TestAllNewAPIsInitialized(t *testing.T) {
	c := integration.LoadClient(t)
	assert.NotNil(t, c.MarginAPI)
	assert.NotNil(t, c.MarginUniAPI)
	assert.NotNil(t, c.UnifiedAPI)
	assert.NotNil(t, c.SubAccountAPI)
	assert.NotNil(t, c.EarnAPI)
	assert.NotNil(t, c.EarnUniAPI)
	assert.NotNil(t, c.FlashSwapAPI)
	assert.NotNil(t, c.MultiCollateralLoanAPI)
	assert.NotNil(t, c.CrossExAPI)
	assert.NotNil(t, c.P2pAPI)
	assert.NotNil(t, c.RebateAPI)
	assert.NotNil(t, c.WithdrawalAPI)
	assert.NotNil(t, c.ActivityAPI)
	assert.NotNil(t, c.CouponAPI)
	assert.NotNil(t, c.LaunchAPI)
	assert.NotNil(t, c.SquareAPI)
	assert.NotNil(t, c.WelfareAPI)
}

package client

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/oauth2"
	"github.com/gate/gate-cli/internal/operations"

	gateapi "github.com/gate/gateapi-go/v7"

	"github.com/gate/gate-cli/internal/config"
	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/useragent"
	"github.com/gate/gate-cli/internal/version"
)

// Client wraps the Gate SDK API client with auth state tracking.
type Client struct {
	SpotAPI                *gateapi.SpotApiService
	FuturesAPI             *gateapi.FuturesApiService
	TradFiAPI              *gateapi.CFDApiService
	StockAPI               *gateapi.StockApiService
	AlphaAPI               *gateapi.AlphaApiService
	AccountAPI             *gateapi.AccountApiService
	WalletAPI              *gateapi.WalletApiService
	OptionsAPI             *gateapi.OptionsApiService
	DeliveryAPI            *gateapi.DeliveryApiService
	MarginAPI              *gateapi.MarginApiService
	MarginUniAPI           *gateapi.MarginUniApiService
	UnifiedAPI             *gateapi.UnifiedApiService
	EarnAPI                *gateapi.EarnApiService
	EarnUniAPI             *gateapi.EarnUniApiService
	FlashSwapAPI           *gateapi.FlashSwapApiService
	MultiCollateralLoanAPI *gateapi.MultiCollateralLoanApiService
	CrossExAPI             *gateapi.CrossExApiService
	P2pAPI                 *gateapi.P2pApiService
	RebateAPI              *gateapi.RebateApiService
	WithdrawalAPI          *gateapi.WithdrawalApiService
	ActivityAPI            *gateapi.ActivityApiService
	CouponAPI              *gateapi.CouponApiService
	SquareAPI              *gateapi.SquareApiService
	WelfareAPI             *gateapi.WelfareApiService
	AssetswapAPI           *gateapi.AssetswapApiService
	BotAPI                 *gateapi.BotApiService
	ctx                    context.Context
	auth                   bool
	oauthAvailable         bool
	resource               string
	userAgent              string
	defaultHeaders         map[string]string

	dualMu    sync.Mutex
	dualCache map[string]bool // settle → isDualMode
}

// New creates a Gate API client from the resolved config.
// cmdPath is the slash-separated command path (e.g. "spot/order/create").
func New(cfg *config.Config, cmdPath string) (*Client, error) {
	return NewWithContext(context.Background(), cfg, cmdPath)
}

// NewWithContext 保留统一SDK构造入口，命令取消直接传给每个API调用。
func NewWithContext(ctx context.Context, cfg *config.Config, cmdPath string) (*Client, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.gateio.ws"
	}
	resource, err := oauth2.APIResource(base)
	if err != nil {
		return nil, err
	}
	hmac := cfg.APIKey != "" && cfg.APISecret != ""
	available := cfg.OAuth2.Token != nil || len(cfg.OAuth2.Tokens) > 0
	gateCfg := gateapi.NewConfiguration()

	sdkUA := gateCfg.UserAgent
	gateCfg.UserAgent = useragent.Build(cmdPath, sdkUA)

	// Custom headers for backend analytics
	agent := useragent.Detect()
	gateCfg.DefaultHeader["X-Gate-Agent"] = agent.Name
	gateCfg.DefaultHeader["X-Gate-Agent-Version"] = agent.Extra
	gateCfg.DefaultHeader["X-Gate-Cli-Name"] = "gate-cli"
	gateCfg.DefaultHeader["X-Gate-Cli-Version"] = version.Version

	if cfg.BaseURL != "" {
		gateCfg.BasePath = cfg.BaseURL + "/api/v4"
	}
	if cfg.APIKey != "" && cfg.APISecret != "" {
		gateCfg.Key = cfg.APIKey
		gateCfg.Secret = cfg.APISecret
	}
	// SDK Debug会Dump完整认证头/请求体；集中到安全transport只输出固定摘要。
	gateCfg.Debug = false
	gateCfg.BasePath = resource
	gateCfg.HTTPClient = &http.Client{Timeout: 30 * time.Second, Transport: &apiTransport{resource: resource, hmac: hmac, oauth: available, debug: cfg.Debug, base: sharedAPITransport}, CheckRedirect: apiRedirect}

	apiClient := gateapi.NewAPIClient(gateCfg)

	return &Client{
		SpotAPI:                apiClient.SpotApi,
		FuturesAPI:             apiClient.FuturesApi,
		TradFiAPI:              apiClient.CFDApi,
		StockAPI:               apiClient.StockApi,
		AlphaAPI:               apiClient.AlphaApi,
		AccountAPI:             apiClient.AccountApi,
		WalletAPI:              apiClient.WalletApi,
		OptionsAPI:             apiClient.OptionsApi,
		DeliveryAPI:            apiClient.DeliveryApi,
		MarginAPI:              apiClient.MarginApi,
		MarginUniAPI:           apiClient.MarginUniApi,
		UnifiedAPI:             apiClient.UnifiedApi,
		EarnAPI:                apiClient.EarnApi,
		EarnUniAPI:             apiClient.EarnUniApi,
		FlashSwapAPI:           apiClient.FlashSwapApi,
		MultiCollateralLoanAPI: apiClient.MultiCollateralLoanApi,
		CrossExAPI:             apiClient.CrossExApi,
		P2pAPI:                 apiClient.P2pApi,
		RebateAPI:              apiClient.RebateApi,
		WithdrawalAPI:          apiClient.WithdrawalApi,
		ActivityAPI:            apiClient.ActivityApi,
		CouponAPI:              apiClient.CouponApi,
		SquareAPI:              apiClient.SquareApi,
		WelfareAPI:             apiClient.WelfareApi,
		AssetswapAPI:           apiClient.AssetswapApi,
		BotAPI:                 apiClient.BotApi,
		ctx:                    ctx,
		oauthAvailable:         available, resource: resource,
		auth:           cfg.APIKey != "" && cfg.APISecret != "",
		userAgent:      gateCfg.UserAgent,
		defaultHeaders: gateCfg.DefaultHeader,
		dualCache:      make(map[string]bool),
	}, nil
}

// Context returns the background context for API calls.
func (c *Client) Context() context.Context {
	return c.ctx
}

// IsAuthenticated returns true when API key + secret are configured.
func (c *Client) IsAuthenticated() bool {
	return c.auth
}

// UserAgent returns the User-Agent string sent with every API request.
func (c *Client) UserAgent() string {
	return c.userAgent
}

// DefaultHeader returns the value of a default HTTP header, or empty string if not set.
func (c *Client) DefaultHeader(key string) string {
	return c.defaultHeaders[key]
}

// RequireAuth returns an error if the client has no API credentials.
func (c *Client) RequireAuth() error {
	if c.auth {
		return nil
	}
	if c.oauthAvailable {
		token, err := oauth2.LoadToken()
		if err != nil {
			return err
		}
		if token != nil && token.AccessToken != "" && token.Resource == c.resource && oauth2.NormalizeDiscoveryURL(token.DiscoveryURL) == oauth2.CurrentDiscoveryURL() {
			return nil
		}
	}
	return exitcode.New(4, fmt.Errorf("OAuth2 authorization or API key required — run: gate-cli cex auth login; configure API key only if needed"))
}

// IsDualMode 返回已确认的仓位模式；查询失败必须传播，不能把未知状态当成单向模式。
// 仅缓存成功结果，锁只保护进程内快照，不跨SDK网络请求持有。
func (c *Client) IsDualMode(settle string) (bool, error) {
	c.dualMu.Lock()
	value, ok := c.dualCache[settle]
	c.dualMu.Unlock()
	if ok {
		return value, nil
	}
	acc, response, err := c.FuturesAPI.ListFuturesAccounts(c.ctx, settle)
	if err != nil {
		return false, CommandError(err, response, "GET", "/api/v4/futures/"+settle+"/accounts", "")
	}
	c.dualMu.Lock()
	c.dualCache[settle] = acc.InDualMode
	c.dualMu.Unlock()
	return acc.InDualMode, nil
}

// GetFuturesPosition returns open position(s) for a contract.
//
// In single mode the slice has at most one element.
// In dual mode the slice has up to two elements (long side, short side);
// positions with size "0" are included to preserve the full dual-mode snapshot.
func (c *Client) GetFuturesPosition(settle, contract string) ([]gateapi.Position, *http.Response, error) {
	isDual, err := c.IsDualMode(settle)
	if err != nil {
		return nil, nil, err
	}
	if isDual {
		return c.FuturesAPI.GetDualModePosition(c.ctx, settle, contract)
	}
	pos, resp, err := c.FuturesAPI.GetPosition(c.ctx, settle, contract)
	if err != nil {
		return nil, resp, err
	}
	return []gateapi.Position{pos}, resp, nil
}

// UpdateFuturesPositionLeverage sets leverage for a contract.
// In dual mode uses UpdateDualModePositionLeverage (returns []Position).
// In single mode uses UpdatePositionLeverage (returns Position, wrapped in a slice).
func (c *Client) UpdateFuturesPositionLeverage(settle, contract, leverage string, opts *gateapi.UpdatePositionLeverageOpts) ([]gateapi.Position, *http.Response, error) {
	isDual, err := c.IsDualMode(settle)
	if err != nil {
		return nil, nil, err
	}
	if isDual {
		var dualOpts *gateapi.UpdateDualModePositionLeverageOpts
		if opts != nil {
			dualOpts = &gateapi.UpdateDualModePositionLeverageOpts{
				CrossLeverageLimit: opts.CrossLeverageLimit,
			}
		}
		return c.FuturesAPI.UpdateDualModePositionLeverage(c.ctx, settle, contract, leverage, dualOpts)
	}
	pos, resp, err := c.FuturesAPI.UpdatePositionLeverage(c.ctx, settle, contract, leverage, opts)
	if err != nil {
		return nil, resp, err
	}
	return []gateapi.Position{pos}, resp, nil
}

// ParseGateError converts a Gate SDK error + HTTP response into a GateError.
// It extracts: status code, Gate label/message, x-gate-trace-id header, and request info.
func ParseGateError(err error, httpResp *http.Response, method, url, body string) *output.GateError {
	// SDK errors have value receivers; a typed nil pointer must not reach Error().
	switch e := err.(type) {
	case *gateapi.GateAPIError:
		if e == nil {
			err = nil
		}
	case *gateapi.GenericOpenAPIError:
		if e == nil {
			err = nil
		}
	}
	gateErr := &output.GateError{
		Status:  500,
		Message: "unknown API error",
		Request: &output.RequestInfo{Method: method, URL: url, Body: body},
	}
	if err != nil {
		gateErr.Message = err.Error()
	}

	if httpResp != nil {
		gateErr.Status = httpResp.StatusCode
		gateErr.TraceID = httpResp.Header.Get("x-gate-trace-id")
	}

	// GateAPIError carries structured label + message from Gate standard error format.
	var apiErr gateapi.GateAPIError
	var apiPtr *gateapi.GateAPIError
	if errors.As(err, &apiErr) || (errors.As(err, &apiPtr) && apiPtr != nil) {
		if apiPtr != nil {
			apiErr = *apiPtr
		}
		gateErr.Label = apiErr.Label
		gateErr.Message = apiErr.GetMessage()
		return gateErr
	}

	// GenericOpenAPIError: HTTP-level error without Gate structure.
	var genErr gateapi.GenericOpenAPIError
	var genPtr *gateapi.GenericOpenAPIError
	if errors.As(err, &genErr) || (errors.As(err, &genPtr) && genPtr != nil) {
		if genPtr != nil {
			genErr = *genPtr
		}
		if httpResp != nil && httpResp.StatusCode >= 200 && httpResp.StatusCode < 300 {
			// Some endpoints return a business rejection with HTTP 200. The SDK
			// retains its body on decode failure; recover only the error contract.
			if label, message, ok := decodeBusinessError(genErr.Body()); ok {
				gateErr.Label, gateErr.Message = label, message
				return gateErr
			}
			if httpResp.StatusCode == http.StatusResetContent {
				// Only SDK 176's declared detail error gets this presentation;
				// unrelated operations retain their existing decoding failures.
				op, lookupErr := operations.Lookup(method, url)
				if lookupErr == nil && op != nil && op.OperationID == "getPriceTriggeredOrder" {
					// Generic SDK errors may embed raw bodies; do not expose them.
					gateErr.Label = "API_ERROR"
					gateErr.Message = "API returned an unrecognized error response"
					return gateErr
				}
			}
			// A successful HTTP status can still contain an incompatible SDK DTO.
			// Keep the decoding cause; replacing it with the body hides the failure
			// and can dump an entire account response into stderr.
			gateErr.Label = "INVALID_RESPONSE"
			gateErr.Message = "cannot decode API response: " + genErr.Error()
		} else {
			gateErr.Message = string(genErr.Body())
		}
	}

	return gateErr
}

// CommandError 保留Gate协议错误并统一失败退出码；新股票和本期CFD路径复用此入口。
func CommandError(err error, response *http.Response, method, path, body string) error {
	return commandError(err, response, &output.RequestInfo{Method: method, URL: path, Body: body})
}

// A missing diagnostic request is distinct from an invented GET/POST. The old
// public entry always supplies its original metadata and retains its behavior.
func commandError(err error, response *http.Response, request *output.RequestInfo) error {
	// 多步骤调用已在真正失败的边界构造错误；不得覆盖其状态、退出码和请求位置，
	// 更不能把模式查询失败误标为后续交易写入结果未知。
	var existing *output.GateError
	if errors.As(err, &existing) {
		return err
	}
	method, path, body := "", "", ""
	if request != nil {
		method, path, body = request.Method, request.URL, request.Body
	}
	detail := ParseGateError(err, response, method, path, body)
	detail.Request = request
	code := 1
	var coded *exitcode.Error
	if errors.As(err, &coded) {
		code = coded.Code
		if code == 4 {
			detail.Status = 401
		}
	}
	var expired *oauth2.TokenExpiredError
	if errors.As(err, &expired) {
		detail.Status = 401
		code = 4
	}
	if detail.Status == 401 || detail.Status == 403 {
		code = 4
	}
	// 权限失败的恢复建议区分CEX与Intel；不因401/403自动改认证或重放业务请求。
	if detail.Status == 401 {
		detail.SuggestedNextAction = "stop retrying; run gate-cli cex auth login --force, or verify the configured API key"
	}
	if detail.Status == 403 {
		detail.SuggestedNextAction = "stop retrying; verify granted scopes and account/API key permissions"
	}
	// 已收到写响应但无法取得可用结果时，服务端可能已提交；先对账/查询，不直接补发。
	if request != nil && method != "GET" && method != "HEAD" && response != nil && ((response.StatusCode >= 200 && response.StatusCode < 300) || response.StatusCode >= 500) && !strings.Contains(detail.Message, "outcome unknown") {
		detail.Message = "write outcome unknown; reconcile before submitting again: " + detail.Message
		detail.SuggestedNextAction = "query the original order or business reference before submitting any further write"
	}
	if request == nil {
		// Preserve upstream facts and auth exit codes even if local catalog
		// diagnostics fail. Unknown operation metadata must never invite replay.
		detail.ErrorType = "UNKNOWN"
		if detail.Status == 401 {
			detail.ErrorType = "AUTH_ERROR"
		}
		if detail.Status == 403 {
			detail.ErrorType = "PERMISSION_DENIED"
		}
		detail.SuggestedNextAction = "operation metadata unavailable; verify the CLI installation and original error; do not resubmit writes automatically"
	}
	return exitcode.New(code, detail)
}

package client

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	gateapi "github.com/gate/gateapi-go/v7"

	"github.com/gate/gate-cli/internal/config"
	"github.com/gate/gate-cli/internal/output"
	"github.com/gate/gate-cli/internal/version"
)

// Client wraps the Gate SDK API client with auth state tracking.
type Client struct {
	SpotAPI    *gateapi.SpotApiService
	FuturesAPI *gateapi.FuturesApiService
	TradFiAPI  *gateapi.TradFiApiService
	AlphaAPI   *gateapi.AlphaApiService
	AccountAPI *gateapi.AccountApiService
	ctx        context.Context
	auth       bool
	userAgent  string

	dualMu    sync.Mutex
	dualCache map[string]bool // settle → isDualMode
}

// New creates a Gate API client from the resolved config.
func New(cfg *config.Config) (*Client, error) {
	gateCfg := gateapi.NewConfiguration()
	gateCfg.UserAgent = "gate-cli/" + version.Version
	if cfg.BaseURL != "" {
		gateCfg.BasePath = cfg.BaseURL + "/api/v4"
	}
	if cfg.APIKey != "" && cfg.APISecret != "" {
		gateCfg.Key = cfg.APIKey
		gateCfg.Secret = cfg.APISecret
	}
	gateCfg.Debug = cfg.Debug

	apiClient := gateapi.NewAPIClient(gateCfg)

	return &Client{
		SpotAPI:    apiClient.SpotApi,
		FuturesAPI: apiClient.FuturesApi,
		TradFiAPI:  apiClient.TradFiApi,
		AlphaAPI:   apiClient.AlphaApi,
		AccountAPI: apiClient.AccountApi,
		ctx:        context.Background(),
		auth:       cfg.APIKey != "" && cfg.APISecret != "",
		userAgent:  gateCfg.UserAgent,
		dualCache:  make(map[string]bool),
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

// RequireAuth returns an error if the client has no API credentials.
func (c *Client) RequireAuth() error {
	if !c.auth {
		return fmt.Errorf("API key and secret required — set GATE_API_KEY/GATE_API_SECRET or run: gate-cli config init")
	}
	return nil
}

// IsDualMode returns true when the futures account uses dual-position (hedge) mode
// for the given settle currency. The result is cached after the first successful call.
func (c *Client) IsDualMode(settle string) bool {
	c.dualMu.Lock()
	defer c.dualMu.Unlock()

	if v, ok := c.dualCache[settle]; ok {
		return v
	}

	acc, _, err := c.FuturesAPI.ListFuturesAccounts(c.ctx, settle)
	if err != nil {
		return false // assume single mode on error
	}
	c.dualCache[settle] = acc.InDualMode
	return acc.InDualMode
}

// GetFuturesPosition returns open position(s) for a contract.
//
// In single mode the slice has at most one element.
// In dual mode the slice has up to two elements (long side, short side);
// positions with size "0" are included to preserve the full dual-mode snapshot.
func (c *Client) GetFuturesPosition(settle, contract string) ([]gateapi.Position, *http.Response, error) {
	if c.IsDualMode(settle) {
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
	if c.IsDualMode(settle) {
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
	gateErr := &output.GateError{
		Status:  500,
		Message: err.Error(),
		Request: &output.RequestInfo{Method: method, URL: url, Body: body},
	}

	if httpResp != nil {
		gateErr.Status = httpResp.StatusCode
		gateErr.TraceID = httpResp.Header.Get("x-gate-trace-id")
	}

	// GateAPIError carries structured label + message from Gate standard error format.
	if apiErr, ok := err.(gateapi.GateAPIError); ok {
		gateErr.Label = apiErr.Label
		gateErr.Message = apiErr.GetMessage()
		return gateErr
	}

	// GenericOpenAPIError: HTTP-level error without Gate structure.
	if genErr, ok := err.(gateapi.GenericOpenAPIError); ok {
		gateErr.Message = string(genErr.Body())
	}

	return gateErr
}

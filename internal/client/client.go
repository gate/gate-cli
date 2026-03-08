package client

import (
	"context"
	"fmt"
	"net/http"

	gateapi "github.com/gate/gateapi-go/v7"

	"github.com/gate/gate-cli/internal/config"
	"github.com/gate/gate-cli/internal/output"
)

// Client wraps the Gate SDK API client with auth state tracking.
type Client struct {
	SpotAPI    *gateapi.SpotApiService
	FuturesAPI *gateapi.FuturesApiService
	ctx        context.Context
	auth       bool
}

// New creates a Gate API client from the resolved config.
func New(cfg *config.Config) (*Client, error) {
	gateCfg := gateapi.NewConfiguration()
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
		ctx:        context.Background(),
		auth:       cfg.APIKey != "" && cfg.APISecret != "",
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

// RequireAuth returns an error if the client has no API credentials.
func (c *Client) RequireAuth() error {
	if !c.auth {
		return fmt.Errorf("API key and secret required — set GATE_API_KEY/GATE_API_SECRET or run: gate-cli config init")
	}
	return nil
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

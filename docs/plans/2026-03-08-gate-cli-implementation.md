# gate-cli Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a Go CLI tool that wraps Gate.io API (spot + futures) with human-friendly table output, JSON output for agents/scripts, and intuitive trading subcommands.

**Architecture:** Cobra command tree with `spot` and `futures` subgroups. Internal packages handle config (file + env + flag priority), Gate API client initialization, and unified output rendering (table/JSON). Error handling captures Gate standard error format (`label`/`message`), `x-gate-trace-id`, and request details.

**Tech Stack:** Go 1.21+, Cobra, Gate Go SDK (`github.com/gate/gateapi-go/v7`), `github.com/olekukonko/tablewriter`, `github.com/spf13/viper` (config), `github.com/stretchr/testify` (tests), goreleaser (release).

---

## Task 1: Project Bootstrap

**Files:**
- Create: `main.go`
- Create: `go.mod`
- Create: `cmd/root.go`
- Create: `cmd/spot/spot.go`
- Create: `cmd/futures/futures.go`

**Step 1: Initialize Go module**

```bash
cd /Users/revil/projects/gate-cli
go mod init github.com/yourname/gate-cli
```

**Step 2: Install core dependencies**

```bash
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
go get github.com/gate/gateapi-go/v7@latest
go get github.com/olekukonko/tablewriter@latest
go get github.com/stretchr/testify@latest
```

**Step 3: Create `main.go`**

```go
package main

import "github.com/yourname/gate-cli/cmd"

func main() {
	cmd.Execute()
}
```

**Step 4: Create `cmd/root.go`**

```go
package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/yourname/gate-cli/cmd/futures"
	"github.com/yourname/gate-cli/cmd/spot"
)

var rootCmd = &cobra.Command{
	Use:   "gate-cli",
	Short: "Gate.io API command-line interface",
	Long:  "gate-cli wraps the Gate.io API for easy use from the terminal and in scripts.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("format", "table", "Output format: table or json")
	rootCmd.PersistentFlags().String("profile", "default", "Config profile to use")
	rootCmd.PersistentFlags().Bool("debug", false, "Print raw HTTP request/response")

	rootCmd.AddCommand(spot.Cmd)
	rootCmd.AddCommand(futures.Cmd)
}
```

**Step 5: Create `cmd/spot/spot.go`**

```go
package spot

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "spot",
	Short: "Spot trading commands",
}
```

**Step 6: Create `cmd/futures/futures.go`**

```go
package futures

import "github.com/spf13/cobra"

var Cmd = &cobra.Command{
	Use:   "futures",
	Short: "Futures trading commands",
}
```

**Step 7: Verify it builds and runs**

```bash
go build -o gate-cli .
./gate-cli --help
```

Expected output: help text showing `spot` and `futures` subcommands.

**Step 8: Commit**

```bash
git init
git add .
git commit -m "feat: bootstrap project with cobra root, spot and futures command groups"
```

---

## Task 2: Config Package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write failing tests**

```go
// internal/config/config_test.go
package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourname/gate-cli/internal/config"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgFile, []byte(`
default_profile: main
default_settle: usdt
profiles:
  main:
    api_key: "key123"
    api_secret: "secret456"
    base_url: "https://api.gateio.ws"
`), 0600)

	cfg, err := config.Load(config.Options{ConfigFile: cfgFile, Profile: "main"})
	require.NoError(t, err)
	assert.Equal(t, "key123", cfg.APIKey)
	assert.Equal(t, "secret456", cfg.APISecret)
	assert.Equal(t, "https://api.gateio.ws", cfg.BaseURL)
	assert.Equal(t, "usdt", cfg.DefaultSettle)
}

func TestEnvVarOverridesFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgFile, []byte(`
profiles:
  default:
    api_key: "file-key"
    api_secret: "file-secret"
`), 0600)

	t.Setenv("GATE_API_KEY", "env-key")
	t.Setenv("GATE_API_SECRET", "env-secret")

	cfg, err := config.Load(config.Options{ConfigFile: cfgFile, Profile: "default"})
	require.NoError(t, err)
	assert.Equal(t, "env-key", cfg.APIKey)
	assert.Equal(t, "env-secret", cfg.APISecret)
}

func TestFlagOverridesEnv(t *testing.T) {
	t.Setenv("GATE_API_KEY", "env-key")

	cfg, err := config.Load(config.Options{
		Profile:   "default",
		FlagAPIKey: "flag-key",
	})
	require.NoError(t, err)
	assert.Equal(t, "flag-key", cfg.APIKey)
}

func TestDefaultBaseURL(t *testing.T) {
	cfg, err := config.Load(config.Options{Profile: "default"})
	require.NoError(t, err)
	assert.Equal(t, "https://api.gateio.ws", cfg.BaseURL)
}
```

**Step 2: Run to verify failure**

```bash
go test ./internal/config/... -v
```

Expected: compile error (package doesn't exist yet).

**Step 3: Implement `internal/config/config.go`**

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const defaultBaseURL = "https://api.gateio.ws"
const defaultSettle = "usdt"

type Options struct {
	ConfigFile  string
	Profile     string
	FlagAPIKey  string
	FlagAPISecret string
}

type Config struct {
	APIKey        string
	APISecret     string
	BaseURL       string
	DefaultSettle string
	Debug         bool
}

type fileConfig struct {
	DefaultProfile string             `yaml:"default_profile"`
	DefaultSettle  string             `yaml:"default_settle"`
	Profiles       map[string]profile `yaml:"profiles"`
}

type profile struct {
	APIKey    string `yaml:"api_key"`
	APISecret string `yaml:"api_secret"`
	BaseURL   string `yaml:"base_url"`
}

func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".gate-cli", "config.yaml")
}

func Load(opts Options) (*Config, error) {
	cfg := &Config{
		BaseURL:       defaultBaseURL,
		DefaultSettle: defaultSettle,
	}

	// Load from file
	cfgPath := opts.ConfigFile
	if cfgPath == "" {
		cfgPath = DefaultConfigPath()
	}

	if data, err := os.ReadFile(cfgPath); err == nil {
		var fc fileConfig
		if err := yaml.Unmarshal(data, &fc); err != nil {
			return nil, fmt.Errorf("invalid config file: %w", err)
		}
		if fc.DefaultSettle != "" {
			cfg.DefaultSettle = fc.DefaultSettle
		}
		profileName := opts.Profile
		if profileName == "" || profileName == "default" {
			profileName = fc.DefaultProfile
		}
		if p, ok := fc.Profiles[profileName]; ok {
			cfg.APIKey = p.APIKey
			cfg.APISecret = p.APISecret
			if p.BaseURL != "" {
				cfg.BaseURL = p.BaseURL
			}
		}
	}

	// Env vars override file
	if v := os.Getenv("GATE_API_KEY"); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv("GATE_API_SECRET"); v != "" {
		cfg.APISecret = v
	}
	if v := os.Getenv("GATE_BASE_URL"); v != "" {
		cfg.BaseURL = v
	}

	// CLI flags override env
	if opts.FlagAPIKey != "" {
		cfg.APIKey = opts.FlagAPIKey
	}
	if opts.FlagAPISecret != "" {
		cfg.APISecret = opts.FlagAPISecret
	}

	return cfg, nil
}
```

**Step 4: Add yaml dependency**

```bash
go get gopkg.in/yaml.v3
```

**Step 5: Run tests to verify they pass**

```bash
go test ./internal/config/... -v
```

Expected: all tests PASS.

**Step 6: Commit**

```bash
git add internal/config/
git commit -m "feat: add config package with file/env/flag priority loading"
```

---

## Task 3: Output Package

**Files:**
- Create: `internal/output/output.go`
- Create: `internal/output/output_test.go`

**Step 1: Write failing tests**

```go
// internal/output/output_test.go
package output_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourname/gate-cli/internal/output"
)

func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	p := output.New(&buf, output.FormatJSON)

	data := map[string]string{"currency": "BTC", "available": "0.1"}
	err := p.Print(data)
	require.NoError(t, err)

	var result map[string]string
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "BTC", result["currency"])
}

func TestTableOutput(t *testing.T) {
	var buf bytes.Buffer
	p := output.New(&buf, output.FormatTable)

	rows := [][]string{{"BTC", "0.1", "0.0"}, {"USDT", "1000", "0"}}
	err := p.Table([]string{"Currency", "Available", "Locked"}, rows)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Currency")
	assert.Contains(t, out, "BTC")
	assert.Contains(t, out, "USDT")
}

func TestErrorJSON(t *testing.T) {
	var buf bytes.Buffer
	p := output.New(&buf, output.FormatJSON)

	gateErr := &output.GateError{
		Status:  400,
		Label:   "INVALID_PARAM_VALUE",
		Message: "Invalid currency pair",
		TraceID: "abc123",
		Request: &output.RequestInfo{
			Method: "POST",
			URL:    "https://api.gateio.ws/api/v4/spot/orders",
			Body:   `{"currency_pair":"INVALID"}`,
		},
	}
	p.PrintError(gateErr)

	var result map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	errObj := result["error"].(map[string]interface{})
	assert.Equal(t, float64(400), errObj["status"])
	assert.Equal(t, "INVALID_PARAM_VALUE", errObj["label"])
	assert.Equal(t, "abc123", errObj["trace_id"])
}
```

**Step 2: Run to verify failure**

```bash
go test ./internal/output/... -v
```

Expected: compile error.

**Step 3: Implement `internal/output/output.go`**

```go
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/olekukonko/tablewriter"
)

type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
)

type RequestInfo struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Body   string `json:"body,omitempty"`
}

type GateError struct {
	Status  int          `json:"status"`
	Label   string       `json:"label,omitempty"`
	Message string       `json:"message"`
	TraceID string       `json:"trace_id,omitempty"`
	Request *RequestInfo `json:"request,omitempty"`
}

type Printer struct {
	out    io.Writer
	errOut io.Writer
	format Format
}

func New(out io.Writer, format Format) *Printer {
	return &Printer{out: out, errOut: os.Stderr, format: format}
}

func NewWithStderr(out, errOut io.Writer, format Format) *Printer {
	return &Printer{out: out, errOut: errOut, format: format}
}

func (p *Printer) Print(data interface{}) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(p.out, string(b))
	return err
}

func (p *Printer) Table(headers []string, rows [][]string) error {
	if p.format == FormatJSON {
		// Convert to list of maps
		var result []map[string]string
		for _, row := range rows {
			m := make(map[string]string)
			for i, h := range headers {
				if i < len(row) {
					m[h] = row[i]
				}
			}
			result = append(result, m)
		}
		return p.Print(result)
	}

	tw := tablewriter.NewWriter(p.out)
	tw.SetHeader(headers)
	tw.SetBorder(false)
	tw.SetColumnSeparator("    ")
	tw.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	tw.SetAlignment(tablewriter.ALIGN_LEFT)
	tw.AppendBulk(rows)
	tw.Render()
	return nil
}

func (p *Printer) PrintError(gateErr *GateError) {
	if p.format == FormatJSON {
		out := map[string]interface{}{"error": gateErr}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(p.errOut, string(b))
		return
	}

	// Table mode: human-friendly stderr
	label := gateErr.Label
	if label == "" {
		label = http.StatusText(gateErr.Status)
	}
	fmt.Fprintf(p.errOut, "Error [%d %s]: %s\n", gateErr.Status, label, gateErr.Message)
	if gateErr.TraceID != "" {
		fmt.Fprintf(p.errOut, "Trace ID: %s\n", gateErr.TraceID)
	}
	if gateErr.Request != nil {
		fmt.Fprintf(p.errOut, "Request: %s %s\n", gateErr.Request.Method, gateErr.Request.URL)
	}
}

func ParseFormat(s string) Format {
	if s == "json" {
		return FormatJSON
	}
	return FormatTable
}
```

Note: add `"net/http"` import for `http.StatusText`.

**Step 4: Run tests**

```bash
go test ./internal/output/... -v
```

Expected: all tests PASS.

**Step 5: Commit**

```bash
git add internal/output/
git commit -m "feat: add output package with table/JSON rendering and structured error format"
```

---

## Task 4: Gate API Client Package

**Files:**
- Create: `internal/client/client.go`
- Create: `internal/client/client_test.go`

**Step 1: Write failing test**

```go
// internal/client/client_test.go
package client_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yourname/gate-cli/internal/client"
	"github.com/yourname/gate-cli/internal/config"
)

func TestNewClientNoAuth(t *testing.T) {
	cfg := &config.Config{BaseURL: "https://api.gateio.ws"}
	c, err := client.New(cfg)
	require.NoError(t, err)
	assert.NotNil(t, c)
	assert.False(t, c.IsAuthenticated())
}

func TestNewClientWithAuth(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.gateio.ws",
		APIKey:    "key",
		APISecret: "secret",
	}
	c, err := client.New(cfg)
	require.NoError(t, err)
	assert.True(t, c.IsAuthenticated())
}

func TestRequireAuthFails(t *testing.T) {
	cfg := &config.Config{BaseURL: "https://api.gateio.ws"}
	c, _ := client.New(cfg)
	err := c.RequireAuth()
	assert.ErrorContains(t, err, "API key")
}
```

**Step 2: Run to verify failure**

```bash
go test ./internal/client/... -v
```

Expected: compile error.

**Step 3: Implement `internal/client/client.go`**

```go
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/yourname/gate-cli/internal/config"
	"github.com/yourname/gate-cli/internal/output"
)

type Client struct {
	SpotAPI    *gateapi.SpotApiService
	FuturesAPI *gateapi.FuturesApiService
	ctx        context.Context
	debug      bool
	auth       bool
}

func New(cfg *config.Config) (*Client, error) {
	gateConfig := gateapi.NewConfiguration()
	gateConfig.Host = cfg.BaseURL

	if cfg.APIKey != "" && cfg.APISecret != "" {
		gateConfig.Key = cfg.APIKey
		gateConfig.Secret = cfg.APISecret
	}

	if cfg.Debug {
		gateConfig.HTTPClient = &http.Client{
			Transport: &debugTransport{base: http.DefaultTransport},
		}
	}

	apiClient := gateapi.NewAPIClient(gateConfig)
	ctx := context.Background()

	return &Client{
		SpotAPI:    apiClient.SpotApi,
		FuturesAPI: apiClient.FuturesApi,
		ctx:        ctx,
		debug:      cfg.Debug,
		auth:       cfg.APIKey != "" && cfg.APISecret != "",
	}, nil
}

func (c *Client) Context() context.Context {
	return c.ctx
}

func (c *Client) IsAuthenticated() bool {
	return c.auth
}

func (c *Client) RequireAuth() error {
	if !c.auth {
		return fmt.Errorf("API key and secret required. Set GATE_API_KEY/GATE_API_SECRET or run: gate-cli config init")
	}
	return nil
}

// ParseGateError attempts to extract Gate standard error format from an API error response.
// Returns a GateError populated with status, label, message, trace_id, and request info.
func ParseGateError(err error, method, url, body string) *output.GateError {
	gateErr := &output.GateError{
		Status:  500,
		Message: err.Error(),
		Request: &output.RequestInfo{Method: method, URL: url, Body: body},
	}

	// Gate SDK wraps errors as gateapi.GateAPIError
	if apiErr, ok := err.(gateapi.GateAPIError); ok {
		gateErr.Status = apiErr.Status
		gateErr.Label = apiErr.Label
		gateErr.Message = apiErr.Message
		gateErr.TraceID = apiErr.TraceID
		return gateErr
	}

	// Try to parse generic HTTP error response body
	type genericError struct {
		Label   string `json:"label"`
		Message string `json:"message"`
	}
	// If we can extract body from http response, attempt parse
	if resp, ok := err.(interface{ Body() io.Reader }); ok {
		var ge genericError
		if json.NewDecoder(resp.Body()).Decode(&ge) == nil && ge.Label != "" {
			gateErr.Label = ge.Label
			gateErr.Message = ge.Message
		}
	}

	return gateErr
}

type debugTransport struct {
	base http.RoundTripper
}

func (t *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Printf("→ %s %s\n", req.Method, req.URL)
	resp, err := t.base.RoundTrip(req)
	if resp != nil {
		fmt.Printf("← %d\n", resp.StatusCode)
	}
	return resp, err
}
```

**Step 4: Run tests**

```bash
go test ./internal/client/... -v
```

Expected: all tests PASS.

**Step 5: Commit**

```bash
git add internal/client/
git commit -m "feat: add Gate API client wrapper with auth check and error parsing"
```

---

## Task 5: Config CLI Commands

**Files:**
- Create: `cmd/config/config.go`
- Modify: `cmd/root.go` — add config subcommand

**Step 1: Implement `cmd/config/config.go`**

```go
package configcmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yourname/gate-cli/internal/config"
	"gopkg.in/yaml.v3"
)

var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Manage gate-cli configuration",
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactively create config file",
	RunE:  runInit,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured profiles",
	RunE:  runList,
}

var setCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value (api-key, api-secret, base-url)",
	Args:  cobra.ExactArgs(2),
	RunE:  runSet,
}

func init() {
	setCmd.Flags().String("profile", "default", "Profile to update")
	Cmd.AddCommand(initCmd, listCmd, setCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Profile name [default]: ")
	profile, _ := reader.ReadString('\n')
	profile = strings.TrimSpace(profile)
	if profile == "" {
		profile = "default"
	}
	fmt.Print("API Key: ")
	apiKey, _ := reader.ReadString('\n')
	apiKey = strings.TrimSpace(apiKey)
	fmt.Print("API Secret: ")
	apiSecret, _ := reader.ReadString('\n')
	apiSecret = strings.TrimSpace(apiSecret)

	cfgPath := config.DefaultConfigPath()
	os.MkdirAll(filepath.Dir(cfgPath), 0700)

	type profileEntry struct {
		APIKey    string `yaml:"api_key"`
		APISecret string `yaml:"api_secret"`
	}
	type fileLayout struct {
		DefaultProfile string                  `yaml:"default_profile"`
		DefaultSettle  string                  `yaml:"default_settle"`
		Profiles       map[string]profileEntry `yaml:"profiles"`
	}

	fc := fileLayout{
		DefaultProfile: profile,
		DefaultSettle:  "usdt",
		Profiles: map[string]profileEntry{
			profile: {APIKey: apiKey, APISecret: apiSecret},
		},
	}

	data, _ := yaml.Marshal(fc)
	if err := os.WriteFile(cfgPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	fmt.Printf("Config written to %s\n", cfgPath)
	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	data, err := os.ReadFile(config.DefaultConfigPath())
	if err != nil {
		return fmt.Errorf("no config file found. Run: gate-cli config init")
	}
	fmt.Println(string(data))
	return nil
}

func runSet(cmd *cobra.Command, args []string) error {
	// Minimal implementation: print guidance
	fmt.Printf("Set %s = %s in profile %s\n", args[0], args[1], cmd.Flag("profile").Value)
	fmt.Println("(Full file-editing implementation in next iteration)")
	return nil
}
```

**Step 2: Add to root**

In `cmd/root.go`, add:
```go
import configcmd "github.com/yourname/gate-cli/cmd/config"
// in init():
rootCmd.AddCommand(configcmd.Cmd)
```

**Step 3: Build and test manually**

```bash
go build -o gate-cli . && ./gate-cli config --help
```

Expected: shows `init`, `list`, `set` subcommands.

**Step 4: Commit**

```bash
git add cmd/config/ cmd/root.go
git commit -m "feat: add config command with init, list, set subcommands"
```

---

## Task 6: Spot Market Commands

**Files:**
- Create: `cmd/spot/market.go`
- Modify: `cmd/spot/spot.go` — register market subcommand

**Step 1: Create helper for getting printer + client in commands**

Add to `cmd/root.go`:

```go
// GetPrinter returns an output.Printer based on --format flag
func GetPrinter(cmd *cobra.Command) *output.Printer {
	format, _ := cmd.Flags().GetString("format")
	if format == "" {
		format, _ = cmd.Root().PersistentFlags().GetString("format")
	}
	return output.New(os.Stdout, output.ParseFormat(format))
}

// GetClient builds a Gate API client from flags + config
func GetClient(cmd *cobra.Command) (*client.Client, error) {
	profile, _ := cmd.Root().PersistentFlags().GetString("profile")
	debug, _ := cmd.Root().PersistentFlags().GetBool("debug")
	cfg, err := config.Load(config.Options{Profile: profile})
	if err != nil {
		return nil, err
	}
	cfg.Debug = debug
	return client.New(cfg)
}
```

**Step 2: Implement `cmd/spot/market.go`**

```go
package spot

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yourname/gate-cli/cmd"
	"github.com/yourname/gate-cli/internal/client"
)

var marketCmd = &cobra.Command{
	Use:   "market",
	Short: "Spot market data (public)",
}

var marketTickerCmd = &cobra.Command{
	Use:   "ticker",
	Short: "Get ticker for a currency pair",
	RunE:  runSpotTicker,
}

var marketTickersCmd = &cobra.Command{
	Use:   "tickers",
	Short: "Get all spot tickers",
	RunE:  runSpotTickers,
}

var marketOrderbookCmd = &cobra.Command{
	Use:   "orderbook",
	Short: "Get order book for a currency pair",
	RunE:  runSpotOrderbook,
}

var marketTradesCmd = &cobra.Command{
	Use:   "trades",
	Short: "Get recent trades for a currency pair",
	RunE:  runSpotTrades,
}

var marketCandlesticksCmd = &cobra.Command{
	Use:   "candlesticks",
	Short: "Get candlestick data",
	RunE:  runSpotCandlesticks,
}

func init() {
	marketTickerCmd.Flags().String("pair", "", "Currency pair, e.g. BTC_USDT (required)")
	marketTickerCmd.MarkFlagRequired("pair")

	marketOrderbookCmd.Flags().String("pair", "", "Currency pair (required)")
	marketOrderbookCmd.Flags().Int32("depth", 20, "Order book depth")
	marketOrderbookCmd.MarkFlagRequired("pair")

	marketTradesCmd.Flags().String("pair", "", "Currency pair (required)")
	marketTradesCmd.Flags().Int32("limit", 20, "Number of trades")
	marketTradesCmd.MarkFlagRequired("pair")

	marketCandlesticksCmd.Flags().String("pair", "", "Currency pair (required)")
	marketCandlesticksCmd.Flags().String("interval", "1h", "Interval: 10s,1m,5m,15m,30m,1h,4h,8h,1d,7d,30d")
	marketCandlesticksCmd.Flags().Int32("limit", 100, "Number of candlesticks")
	marketCandlesticksCmd.MarkFlagRequired("pair")

	marketCmd.AddCommand(marketTickerCmd, marketTickersCmd, marketOrderbookCmd, marketTradesCmd, marketCandlesticksCmd)
	Cmd.AddCommand(marketCmd)
}

func runSpotTicker(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}

	tickers, _, err := c.SpotAPI.ListTickers(c.Context()).CurrencyPair(pair).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "GET", "/api/v4/spot/tickers", ""))
		return nil
	}
	if len(tickers) == 0 {
		return fmt.Errorf("no ticker found for %s", pair)
	}
	t := tickers[0]
	if p.IsJSON() {
		return p.Print(t)
	}
	return p.Table(
		[]string{"Pair", "Last", "Change %", "Volume", "High 24h", "Low 24h"},
		[][]string{{t.CurrencyPair, t.Last, t.ChangePercentage, t.BaseVolume, t.High24h, t.Low24h}},
	)
}

func runSpotTickers(cmd *cobra.Command, args []string) error {
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}

	tickers, _, err := c.SpotAPI.ListTickers(c.Context()).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "GET", "/api/v4/spot/tickers", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(tickers)
	}
	rows := make([][]string, len(tickers))
	for i, t := range tickers {
		rows[i] = []string{t.CurrencyPair, t.Last, t.ChangePercentage, t.BaseVolume}
	}
	return p.Table([]string{"Pair", "Last", "Change %", "Volume"}, rows)
}

func runSpotOrderbook(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	depth, _ := cmd.Flags().GetInt32("depth")
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}

	ob, _, err := c.SpotAPI.ListOrderBook(c.Context()).CurrencyPair(pair).Depth(depth).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "GET", "/api/v4/spot/order_book", ""))
		return nil
	}
	return p.Print(ob)
}

func runSpotTrades(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}

	trades, _, err := c.SpotAPI.ListTrades(c.Context()).CurrencyPair(pair).Limit(limit).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "GET", "/api/v4/spot/trades", ""))
		return nil
	}
	return p.Print(trades)
}

func runSpotCandlesticks(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	interval, _ := cmd.Flags().GetString("interval")
	limit, _ := cmd.Flags().GetInt32("limit")
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}

	candles, _, err := c.SpotAPI.ListCandlesticks(c.Context()).
		CurrencyPair(pair).Interval(interval).Limit(limit).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "GET", "/api/v4/spot/candlesticks", ""))
		return nil
	}
	return p.Print(candles)
}
```

Add helpers to `cmd/spot/spot.go`:

```go
func getCmdPrinter(cmd *cobra.Command) *output.Printer { return cmd_helpers.GetPrinter(cmd) }
func getCmdClient(cmd *cobra.Command) (*client.Client, error) { return cmd_helpers.GetClient(cmd) }
```

(Or inline — keep it simple, no premature abstraction.)

**Step 3: Build and smoke test**

```bash
go build -o gate-cli . && ./gate-cli spot market ticker --pair BTC_USDT
./gate-cli spot market ticker --pair BTC_USDT --format json
```

**Step 4: Commit**

```bash
git add cmd/spot/
git commit -m "feat: add spot market commands (ticker, tickers, orderbook, trades, candlesticks)"
```

---

## Task 7: Spot Account Commands

**Files:**
- Create: `cmd/spot/account.go`

**Step 1: Implement `cmd/spot/account.go`**

```go
package spot

import (
	"github.com/spf13/cobra"
	"github.com/yourname/gate-cli/internal/client"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Spot account commands",
}

var accountListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all spot account balances",
	RunE:  runSpotAccountList,
}

var accountGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get balance for a specific currency",
	RunE:  runSpotAccountGet,
}

func init() {
	accountGetCmd.Flags().String("currency", "", "Currency symbol, e.g. BTC (required)")
	accountGetCmd.MarkFlagRequired("currency")
	accountCmd.AddCommand(accountListCmd, accountGetCmd)
	Cmd.AddCommand(accountCmd)
}

func runSpotAccountList(cmd *cobra.Command, args []string) error {
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	accounts, _, err := c.SpotAPI.ListSpotAccounts(c.Context()).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "GET", "/api/v4/spot/accounts", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(accounts)
	}
	rows := make([][]string, 0)
	for _, a := range accounts {
		if a.Available != "0" || a.Locked != "0" {
			rows = append(rows, []string{a.Currency, a.Available, a.Locked})
		}
	}
	return p.Table([]string{"Currency", "Available", "Locked"}, rows)
}

func runSpotAccountGet(cmd *cobra.Command, args []string) error {
	currency, _ := cmd.Flags().GetString("currency")
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	accounts, _, err := c.SpotAPI.ListSpotAccounts(c.Context()).Currency(currency).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "GET", "/api/v4/spot/accounts", ""))
		return nil
	}
	return p.Print(accounts)
}
```

**Step 2: Build and smoke test**

```bash
go build -o gate-cli . && ./gate-cli spot account list
```

**Step 3: Commit**

```bash
git add cmd/spot/account.go
git commit -m "feat: add spot account list and get commands"
```

---

## Task 8: Spot Order Commands

**Files:**
- Create: `cmd/spot/order.go`

**Step 1: Implement `cmd/spot/order.go`**

```go
package spot

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/yourname/gate-cli/internal/client"
)

var orderCmd = &cobra.Command{
	Use:   "order",
	Short: "Spot order commands",
}

func init() {
	// buy
	buyCmd := &cobra.Command{Use: "buy", Short: "Place a buy order", RunE: runSpotBuy}
	buyCmd.Flags().String("pair", "", "Currency pair (required)")
	buyCmd.Flags().String("amount", "", "Amount to buy (required)")
	buyCmd.Flags().String("price", "", "Price (omit for market order)")
	buyCmd.MarkFlagRequired("pair")
	buyCmd.MarkFlagRequired("amount")

	// sell
	sellCmd := &cobra.Command{Use: "sell", Short: "Place a sell order", RunE: runSpotSell}
	sellCmd.Flags().String("pair", "", "Currency pair (required)")
	sellCmd.Flags().String("amount", "", "Amount to sell (required)")
	sellCmd.Flags().String("price", "", "Price (omit for market order)")
	sellCmd.MarkFlagRequired("pair")
	sellCmd.MarkFlagRequired("amount")

	// get
	getCmd := &cobra.Command{Use: "get", Short: "Get an order by ID", RunE: runSpotOrderGet}
	getCmd.Flags().String("id", "", "Order ID (required)")
	getCmd.Flags().String("pair", "", "Currency pair (required)")
	getCmd.MarkFlagRequired("id")
	getCmd.MarkFlagRequired("pair")

	// list
	listCmd := &cobra.Command{Use: "list", Short: "List orders", RunE: runSpotOrderList}
	listCmd.Flags().String("pair", "", "Currency pair (required)")
	listCmd.Flags().String("status", "open", "Status: open, closed, cancelled")
	listCmd.MarkFlagRequired("pair")

	// cancel
	cancelCmd := &cobra.Command{Use: "cancel", Short: "Cancel order(s)", RunE: runSpotOrderCancel}
	cancelCmd.Flags().String("id", "", "Order ID")
	cancelCmd.Flags().String("pair", "", "Currency pair (required)")
	cancelCmd.Flags().Bool("all", false, "Cancel all open orders for the pair")
	cancelCmd.MarkFlagRequired("pair")

	orderCmd.AddCommand(buyCmd, sellCmd, getCmd, listCmd, cancelCmd)
	Cmd.AddCommand(orderCmd)
}

func createSpotOrder(cmd *cobra.Command, side, pair, amount, price string) error {
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	order := gateapi.Order{
		CurrencyPair: pair,
		Side:         side,
		Amount:       amount,
		Type:         "limit",
	}
	if price == "" {
		order.Type = "market"
	} else {
		order.Price = price
	}

	result, _, err := c.SpotAPI.CreateOrder(c.Context()).Order(order).Execute()
	if err != nil {
		body, _ := json.Marshal(order)
		p.PrintError(client.ParseGateError(err, "POST", "/api/v4/spot/orders", string(body)))
		return nil
	}
	return p.Print(result)
}

func runSpotBuy(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	amount, _ := cmd.Flags().GetString("amount")
	price, _ := cmd.Flags().GetString("price")
	return createSpotOrder(cmd, "buy", pair, amount, price)
}

func runSpotSell(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	amount, _ := cmd.Flags().GetString("amount")
	price, _ := cmd.Flags().GetString("price")
	return createSpotOrder(cmd, "sell", pair, amount, price)
}

func runSpotOrderGet(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetString("id")
	pair, _ := cmd.Flags().GetString("pair")
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	order, _, err := c.SpotAPI.GetOrder(c.Context(), id, pair).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "GET", fmt.Sprintf("/api/v4/spot/orders/%s", id), ""))
		return nil
	}
	return p.Print(order)
}

func runSpotOrderList(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	status, _ := cmd.Flags().GetString("status")
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	orders, _, err := c.SpotAPI.ListOrders(c.Context(), pair, status).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "GET", "/api/v4/spot/orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(orders)
	}
	rows := make([][]string, len(orders))
	for i, o := range orders {
		rows[i] = []string{o.Id, o.CurrencyPair, o.Side, o.Amount, o.Price, o.Status}
	}
	return p.Table([]string{"ID", "Pair", "Side", "Amount", "Price", "Status"}, rows)
}

func runSpotOrderCancel(cmd *cobra.Command, args []string) error {
	pair, _ := cmd.Flags().GetString("pair")
	all, _ := cmd.Flags().GetBool("all")
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	if all {
		cancelled, _, err := c.SpotAPI.CancelOrders(c.Context()).CurrencyPair(pair).Execute()
		if err != nil {
			p.PrintError(client.ParseGateError(err, "DELETE", "/api/v4/spot/orders", ""))
			return nil
		}
		return p.Print(cancelled)
	}

	id, _ := cmd.Flags().GetString("id")
	if id == "" {
		return fmt.Errorf("provide --id <order-id> or --all")
	}
	result, _, err := c.SpotAPI.CancelOrder(c.Context(), id, pair).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "DELETE", fmt.Sprintf("/api/v4/spot/orders/%s", id), ""))
		return nil
	}
	return p.Print(result)
}
```

**Step 2: Build and smoke test**

```bash
go build -o gate-cli . && ./gate-cli spot order --help
./gate-cli spot order list --pair BTC_USDT --format json
```

**Step 3: Commit**

```bash
git add cmd/spot/order.go
git commit -m "feat: add spot order commands (buy, sell, get, list, cancel)"
```

---

## Task 9: Futures Market Commands

**Files:**
- Create: `cmd/futures/market.go`

Follow the same pattern as Task 6, using `FuturesAPI` instead of `SpotAPI`. All futures commands accept `--settle` (default: `usdt`).

```go
package futures

import (
	"github.com/spf13/cobra"
	"github.com/yourname/gate-cli/internal/client"
)

var marketCmd = &cobra.Command{
	Use:   "market",
	Short: "Futures market data (public)",
}

func getSettle(cmd *cobra.Command) string {
	s, _ := cmd.Flags().GetString("settle")
	if s == "" {
		return "usdt"
	}
	return s
}

func addSettleFlag(cmd *cobra.Command) {
	cmd.Flags().String("settle", "usdt", "Settlement currency: usdt, btc")
}

func init() {
	tickerCmd := &cobra.Command{Use: "ticker", Short: "Get futures ticker", RunE: runFuturesTicker}
	tickerCmd.Flags().String("contract", "", "Contract name, e.g. BTC_USDT (required)")
	tickerCmd.MarkFlagRequired("contract")
	addSettleFlag(tickerCmd)

	tickersCmd := &cobra.Command{Use: "tickers", Short: "Get all futures tickers", RunE: runFuturesTickers}
	addSettleFlag(tickersCmd)

	orderbookCmd := &cobra.Command{Use: "orderbook", Short: "Get futures order book", RunE: runFuturesOrderbook}
	orderbookCmd.Flags().String("contract", "", "Contract name (required)")
	orderbookCmd.MarkFlagRequired("contract")
	addSettleFlag(orderbookCmd)

	tradesCmd := &cobra.Command{Use: "trades", Short: "Get recent futures trades", RunE: runFuturesTrades}
	tradesCmd.Flags().String("contract", "", "Contract name (required)")
	tradesCmd.Flags().Int32("limit", 20, "Number of trades")
	tradesCmd.MarkFlagRequired("contract")
	addSettleFlag(tradesCmd)

	candlesCmd := &cobra.Command{Use: "candlesticks", Short: "Get futures candlesticks", RunE: runFuturesCandlesticks}
	candlesCmd.Flags().String("contract", "", "Contract name (required)")
	candlesCmd.Flags().String("interval", "1h", "Interval")
	candlesCmd.Flags().Int32("limit", 100, "Number of candles")
	candlesCmd.MarkFlagRequired("contract")
	addSettleFlag(candlesCmd)

	fundingCmd := &cobra.Command{Use: "funding-rate", Short: "Get funding rate history", RunE: runFuturesFundingRate}
	fundingCmd.Flags().String("contract", "", "Contract name (required)")
	fundingCmd.Flags().Int32("limit", 20, "Number of records")
	fundingCmd.MarkFlagRequired("contract")
	addSettleFlag(fundingCmd)

	marketCmd.AddCommand(tickerCmd, tickersCmd, orderbookCmd, tradesCmd, candlesCmd, fundingCmd)
	Cmd.AddCommand(marketCmd)
}

func runFuturesTicker(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	settle := getSettle(cmd)
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}
	tickers, _, err := c.FuturesAPI.ListFuturesTickers(c.Context(), settle).Contract(contract).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "GET", "/api/v4/futures/"+settle+"/tickers", ""))
		return nil
	}
	return p.Print(tickers)
}

// ... implement runFuturesTickers, runFuturesOrderbook, runFuturesTrades,
//     runFuturesCandlesticks, runFuturesFundingRate following same pattern
```

**Step 2: Build and smoke test**

```bash
go build -o gate-cli . && ./gate-cli futures market ticker --contract BTC_USDT
```

**Step 3: Commit**

```bash
git add cmd/futures/market.go
git commit -m "feat: add futures market commands (ticker, tickers, orderbook, trades, candlesticks, funding-rate)"
```

---

## Task 10: Futures Account & Position Commands

**Files:**
- Create: `cmd/futures/account.go`
- Create: `cmd/futures/position.go`

**Step 1: Implement account.go**

```go
package futures

import "github.com/spf13/cobra"

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Futures account commands",
}

func init() {
	getCmd := &cobra.Command{Use: "get", Short: "Get futures account", RunE: runFuturesAccountGet}
	addSettleFlag(getCmd)
	accountCmd.AddCommand(getCmd)
	Cmd.AddCommand(accountCmd)
}

func runFuturesAccountGet(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	account, _, err := c.FuturesAPI.GetFuturesAccount(c.Context(), settle).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "GET", "/api/v4/futures/"+settle+"/accounts", ""))
		return nil
	}
	return p.Print(account)
}
```

**Step 2: Implement position.go**

```go
package futures

import "github.com/spf13/cobra"

var positionCmd = &cobra.Command{
	Use:   "position",
	Short: "Futures position commands",
}

func init() {
	listCmd := &cobra.Command{Use: "list", Short: "List open positions", RunE: runFuturesPositionList}
	addSettleFlag(listCmd)

	getCmd := &cobra.Command{Use: "get", Short: "Get a specific position", RunE: runFuturesPositionGet}
	getCmd.Flags().String("contract", "", "Contract name (required)")
	getCmd.MarkFlagRequired("contract")
	addSettleFlag(getCmd)

	positionCmd.AddCommand(listCmd, getCmd)
	Cmd.AddCommand(positionCmd)
}

func runFuturesPositionList(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	positions, _, err := c.FuturesAPI.ListPositions(c.Context(), settle).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "GET", "/api/v4/futures/"+settle+"/positions", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(positions)
	}
	rows := make([][]string, len(positions))
	for i, pos := range positions {
		rows[i] = []string{pos.Contract, pos.Size, pos.EntryPrice, pos.UnrealisedPnl, pos.Mode}
	}
	return p.Table([]string{"Contract", "Size", "Entry Price", "Unrealised PnL", "Mode"}, rows)
}

func runFuturesPositionGet(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	settle := getSettle(cmd)
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	pos, _, err := c.FuturesAPI.GetPosition(c.Context(), settle, contract).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "GET", "/api/v4/futures/"+settle+"/positions/"+contract, ""))
		return nil
	}
	return p.Print(pos)
}
```

**Step 3: Build and smoke test**

```bash
go build -o gate-cli . && ./gate-cli futures position list
```

**Step 4: Commit**

```bash
git add cmd/futures/account.go cmd/futures/position.go
git commit -m "feat: add futures account get and position list/get commands"
```

---

## Task 11: Futures Order Commands

**Files:**
- Create: `cmd/futures/order.go`

Gate Futures uses `size` (integer, positive = long/buy, negative = short/sell) to express direction.

| Command | size sign |
|---------|-----------|
| `long`  | +size     |
| `short` | -size     |
| `add`   | same as current position side |
| `remove`| opposite of current position side |
| `close` | size=0 (close_side=true) |

```go
package futures

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/yourname/gate-cli/internal/client"
)

var orderCmd = &cobra.Command{
	Use:   "order",
	Short: "Futures order commands",
}

func init() {
	// long
	longCmd := newFuturesDirectionCmd("long", "Open a long position")
	// short
	shortCmd := newFuturesDirectionCmd("short", "Open a short position")
	// add
	addCmd := newFuturesDirectionCmd("add", "Add to existing position")
	// remove
	removeCmd := newFuturesDirectionCmd("remove", "Reduce existing position")

	// close
	closeCmd := &cobra.Command{Use: "close", Short: "Close position", RunE: runFuturesClose}
	closeCmd.Flags().String("contract", "", "Contract name (required)")
	closeCmd.Flags().Int64("size", 0, "Partial close size (0 = full close)")
	closeCmd.MarkFlagRequired("contract")
	addSettleFlag(closeCmd)

	// get
	getCmd := &cobra.Command{Use: "get", Short: "Get an order", RunE: runFuturesOrderGet}
	getCmd.Flags().Int64("id", 0, "Order ID (required)")
	getCmd.MarkFlagRequired("id")
	addSettleFlag(getCmd)

	// list
	listCmd := &cobra.Command{Use: "list", Short: "List orders", RunE: runFuturesOrderList}
	listCmd.Flags().String("contract", "", "Contract name")
	listCmd.Flags().String("status", "open", "Status: open, finished")
	addSettleFlag(listCmd)

	// cancel
	cancelCmd := &cobra.Command{Use: "cancel", Short: "Cancel order(s)", RunE: runFuturesOrderCancel}
	cancelCmd.Flags().Int64("id", 0, "Order ID")
	cancelCmd.Flags().String("contract", "", "Contract (required for --all)")
	cancelCmd.Flags().Bool("all", false, "Cancel all open orders for contract")
	addSettleFlag(cancelCmd)

	orderCmd.AddCommand(longCmd, shortCmd, addCmd, removeCmd, closeCmd, getCmd, listCmd, cancelCmd)
	Cmd.AddCommand(orderCmd)
}

func newFuturesDirectionCmd(use, short string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFuturesDirectionOrder(cmd, use)
		},
	}
	cmd.Flags().String("contract", "", "Contract name (required)")
	cmd.Flags().Int64("size", 0, "Number of contracts (required)")
	cmd.Flags().String("price", "", "Price (omit for market order)")
	cmd.MarkFlagRequired("contract")
	cmd.MarkFlagRequired("size")
	addSettleFlag(cmd)
	return cmd
}

func runFuturesDirectionOrder(cmd *cobra.Command, direction string) error {
	contract, _ := cmd.Flags().GetString("contract")
	size, _ := cmd.Flags().GetInt64("size")
	price, _ := cmd.Flags().GetString("price")
	settle := getSettle(cmd)
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	// Convert direction to signed size
	switch direction {
	case "short", "remove":
		size = -size
	}

	order := gateapi.FuturesOrder{
		Contract: contract,
		Size:     size,
	}
	if price == "" {
		order.Price = "0"
		order.Tif = "ioc"
	} else {
		order.Price = price
		order.Tif = "gtc"
	}

	result, _, err := c.FuturesAPI.CreateFuturesOrder(c.Context(), settle).FuturesOrder(order).Execute()
	if err != nil {
		body, _ := json.Marshal(order)
		p.PrintError(client.ParseGateError(err, "POST", "/api/v4/futures/"+settle+"/orders", string(body)))
		return nil
	}
	return p.Print(result)
}

func runFuturesClose(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	size, _ := cmd.Flags().GetInt64("size")
	settle := getSettle(cmd)
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	order := gateapi.FuturesOrder{
		Contract: contract,
		Size:     size,
		Price:    "0",
		Tif:      "ioc",
		Close:    true,
	}

	result, _, err := c.FuturesAPI.CreateFuturesOrder(c.Context(), settle).FuturesOrder(order).Execute()
	if err != nil {
		body, _ := json.Marshal(order)
		p.PrintError(client.ParseGateError(err, "POST", "/api/v4/futures/"+settle+"/orders", string(body)))
		return nil
	}
	return p.Print(result)
}

func runFuturesOrderGet(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt64("id")
	settle := getSettle(cmd)
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	order, _, err := c.FuturesAPI.GetFuturesOrder(c.Context(), settle, fmt.Sprintf("%d", id)).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "GET", fmt.Sprintf("/api/v4/futures/%s/orders/%d", settle, id), ""))
		return nil
	}
	return p.Print(order)
}

func runFuturesOrderList(cmd *cobra.Command, args []string) error {
	contract, _ := cmd.Flags().GetString("contract")
	status, _ := cmd.Flags().GetString("status")
	settle := getSettle(cmd)
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}
	req := c.FuturesAPI.ListFuturesOrders(c.Context(), settle, status)
	if contract != "" {
		req = req.Contract(contract)
	}
	orders, _, err := req.Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "GET", "/api/v4/futures/"+settle+"/orders", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(orders)
	}
	rows := make([][]string, len(orders))
	for i, o := range orders {
		rows[i] = []string{fmt.Sprintf("%d", o.Id), o.Contract, fmt.Sprintf("%d", o.Size), o.Price, o.Status}
	}
	return p.Table([]string{"ID", "Contract", "Size", "Price", "Status"}, rows)
}

func runFuturesOrderCancel(cmd *cobra.Command, args []string) error {
	settle := getSettle(cmd)
	all, _ := cmd.Flags().GetBool("all")
	p := getCmdPrinter(cmd)
	c, err := getCmdClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	if all {
		contract, _ := cmd.Flags().GetString("contract")
		if contract == "" {
			return fmt.Errorf("--contract is required when using --all")
		}
		cancelled, _, err := c.FuturesAPI.CancelFuturesOrders(c.Context(), settle, contract).Execute()
		if err != nil {
			p.PrintError(client.ParseGateError(err, "DELETE", "/api/v4/futures/"+settle+"/orders", ""))
			return nil
		}
		return p.Print(cancelled)
	}

	id, _ := cmd.Flags().GetInt64("id")
	if id == 0 {
		return fmt.Errorf("provide --id <order-id> or --all")
	}
	result, _, err := c.FuturesAPI.CancelFuturesOrder(c.Context(), settle, fmt.Sprintf("%d", id)).Execute()
	if err != nil {
		p.PrintError(client.ParseGateError(err, "DELETE", fmt.Sprintf("/api/v4/futures/%s/orders/%d", settle, id), ""))
		return nil
	}
	return p.Print(result)
}
```

**Step 2: Build and smoke test**

```bash
go build -o gate-cli . && ./gate-cli futures order --help
./gate-cli futures order long --contract BTC_USDT --size 1 --price 80000
```

**Step 3: Commit**

```bash
git add cmd/futures/order.go
git commit -m "feat: add futures order commands (long, short, add, remove, close, get, list, cancel)"
```

---

## Task 12: Release Pipeline

**Files:**
- Create: `.goreleaser.yaml`
- Create: `.github/workflows/release.yaml`

**Step 1: Create `.goreleaser.yaml`**

```yaml
version: 2

project_name: gate-cli

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - windows
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w -X main.version={{.Version}}

archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
```

**Step 2: Create `.github/workflows/release.yaml`**

```yaml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Step 3: Verify goreleaser config locally**

```bash
# Install goreleaser if not present
brew install goreleaser

# Dry run
goreleaser build --snapshot --clean
```

Expected: builds appear in `dist/` for all platforms.

**Step 4: Commit**

```bash
git add .goreleaser.yaml .github/
git commit -m "feat: add goreleaser config and GitHub Actions release pipeline"
```

---

## Verification Checklist

After all tasks complete, run:

```bash
# Build
go build -o gate-cli .

# All public commands (no API key needed)
./gate-cli spot market ticker --pair BTC_USDT
./gate-cli spot market ticker --pair BTC_USDT --format json
./gate-cli futures market ticker --contract BTC_USDT

# Error format
./gate-cli spot market ticker --pair INVALID_PAIR
./gate-cli spot market ticker --pair INVALID_PAIR --format json

# Auth commands (requires real API key)
GATE_API_KEY=xxx GATE_API_SECRET=yyy ./gate-cli spot account list
GATE_API_KEY=xxx GATE_API_SECRET=yyy ./gate-cli futures position list

# Help
./gate-cli --help
./gate-cli spot --help
./gate-cli futures order --help
```

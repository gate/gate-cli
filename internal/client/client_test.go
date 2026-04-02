package client_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/config"
)

func TestNewClientNoAuth(t *testing.T) {
	cfg := &config.Config{BaseURL: "https://api.gateio.ws"}
	c, err := client.New(cfg, "test")
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
	c, err := client.New(cfg, "test")
	require.NoError(t, err)
	assert.True(t, c.IsAuthenticated())
}

func TestNewClientSetsUserAgent(t *testing.T) {
	cfg := &config.Config{BaseURL: "https://api.gateio.ws"}
	c, err := client.New(cfg, "spot/order/create")
	require.NoError(t, err)

	ua := c.UserAgent()
	assert.True(t, strings.HasPrefix(ua, "gate-cli/"), "UserAgent should start with gate-cli/, got: %s", ua)
	assert.Contains(t, ua, "/spot/order/create/", "UserAgent should contain command path")
	assert.Contains(t, ua, "OpenAPI-Generator/", "UserAgent should contain SDK UA")
}

func TestNewClientSetsCustomHeaders(t *testing.T) {
	cfg := &config.Config{BaseURL: "https://api.gateio.ws"}
	c, err := client.New(cfg, "spot/order/create")
	require.NoError(t, err)

	// X-Gate-Cli-Name is always "gate-cli"
	assert.Equal(t, "gate-cli", c.DefaultHeader("X-Gate-Cli-Name"))

	// X-Gate-Cli-Version should be the version string (at least non-empty)
	assert.NotEmpty(t, c.DefaultHeader("X-Gate-Cli-Version"))

	// X-Gate-Agent should be a detected agent name (non-empty)
	assert.NotEmpty(t, c.DefaultHeader("X-Gate-Agent"))

	// X-Gate-Agent-Version should be set (could be "-" for unknown)
	assert.NotEmpty(t, c.DefaultHeader("X-Gate-Agent-Version"))

	// SDK's built-in header should still be present
	assert.Equal(t, "1", c.DefaultHeader("X-Gate-Size-Decimal"))
}

func TestDefaultHeaderReturnsEmptyForMissing(t *testing.T) {
	cfg := &config.Config{BaseURL: "https://api.gateio.ws"}
	c, err := client.New(cfg, "test")
	require.NoError(t, err)

	assert.Empty(t, c.DefaultHeader("X-Nonexistent-Header"))
}

func TestRequireAuthFailsWhenNoKey(t *testing.T) {
	cfg := &config.Config{BaseURL: "https://api.gateio.ws"}
	c, _ := client.New(cfg, "test")
	err := c.RequireAuth()
	assert.ErrorContains(t, err, "API key")
}

func TestRequireAuthSucceedsWhenKeySet(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.gateio.ws",
		APIKey:    "key",
		APISecret: "secret",
	}
	c, _ := client.New(cfg, "test")
	assert.NoError(t, c.RequireAuth())
}

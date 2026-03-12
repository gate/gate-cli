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

func TestNewClientSetsUserAgent(t *testing.T) {
	cfg := &config.Config{BaseURL: "https://api.gateio.ws"}
	c, err := client.New(cfg)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(c.UserAgent(), "gate-cli/"), "UserAgent should start with gate-cli/, got: %s", c.UserAgent())
}

func TestRequireAuthFailsWhenNoKey(t *testing.T) {
	cfg := &config.Config{BaseURL: "https://api.gateio.ws"}
	c, _ := client.New(cfg)
	err := c.RequireAuth()
	assert.ErrorContains(t, err, "API key")
}

func TestRequireAuthSucceedsWhenKeySet(t *testing.T) {
	cfg := &config.Config{
		BaseURL:   "https://api.gateio.ws",
		APIKey:    "key",
		APISecret: "secret",
	}
	c, _ := client.New(cfg)
	assert.NoError(t, c.RequireAuth())
}

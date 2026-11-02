package oauth2

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveAndLoadToken(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	td := &TokenData{
		ClientID:     "pkce_test123",
		DiscoveryURL: "https://example.invalid/.well-known/oauth-authorization-server",
		AccessToken:  "at_xxx",
		RefreshToken: "rt_xxx",
		ExpiresAt:    time.Now().Add(time.Hour).Truncate(time.Second),
		Scopes:       "read spot_trade",
	}

	// 新的 LoadToken 按当前 oauth2.discovery_url 匹配，需要先把 config.yaml
	// 的 discovery_url 对齐到 td.DiscoveryURL，否则 load 回返回 nil。
	ensureActiveDiscoveryURL(t, td.DiscoveryURL)
	require.NoError(t, SaveToken(td))

	loaded, err := LoadToken()
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, td.ClientID, loaded.ClientID)
	assert.Equal(t, td.DiscoveryURL, loaded.DiscoveryURL)
	assert.Equal(t, td.AccessToken, loaded.AccessToken)
	assert.Equal(t, td.RefreshToken, loaded.RefreshToken)
	assert.Equal(t, td.Scopes, loaded.Scopes)
}

func TestSaveToken_AtomicWrite(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	ensureActiveDiscoveryURL(t, "")
	td := &TokenData{ClientID: "test", AccessToken: "at", ExpiresAt: time.Now().Add(time.Hour)}
	require.NoError(t, SaveToken(td))

	// 不应残留 tmp 文件（SaveToken 走 rename 原子写）。
	cfgPath, err := ConfigYAMLPath()
	require.NoError(t, err)
	entries, err := os.ReadDir(filepath.Dir(cfgPath))
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t,
			filepath.Ext(e.Name()) == ".tmp" || contains(e.Name(), ".tmp"),
			"stray tmp file: %s", e.Name(),
		)
	}
}

func TestSaveToken_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission check not applicable on Windows")
	}

	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	ensureActiveDiscoveryURL(t, "")
	td := &TokenData{ClientID: "test", AccessToken: "at", ExpiresAt: time.Now().Add(time.Hour)}
	require.NoError(t, SaveToken(td))

	cfgPath, err := ConfigYAMLPath()
	require.NoError(t, err)
	info, err := os.Stat(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestCheckTokenPermissions_Insecure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission check not applicable on Windows")
	}

	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	// Write with insecure permissions
	path, err := TokenPath()
	require.NoError(t, err)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("oauth2:\n  client_id: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err = CheckTokenPermissions()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insecure permissions")
	assert.Contains(t, err.Error(), "0644")
}

func TestCheckTokenPermissions_FileNotExists(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	err := CheckTokenPermissions()
	assert.NoError(t, err) // no file = no error
}

func TestLoadToken_FileNotExists(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	tok, err := LoadToken()
	assert.NoError(t, err)
	assert.Nil(t, tok)
}

func TestRemoveToken_PreservesClientID(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	writeTestToken(t, &TokenData{
		ClientID:     "pkce_keep_me",
		AccessToken:  "at_delete",
		RefreshToken: "rt_delete",
		ExpiresAt:    time.Now().Add(time.Hour),
		Scopes:       "read",
	})

	require.NoError(t, RemoveToken())

	loaded, err := LoadToken()
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "pkce_keep_me", loaded.ClientID, "client_id should be preserved")
	assert.Empty(t, loaded.AccessToken, "access_token should be cleared")
	assert.Empty(t, loaded.RefreshToken, "refresh_token should be cleared")
	assert.Empty(t, loaded.Scopes, "scopes should be cleared")
}

func TestRemoveToken_PreservesDiscoveryURL(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	writeTestToken(t, &TokenData{
		ClientID:     "pkce_keep_me",
		DiscoveryURL: "https://example.invalid/.well-known/oauth-authorization-server",
		AccessToken:  "at_delete",
		RefreshToken: "rt_delete",
		ExpiresAt:    time.Now().Add(time.Hour),
		Scopes:       "read",
	})

	require.NoError(t, RemoveToken())

	loaded, err := LoadToken()
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t,
		"https://example.invalid/.well-known/oauth-authorization-server",
		loaded.DiscoveryURL,
		"discovery_url should be preserved so next login can reuse the client when discovery matches",
	)
}

func TestTokenData_IsValid(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		td := &TokenData{AccessToken: "at", ExpiresAt: time.Now().Add(time.Hour)}
		assert.True(t, td.IsValid())
	})
	t.Run("expired", func(t *testing.T) {
		td := &TokenData{AccessToken: "at", ExpiresAt: time.Now().Add(-time.Hour)}
		assert.False(t, td.IsValid())
	})
	t.Run("empty token", func(t *testing.T) {
		td := &TokenData{ExpiresAt: time.Now().Add(time.Hour)}
		assert.False(t, td.IsValid())
	})
}

func TestTokenData_HasRefreshToken(t *testing.T) {
	assert.True(t, (&TokenData{RefreshToken: "rt"}).HasRefreshToken())
	assert.False(t, (&TokenData{}).HasRefreshToken())
}

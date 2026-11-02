package oauth2

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMigrateLegacyTokenFile_MovesIntoConfigYAML 覆盖核心路径：
// ~/.gate-cli/tokens.yaml 存在 → 迁移到 config.yaml oauth2.tokens[] → 归档旧文件。
func TestMigrateLegacyTokenFile_MovesIntoConfigYAML(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	home, _ := resolveHome()
	legacy := filepath.Join(home, ".gate-cli", "tokens.yaml")
	legacyYAML := `oauth2:
  client_id: pkce_legacy
  discovery_url: https://legacy.example/.well-known/oauth-authorization-server
  access_token: at_legacy
  refresh_token: rt_legacy
  expires_at: 2099-01-01T00:00:00Z
  scopes: read
`
	require.NoError(t, os.WriteFile(legacy, []byte(legacyYAML), 0600))

	// config.yaml 点向同一 discovery_url 以便 LoadToken 能匹配。
	ensureActiveDiscoveryURL(t, "https://legacy.example/.well-known/oauth-authorization-server")

	var stderr bytes.Buffer
	require.NoError(t, MigrateLegacyTokenFile(&stderr))

	// 旧文件被归档（重命名），不再存在原路径。
	_, err := os.Stat(legacy)
	assert.True(t, os.IsNotExist(err), "legacy file should be renamed")

	entries, _ := os.ReadDir(filepath.Dir(legacy))
	var archived bool
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "tokens.yaml.migrated-") {
			archived = true
			break
		}
	}
	assert.True(t, archived, "archived copy should exist")

	// stderr 给出一次性提示。
	assert.Contains(t, stderr.String(), "tokens.yaml")
	assert.Contains(t, stderr.String(), "migrated")

	// LoadToken 能从 config.yaml 读到迁移来的凭据。
	loaded, err := LoadToken()
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "pkce_legacy", loaded.ClientID)
	assert.Equal(t, "at_legacy", loaded.AccessToken)
	assert.Equal(t, "rt_legacy", loaded.RefreshToken)
}

func TestMigrateLegacyTokenFile_NoLegacyFile_NoOp(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	var stderr bytes.Buffer
	require.NoError(t, MigrateLegacyTokenFile(&stderr))
	assert.Empty(t, stderr.String())
}

// TestMigrateLegacyTokenFile_ConflictKeepsConfig 当 config.yaml 已经有对应
// discovery_url 的条目时，不覆写，只归档旧文件。
func TestMigrateLegacyTokenFile_ConflictKeepsConfig(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()

	home, _ := resolveHome()
	url := "https://same.example/.well-known/oauth-authorization-server"
	ensureActiveDiscoveryURL(t, url)
	// 先往 config.yaml 写一个"新登录"的 entry。
	require.NoError(t, SaveToken(&TokenData{
		ClientID:     "pkce_new",
		DiscoveryURL: url,
		AccessToken:  "at_new",
		RefreshToken: "rt_new",
	}))

	// 再放一份过时的 legacy tokens.yaml。
	legacy := filepath.Join(home, ".gate-cli", "tokens.yaml")
	require.NoError(t, os.WriteFile(legacy, []byte(`oauth2:
  client_id: pkce_stale
  discovery_url: `+url+`
  access_token: at_stale
  refresh_token: rt_stale
`), 0600))

	var stderr bytes.Buffer
	require.NoError(t, MigrateLegacyTokenFile(&stderr))

	// config.yaml 里的"新"条目应原样保留。
	loaded, err := LoadToken()
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, "pkce_new", loaded.ClientID)
	assert.Equal(t, "at_new", loaded.AccessToken)

	// legacy 被归档。
	_, err = os.Stat(legacy)
	assert.True(t, os.IsNotExist(err))
}

// TestLegacyMigrationRespectsActiveAuthorization 迁移不能在浏览器授权已领取操作权后插入旧凭据。
func TestLegacyMigrationRespectsActiveAuthorization(t *testing.T) {
	home, cleanup := setupTestTokenDir(t)
	defer cleanup()
	op, err := BeginAuthOperation(context.Background(), "login", nil)
	require.NoError(t, err)
	defer func() { require.NoError(t, op.Finish(context.Background())) }()
	path := filepath.Join(home, ".gate-cli", "tokens.yaml")
	require.NoError(t, os.WriteFile(path, []byte("oauth2:\n  client_id: old\n  access_token: old-at\n"), 0600))
	require.Error(t, MigrateLegacyTokenFile(io.Discard))
	actual, err := LoadToken()
	require.NoError(t, err)
	require.Nil(t, actual)
	_, err = os.Stat(path)
	require.NoError(t, err)
}

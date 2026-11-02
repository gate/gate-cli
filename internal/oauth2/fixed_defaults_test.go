package oauth2

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFixedOAuthDefaultsIgnoreLegacyOverrides 防止旧分支遗留配置改变生产信任源或注册回调。
func TestFixedOAuthDefaultsIgnoreLegacyOverrides(t *testing.T) {
	home := isolateHome(t)
	dir := filepath.Join(home, ".gate-cli")
	require.NoError(t, os.MkdirAll(dir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("oauth2:\n  discovery_url: https://untrusted.example.invalid\n  custom_redirect_uri: https://untrusted.example.invalid/callback\n"), 0600))
	require.Equal(t, DefaultDiscoveryURL, CurrentDiscoveryURL())
	require.Equal(t, DefaultDisplayCallbackURL, ResolveManualRedirectURI())
	uris := BuildDCRRedirectURIs()
	require.Len(t, uris, 6)
	require.Equal(t, DefaultDisplayCallbackURL, uris[5])
}

//go:build agentic_e2e

package oauth2

import "os"

// init 仅存在于隔离 E2E 构建；普通发布构建不含测试环境变量读取逻辑。
func init() {
	discoveryURLSource = func() string {
		if issuer := os.Getenv("CLI_AGENTIC_TEST_ISSUER"); issuer != "" {
			return NormalizeDiscoveryURL(issuer)
		}
		return DefaultDiscoveryURL
	}
}

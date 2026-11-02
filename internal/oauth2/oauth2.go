// Package oauth2 implements OAuth2 PKCE authorization for Gate CLI.
// Endpoint URLs are resolved at startup by fetching the authorization
// server's RFC 8414 metadata document. The discovery URL itself is the
// fixed production trust anchor; no runtime override is accepted.
// see internal/oauth2/discovery.go and internal/oauth2/config.go.
package oauth2

import (
	"fmt"
	"strings"
)

const (
	// DiscoveryMetadataSuffix 是 RFC 8414 元数据路径后缀，用于凭据 issuer 归一化。
	DiscoveryMetadataSuffix = "/.well-known/oauth-authorization-server"

	// DefaultDiscoveryURL 是唯一的生产发现入口，不读取本地覆盖项。
	DefaultDiscoveryURL = "https://www.gate.com" + DiscoveryMetadataSuffix

	// DefaultDisplayCallbackURL 是手动登录的固定展示回调，DCR 与换码必须一致。
	DefaultDisplayCallbackURL = "https://www.gate.com/cli-oauth-callback"

	// Default scopes requested during authorization
	DefaultScopes = "base spot_trade futures_trade earn cfd stock"

	// Client name used in DCR
	ClientName = "gate-cli"
)

// NormalizeDiscoveryURL 接受以下任意形式并返回完整 RFC 8414 元数据 URL：
//   - "https://www.gate.com"
//   - "https://www.gate.com/"
//   - "https://www.gate.com/.well-known/oauth-authorization-server"
//
// 空串原样返回（让上游走默认回退）。不做 scheme 校验，后续 HTTP 层会给出
// 具体错误。
func NormalizeDiscoveryURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if strings.HasSuffix(s, DiscoveryMetadataSuffix) {
		return s
	}
	return strings.TrimRight(s, "/") + DiscoveryMetadataSuffix
}

// ShortDiscoveryURL 是 NormalizeDiscoveryURL 的逆操作：返回短前缀形式（去掉
// RFC 8414 suffix 与尾部斜杠）。仅用于历史凭据 tokens[] 的落盘元数据。
//
// 匹配逻辑应同时对两侧调用 ShortDiscoveryURL，确保新老数据（短/长）混存也能正确对齐。
func ShortDiscoveryURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	s = strings.TrimSuffix(s, DiscoveryMetadataSuffix)
	return strings.TrimRight(s, "/")
}

// isProductionDiscovery 判断传入 discovery URL（任意短/长形式）是否为正式环境。
// 生产 URL 语义上等于 DefaultDiscoveryURL；斜杠自动归一，两种写法等效。
func isProductionDiscovery(url string) bool {
	return ShortDiscoveryURL(url) == ShortDiscoveryURL(DefaultDiscoveryURL)
}

// Candidate ports for local callback server (§4.3 spec)
var CallbackPorts = []int{18991, 18992, 18993, 18994, 18995}

// BuildDCRRedirectURIs 仅注册五个 loopback 回调和固定的手动展示页。
func BuildDCRRedirectURIs() []string {
	uris := make([]string, 0, len(CallbackPorts)+1)
	for _, port := range CallbackPorts {
		uris = append(uris, fmt.Sprintf("http://127.0.0.1:%d/callback", port))
	}
	return append(uris, DefaultDisplayCallbackURL)
}

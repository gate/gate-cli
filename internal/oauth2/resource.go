package oauth2

import (
	"fmt"
	"net/url"
	"strings"
)

// APIResource 从已解析的API base生成唯一受众；拒绝含身份信息或路径歧义的地址。
func APIResource(base string) (string, error) {
	u, err := url.Parse(base)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Opaque != "" || strings.ContainsAny(base, "%\\") || strings.ContainsAny(u.Path, "\r\n\t") {
		return "", fmt.Errorf("invalid API base URL")
	}
	// 测试/预发允许非回环HTTP；受众仍包含协议，不能跨协议或站点复用凭据。
	// 这里只开放HTTP(S)，保留上面的URL校验和transport统一的发送/重定向边界。
	if u.Scheme != "https" && u.Scheme != "http" {
		return "", fmt.Errorf("API base URL must use http or https")
	}
	p := strings.TrimRight(u.Path, "/")
	if strings.Contains(p, "//") || strings.HasSuffix(p, "/api/v4") {
		return "", fmt.Errorf("ambiguous API base path")
	}
	for _, part := range strings.Split(p, "/") {
		if part == "." || part == ".." {
			return "", fmt.Errorf("ambiguous API base path")
		}
	}
	u.Path = p + "/api/v4"
	return u.String(), nil
}

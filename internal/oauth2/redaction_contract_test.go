package oauth2

import (
	"fmt"
	"strings"
	"testing"
)

// TestCredentialFormattingRedactsSecrets 防止结构体调试输出或短Token回显完整凭据。
func TestCredentialFormattingRedactsSecrets(t *testing.T) {
	for _, token := range []string{"", "short", "synthetic-sensitive-access-123456"} {
		masked := RedactToken(token)
		if token != "" && strings.Contains(masked, token) {
			t.Fatal("token escaped redaction")
		}
	}
	token := &TokenData{AccessToken: "synthetic-sensitive-access-123456", RefreshToken: "synthetic-sensitive-refresh-654321"}
	for _, out := range []string{fmt.Sprint(token), fmt.Sprintf("%#v", token), (*TokenData)(nil).String()} {
		if strings.Contains(out, token.AccessToken) || strings.Contains(out, token.RefreshToken) {
			t.Fatal("formatter leaked credentials")
		}
	}
}

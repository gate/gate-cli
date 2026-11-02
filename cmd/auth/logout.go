package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gate/gate-cli/internal/oauth2"
	"github.com/spf13/cobra"
)

// logoutCmd 本地清理先持久化；远端失败明确报告，不能宣称全局断开成功。
var logoutCmd = &cobra.Command{Use: "logout", Short: "Clear local credentials and revoke this connection", RunE: runLogout}

func runLogout(cmd *cobra.Command, args []string) error {
	old, err := oauth2.LoadToken()
	if err != nil {
		return err
	}
	if old == nil || (old.AccessToken == "" && old.RefreshToken == "") {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Not logged in."); err != nil {
			return err
		}
		return nil
	}
	// 发现失败也不能阻止本地退出；仍需取得统一操作权，不能与换码/刷新交错。
	discoveryErr := oauth2.InitEndpointsContext(cmd.Context())
	operation, err := oauth2.BeginAuthOperation(cmd.Context(), "logout", old)
	if err != nil {
		return err
	}
	local := *old
	local.AccessToken = ""
	local.RefreshToken = ""
	local.ExpiresAt = time.Time{}
	local.TokenType = ""
	local.Scopes = ""
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(cmd.Context()), 3*time.Second)
	defer cancel()
	if err := operation.Dispatch(cleanupCtx, &local); err != nil {
		return err
	}
	revokeErr := discoveryErr
	if revokeErr == nil {
		token := old.RefreshToken
		if token == "" {
			token = old.AccessToken
		}
		revokeErr = oauth2.RevokeToken(cmd.Context(), token, old.ClientID)
	}
	finishErr := operation.Finish(cleanupCtx)
	if revokeErr != nil {
		return errors.Join(fmt.Errorf("local credentials cleared; server revocation unconfirmed. Disconnect on the Connections page: %w", revokeErr), finishErr)
	}
	if finishErr != nil {
		return finishErr
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Logged out; server revocation confirmed."); err != nil {
		return err
	}
	return nil
}

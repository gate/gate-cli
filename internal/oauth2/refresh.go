package oauth2

import (
	"context"
	"errors"
	"fmt"
	"time"
)

const HardThreshold = 60 * time.Second
const LockTimeout = 10 * time.Second

// TokenExpiredError 表示需要用户重新授权；不触发交易请求自动重放。
type TokenExpiredError struct{ Message string }

func (e *TokenExpiredError) Error() string { return e.Message }
func validateBinding(token *TokenData, resource string) error {
	if token == nil || token.Resource == "" || token.Resource != resource || NormalizeDiscoveryURL(token.DiscoveryURL) != CurrentDiscoveryURL() {
		return &TokenExpiredError{Message: "Credential audience/issuer changed. Run 'gate-cli cex auth login --force'."}
	}
	return nil
}

// EnsureTokenValid 只在临近到期时同步刷新，一小时AT不会产生后台轮换。
func EnsureTokenValid(ctx context.Context, resource string) (*TokenData, error) {
	token, err := LoadToken()
	if err != nil {
		return nil, err
	}
	if err := validateBinding(token, resource); err != nil {
		return nil, err
	}
	if token.AccessToken == "" {
		return nil, &TokenExpiredError{Message: "No access token. Run 'gate-cli cex auth login'."}
	}
	if time.Until(token.ExpiresAt) > HardThreshold {
		return token, nil
	}
	return refreshInternal(ctx, resource, false)
}
func RefreshIfNeeded(ctx context.Context, resource string) (*TokenData, error) {
	return refreshInternal(ctx, resource, false)
}
func ForceRefresh(ctx context.Context, resource string) (*TokenData, error) {
	return refreshInternal(ctx, resource, true)
}
func refreshInternal(parent context.Context, resource string, force bool) (*TokenData, error) {
	ctx, cancel := context.WithTimeout(parent, LockTimeout)
	defer cancel()
	token, err := LoadToken()
	if err != nil {
		return nil, err
	}
	if err := validateBinding(token, resource); err != nil {
		return nil, err
	}
	if !force && token.AccessToken != "" && time.Until(token.ExpiresAt) > HardThreshold {
		return token, nil
	}
	if token.RefreshToken == "" || token.ClientID == "" {
		return nil, &TokenExpiredError{Message: "No refresh token available; authentication may be in progress. Run 'gate-cli cex auth login --force' if needed."}
	}
	// 业务命令进程不会全局做发现；只在确需刷新时解析当前issuer，且在领取/烧RT之前完成。
	// 发现失败是尚未发送写请求的明确失败，必须保留旧RT。
	endpoints := ActiveEndpoints()
	if endpoints.TokenEndpoint == "" || NormalizeDiscoveryURL(endpoints.BaseURL) != CurrentDiscoveryURL() {
		if err := InitEndpointsContext(ctx); err != nil {
			return nil, fmt.Errorf("resolve refresh issuer: %w", err)
		}
	}
	operation, err := BeginAuthOperation(ctx, "refresh", token)
	if err != nil {
		return nil, err
	}
	cleanup := func(cause error, unknown bool) error {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), 3*time.Second)
		defer cancel()
		return errors.Join(cause, operation.Fail(ctx, unknown))
	}
	claimed := *token
	claimed.RefreshToken = ""
	if err := operation.Dispatch(ctx, &claimed); err != nil {
		return nil, cleanup(err, false)
	}
	response, err := RefreshToken(ctx, token.RefreshToken, token.ClientID, resource)
	if err != nil {
		return nil, cleanup(fmt.Errorf("refresh result not usable; reauthorize: %w", err), true)
	}
	next := *token
	next.AccessToken = response.AccessToken
	next.RefreshToken = response.RefreshToken
	next.TokenType = response.TokenType
	next.Scopes = response.Scope
	next.ExpiresAt = time.Now().Add(time.Duration(response.ExpiresIn) * time.Second)
	if err := operation.SaveResult(ctx, &next); err != nil {
		return nil, err // 保留dispatched记录，保存失败时不得在defer里盲删。
	}
	if err := operation.Finish(ctx); err != nil {
		return nil, err
	}
	return &next, nil
}

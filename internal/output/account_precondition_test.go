package output

import (
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// A precise account-precondition rejection needs a business recovery action,
// while auth, route, write-outcome and unrelated parameter semantics stay intact.
func TestAccountPreconditionConvergence(t *testing.T) {
	for _, tc := range []struct {
		name, method, path, label, message, kind string
		status                                   int
		retry                                    bool
	}{
		{"futures account", "GET", "/api/v4/futures/usdt/accounts", "USER_NOT_FOUND", "please transfer funds first to create futures account", "ACCOUNT_NOT_READY", 400, false},
		{"delivery account", "GET", "/api/v4/delivery/btc/accounts", "USER_NOT_FOUND", "账户未初始化", "ACCOUNT_NOT_READY", 400, false},
		{"futures orders", "GET", "/api/v4/futures/btc/orders", "USER_NOT_FOUND", "", "ACCOUNT_NOT_READY", 400, false},
		{"delivery orders", "GET", "/api/v4/delivery/usdt/orders", "USER_NOT_FOUND", "user not found", "ACCOUNT_NOT_READY", 400, false},
		{"option unknown prerequisite", "GET", "/api/v4/options/accounts", "USER_NOT_FOUND", "", "UNKNOWN", 400, false},
		{"unknown route", "GET", "/api/v4/futures/usdt/not_accounts", "USER_NOT_FOUND", "user not found", "UNKNOWN", 400, false},
		{"individual order", "GET", "/api/v4/futures/usdt/orders/123", "USER_NOT_FOUND", "user not found", "UNKNOWN", 400, false},
		{"write", "POST", "/api/v4/futures/usdt/orders", "USER_NOT_FOUND", "user not found", "UNKNOWN", 400, false},
		{"no metadata", "", "", "USER_NOT_FOUND", "user not found", "UNKNOWN", 400, false},
		{"unauthorized", "GET", "/api/v4/futures/usdt/accounts", "USER_NOT_FOUND", "user not found", "AUTH_ERROR", 401, false},
		{"forbidden", "GET", "/api/v4/futures/usdt/accounts", "USER_NOT_FOUND", "user not found", "PERMISSION_DENIED", 403, false},
		{"missing route", "GET", "/api/v4/futures/usdt/accounts", "USER_NOT_FOUND", "user not found", "API_NOT_FOUND", 404, false},
		{"rate limited", "GET", "/api/v4/futures/usdt/accounts", "USER_NOT_FOUND", "user not found", "NETWORK_ERROR", 429, false},
		{"server error", "GET", "/api/v4/futures/usdt/accounts", "USER_NOT_FOUND", "user not found", "NETWORK_ERROR", 500, false},
		{"unavailable", "GET", "/api/v4/futures/usdt/accounts", "USER_NOT_FOUND", "user not found", "NETWORK_ERROR", 503, false},
		{"ordinary invalid", "GET", "/api/v4/futures/usdt/accounts", "INVALID_ARGUMENTS", "bad settle", "INVALID_ARGS", 400, true},
		{"similar label", "GET", "/api/v4/futures/usdt/accounts", "USER_NOT_FOUND_OTHER", "rejected", "INVALID_ARGS", 400, true},
		{"unsupported amend", "PUT", "/api/v4/futures/usdt/price_orders/amend/1", "APIOrderNotSupportUpdateTouchOrder", "rejected", "UNSUPPORTED_OPERATION", 400, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ge := &GateError{Status: tc.status, Label: tc.label, Message: tc.message, TraceID: "fixture-trace"}
			if tc.method != "" {
				ge.Request = &RequestInfo{Method: tc.method, URL: tc.path}
			}
			FillAgentErrorConvergence(ge)
			if ge.ErrorType != tc.kind || ge.Retryable != tc.retry || ge.Message != tc.message || ge.Label != tc.label || ge.TraceID != "fixture-trace" {
				t.Fatalf("unexpected classification: %+v; want %s retry=%t", ge, tc.kind, tc.retry)
			}
			if tc.kind == "ACCOUNT_NOT_READY" && (!strings.Contains(ge.SuggestedNextAction, "account") || !strings.Contains(ge.SuggestedNextAction, "stop") || strings.Contains(ge.SuggestedNextAction, "fix flags")) {
				t.Fatalf("misleading account advice: %s", ge.SuggestedNextAction)
			}
			before := *ge
			FillAgentErrorConvergence(ge)
			if !reflect.DeepEqual(*ge, before) {
				t.Fatal("convergence must be idempotent")
			}
		})
	}
}

func TestAccountPreconditionPreservesExplicitTypeAndWriteGuard(t *testing.T) {
	t.Parallel()
	FillAgentErrorConvergence(nil)
	ge := &GateError{Status: 400, Label: "USER_NOT_FOUND", ErrorType: "PERMISSION_DENIED", Request: &RequestInfo{Method: "GET", URL: "/api/v4/futures/usdt/accounts"}}
	FillAgentErrorConvergence(ge)
	if ge.ErrorType != "PERMISSION_DENIED" || ge.Retryable {
		t.Fatalf("explicit type overwritten: %+v", ge)
	}
	for _, status := range []int{http.StatusOK, http.StatusInternalServerError} {
		ge := &GateError{Status: status, Label: "USER_NOT_FOUND", Request: &RequestInfo{Method: "POST", URL: "/api/v4/futures/usdt/orders"}}
		FillAgentErrorConvergence(ge)
		if ge.Retryable || !strings.Contains(ge.SuggestedNextAction, "query the original") {
			t.Fatalf("unknown write outcome lost: %+v", ge)
		}
	}
	if kind := ClassifyCLIError(400, "USER_NOT_FOUND", "user not found"); kind != "UNKNOWN" {
		t.Fatalf("no route must not fabricate a reason or empty result: %s", kind)
	}
}

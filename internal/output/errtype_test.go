package output

import (
	"strings"
	"testing"
)

func TestClassifyCLIErrorInvalidArgs(t *testing.T) {
	t.Parallel()
	if got := ClassifyCLIError(400, "INVALID_ARGUMENTS", "bad"); got != "INVALID_ARGS" {
		t.Fatalf("got %q", got)
	}
}

func TestInvalidArgsErrorHasErrorType(t *testing.T) {
	t.Parallel()
	ge := InvalidArgsError("missing symbol")
	if ge.ErrorType != "INVALID_ARGS" {
		t.Fatalf("got %q", ge.ErrorType)
	}
}

func TestClassifyGateAPIError(t *testing.T) {
	t.Parallel()
	if got := ClassifyGateAPIError(401, "INVALID_KEY", ""); got != "AUTH_ERROR" {
		t.Fatalf("got %q", got)
	}
	if got := ClassifyGateAPIError(403, "", ""); got != "PERMISSION_DENIED" {
		t.Fatalf("got %q", got)
	}
	if got := ClassifyGateAPIError(404, "", ""); got != "API_NOT_FOUND" {
		t.Fatalf("got %q", got)
	}
}

// TestMissingRouteIsNotEmpty guards the Agent's business interpretation of 404.
func TestMissingRouteIsNotEmpty(t *testing.T) {
	t.Parallel()
	for _, label := range []string{"", "NOT_FOUND", "SERVER_ERROR"} {
		ge := &GateError{Status: 404, Label: label, Message: "404 page not found\n"}
		FillAgentErrorConvergence(ge)
		if ge.ErrorType != "API_NOT_FOUND" || ge.Retryable || !strings.Contains(ge.SuggestedNextAction, "gateway route") {
			t.Fatalf("misleading route failure: %#v", ge)
		}
	}
	for _, label := range []string{"ORDER_NOT_FOUND", "POSITION_NOT_FOUND", "RECORD_NOT_FOUND"} {
		if got := ClassifyGateAPIError(404, label, "missing business object"); got != "EMPTY_RESULT" {
			t.Fatalf("%s lost business absence semantics: %s", label, got)
		}
	}
}

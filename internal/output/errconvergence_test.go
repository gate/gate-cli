package output

import (
	"testing"
)

func TestFillAgentErrorConvergenceInvalidArgs(t *testing.T) {
	t.Parallel()
	ge := &GateError{
		Status:  400,
		Label:   "INVALID_ARGUMENTS",
		Message: "size must be <= 500",
	}
	FillAgentErrorConvergence(ge)
	if ge.ErrorType != "INVALID_ARGS" {
		t.Fatalf("error_type=%q", ge.ErrorType)
	}
	if !ge.Retryable {
		t.Fatal("expected retryable for INVALID_ARGS")
	}
	if ge.SuggestedNextAction == "" {
		t.Fatal("expected suggested_next_action")
	}
}

func TestFillAgentErrorConvergenceAuthError(t *testing.T) {
	t.Parallel()
	ge := &GateError{Status: 401, Label: "AUTH", Message: "api key required"}
	FillAgentErrorConvergence(ge)
	if ge.ErrorType != "AUTH_ERROR" {
		t.Fatalf("error_type=%q", ge.ErrorType)
	}
	if ge.Retryable {
		t.Fatal("AUTH_ERROR must not be retryable")
	}
}

func TestFillAgentErrorConvergenceGovernanceNotRetryable(t *testing.T) {
	t.Parallel()
	for _, label := range []string{"PREFLIGHT_BLOCKED", "DOCTOR_FAILED", "MIGRATE_FAILED"} {
		ge := &GateError{Status: 422, Label: label, Message: "blocked"}
		FillAgentErrorConvergence(ge)
		if ge.Retryable {
			t.Fatalf("%s must not be retryable", label)
		}
		if ge.SuggestedNextAction == "" {
			t.Fatalf("%s missing suggested_next_action", label)
		}
	}
}

func TestInvalidArgsErrorIncludesConvergenceFields(t *testing.T) {
	t.Parallel()
	ge := InvalidArgsError("coin is required")
	if ge.ErrorType != "INVALID_ARGS" || !ge.Retryable || ge.SuggestedNextAction == "" {
		t.Fatalf("got %#v", ge)
	}
}

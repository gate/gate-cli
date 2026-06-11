package output

import "testing"

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
	if got := ClassifyGateAPIError(404, "", ""); got != "EMPTY_RESULT" {
		t.Fatalf("got %q", got)
	}
}

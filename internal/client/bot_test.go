package client

import (
	"encoding/json"
	"errors"
	"github.com/gate/gate-cli/internal/output"
	"net/http"
	"testing"
)

// Guard HTTP-success/business-failure, partial writes and missing identities independently.
func TestBotResultError(t *testing.T) {
	for _, tc := range []struct {
		raw, method string
		wantError   bool
	}{
		{`{"code":200,"data":{}}`, "GET", false},
		{`{"code":400,"label":"INVALID_USER","trace_id":"fixture","message":"invalid user"}`, "GET", true},
		{`{}`, "GET", true},
		{`{"code":200,"data":{}}`, "POST", true},
		{`{"code":200,"data":{"strategy_id":"1","status":"starting","recovery_required":true}}`, "POST", false},
		{`{"code":500,"data":{"strategy_id":"9223372036854775807","status":"unknown","operation_status":"pending","recovery_required":false}}`, "POST", true},
	} {
		var result map[string]json.RawMessage
		if err := json.Unmarshal([]byte(tc.raw), &result); err != nil {
			t.Fatal(err)
		}
		err := BotResultError(result, nil, &http.Response{StatusCode: 200}, tc.method, "/api/v4/bot/fixture", "")
		if (err != nil) != tc.wantError {
			t.Errorf("%s: %v", tc.raw, err)
		}
		if err != nil {
			var detail *output.GateError
			if !errors.As(err, &detail) {
				t.Errorf("lost typed error: %v", err)
			}
		}
	}
}

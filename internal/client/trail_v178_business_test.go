package client

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/gate/gate-cli/internal/output"
)

// v178 success DTOs now retain envelope codes. The existing error boundary must
// reject business failures while leaving successful bytes for the native SDK.
func TestSDK178TrailBusinessBoundary(t *testing.T) {
	for _, operation := range []string{"update", "stop", "change_log"} {
		for _, tc := range []struct{ name, raw, label string }{
			{"success extension", `{"code":0,"message":"ok","label":"future","data":{},"unknown":true}`, ""},
			{"business", `{"code":-1,"label":"TRAIL_REJECTED","message":"fixture rejection","data":{},"private_debug":"private-canary"}`, "TRAIL_REJECTED"},
			{"message only", `{"code":1,"message":"fixture rejection","data":{},"private_debug":"private-canary"}`, ""},
			{"ambiguous code", `{"code":1,"code":0,"message":"fixture rejection"}`, "INVALID_RESPONSE"},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				method := "POST"
				if operation == "change_log" {
					method = "GET"
				}
				response, _ := futuresTransportResponse(t, method, "/api/v4/futures/usdt/autoorder/v1/trail/"+operation, tc.raw, 200, nil)
				raw, err := io.ReadAll(response.Body)
				if closeErr := response.Body.Close(); closeErr != nil {
					t.Fatal(closeErr)
				}
				if tc.name == "success extension" {
					if err != nil || string(raw) != tc.raw {
						t.Fatalf("success was reconstructed: %s %v", raw, err)
					}
					return
				}
				var detail *output.GateError
				if !errors.As(err, &detail) {
					t.Fatalf("business failure was treated as SDK success: %s %v", raw, err)
				}
				if detail.Label != tc.label || detail.Request == nil || detail.Request.Method != method {
					t.Fatalf("business diagnostics changed: %#v", detail)
				}
				if strings.Contains(detail.Error(), "private-canary") {
					t.Fatal("private response leaked")
				}
			})
		}
	}
}

package oauth2

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestDeadLoginRecoveryRejectsStaleSnapshot 覆盖三个刷新入口：烧毁RT后不能再发送调用前的内存副本。
func TestDeadLoginRecoveryRejectsStaleSnapshot(t *testing.T) {
	for _, entry := range []string{"force", "if-needed", "ensure"} {
		t.Run(entry, func(t *testing.T) {
			_, cleanup := setupTestTokenDir(t)
			defer cleanup()
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				if err := json.NewEncoder(w).Encode(TokenResponse{AccessToken: "new-at", RefreshToken: "new-rt", TokenType: "Bearer", ExpiresIn: 3600, Scope: "stock"}); err != nil {
					t.Error(err)
				}
			}))
			defer server.Close()
			previous := setActiveEndpointsForTest(&Endpoints{BaseURL: server.URL, TokenEndpoint: server.URL + "/token"})
			defer setActiveEndpointsForTest(previous)
			resource := "https://api.gateio.ws/api/v4"
			writeTestToken(t, &TokenData{ClientID: "synthetic-client", DiscoveryURL: server.URL, Resource: resource, AccessToken: "old-at", RefreshToken: "old-rt", Scopes: "stock", ExpiresAt: time.Now().Add(-time.Minute)})
			old, err := LoadToken()
			if err != nil {
				t.Fatal(err)
			}
			op, err := BeginAuthOperation(t.Context(), "login", old)
			if err != nil {
				t.Fatal(err)
			}
			if err := op.Dispatch(t.Context(), nil); err != nil {
				t.Fatal(err)
			}
			record, err := readOperation()
			if err != nil {
				t.Fatal(err)
			}
			record.PID = 99999999
			if processAlive(record.PID) {
				t.Fatal("synthetic dead PID is alive")
			}
			if err := writeOperation(record); err != nil {
				t.Fatal(err)
			}
			switch entry {
			case "force":
				_, err = ForceRefresh(t.Context(), resource)
			case "if-needed":
				_, err = RefreshIfNeeded(t.Context(), resource)
			case "ensure":
				_, err = EnsureTokenValid(t.Context(), resource)
			}
			var expired *TokenExpiredError
			if !errors.As(err, &expired) || calls.Load() != 0 {
				t.Errorf("stale snapshot used: err=%v HTTP=%d", err, calls.Load())
			}
			current, err := LoadToken()
			if err != nil || current.RefreshToken != "" || current.AccessToken != "old-at" {
				t.Fatalf("burned RT revived or AT lost: err=%v", err)
			}
			pending, err := readOperation()
			if err != nil || pending != nil {
				t.Fatalf("rejected snapshot left ownership: %v", err)
			}
			// 新授权从恢复后的权威快照开始，不能被本次拒绝留下的记录锁住。
			next, err := BeginAuthOperation(t.Context(), "login", current)
			if err != nil {
				t.Fatal(err)
			}
			if err := next.Finish(t.Context()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

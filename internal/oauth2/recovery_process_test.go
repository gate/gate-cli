package oauth2

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// TestAuthRecoveryCrashHelper 仅测试子进程使用；停在真实Dispatch之后供父进程强杀。
func TestAuthRecoveryCrashHelper(t *testing.T) {
	ready := os.Getenv("GATE_CLI_TEST_AUTH_CRASH_READY")
	if ready == "" {
		return
	}
	// 子进程沿用父测试的隔离issuer，避免读取到另一个环境的凭据条目。
	ensureActiveDiscoveryURL(t, os.Getenv("GATE_CLI_TEST_AUTH_CRASH_ISSUER"))
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
	if err := os.WriteFile(ready, []byte("ready"), 0600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(8 * time.Second)
	t.Fatal("parent did not kill the crashed-login fixture")
}

// TestKilledLoginCannotRefreshBurnedRT 使用真实子进程PID和强杀，验证重启后不重发旧RT。
func TestKilledLoginCannotRefreshBurnedRT(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"access_token":"new-at","refresh_token":"new-rt","token_type":"Bearer","expires_in":3600,"scope":"stock"}`)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	previous := setActiveEndpointsForTest(&Endpoints{BaseURL: server.URL, TokenEndpoint: server.URL + "/token"})
	defer setActiveEndpointsForTest(previous)
	resource := "https://api.gateio.ws/api/v4"
	writeTestToken(t, &TokenData{ClientID: "synthetic-client", DiscoveryURL: server.URL, Resource: resource, AccessToken: "old-at", RefreshToken: "old-rt", Scopes: "stock", ExpiresAt: time.Now().Add(-time.Minute)})
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ready := filepath.Join(t.TempDir(), "ready")
	child := exec.CommandContext(t.Context(), executable, "-test.run=^TestAuthRecoveryCrashHelper$", "-test.timeout=10s")
	child.Env = append(os.Environ(), "GATE_CLI_TEST_AUTH_CRASH_READY="+ready, "GATE_CLI_TEST_AUTH_CRASH_ISSUER="+server.URL)
	var log bytes.Buffer
	child.Stdout, child.Stderr = &log, &log
	if err := child.Start(); err != nil {
		t.Fatal(err)
	}
	joined := false
	defer func() {
		if !joined {
			_ = child.Process.Kill()
			_ = child.Wait()
		}
	}()
	deadline := time.Now().Add(4 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("child did not reach dispatched state")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := child.Wait(); err == nil {
		t.Fatal("crash fixture exited successfully")
	}
	joined = true
	_, err = ForceRefresh(t.Context(), resource)
	var expired *TokenExpiredError
	if !errors.As(err, &expired) || calls.Load() != 0 {
		t.Fatalf("killed login reused RT: err=%v requests=%d", err, calls.Load())
	}
	current, err := LoadToken()
	if err != nil || current.RefreshToken != "" {
		t.Fatalf("burned RT survived: err=%v", err)
	}
}

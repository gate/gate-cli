package oauth2

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestAuthOperationOwnsCredentialMutation 保证共享文件锁只保护短事务，HTTP期间通过记录互斥。
func TestAuthOperationOwnsCredentialMutation(t *testing.T) {
	home, cleanup := setupTestTokenDir(t)
	defer cleanup()
	old := &TokenData{ClientID: "synthetic", AccessToken: "synthetic-at", RefreshToken: "synthetic-rt", Resource: "https://api.gateio.ws/api/v4", ExpiresAt: time.Now().Add(time.Hour)}
	writeTestToken(t, old)
	old, _ = LoadToken()
	operation, err := BeginAuthOperation(t.Context(), "refresh", old)
	if err != nil {
		t.Fatal(err)
	}
	claimed := *old
	claimed.RefreshToken = ""
	if err := operation.Dispatch(t.Context(), &claimed); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(home, ".gate-cli/auth-operation.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{old.AccessToken, old.RefreshToken} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("operation record contains credential")
		}
	}
	// 已释放flock，普通配置可编辑；第二个凭据变更仍不能消费同一RT。
	if err := WithConfigLock(t.Context(), func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := BeginAuthOperation(t.Context(), "logout", &claimed); err == nil {
		t.Fatal("concurrent operation claimed execution")
	}
	next := claimed
	next.AccessToken = "next-at"
	next.RefreshToken = "next-rt"
	if err := operation.SaveResult(t.Context(), &next); err != nil {
		t.Fatal(err)
	}
	if _, err := BeginAuthOperation(t.Context(), "logout", &next); err == nil {
		t.Fatal("result save prematurely released migration cleanup ownership")
	}
	if err := operation.Finish(t.Context()); err != nil {
		t.Fatal(err)
	}
	actual, err := LoadToken()
	if err != nil || actual.RefreshToken != "next-rt" {
		t.Fatal("committed credential lost", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".gate-cli/auth-operation.json")); !os.IsNotExist(err) {
		t.Fatal("completed operation record retained", err)
	}
}

func TestAuthOperationStaleLoginDoesNotDispatch(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()
	old := &TokenData{ClientID: "synthetic", AccessToken: "old", Resource: "https://api.gateio.ws/api/v4"}
	writeTestToken(t, old)
	old, _ = LoadToken()
	next := *old
	next.AccessToken = "new"
	if err := SaveToken(&next); err != nil {
		t.Fatal(err)
	}
	if _, err := BeginAuthOperation(t.Context(), "login", old); err == nil {
		t.Fatal("stale browser result can overwrite newer credentials")
	}
}

func TestAuthOperationUnknownOutcomeBurnsRT(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()
	old := &TokenData{ClientID: "synthetic", AccessToken: "old-at", RefreshToken: "one-use-rt", Resource: "https://api.gateio.ws/api/v4"}
	writeTestToken(t, old)
	old, _ = LoadToken()
	operation, err := BeginAuthOperation(t.Context(), "login", old)
	if err != nil {
		t.Fatal(err)
	}
	if err := operation.Dispatch(t.Context(), nil); err != nil {
		t.Fatal(err)
	}
	if err := operation.Fail(t.Context(), true); err != nil {
		t.Fatal(err)
	}
	actual, err := LoadToken()
	if err != nil || actual.RefreshToken != "" || actual.AccessToken != "old-at" {
		t.Fatal("ambiguous exchange retained suspect RT", err)
	}
}

// TestAuthOperationDeadOwnerRecovery 模拟进程已退出的每个可识别窗口，不用TTL接管活进程。
func TestAuthOperationDeadOwnerRecovery(t *testing.T) {
	for _, phase := range []string{"reserved", "dispatched", "committed"} {
		t.Run(phase, func(t *testing.T) {
			home, cleanup := setupTestTokenDir(t)
			defer cleanup()
			old := &TokenData{ClientID: "synthetic", AccessToken: "old-at", RefreshToken: "old-rt", Resource: "https://api.gateio.ws/api/v4"}
			writeTestToken(t, old)
			old, _ = LoadToken()
			operation, err := BeginAuthOperation(t.Context(), "refresh", old)
			if err != nil {
				t.Fatal(err)
			}
			if phase != "reserved" {
				claimed := *old
				claimed.RefreshToken = ""
				if err := operation.Dispatch(t.Context(), &claimed); err != nil {
					t.Fatal(err)
				}
			}
			if phase == "committed" {
				next := *old
				next.AccessToken = "new-at"
				next.RefreshToken = "new-rt"
				if err := operation.SaveResult(t.Context(), &next); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(home, ".gate-cli/auth-operation.json")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var record map[string]interface{}
			if json.Unmarshal(raw, &record) != nil {
				t.Fatal("bad operation JSON")
			}
			record["pid"] = 99999999
			raw, err = json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			current, _ := LoadToken()
			recovered, err := BeginAuthOperation(t.Context(), "login", current)
			if err != nil {
				t.Fatal(err)
			}
			actual, _ := LoadToken()
			if phase == "committed" && actual.RefreshToken != "new-rt" {
				t.Fatal("recovery erased committed new pair")
			}
			if phase == "dispatched" && actual.RefreshToken != "" {
				t.Fatal("recovery restored spent RT")
			}
			if err := recovered.Finish(t.Context()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestAuthOperationMustDispatchBeforeResult 防止尚未领取执行权的操作直接覆盖凭据。
func TestAuthOperationMustDispatchBeforeResult(t *testing.T) {
	_, cleanup := setupTestTokenDir(t)
	defer cleanup()
	old := &TokenData{ClientID: "synthetic", AccessToken: "old-at", Resource: "https://api.gateio.ws/api/v4"}
	writeTestToken(t, old)
	old, err := LoadToken()
	if err != nil {
		t.Fatal(err)
	}
	op, err := BeginAuthOperation(t.Context(), "login", old)
	if err != nil {
		t.Fatal(err)
	}
	next := *old
	next.AccessToken = "new-at"
	if err := op.SaveResult(t.Context(), &next); err == nil {
		t.Fatal("reserved operation wrote a result without dispatch")
	}
	if err := op.Finish(t.Context()); err != nil {
		t.Fatal(err)
	}
}

// TestAuthOperationFailedPersistenceRetainsOwnership 真实文件失败不能删掉已发送操作或恢复RT。
func TestAuthOperationFailedPersistenceRetainsOwnership(t *testing.T) {
	home, cleanup := setupTestTokenDir(t)
	defer cleanup()
	old := &TokenData{ClientID: "synthetic", AccessToken: "old-at", RefreshToken: "old-rt", Resource: "https://api.gateio.ws/api/v4"}
	writeTestToken(t, old)
	old, err := LoadToken()
	if err != nil {
		t.Fatal(err)
	}
	op, err := BeginAuthOperation(t.Context(), "refresh", old)
	if err != nil {
		t.Fatal(err)
	}
	claimed := *old
	claimed.RefreshToken = ""
	if err := op.Dispatch(t.Context(), &claimed); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".gate-cli/config.yaml")
	backup := path + ".test-backup"
	if err := os.Rename(path, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	next := claimed
	next.AccessToken = "new-at"
	next.RefreshToken = "new-rt"
	if err := op.SaveResult(t.Context(), &next); err == nil {
		t.Fatal("directory target unexpectedly saved")
	}
	record, err := readOperation()
	if err != nil || record == nil || record.Phase != "dispatched" {
		t.Fatal("failed save lost ownership", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(backup, path); err != nil {
		t.Fatal(err)
	}
	current, err := LoadToken()
	if err != nil || current.RefreshToken != "" {
		t.Fatal("failed save revived spent RT", err)
	}
	wrong := &AuthOperation{nonce: strings.Repeat("0", 64)}
	if err := wrong.Finish(t.Context()); err == nil {
		t.Fatal("foreign nonce finished operation")
	}
	if err := op.Dispatch(t.Context(), nil); err == nil {
		t.Fatal("operation dispatched twice")
	}
	if err := op.SaveResult(t.Context(), nil); err == nil {
		t.Fatal("nil result accepted")
	}
	if err := op.Fail(t.Context(), true); err != nil {
		t.Fatal(err)
	}
}

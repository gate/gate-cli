package oauth2

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gate/gate-cli/internal/config"
)

// AuthOperation 是本机一次凭据变更的所有权，不跨网络持文件锁、不保存Token或授权码。
// 所有凭据发送者先Dispatch，保存结果后Finish；失败必须Fail，不能无条件defer删记录。
type AuthOperation struct{ nonce string }
type authOperationRecord struct {
	Version int       `json:"version"`
	Action  string    `json:"action"`
	Nonce   string    `json:"nonce"`
	PID     int       `json:"pid"`
	Started time.Time `json:"started"`
	Phase   string    `json:"phase"`
	Before  string    `json:"before"`
	Claimed string    `json:"claimed"`
	Result  string    `json:"result,omitempty"`
}

func operationPath() (string, error) {
	dir, err := tokenDir()
	return filepath.Join(dir, "auth-operation.json"), err
}
func credentialDigest(token *TokenData) string {
	// 注册元数据也参与摘要；非凭据配置编辑不会与浏览器授权产生伪冲突。
	body, _ := json.Marshal(token)
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}
func readOperation() (*authOperationRecord, error) {
	path, err := operationPath()
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Windows的POSIX权限位不代表ACL；与Token文件检查一致，依赖私有用户目录继承的ACL。
	if !info.Mode().IsRegular() || info.Size() > 8192 || (runtime.GOOS != "windows" && info.Mode().Perm()&0077 != 0) {
		return nil, fmt.Errorf("unsafe auth operation record; stop old CLI process before recovery")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var record authOperationRecord
	if json.Unmarshal(raw, &record) != nil || record.Version != 1 || len(record.Nonce) != 64 || record.PID <= 0 || len(record.Before) != 64 || len(record.Claimed) != 64 || (record.Phase != "reserved" && record.Phase != "dispatched") {
		return nil, fmt.Errorf("invalid auth operation record; manual recovery required after stopping old process")
	}
	return &record, nil
}
func writeOperation(record *authOperationRecord) error {
	path, err := operationPath()
	if err != nil {
		return err
	}
	raw, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return config.WriteAtomic(path, raw, 0600)
}
func deleteOperation() error {
	path, err := operationPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return config.SyncDirectory(filepath.Dir(path))
}

// BeginAuthOperation 在短锁内检查浏览器开始时的快照，避免旧code覆盖其他进程的新授权。
func BeginAuthOperation(ctx context.Context, action string, expected *TokenData) (*AuthOperation, error) {
	if action != "login" && action != "refresh" && action != "logout" {
		return nil, fmt.Errorf("invalid auth operation")
	}
	var operation *AuthOperation
	err := WithConfigLock(ctx, func() error {
		current, err := LoadToken()
		if err != nil {
			return err
		}
		if credentialDigest(current) != credentialDigest(expected) {
			return fmt.Errorf("credentials changed; restart authorization")
		}
		record, err := readOperation()
		if err != nil {
			return err
		}
		if record != nil {
			if processAlive(record.PID) {
				return fmt.Errorf("authentication operation in progress; retry after it finishes")
			}
			currentHash := credentialDigest(current)
			switch {
			case record.Result != "" && currentHash == record.Result:
				// 新凭据已提交，仅清理残留所有权；绝不能清掉这对新Token。
			case record.Phase == "reserved" && (currentHash == record.Before || currentHash == record.Claimed):
				// 尚未发送HTTP；已领取清空的RT也不恢复。
			case record.Phase == "dispatched" && currentHash == record.Claimed:
				if current != nil && current.RefreshToken != "" {
					copy := *current
					copy.RefreshToken = ""
					if err := saveOperationResultLocked(record, &copy); err != nil {
						return err
					}
					current = &copy
				}
			default:
				return fmt.Errorf("cannot establish auth operation provenance; manual recovery required")
			}
			if err := deleteOperation(); err != nil {
				return err
			}
		}
		// 恢复可能烧毁RT；调用方仍持有恢复前的快照时，不能为它签发新的发送权。
		// 在同一短锁内重验，既避免内存旧RT复活，也不新增跨进程写入口。
		if credentialDigest(current) != credentialDigest(expected) {
			return &TokenExpiredError{Message: "Credentials changed during crash recovery. Run 'gate-cli cex auth login --force'."}
		}
		nonce := make([]byte, 32)
		if _, err := rand.Read(nonce); err != nil {
			return err
		}
		hash := credentialDigest(current)
		record = &authOperationRecord{Version: 1, Action: action, Nonce: hex.EncodeToString(nonce), PID: os.Getpid(), Started: time.Now().UTC(), Phase: "reserved", Before: hash, Claimed: hash}
		if err := writeOperation(record); err != nil {
			return err
		}
		operation = &AuthOperation{nonce: record.Nonce}
		return nil
	})
	return operation, err
}
func (operation *AuthOperation) owned() (*authOperationRecord, error) {
	record, err := readOperation()
	if err != nil {
		return nil, err
	}
	if record == nil || record.Nonce != operation.nonce || record.PID != os.Getpid() {
		return nil, fmt.Errorf("authentication operation ownership changed")
	}
	return record, nil
}

// Dispatch 先落盘领取后的状态（refresh烧掉RT、logout清凭据），再标记可发送。
func (operation *AuthOperation) Dispatch(ctx context.Context, state *TokenData) error {
	return WithConfigLock(ctx, func() error {
		record, err := operation.owned()
		if err != nil {
			return err
		}
		if record.Phase != "reserved" {
			return fmt.Errorf("authentication operation already dispatched")
		}
		current, err := LoadToken()
		if err != nil {
			return err
		}
		if credentialDigest(current) != record.Before {
			return fmt.Errorf("credentials changed before dispatch")
		}
		if state != nil {
			record.Claimed = credentialDigest(state)
		}
		// 先记录两个可恢复摘要，崩溃时不会把已清空的RT恢复回来。
		if err := writeOperation(record); err != nil {
			return err
		}
		if state != nil {
			if err := saveTokenLocked(state); err != nil {
				return err
			}
		}
		record.Phase = "dispatched"
		return writeOperation(record)
	})
}

// SaveResult 先记录结果摘要再保存凭据，专门覆盖保存成功但清理前崩溃的窗口。
// 保留执行权，允许同次login继续定向撤销旧连接，再显式Finish。
func (operation *AuthOperation) SaveResult(ctx context.Context, result *TokenData) error {
	if result == nil {
		return fmt.Errorf("nil operation result")
	}
	return WithConfigLock(ctx, func() error {
		record, err := operation.owned()
		if err != nil {
			return err
		}
		// 结果写入必须属于已领取并发送的登录/刷新，不能绕过Dispatch的先落盘约束。
		if record.Phase != "dispatched" || record.Action == "logout" {
			return fmt.Errorf("authentication result requires a dispatched login or refresh")
		}
		current, err := LoadToken()
		if err != nil {
			return err
		}
		if credentialDigest(current) != record.Claimed {
			return fmt.Errorf("credentials changed while operation was in flight")
		}
		return saveOperationResultLocked(record, result)
	})
}
func (operation *AuthOperation) Finish(ctx context.Context) error {
	return WithConfigLock(ctx, func() error {
		record, err := operation.owned()
		if err != nil {
			return err
		}
		current, err := LoadToken()
		if err != nil {
			return err
		}
		hash := credentialDigest(current)
		if hash != record.Before && hash != record.Claimed && hash != record.Result {
			return fmt.Errorf("credentials changed before finishing operation")
		}
		return deleteOperation()
	})
}

// Fail 对未知换码/刷新结果先清可疑RT，再回收本人记录；已提交结果始终保留。
func (operation *AuthOperation) Fail(ctx context.Context, unknown bool) error {
	err := WithConfigLock(ctx, func() error {
		record, err := operation.owned()
		if err != nil {
			return err
		}
		current, err := LoadToken()
		if err != nil {
			return err
		}
		hash := credentialDigest(current)
		if record.Result != "" && hash == record.Result {
			return nil
		}
		if hash != record.Before && hash != record.Claimed {
			return fmt.Errorf("credentials changed during failed operation")
		}
		if unknown && current != nil && current.RefreshToken != "" {
			copy := *current
			copy.RefreshToken = ""
			return saveOperationResultLocked(record, &copy)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return operation.Finish(ctx)
}

// saveOperationResultLocked is the existing result-before-credentials barrier,
// shared by successful results, uncertain failure and crash recovery. The caller
// holds WithConfigLock and has validated ownership/provenance. In particular,
// recovery must journal the burned snapshot before saving it, so another crash
// cannot leave a legitimate local state with no matching provenance digest.
func saveOperationResultLocked(record *authOperationRecord, result *TokenData) error {
	record.Result = credentialDigest(result)
	if err := writeOperation(record); err != nil {
		return err
	}
	return saveTokenLocked(result)
}

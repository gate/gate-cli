//go:build agentic_e2e

package agentic_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestRecoveryCanRestartAfterSecondInterruption 在构建 overlay 注入一次恢复中断，保持生产源码不变。
// 不手造互相矛盾的 journal；子用例通过真实 Begin/Dispatch 和恢复落盘产生中断窗口。
func TestRecoveryCanRestartAfterSecondInterruption(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	source := filepath.Join(root, "internal/oauth2/operation.go")
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	marker := "current = &copy"
	if strings.Count(string(raw), marker) != 1 {
		t.Fatal("recovery window changed; review fault placement before running")
	}
	body := strings.Replace(string(raw), marker, marker+"\nreturn fmt.Errorf(\"injected recovery interruption\")", 1)
	dir := t.TempDir()
	replacement := filepath.Join(dir, "operation.go")
	if err := os.WriteFile(replacement, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(root, "tests/agentic/testdata/recovery_interruption.go.fixture")
	overlay := map[string]interface{}{"Replace": map[string]string{source: replacement, filepath.Join(root, "internal/oauth2/recovery_second_interruption_test.go"): fixture}}
	encoded, err := json.Marshal(overlay)
	if err != nil {
		t.Fatal(err)
	}
	overlayPath := filepath.Join(dir, "overlay.json")
	if err := os.WriteFile(overlayPath, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	// 与父用例同一 Go 工具链；此子进程没有网络业务，只运行一个私有目录状态机用例。
	//nolint:staticcheck // 本地构建夹具不分发；PATH 中 Go 1.25 用于 issuer，此处必须沿用父用例 Go 1.24。
	cmd := exec.CommandContext(ctx, filepath.Join(runtime.GOROOT(), "bin/go"), "test", "-overlay="+overlayPath, "./internal/oauth2", "-run", "^TestRecoverySecondInterruptionContract$", "-count=1", "-timeout=30s")
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("recovery restart contract failed: %v\n%s", err, output)
	}
}

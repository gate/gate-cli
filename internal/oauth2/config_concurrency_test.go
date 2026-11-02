package oauth2

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/gate/gate-cli/internal/config"
	"github.com/gate/gate-cli/internal/configcmd"
	"github.com/spf13/cobra"
)

// TestConfigFirstCreateAndTokenWriter 同时首次创建文件；两位写者必须在同一锁内重读并合并。
func TestConfigFirstCreateAndTokenWriter(t *testing.T) {
	home, cleanup := setupTestTokenDir(t)
	defer cleanup()
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetIn(bytes.NewBufferString("manual\nsynthetic-key\nsynthetic-secret\n"))
	cmd.SetOut(io.Discard)
	token := &TokenData{ClientID: "synthetic", AccessToken: "synthetic-at", RefreshToken: "synthetic-rt", Resource: "https://api.gateio.ws/api/v4", Scopes: "stock", ExpiresAt: time.Now().Add(time.Hour)}
	for _, fn := range []func() error{func() error { return configcmd.RunInit(cmd) }, func() error { return SaveToken(token) }} {
		wg.Add(1)
		go func(fn func() error) { defer wg.Done(); <-start; errs <- fn() }(fn)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	actual, err := LoadToken()
	if err != nil || actual == nil || actual.AccessToken != token.AccessToken {
		t.Fatal("first create lost token", err)
	}
	cfg, err := config.Load(config.Options{Profile: "manual"})
	if err != nil || cfg.APIKey != "synthetic-key" {
		t.Fatal("first create lost profile", err)
	}
	if info, err := os.Stat(filepath.Join(home, ".gate-cli/config.yaml")); err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("credential file permissions changed", err)
	}
}

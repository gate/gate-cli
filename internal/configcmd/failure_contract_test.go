package configcmd

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

type failedWriter struct{}

func (failedWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

// TestConfigurationFailureBoundaries 输入/输出/坏YAML失败不得静默成功或回显未脱敏配置。
func TestConfigurationFailureBoundaries(t *testing.T) {
	for _, input := range []string{"", "name\n", "name\nkey\n", strings.Repeat("x", 129) + "\nkey\nsecret\n"} {
		t.Run("incomplete-"+strings.Repeat("x", len(input)%5), func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			cmd.SetIn(strings.NewReader(input))
			cmd.SetOut(io.Discard)
			if err := RunInit(cmd); err == nil {
				t.Fatal("invalid input succeeded")
			}
		})
	}
	t.Setenv("HOME", t.TempDir())
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(failedWriter{})
	cmd.SetIn(strings.NewReader("x\nk\ns\n"))
	if err := RunInit(cmd); err == nil {
		t.Fatal("ignored prompt failure")
	}
	if err := RunSet(cmd, nil); err == nil {
		t.Fatal("missing args")
	}
	if err := RunSet(cmd, []string{"api-key", "x"}); err == nil {
		t.Fatal("missing config")
	}
	path := filepath.Join(os.Getenv("HOME"), ".gate-cli", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("oauth2: [unterminated synthetic-secret"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd.Flags().Bool("show-secrets", false, "")
	if err := RunList(cmd); err == nil {
		t.Fatal("invalid YAML displayed")
	}
	if strings.Contains(MaskSecrets("oauth2: [unterminated synthetic-secret"), "synthetic-secret") {
		t.Fatal("unsafe fallback")
	}
	if err := RunSet(cmd, []string{"unknown", "x"}); err == nil {
		t.Fatal("unknown key")
	}
}

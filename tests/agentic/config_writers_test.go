//go:build agentic_e2e

package agentic_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// TestConfigWritersPreserveCredentials 覆盖统一的根config入口，不能用裁剪后的结构整体覆盖OAuth状态。
func TestConfigWritersPreserveCredentials(t *testing.T) {
	for _, prefix := range [][]string{{"config"}} {
		for _, action := range []string{"set", "init"} {
			t.Run(strings.Join(prefix, " ")+"/"+action, func(t *testing.T) {
				home := t.TempDir()
				dir := filepath.Join(home, ".gate-cli")
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
				original := `# Preserve this comment
custom: keep
intel:
  info_mcp_url: https://custom.example.test/mcp
oauth2:
  discovery_url: https://issuer.example.test
  tokens:
    - discovery_url: https://issuer.example.test
      resource: https://api.gateio.ws/api/v4
      client_id: synthetic
      access_token: synthetic-at
      refresh_token: synthetic-rt
profiles:
  untouched:
    api_key: keep-key
`
				path := filepath.Join(dir, "config.yaml")
				if err := os.WriteFile(path, []byte(original), 0600); err != nil {
					t.Fatal(err)
				}
				args := append(append([]string{}, prefix...), action)
				if action == "set" {
					args = append(args, "base-url", "https://new.example.test", "--profile", "new")
				}
				ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
				defer cancel()
				cmd := exec.CommandContext(ctx, cliBinary, args...)
				cmd.Env = childEnv(home, "", false)
				cmd.Stdin = strings.NewReader("new\nnew-key\nnew-secret\n")
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("config writer: %v %s", err, out)
				}
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Contains(raw, []byte("# Preserve this comment")) {
					t.Error("comment lost")
				}
				var document map[string]interface{}
				if err := yaml.Unmarshal(raw, &document); err != nil {
					t.Fatal(err)
				}
				profiles := document["profiles"].(map[string]interface{})
				if profiles["new"] == nil {
					t.Fatal("command did not update selected profile")
				}
				if document["oauth2"] == nil || document["custom"] != "keep" {
					t.Fatal("writer erased OAuth or unknown configuration")
				}
				if !bytes.Contains(raw, []byte("synthetic-rt")) || !bytes.Contains(raw, []byte("https://custom.example.test/mcp")) || !bytes.Contains(raw, []byte("keep-key")) {
					t.Fatal("writer overwrote unrelated state")
				}
			})
		}
	}
}

func TestConfigListMasksOAuthCredentials(t *testing.T) {
	for _, prefix := range [][]string{{"config"}} {
		for _, body := range []string{
			"oauth2: {token: {access_token: synthetic-sensitive-at, refresh_token: synthetic-sensitive-rt}}\n",
			"oauth2:\n  tokens:\n    - access_token: synthetic-sensitive-at\n      refresh_token: synthetic-sensitive-rt\n",
		} {
			t.Run(strings.Join(prefix, " ")+body[:6], func(t *testing.T) {
				home := t.TempDir()
				if err := os.Mkdir(filepath.Join(home, ".gate-cli"), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(home, ".gate-cli/config.yaml"), []byte(body), 0600); err != nil {
					t.Fatal(err)
				}
				args := append(append([]string{}, prefix...), "list")
				stdout, stderr, err := sameHomeCLI(t, home, "", args...)
				if err != nil {
					t.Fatalf("list: %v %s", err, stderr)
				}
				if bytes.Contains(stdout, []byte("synthetic-sensitive-")) {
					t.Fatal("config list exposed OAuth credential")
				}
			})
		}
	}
}

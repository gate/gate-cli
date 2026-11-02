package cmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gate/gate-cli/tests/contracts"
)

// The isolated command-package child exercises Execute itself without binding
// these tests to any not-yet-implemented helper or changing an existing signature.
func TestCommandPolicyHelperProcess(t *testing.T) {
	if os.Getenv("CLI_COMMAND_TEST_HELPER") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{"gate-cli"}, os.Args[i+1:]...)
			Execute()
			os.Exit(0)
		}
	}
	os.Exit(90)
}

func TestCommandPolicyBooleanCompatibility(t *testing.T) {
	for _, tc := range contracts.ArgumentScenarios(t) {
		t.Run(tc.ID, func(t *testing.T) {
			peer := contracts.NewPeer(t, tc.Case, "hmac", "success")
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			args := append([]string{"-test.run=^TestCommandPolicyHelperProcess$", "--", "--format", "json"}, tc.Case.Args...)
			child := exec.CommandContext(ctx, executable, args...)
			home := t.TempDir()
			env := []string{}
			for _, v := range os.Environ() {
				key, _, _ := strings.Cut(v, "=")
				if !strings.HasPrefix(key, "GATE_") && key != "CLI_COMMAND_TEST_HELPER" && key != "HOME" && key != "USERPROFILE" && key != "XDG_CONFIG_HOME" && key != "XDG_CACHE_HOME" && key != "LOCALAPPDATA" && key != "APPDATA" {
					env = append(env, v)
				}
			}
			for _, key := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "LOCALAPPDATA", "APPDATA"} {
				env = append(env, key+"="+home)
			}
			child.Env = append(env, "CLI_COMMAND_TEST_HELPER=1", "GATE_BASE_URL="+peer.URL, "GATE_API_KEY=synthetic-key", "GATE_API_SECRET=synthetic-secret")
			var stdout, stderr bytes.Buffer
			child.Stdout = &stdout
			child.Stderr = &stderr
			err = child.Run()
			if tc.Reject {
				peer.AssertCalls(0)
				if err == nil || stdout.Len() != 0 {
					t.Errorf("invalid argv accepted: %v", err)
				}
				return
			}
			peer.AssertCalls(len(tc.Case.Exchanges))
			if err != nil || stderr.Len() != 0 || !contracts.EqualJSON(stdout.Bytes(), tc.Case.Want) {
				t.Errorf("boolean/argv changed: %v %s %s", err, &stdout, &stderr)
			}
		})
	}
}

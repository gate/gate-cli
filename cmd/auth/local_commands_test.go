package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gate/gate-cli/internal/oauth2"
	"github.com/spf13/cobra"
)

// TestLocalAuthCommands 覆盖无凭据、三种本地状态；任何状态输出都不能含Token。
func TestLocalAuthCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	oauth2.ResetHomeCacheForTest()
	defer oauth2.ResetHomeCacheForTest()
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(&out)
	if err := runStatus(cmd, nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Not logged in") {
		t.Fatal("missing logged out status")
	}
	for _, state := range []string{"authenticated", "expired (refreshable)", "expired"} {
		token := &oauth2.TokenData{ClientID: "synthetic", AccessToken: "sensitive-at", RefreshToken: "sensitive-rt", Scopes: "stock", Resource: "https://api.gateio.ws/api/v4", ExpiresAt: time.Now().Add(time.Hour)}
		if state != "authenticated" {
			token.ExpiresAt = time.Now().Add(-time.Hour)
		}
		if state == "expired" {
			token.RefreshToken = ""
		}
		if err := oauth2.SaveToken(token); err != nil {
			t.Fatal(err)
		}
		out.Reset()
		if err := runStatus(cmd, nil); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), state) || strings.Contains(out.String(), "sensitive-") {
			t.Fatalf("invalid or sensitive status: %s", out.String())
		}
	}
}

// TestLocalStatusJSON verifies the public machine contract without contacting
// the issuer; a local expiry observation never claims remote authorization.
func TestLocalStatusJSON(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	oauth2.ResetHomeCacheForTest()
	defer oauth2.ResetHomeCacheForTest()
	cmd := &cobra.Command{}
	cmd.PersistentFlags().String("format", "json", "")
	var out bytes.Buffer
	cmd.SetOut(&out)
	for _, want := range []string{"not_logged_in", "authenticated", "expired (refreshable)", "expired"} {
		if want != "not_logged_in" {
			token := &oauth2.TokenData{ClientID: "synthetic", AccessToken: "private-at", RefreshToken: "private-rt", Scopes: "base earn", Resource: "https://api.gateio.ws/api/v4", ExpiresAt: time.Now().Add(time.Hour)}
			if strings.HasPrefix(want, "expired") {
				token.ExpiresAt = time.Now().Add(-time.Hour)
			}
			if want == "expired" {
				token.RefreshToken = ""
			}
			if err := oauth2.SaveToken(token); err != nil {
				t.Fatal(err)
			}
		}
		out.Reset()
		if err := runStatus(cmd, nil); err != nil {
			t.Fatal(err)
		}
		var state map[string]interface{}
		if err := json.Unmarshal(out.Bytes(), &state); err != nil {
			t.Fatal(err)
		}
		if state["status"] != want || state["remote_checked"] != false || state["issuer"] != oauth2.ShortDiscoveryURL(oauth2.CurrentDiscoveryURL()) || strings.Contains(out.String(), "private-") {
			t.Fatalf("invalid public status projection: %s", out.String())
		}
	}
	cmd.SetOut(statusBrokenWriter{})
	if err := runStatus(cmd, nil); !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("output failure lost: %v", err)
	}
}

// statusBrokenWriter exercises the output failure branch without global IO.
type statusBrokenWriter struct{}

func (statusBrokenWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestRequestedScopes(t *testing.T) {
	for _, raw := range []string{"stock,stock,base", "", "trade", "read", "mcp:tools"} {
		cmd := &cobra.Command{}
		cmd.Flags().String("scope", "", "")
		if err := cmd.Flags().Set("scope", raw); err != nil {
			t.Fatal(err)
		}
		actual, err := requestedScopes(cmd)
		if raw == "stock,stock,base" {
			if err != nil || actual != "base stock" {
				t.Fatal(actual, err)
			}
		} else if err == nil {
			t.Errorf("invalid scope %q", raw)
		}
	}
}

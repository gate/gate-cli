package oauth2

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Result persistence is a write-ahead barrier: a Token write may not happen
// until its digest has a durable journal record. No secrets belong in that record.
func TestRecoveryResultWriteAhead(t *testing.T) {
	for _, failure := range []string{"none", "journal", "token"} {
		t.Run(failure, func(t *testing.T) {
			home, cleanup := setupTestTokenDir(t)
			defer cleanup()
			writeTestToken(t, &TokenData{ClientID: "synthetic", AccessToken: "synthetic-at", RefreshToken: "synthetic-rt", Resource: "https://api.gateio.ws/api/v4"})
			old, err := LoadToken()
			if err != nil {
				t.Fatal(err)
			}
			op, err := BeginAuthOperation(t.Context(), "login", old)
			if err != nil {
				t.Fatal(err)
			}
			if err = op.Dispatch(t.Context(), nil); err != nil {
				t.Fatal(err)
			}
			record, err := readOperation()
			if err != nil {
				t.Fatal(err)
			}
			journal, _ := operationPath()
			configPath := filepath.Join(home, ".gate-cli", "config.yaml")
			if failure == "journal" {
				if err = os.Remove(journal); err != nil {
					t.Fatal(err)
				}
				if err = os.Mkdir(journal, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if failure == "token" {
				if err = os.WriteFile(configPath, []byte("broken: ["), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			burned := *old
			burned.RefreshToken = ""
			err = WithConfigLock(t.Context(), func() error { return saveOperationResultLocked(record, &burned) })
			if (err != nil) != (failure != "none") {
				t.Fatalf("failure=%s err=%v", failure, err)
			}
			if failure == "journal" || failure == "token" {
				after, e := os.ReadFile(configPath)
				if e != nil || !bytes.Equal(before, after) {
					t.Fatal("failed save mutated token", e)
				}
			}
			if failure != "journal" {
				record, e := readOperation()
				if e != nil || record.Result != credentialDigest(&burned) {
					t.Fatal("write-ahead digest missing", e)
				}
				raw, e := os.ReadFile(journal)
				if e != nil || bytes.Contains(raw, []byte("synthetic-at")) || bytes.Contains(raw, []byte("synthetic-rt")) {
					t.Fatal("secret in journal", e)
				}
			}
		})
	}
}

// These are the actual write-ahead boundaries, built from a real dispatched
// operation. Missing evidence must still reject an otherwise plausible RT burn.
func TestRecoveryResultBoundaries(t *testing.T) {
	for _, stage := range []string{"before-token", "after-token", "unjournaled", "foreign"} {
		t.Run(stage, func(t *testing.T) {
			_, cleanup := setupTestTokenDir(t)
			defer cleanup()
			writeTestToken(t, &TokenData{ClientID: "synthetic", AccessToken: "synthetic-at", RefreshToken: "synthetic-rt", Resource: "https://api.gateio.ws/api/v4"})
			old, err := LoadToken()
			if err != nil {
				t.Fatal(err)
			}
			op, err := BeginAuthOperation(t.Context(), "login", old)
			if err != nil {
				t.Fatal(err)
			}
			if err = op.Dispatch(t.Context(), nil); err != nil {
				t.Fatal(err)
			}
			record, err := readOperation()
			if err != nil {
				t.Fatal(err)
			}
			record.PID = 99999999
			if processAlive(record.PID) {
				t.Fatal("dead PID fixture is alive")
			}
			burned := *old
			burned.RefreshToken = ""
			if stage != "unjournaled" {
				record.Result = credentialDigest(&burned)
			}
			if stage == "foreign" {
				record.Result = strings.Repeat("0", 64)
			}
			if err = writeOperation(record); err != nil {
				t.Fatal(err)
			}
			expected := old
			if stage != "before-token" {
				if err = SaveToken(&burned); err != nil {
					t.Fatal(err)
				}
				expected, err = LoadToken()
				if err != nil {
					t.Fatal(err)
				}
			}
			next, err := BeginAuthOperation(t.Context(), "login", expected)
			if stage == "unjournaled" || stage == "foreign" {
				if err == nil || !strings.Contains(err.Error(), "provenance") {
					t.Fatal("unproven state accepted", err)
				}
				return
			}
			if stage == "before-token" {
				var expired *TokenExpiredError
				if !errors.As(err, &expired) {
					t.Fatal("old snapshot dispatched after recovery", err)
				}
				expected, err = LoadToken()
				if err != nil {
					t.Fatal(err)
				}
				next, err = BeginAuthOperation(t.Context(), "login", expected)
			}
			if err != nil || next == nil {
				t.Fatal("proven state cannot recover", err)
			}
			if expected.RefreshToken != "" || expected.AccessToken != "synthetic-at" {
				t.Fatal("burned RT revived or AT lost")
			}
			if err = next.Finish(t.Context()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

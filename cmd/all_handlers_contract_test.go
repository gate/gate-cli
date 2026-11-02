package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/gate/gate-cli/internal/oauth2"
	"github.com/gate/gate-cli/tests/contracts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

// Handler tests execute each command's real RunE and SDK against a strict local
// HTTP peer. Environment and global Cobra flags are restored after every case;
// no subtest runs in parallel because the production tree/stdout are globals.
func invokeContractHandler(t *testing.T, tc contracts.Case, base string, ctx context.Context, brokenOutput ...bool) ([]byte, error) {
	t.Helper()
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if strings.HasPrefix(key, "GATE_") {
			t.Setenv(key, "")
		}
	}
	home := t.TempDir()
	for _, key := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "LOCALAPPDATA", "APPDATA"} {
		t.Setenv(key, home)
	}
	t.Setenv("GATE_BASE_URL", base)
	t.Setenv("GATE_API_KEY", "synthetic-key")
	t.Setenv("GATE_API_SECRET", "synthetic-secret")
	oauth2.ResetHomeCacheForTest()
	t.Cleanup(oauth2.ResetHomeCacheForTest)
	c, rest, err := rootCmd.Find(strings.Fields(tc.Name))
	require.NoError(t, err)
	require.Empty(t, rest)
	require.NotNil(t, c.RunE)
	// Slice flags append on Set, so restoration must use SliceValue.Replace.
	seen := map[*pflag.Flag]bool{}
	save := func(f *pflag.Flag) {
		if seen[f] {
			return
		}
		seen[f] = true
		value, changed := f.Value.String(), f.Changed
		if slice, ok := f.Value.(pflag.SliceValue); ok {
			original := append([]string{}, slice.GetSlice()...)
			t.Cleanup(func() { require.NoError(t, slice.Replace(original)); f.Changed = changed })
		} else {
			t.Cleanup(func() { require.NoError(t, f.Value.Set(value)); f.Changed = changed })
		}
	}
	for parent := c; parent != nil; parent = parent.Parent() {
		parent.Flags().VisitAll(save)
		parent.PersistentFlags().VisitAll(save)
	}
	require.NoError(t, rootCmd.PersistentFlags().Set("format", "json"))
	previousContext := c.Context()
	c.SetContext(ctx)
	t.Cleanup(func() { c.SetContext(previousContext) })
	require.NoError(t, c.ParseFlags(tc.Args[len(strings.Fields(tc.Name)):]))
	require.NoError(t, c.ValidateArgs(c.Flags().Args()))
	require.NoError(t, c.ValidateRequiredFlags())
	// File-backed capture cannot deadlock when a handler emits more than a pipe
	// buffer. The bounded fixture/output check keeps test artifacts small.
	f, err := os.CreateTemp(t.TempDir(), "handler-output-")
	require.NoError(t, err)
	previous := os.Stdout
	os.Stdout = f
	closed := len(brokenOutput) != 0 && brokenOutput[0]
	if closed {
		require.NoError(t, f.Close())
	}
	defer func() {
		os.Stdout = previous
		if !closed {
			require.NoError(t, f.Close())
		}
	}()
	err = c.RunE(c, c.Flags().Args())
	if closed {
		return nil, err
	}
	_, seekErr := f.Seek(0, io.SeekStart)
	require.NoError(t, seekErr)
	out, readErr := io.ReadAll(io.LimitReader(f, 65537))
	require.NoError(t, readErr)
	require.LessOrEqual(t, len(out), 65536)
	return out, err
}

// An empty successful list is a valid business outcome, distinct from the
// malformed-body and HTTP-failure cases. Detail/ticker lookups are excluded:
// those commands may legitimately require an entity to exist.
func TestAllCEXEmptyListResults(t *testing.T) {
	listNames := map[string]bool{"list": true, "orders": true, "history": true, "records": true, "repay-records": true, "currencies": true, "contracts": true, "pairs": true, "plans": true, "lends": true, "interest-records": true, "settlements": true, "transactions": true, "positions": true}
	for _, original := range contracts.Cases(t) {
		parts := strings.Fields(original.Name)
		last := original.Exchanges[len(original.Exchanges)-1]
		if !listNames[parts[len(parts)-1]] || last.Method != "GET" || !bytes.HasPrefix(bytes.TrimSpace(last.Reply), []byte("[")) {
			continue
		}
		t.Run(original.Name, func(t *testing.T) {
			tc := original
			tc.Exchanges = append([]contracts.Exchange{}, original.Exchanges...)
			tc.Exchanges[len(tc.Exchanges)-1].Reply = []byte(`[]`)
			auth := "hmac"
			if tc.Public {
				auth = "anonymous"
			}
			peer := contracts.NewPeer(t, tc, auth, "success")
			out, err := invokeContractHandler(t, tc, peer.URL, t.Context())
			require.NoError(t, err)
			require.True(t, contracts.EqualJSON(out, []byte(`[]`)), "empty list output=%s", out)
			peer.AssertCalls(len(tc.Exchanges))
		})
	}
}

// Every handler must propagate local setup/output failures as well as HTTP
// errors. A successful write followed by a broken stdout must not be replayed.
func TestAllCEXHandlerLocalFailures(t *testing.T) {
	for _, tc := range contracts.Cases(t) {
		for _, failure := range []string{"configuration", "stdout"} {
			t.Run(tc.Name+"/"+failure, func(t *testing.T) {
				auth := "hmac"
				if tc.Public {
					auth = "anonymous"
				}
				peer := contracts.NewPeer(t, tc, auth, "success")
				base := peer.URL
				if failure == "configuration" {
					base = "invalid://fixture"
				}
				out, err := invokeContractHandler(t, tc, base, t.Context(), failure == "stdout")
				require.Error(t, err)
				require.Empty(t, out)
				want := len(tc.Exchanges)
				if failure == "configuration" {
					want = 0
				}
				peer.AssertCalls(want)
			})
		}
	}
}

func TestAllCEXMalformedJSONInputs(t *testing.T) {
	for _, original := range contracts.Cases(t) {
		for _, flag := range []string{"--json", "--orders-json", "--input"} {
			for i, arg := range original.Args {
				if arg != flag {
					continue
				}
				t.Run(original.Name+"/"+flag, func(t *testing.T) {
					tc := original
					tc.Args = append([]string{}, original.Args...)
					tc.Args[i+1] = "{invalid"
					peer := contracts.NewPeer(t, tc, "hmac", "success")
					out, err := invokeContractHandler(t, tc, peer.URL, t.Context())
					require.Error(t, err)
					require.Empty(t, out)
					peer.AssertCalls(0)
				})
			}
		}
	}
}

func TestAllCEXHandlerContracts(t *testing.T) {
	for _, tc := range contracts.Cases(t) {
		t.Run(tc.Name, func(t *testing.T) {
			for _, fault := range []string{"success", "403", "503"} {
				t.Run(fault, func(t *testing.T) {
					auth := "hmac"
					if tc.Public {
						auth = "anonymous"
					}
					peer := contracts.NewPeer(t, tc, auth, fault)
					out, err := invokeContractHandler(t, tc, peer.URL, t.Context())
					peer.AssertCalls(len(tc.Exchanges))
					if fault == "success" {
						require.NoError(t, err)
						require.True(t, contracts.EqualJSON(out, tc.Want), "output=%s want=%s", out, tc.Want)
					} else {
						require.Error(t, err)
						require.Empty(t, out, "a rejected dependency must not produce success output")
					}
				})
			}
		})
	}
}

// Pretty mode reaches the per-command table projections that JSON mode skips.
// Keep the same strict wire contract; rendering must not change a trade request.
func TestAllCEXHandlerHumanRendering(t *testing.T) {
	for _, original := range contracts.Cases(t) {
		t.Run(original.Name, func(t *testing.T) {
			tc := original
			tc.Args = append(append([]string{}, tc.Args...), "--format", "pretty")
			auth := "hmac"
			if tc.Public {
				auth = "anonymous"
			}
			peer := contracts.NewPeer(t, tc, auth, "success")
			out, err := invokeContractHandler(t, tc, peer.URL, t.Context())
			require.NoError(t, err)
			// Freeze the complete headers/cell projection, so a successful
			// request followed by a missing or swapped column cannot pass.
			require.Equal(t, tc.WantPretty, string(out))
			peer.AssertCalls(len(tc.Exchanges))
		})
	}
}

func TestCEXBusinessContractCompleteness(t *testing.T) {
	var declared, exercised []string
	for _, c := range loadCommandContracts(t) {
		if strings.HasPrefix(c.Path, "cex ") && !strings.HasPrefix(c.Path, "cex auth ") {
			declared = append(declared, c.Path)
		}
	}
	for _, c := range contracts.Cases(t) {
		exercised = append(exercised, c.Name)
	}
	sort.Strings(declared)
	sort.Strings(exercised)
	require.Equal(t, declared, exercised, "every published CEX business handler needs an executable wire contract")
}

// Cancellation must propagate into every SDK handler and prevent any request;
// this catches handlers that accidentally create a fresh background context.
func TestAllCEXHandlerCancelledContext(t *testing.T) {
	for _, tc := range contracts.Cases(t) {
		t.Run(tc.Name, func(t *testing.T) {
			peer := contracts.NewPeer(t, tc, "hmac", "success")
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			out, err := invokeContractHandler(t, tc, peer.URL, ctx)
			require.Error(t, err)
			require.Empty(t, out)
			peer.AssertCalls(0)
		})
	}
}

// Keep Cobra's validation separate from RunE tests so missing flags are tested
// even for handlers that correctly assume Cobra already checked annotations.
func TestAllCEXRequiredFlagOmissions(t *testing.T) {
	for _, tc := range contracts.Cases(t) {
		c, _, err := rootCmd.Find(strings.Fields(tc.Name))
		require.NoError(t, err)
		c.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
			if len(f.Annotations[cobra.BashCompOneRequiredFlag]) == 0 {
				return
			}
			t.Run(tc.Name+"/"+f.Name, func(t *testing.T) {
				old := f.Changed
				f.Changed = false
				defer func() { f.Changed = old }()
				require.ErrorContains(t, c.ValidateRequiredFlags(), f.Name)
			})
		})
	}
}

package auth

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/gate/gate-cli/internal/oauth2"
)

const fakeAuthorizeURL = "https://example.invalid/oauth/authorize?client_id=abc&state=xyz&code_challenge=pqr"

func TestAnnounceAuthorizeURL_TTYMode(t *testing.T) {
	// TTY mode: URL must appear on stdout, stderr must be empty.
	var out, stderr bytes.Buffer
	assert.NoError(t, announceAuthorizeURL(&out, &stderr, fakeAuthorizeURL, "auto", false, false))

	stdoutStr := out.String()
	assert.Contains(t, stdoutStr, "Authorization URL")
	assert.Contains(t, stdoutStr, fakeAuthorizeURL)
	assert.Empty(t, stderr.String(), "agent markers must not leak to stderr when stderr is a TTY")
}

func TestAnnounceAuthorizeURL_AgentModeAuto(t *testing.T) {
	var out, stderr bytes.Buffer
	assert.NoError(t, announceAuthorizeURL(&out, &stderr, fakeAuthorizeURL, "auto", true, false))

	// Human-readable block still on stdout.
	assert.Contains(t, out.String(), fakeAuthorizeURL)

	// Machine-parseable markers on stderr.
	stderrStr := stderr.String()
	assert.Contains(t, stderrStr, "GATE_CLI_AUTH_URL="+fakeAuthorizeURL+"\n")
	assert.Contains(t, stderrStr, "GATE_CLI_AUTH_HINT=")
	assert.Contains(t, stderrStr, "--manual", "auto-mode hint must mention --manual fallback")

	// The URL must appear on its own line (no additional suffix) so regex
	// extraction `^GATE_CLI_AUTH_URL=(.+)$` captures exactly the URL.
	for _, line := range strings.Split(strings.TrimRight(stderrStr, "\n"), "\n") {
		if strings.HasPrefix(line, "GATE_CLI_AUTH_URL=") {
			assert.Equal(t, "GATE_CLI_AUTH_URL="+fakeAuthorizeURL, line)
		}
	}
}

func TestAnnounceAuthorizeURL_AgentModeManual(t *testing.T) {
	var out, stderr bytes.Buffer
	assert.NoError(t, announceAuthorizeURL(&out, &stderr, fakeAuthorizeURL, "manual", true, false))

	stderrStr := stderr.String()
	assert.Contains(t, stderrStr, "GATE_CLI_AUTH_URL="+fakeAuthorizeURL)
	assert.Contains(t, stderrStr, "GATE_CLI_AUTH_HINT=")
	// Manual-mode hint should talk about pasting the code back, NOT about --manual itself.
	assert.Contains(t, stderrStr, "paste the authorization code")
	assert.NotContains(t, stderrStr, "--manual")
}

func TestAnnounceAuthorizeURL_UnknownModeFallsBackToAuto(t *testing.T) {
	var out, stderr bytes.Buffer
	assert.NoError(t, announceAuthorizeURL(&out, &stderr, fakeAuthorizeURL, "weird-mode", true, false))

	// Any unknown mode should behave like "auto" (mentions --manual).
	assert.Contains(t, stderr.String(), "--manual")
}

func TestFormatClickableURL_DisabledReturnsPlain(t *testing.T) {
	got := formatClickableURL(fakeAuthorizeURL, false)
	assert.Equal(t, fakeAuthorizeURL, got)
	assert.NotContains(t, got, "\x1b")
}

func TestFormatClickableURL_EnabledWrapsInOSC8(t *testing.T) {
	got := formatClickableURL(fakeAuthorizeURL, true)
	// Expected: ESC ] 8 ; ; URL BEL URL ESC ] 8 ; ; BEL
	want := "\x1b]8;;" + fakeAuthorizeURL + "\x07" + fakeAuthorizeURL + "\x1b]8;;\x07"
	assert.Equal(t, want, got)
	// The visible URL text is still present (for non-OSC-8 terminals).
	assert.Contains(t, got, fakeAuthorizeURL)
}

func TestAnnounceAuthorizeURL_HyperlinksWrapStdoutOnly(t *testing.T) {
	var out, stderr bytes.Buffer
	assert.NoError(t, announceAuthorizeURL(&out, &stderr, fakeAuthorizeURL, "auto", true, true))

	stdoutStr := out.String()
	// stdout should contain the OSC 8 prefix and the full escape sequence.
	assert.Contains(t, stdoutStr, "\x1b]8;;"+fakeAuthorizeURL+"\x07")
	assert.Contains(t, stdoutStr, fakeAuthorizeURL+"\x1b]8;;\x07")

	// stderr marker MUST carry the raw URL — no escape bytes — so agents can
	// extract it with a plain regex.
	stderrStr := stderr.String()
	assert.Contains(t, stderrStr, "GATE_CLI_AUTH_URL="+fakeAuthorizeURL+"\n")
	for _, line := range strings.Split(strings.TrimRight(stderrStr, "\n"), "\n") {
		if strings.HasPrefix(line, "GATE_CLI_AUTH_URL=") {
			assert.NotContains(t, line, "\x1b", "stderr marker must not contain ANSI/OSC escapes")
		}
	}
}

func TestWatchStdinEnter_EmptyStdinDoesNotTriggerManual(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	assert.False(t, (&inputSession{in: strings.NewReader("")}).watchEnter(ctx), "EOF stdin must not look like Enter (agent/CI)")
}

func TestWatchStdinEnter_NewlineTriggersManual(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	assert.True(t, (&inputSession{in: strings.NewReader("\n")}).watchEnter(ctx))
}

func TestCanReuseClient(t *testing.T) {
	const discA = "https://a.invalid/.well-known/oauth-authorization-server"
	const discB = "https://b.invalid/.well-known/oauth-authorization-server"
	const redirect = "http://127.0.0.1:18991/callback"

	cached := &oauth2.TokenData{
		ClientID:               "pkce_cached",
		DiscoveryURL:           discA,
		RegisteredRedirectURIs: []string{redirect},
	}

	t.Run("match - reuses", func(t *testing.T) {
		assert.True(t, canReuseClient(cached, redirect, discA))
	})

	t.Run("discovery url mismatch - re-register", func(t *testing.T) {
		assert.False(t, canReuseClient(cached, redirect, discB),
			"switching discovery_url must invalidate the cached client_id")
	})

	t.Run("legacy cache without discovery url - re-register", func(t *testing.T) {
		legacy := &oauth2.TokenData{
			ClientID:               "pkce_legacy",
			RegisteredRedirectURIs: []string{redirect},
		}
		assert.False(t, canReuseClient(legacy, redirect, discA),
			"empty cached DiscoveryURL must not silently match a real discovery_url")
	})

	t.Run("nil cache - re-register", func(t *testing.T) {
		assert.False(t, canReuseClient(nil, redirect, discA))
	})

	t.Run("empty client id - re-register", func(t *testing.T) {
		assert.False(t, canReuseClient(&oauth2.TokenData{DiscoveryURL: discA}, redirect, discA))
	})

	t.Run("required uri not covered - re-register", func(t *testing.T) {
		assert.False(t, canReuseClient(cached, "http://127.0.0.1:22222/callback", discA))
	})

	t.Run("empty required uri - reuses when discovery matches", func(t *testing.T) {
		assert.True(t, canReuseClient(cached, "", discA))
	})
}

func TestWatchStdinEnter_NonNewlineBytesIgnoredUntilNewline(t *testing.T) {
	pr, pw := io.Pipe()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	go func() {
		_, _ = pw.Write([]byte("xy"))
		_ = pw.Close()
	}()
	assert.False(t, (&inputSession{in: pr}).watchEnter(ctx), "partial input without newline should wait until ctx timeout")
}

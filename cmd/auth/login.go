package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gate/gate-cli/internal/cmdutil"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/gate/gate-cli/internal/oauth2"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login via OAuth2 PKCE (auto browser → --manual fallback)",
	Long: `Authenticate with Gate using OAuth2 PKCE and an Agentic subaccount.

The browser selects or creates the subaccount and confirms the granted permissions.
By default all six permissions are requested; --scope requests an explicit subset.
Use --manual to copy the authorization URL into a browser and paste the code here.
Use --force to register a new client and authorize again. Existing credentials are
retained until a new result is saved. A cancelled login keeps the old local state.

Examples:
  gate-cli cex auth login
  gate-cli cex auth login --manual --scope base,stock
  gate-cli cex auth login --force
`,
	RunE: runLogin,
}

func init() {
	loginCmd.Flags().Bool("manual", false, "Manual mode: copy URL to browser, paste code back")
	loginCmd.Flags().Bool("force", false, "Register a new client and re-authorize; retain old credentials until success")
	loginCmd.Flags().String("scope", "", "Requested subset of base,spot_trade,futures_trade,earn,cfd,stock (default: all six)")
}

type loginContextKey struct{}
type loginContext struct {
	resource, scopes string
	before           *oauth2.TokenData
	force            bool
	registration     *dcrResult
}

func loginOptions(cmd *cobra.Command) *loginContext {
	return cmd.Context().Value(loginContextKey{}).(*loginContext)
}

// requestedScopes 复用唯一六项常量，显式空/未知值拒绝，重复值按固定顺序去重。
func requestedScopes(cmd *cobra.Command) (string, error) {
	if !cmd.Flags().Changed("scope") {
		return oauth2.DefaultScopes, nil
	}
	raw, _ := cmd.Flags().GetString("scope")
	items := strings.Fields(strings.ReplaceAll(raw, ",", " "))
	if len(items) == 0 {
		return "", fmt.Errorf("scope cannot be empty")
	}
	selected := map[string]bool{}
	for _, item := range items {
		if !strings.Contains(" "+oauth2.DefaultScopes+" ", " "+item+" ") {
			return "", fmt.Errorf("unsupported scope %q", item)
		}
		selected[item] = true
	}
	ordered := make([]string, 0, len(selected))
	for _, item := range strings.Fields(oauth2.DefaultScopes) {
		if selected[item] {
			ordered = append(ordered, item)
		}
	}
	return strings.Join(ordered, " "), nil
}
func runLogin(cmd *cobra.Command, args []string) error {
	manual, _ := cmd.Flags().GetBool("manual")
	force, _ := cmd.Flags().GetBool("force")
	scopes, err := requestedScopes(cmd)
	if err != nil {
		return err
	}
	cfg, err := cmdutil.GetConfig(cmd)
	if err != nil {
		return err
	}
	resource, err := oauth2.APIResource(cfg.BaseURL)
	if err != nil {
		return err
	}
	// 旧文件只通过 store 的统一写入口迁移，不推断 resource，也不复用旧 RT。
	if err := oauth2.MigrateLegacyTokenFile(cmd.ErrOrStderr()); err != nil {
		return err
	}
	existing, err := oauth2.LoadToken()
	if err != nil {
		return err
	}
	options := &loginContext{resource: resource, scopes: scopes, before: existing, force: force}
	cmd.SetContext(context.WithValue(cmd.Context(), loginContextKey{}, options))
	bound := existing != nil && existing.Resource == resource && oauth2.NormalizeDiscoveryURL(existing.DiscoveryURL) == oauth2.CurrentDiscoveryURL()
	// 明确的scope/force意图必须进入完整授权，不能被本地有效AT短路。
	if bound && !force && !cmd.Flags().Changed("scope") && existing.IsValid() {
		if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Locally logged in (scopes: %s); remote access remains subject to authorization.\n", existing.Scopes); err != nil {
			return err
		}
		return nil
	}
	if err := oauth2.InitEndpointsContext(cmd.Context()); err != nil {
		return err
	}
	if bound && !force && !cmd.Flags().Changed("scope") && existing.HasRefreshToken() {
		_, err := oauth2.RefreshIfNeeded(cmd.Context(), resource)
		return err
	}
	// Step 2: Generate PKCE
	verifier, err := oauth2.GenerateVerifier()
	if err != nil {
		return fmt.Errorf("generate PKCE verifier: %w", err)
	}
	challenge := oauth2.GenerateChallenge(verifier)

	// Generate state for CSRF protection
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return fmt.Errorf("generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	if manual {
		return runManualFlow(cmd, verifier, challenge, state, existing, &inputSession{in: cmd.InOrStdin()})
	}
	return runAutoFlow(cmd, verifier, challenge, state, existing)
}

// runAutoFlow opens a browser and listens on a local callback server.
func runAutoFlow(cmd *cobra.Command, verifier, challenge, state string, existing *oauth2.TokenData) (err error) {
	// Bind to first available port synchronously — we need the redirect_uri before DCR
	cbServer, err := oauth2.NewCallbackServer()
	if err != nil {
		return err
	}
	// Release the bound port on any error path so repeated failed logins don't
	// exhaust 18991-18995. The happy path closes cbServer explicitly below
	// before returning; the defer then becomes a no-op (Close is idempotent).
	defer func() {
		if err != nil {
			cbServer.Close()
		}
	}()
	redirectURI := cbServer.RedirectURI

	// DCR with all 6 redirect_uris (5 local ports + display page) per BR-6.
	// Bind the cached client to the current discovery_url so switching
	// authorization servers (e.g. staging ↔ prod) forces re-registration.
	currentDiscoveryURL := oauth2.CurrentDiscoveryURL()
	dcr, err := ensureClientID(cmd, existing, redirectURI, currentDiscoveryURL)
	if err != nil {
		return err
	}

	// Build authorize URL and announce it to the user *before* trying the
	// browser auto-open. Users on SSH / remote boxes still need the URL copy
	// to paste into a browser on another device, and agents need a parseable
	// marker to surface the link to their human.
	authorizeURL := oauth2.BuildAuthorizeURL(dcr.ClientID, redirectURI, state, challenge, loginOptions(cmd).scopes, loginOptions(cmd).resource)
	out := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	agent := stderrIsAgent()
	if err := announceAuthorizeURL(out, stderr, authorizeURL, "auto", agent, stdoutSupportsHyperlinks()); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(out, "Opening your default browser now. If it doesn't open or you're on another device:"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "  • Copy the Authorization URL above into any browser (this device or another)."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "  • No browser on this machine? Press Enter to switch to manual mode."); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "  • Press Ctrl+C to abort."); err != nil {
		return err
	}
	if err := openBrowser(cmd.Context(), authorizeURL); err != nil {
		if _, err := fmt.Fprintln(out); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "Could not auto-open a browser: %v\n", err); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, "  • Copy the Authorization URL above into any browser to continue, OR"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, "  • Press Enter now to switch to manual mode (paste the code back here), OR"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, "  • Re-run with `--manual` for a fully headless flow."); err != nil {
			return err
		}
		if agent {
			if _, err := fmt.Fprintln(stderr, "GATE_CLI_AUTH_HINT=Browser auto-open failed. Open the URL in any browser, or re-run with --manual."); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintln(out, "Waiting for authorization in your browser... (Enter = manual mode, Ctrl+C = abort)"); err != nil {
		return err
	}

	// 自动和手动阶段共用换行状态；交接前等待stdinDone，保证单读者拥有权。
	input := &inputSession{in: cmd.InOrStdin()}

	// Three-way select: callback / stdin Enter / 180s timeout (§4.3.3 A5)
	ctx, cancel := context.WithTimeout(cmd.Context(), 180*time.Second)
	defer cancel()

	// Start stdin listener for manual mode switch. The goroutine is driven off
	// input.watchEnter which returns when either stdin signals or ctx is done,
	// so we do not leak a goroutine after the select below picks another case.
	stdinCh := make(chan struct{}, 1)
	stdinDone := make(chan struct{})
	defer func() { cancel(); <-stdinDone }()
	go func() {
		defer close(stdinDone)
		defer func() {
			if r := recover(); r != nil {
				log.Printf("stdin watcher panicked: %v", r)
			}
		}()
		if input.watchEnter(ctx) {
			select {
			case stdinCh <- struct{}{}:
			default:
			}
		}
	}()

	// Start callback listener.
	// The goroutine must recover from any panic in WaitForCallback — the inner
	// Serve goroutine already recovers, but a panic between our call here and
	// the Serve launch (e.g. in mux setup) would otherwise crash the CLI.
	callbackCh := make(chan *oauth2.CallbackResult, 1)
	callbackErrCh := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				select {
				case callbackErrCh <- fmt.Errorf("callback goroutine panicked: %v", r):
				default:
				}
			}
		}()
		r, e := cbServer.WaitForCallback(ctx)
		if e != nil {
			callbackErrCh <- e
		} else {
			callbackCh <- r
		}
	}()

	var result *oauth2.CallbackResult
	select {
	case result = <-callbackCh:
		// Got callback — continue below
	case err := <-callbackErrCh:
		return fmt.Errorf("callback server: %w", err)
	case <-stdinCh:
		cancel()         // stop callback server + signal any remaining watchers
		cbServer.Close() // release bound port immediately
		<-stdinDone
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Switching to manual mode..."); err != nil {
			return err
		}
		return runManualFlow(cmd, verifier, challenge, state, existing, input)
	case <-ctx.Done():
		if cmd.Context().Err() != nil {
			return cmd.Context().Err()
		}
		cbServer.Close() // release bound port immediately
		<-stdinDone
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Timeout waiting for callback. Switching to manual mode..."); err != nil {
			return err
		}
		return runManualFlow(cmd, verifier, challenge, state, existing, input)
	}
	// Cancel + let the stdin watcher exit before we hand off to exchangeAndSave,
	// so we don't leave a goroutine alive past the login flow. Best-effort.
	cancel()
	select {
	case <-stdinDone:
	case <-time.After(50 * time.Millisecond):
	}

	// 错误回调也必须先校验state，避免伪造error驱动后续恢复动作。
	if subtle.ConstantTimeCompare([]byte(result.State), []byte(state)) != 1 {
		return fmt.Errorf("state mismatch: possible CSRF attack")
	}
	if result.Error != "" {
		return fmt.Errorf("authorization denied: %s", result.Error)
	}
	if result.Code == "" {
		return fmt.Errorf("no authorization code received")
	}

	return exchangeAndSave(cmd, result.Code, dcr.ClientID, redirectURI, verifier, dcr.RedirectURIs, currentDiscoveryURL)
}

// watchEnter 不预读下一行，CRLF的第二字节留给同会话的手动读者识别。
func (input *inputSession) watchEnter(ctx context.Context) bool {
	_, err := input.readLine(ctx, 4096)
	return err == nil
}

// runManualFlow prints the authorize URL and prompts for the code.
// ACCEPTED RISK: state parameter is embedded in the authorize URL but not validated
// on the return path — the user manually copies only the authorization code from the
// cli-callback page. CSRF is mitigated by the user-mediated copy-paste step: an
// attacker cannot silently inject a code into the user's terminal session.
func runManualFlow(cmd *cobra.Command, verifier, challenge, state string, existing *oauth2.TokenData, input *inputSession) error {
	// 固定手动回调与 DCR 注册保持一致，拒绝本地覆盖带来的授权码外送。
	redirectURI := oauth2.ResolveManualRedirectURI()

	currentDiscoveryURL := oauth2.CurrentDiscoveryURL()
	dcr, err := ensureClientID(cmd, existing, redirectURI, currentDiscoveryURL)
	if err != nil {
		return err
	}

	authorizeURL := oauth2.BuildAuthorizeURL(dcr.ClientID, redirectURI, state, challenge, loginOptions(cmd).scopes, loginOptions(cmd).resource)

	out := cmd.OutOrStdout()
	stderr := cmd.ErrOrStderr()
	agent := stderrIsAgent()
	if err := announceAuthorizeURL(out, stderr, authorizeURL, "manual", agent, stdoutSupportsHyperlinks()); err != nil {
		return err
	}
	if _, err := fmt.Fprint(out, "Paste the authorization code here: "); err != nil {
		return err
	}

	code, err := readAuthCode(cmd, input)
	if err != nil {
		return fmt.Errorf("read authorization code: %w", err)
	}
	if code == "" {
		return fmt.Errorf("read authorization code: empty input")
	}

	return exchangeAndSave(cmd, code, dcr.ClientID, redirectURI, verifier, dcr.RedirectURIs, currentDiscoveryURL)
}

// readAuthCode reads the authorization code from stdin without echoing it back
// when running in a terminal (BR-5). In non-TTY environments (CI, pipes, tests)
// it falls back to a line-based reader so the manual flow stays scriptable.
// 同一会话沿用自动阶段的CRLF状态，在恢复回显前完成读操作。
func readAuthCode(cmd *cobra.Command, input *inputSession) (code string, resultErr error) {
	// 终端只关闭回显；同一个有界、可取消的输入入口覆盖TTY和管道。
	if file, ok := input.in.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		state, err := term.MakeRaw(int(file.Fd()))
		if err != nil {
			return "", err
		}
		defer func() { resultErr = errors.Join(resultErr, term.Restore(int(file.Fd()), state)) }()
		defer func() { _, err := fmt.Fprintln(cmd.OutOrStdout()); resultErr = errors.Join(resultErr, err) }()
	}
	line, err := input.readLine(cmd.Context(), 4096)
	return strings.TrimSpace(line), err
}

// dcrResult holds DCR registration result for persisting redirect_uris alongside token.
type dcrResult struct {
	ClientID     string
	RedirectURIs []string
}

// canReuseClient reports whether a cached registration in existing can be
// reused as-is for a login against currentDiscoveryURL. All conditions must
// hold: a non-empty ClientID, an exact DiscoveryURL match (re-registration is
// required when the user points the CLI at a different authorization server),
// and — when requiredURI is given — the cached registration already covers it.
// Kept as a pure function so the decision is unit-testable without triggering
// a real DCR HTTP call.
func canReuseClient(existing *oauth2.TokenData, requiredURI, currentDiscoveryURL string) bool {
	if existing == nil || existing.ClientID == "" {
		return false
	}
	if existing.DiscoveryURL != currentDiscoveryURL {
		return false
	}
	if requiredURI == "" {
		return true
	}
	return containsURI(existing.RegisteredRedirectURIs, requiredURI)
}

// ensureClientID returns a cached client_id or registers a new one with all 6 redirect_uris (BR-6).
// See canReuseClient for the reuse decision; any failure there triggers a fresh
// DCR against the current discovery target.
func ensureClientID(cmd *cobra.Command, existing *oauth2.TokenData, requiredURI, currentDiscoveryURL string) (*dcrResult, error) {
	if result := loginOptions(cmd).registration; result != nil && containsURI(result.RedirectURIs, requiredURI) {
		return result, nil
	}
	if !loginOptions(cmd).force && existing != nil && existing.Resource == loginOptions(cmd).resource && canReuseClient(existing, requiredURI, currentDiscoveryURL) {
		return &dcrResult{
			ClientID:     existing.ClientID,
			RedirectURIs: existing.RegisteredRedirectURIs,
		}, nil
	}
	uris := oauth2.BuildDCRRedirectURIs()
	resp, err := oauth2.Register(cmd.Context(), uris)
	if err != nil {
		return nil, fmt.Errorf("client registration: %w", err)
	}
	registeredURIs := resp.RedirectURIs
	if len(registeredURIs) == 0 {
		registeredURIs = uris // fallback: server didn't echo back URIs
	}
	result := &dcrResult{ClientID: resp.ClientID, RedirectURIs: registeredURIs}
	loginOptions(cmd).registration = result
	return result, nil
}

func containsURI(uris []string, target string) bool {
	for _, u := range uris {
		if u == target {
			return true
		}
	}
	return false
}

// exchangeAndSave exchanges the code for tokens and persists them.
// discoveryURL is recorded alongside the client registration so a later
// login after a discovery_url switch can detect the mismatch and re-register.
func exchangeAndSave(cmd *cobra.Command, code, clientID, redirectURI, verifier string, registeredURIs []string, discoveryURL string) error {
	options := loginOptions(cmd)
	operation, err := oauth2.BeginAuthOperation(cmd.Context(), "login", options.before)
	if err != nil {
		return err
	}
	failed := func(cause error, unknown bool) error {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(cmd.Context()), 3*time.Second)
		defer cancel()
		return errors.Join(cause, operation.Fail(ctx, unknown))
	}
	if err := operation.Dispatch(cmd.Context(), nil); err != nil {
		return failed(err, false)
	}
	tokenResp, err := oauth2.ExchangeCode(cmd.Context(), code, clientID, redirectURI, verifier, options.resource)
	if err != nil {
		var serverErr *oauth2.OAuth2ServerError
		known := errors.As(err, &serverErr) && serverErr.StatusCode == 400 && strings.HasPrefix(serverErr.ErrorCode, "invalid_")
		return failed(fmt.Errorf("token exchange: %w", err), !known)
	}
	td := &oauth2.TokenData{ClientID: clientID, DiscoveryURL: discoveryURL, Resource: options.resource, AccessToken: tokenResp.AccessToken, RefreshToken: tokenResp.RefreshToken, TokenType: tokenResp.TokenType, ExpiresAt: time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second), Scopes: tokenResp.Scope, RegisteredRedirectURIs: registeredURIs}
	if err := operation.SaveResult(cmd.Context(), td); err != nil {
		return fmt.Errorf("new authorization result could not be persisted; operation retained: %w", err)
	}
	var reportErr error
	// 新client迁移只定向清理本机旧连接；同client重授不能撤销刚激活的新grant。
	if old := options.before; old != nil && old.ClientID != "" && old.ClientID != clientID && (old.AccessToken != "" || old.RefreshToken != "") {
		if oauth2.NormalizeDiscoveryURL(old.DiscoveryURL) == oauth2.CurrentDiscoveryURL() {
			token := old.RefreshToken
			if token == "" {
				token = old.AccessToken
			}
			if err := oauth2.RevokeToken(cmd.Context(), token, old.ClientID); err != nil {
				if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "New login succeeded; old connection revocation is unconfirmed. Manage it on the Connections page."); err != nil {
					reportErr = errors.Join(reportErr, err)
				}
			}
		} else {
			if _, err := fmt.Fprintln(cmd.ErrOrStderr(), "New login succeeded; old issuer differs. Disconnect the old connection on its Connections page."); err != nil {
				reportErr = errors.Join(reportErr, err)
			}
		}
	}
	if err := operation.Finish(cmd.Context()); err != nil {
		return fmt.Errorf("new credentials saved; operation cleanup incomplete: %w", err)
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Login successful! (scopes: %s)\n", td.Scopes)
	return errors.Join(reportErr, err)
}

// stderrIsAgent reports whether stderr is not a TTY — a good proxy for
// "we're being invoked by an agent / CI / piped wrapper" and should emit
// machine-parseable markers alongside the human-readable prose.
func stderrIsAgent() bool {
	return !isatty.IsTerminal(os.Stderr.Fd()) && !isatty.IsCygwinTerminal(os.Stderr.Fd())
}

// stdoutSupportsHyperlinks reports whether stdout is a TTY that we're
// willing to hand OSC 8 hyperlink escapes to. Most modern terminals
// silently ignore unknown OSC sequences, but we still gate on:
//   - stdout is a TTY (don't pollute pipes/redirects with escape bytes)
//   - TERM is not "dumb" or empty
//   - NO_HYPERLINKS is unset (NO_COLOR-style opt-out)
func stdoutSupportsHyperlinks() bool {
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return false
	}
	if os.Getenv("NO_HYPERLINKS") != "" {
		return false
	}
	t := strings.ToLower(os.Getenv("TERM"))
	if t == "" || t == "dumb" {
		return false
	}
	return true
}

// formatClickableURL optionally wraps url in an OSC 8 hyperlink escape so
// that compatible terminals render it as a clickable link. When hyperlinks
// is false, returns the url unchanged. The visible text is the URL itself
// so terminals that don't understand OSC 8 still show something copy-pastable.
//
// OSC 8 format:
//
//	ESC ] 8 ; ; <URL> BEL <visible-text> ESC ] 8 ; ; BEL
func formatClickableURL(url string, hyperlinks bool) string {
	if !hyperlinks {
		return url
	}
	return "\x1b]8;;" + url + "\x07" + url + "\x1b]8;;\x07"
}

// announceAuthorizeURL prints the authorization URL prominently on stdout
// (so TTY users can copy it to another device or Cmd/Ctrl-click to open)
// and, when agent-mode is detected, also emits two grep-friendly lines on
// stderr so wrapping tools can surface the URL and a mode-appropriate hint
// to their end user.
//
//	GATE_CLI_AUTH_URL=<url>
//	GATE_CLI_AUTH_HINT=<mode-specific guidance>
//
// mode must be "auto" or "manual"; any other value is treated as "auto".
// When hyperlinks is true, the stdout URL is wrapped in an OSC 8 escape so
// it renders as a clickable hyperlink in supporting terminals. The stderr
// marker always carries the raw URL — agents need a regex-friendly form.
func announceAuthorizeURL(out io.Writer, stderr io.Writer, url, mode string, agentMode bool, hyperlinks bool) error {
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "Authorization URL (copy into any browser to authorize):"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "  "+formatClickableURL(url, hyperlinks)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}

	if !agentMode {
		return nil
	}
	if _, err := fmt.Fprintf(stderr, "GATE_CLI_AUTH_URL=%s\n", url); err != nil {
		return err
	}
	var hint string
	switch mode {
	case "manual":
		hint = "Open the URL above in a browser, then paste the authorization code back when prompted."
	default:
		hint = "Auto mode: a browser is being opened; if no browser is available on this device, press Enter to switch to manual mode or re-run with --manual."
	}
	if _, err := fmt.Fprintf(stderr, "GATE_CLI_AUTH_HINT=%s\n", hint); err != nil {
		return err
	}
	return nil
}

func openBrowser(parent context.Context, url string) error {
	// 系统启动器也受命令取消及固定短预算约束；Run负责回收子进程。
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()
	// Reject non-HTTP(S) URLs to prevent command injection via crafted schemes
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return fmt.Errorf("refusing to open non-HTTP URL")
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.CommandContext(ctx, "open", url).Run()
	case "linux":
		return exec.CommandContext(ctx, "xdg-open", url).Run()
	case "windows":
		// Use rundll32 to avoid shell metacharacter injection via cmd.exe
		return exec.CommandContext(ctx, "rundll32", "url.dll,FileProtocolHandler", url).Run()
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}
}

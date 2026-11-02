package auth

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/oauth2"
	"github.com/gate/gate-cli/internal/output"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current OAuth2 status (run this FIRST before login)",
	Long: `Print local OAuth2 token status and the selected issuer.
Normal builds use the fixed production issuer; approved test builds may select
a test issuer. API-origin overrides do not select an OAuth issuer.

Output states:
  authenticated          Locally unexpired; remote grant may have been revoked.
  expired (refreshable)  Will auto-refresh on the next API call.
  expired                Re-login required (refresh token absent/expired).
  Not logged in          No token stored; run 'gate-cli cex auth login'.

AGENT GUIDANCE:
  This is the recommended entry point when deciding whether authentication
  is needed. Run 'status' first; only invoke 'login' if the output is
  "Not logged in" or "expired" (non-refreshable). This avoids redundant
  browser launches and unnecessary prompts to the user.

Related:
  gate-cli cex auth login   — authorize (auto browser, --manual fallback)
  gate-cli cex auth logout  — clear stored tokens
`,
	RunE: runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	token, err := oauth2.LoadToken()
	if err != nil {
		return fmt.Errorf("load token: %w", err)
	}
	// Report only this issuer's credential projection. Never auto-select a token
	// from another environment or expose secrets to explain a logged-out state.
	state := struct {
		Status        string `json:"status"`
		Issuer        string `json:"issuer"`
		Resource      string `json:"resource,omitempty"`
		Scopes        string `json:"scopes,omitempty"`
		ExpiresAt     string `json:"expires_at,omitempty"`
		RemoteChecked bool   `json:"remote_checked"`
	}{Status: "not_logged_in", Issuer: oauth2.ShortDiscoveryURL(oauth2.CurrentDiscoveryURL())}
	if token != nil && token.AccessToken != "" {
		state.Status, state.Resource, state.Scopes = tokenStatus(token), token.Resource, token.Scopes
		state.ExpiresAt = token.ExpiresAt.Format(time.RFC3339)
	}
	format, _ := cmd.Root().PersistentFlags().GetString("format")
	if format == "json" {
		return output.New(cmd.OutOrStdout(), output.FormatJSON).Print(state)
	}
	if format == "table" {
		return output.UnsupportedTableFormatError()
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Issuer:    %s\n", state.Issuer); err != nil {
		return err
	}

	if token == nil || token.AccessToken == "" {
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Not logged in."); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), "Run 'gate-cli cex auth login' to authenticate."); err != nil {
			return err
		}
		return nil
	}

	out := cmd.OutOrStdout()
	if _, err := fmt.Fprintf(out, "Resource:  %s\n", state.Resource); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Status:    %s\n", tokenStatus(token)); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "Note: local status only; remote authorization is checked on every protected request."); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Scopes:    %s\n", token.Scopes); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Expires:   %s\n", token.ExpiresAt.Format(time.RFC3339)); err != nil {
		return err
	}
	if token.IsValid() {
		remaining := time.Until(token.ExpiresAt).Truncate(time.Second)
		if _, err := fmt.Fprintf(out, "Remaining: %s\n", remaining); err != nil {
			return err
		}
	}
	return nil
}

func tokenStatus(t *oauth2.TokenData) string {
	if t.IsValid() {
		return "authenticated"
	}
	if t.HasRefreshToken() {
		return "expired (refreshable)"
	}
	return "expired"
}

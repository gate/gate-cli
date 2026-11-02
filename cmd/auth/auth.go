package auth

import "github.com/spf13/cobra"

// Cmd is the auth command group (login / logout / status / refresh).
var Cmd = &cobra.Command{
	Use: "auth",
	// 分组自身展示help，未知子命令必须非零，避免下线入口被误判成功。
	Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	Short: "Authentication management (OAuth2 + API key)",
	Long: `Manage OAuth2 login, status, refresh and logout for Gate API requests.

Run 'gate-cli cex auth login' to authorize in a browser, or add --manual for
headless use. 'status' reports local credential state; each private API request
is still checked by the gateway. API key profiles remain supported.
`,
}

func init() {
	Cmd.AddCommand(loginCmd)
	Cmd.AddCommand(logoutCmd)
	Cmd.AddCommand(statusCmd)
	Cmd.AddCommand(refreshCmd)
}

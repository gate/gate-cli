package auth

import (
	"fmt"

	"github.com/gate/gate-cli/internal/cmdutil"
	"github.com/gate/gate-cli/internal/oauth2"
	"github.com/spf13/cobra"
)

// refreshCmd 强制轮换一次RT；不重试，不自动发起新的浏览器授权。
var refreshCmd = &cobra.Command{Use: "refresh", Short: "Refresh the bound OAuth2 token once", RunE: func(cmd *cobra.Command, args []string) error {
	cfg, err := cmdutil.GetConfig(cmd)
	if err != nil {
		return err
	}
	resource, err := oauth2.APIResource(cfg.BaseURL)
	if err != nil {
		return err
	}
	if err := oauth2.InitEndpointsContext(cmd.Context()); err != nil {
		return err
	}
	token, err := oauth2.ForceRefresh(cmd.Context(), resource)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Refresh: success (scopes: %s)\n", token.Scopes); err != nil {
		return err
	}
	return nil
}}

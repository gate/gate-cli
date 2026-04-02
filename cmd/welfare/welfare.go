package welfare

import (
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

// Cmd is the root command for the welfare module.
var Cmd = &cobra.Command{
	Use:   "welfare",
	Short: "Welfare commands",
}

func init() {
	identityCmd := &cobra.Command{
		Use:   "identity",
		Short: "Get user identity for new user rewards",
		RunE:  runIdentity,
	}

	beginnerTasksCmd := &cobra.Command{
		Use:   "beginner-tasks",
		Short: "Get beginner task list",
		RunE:  runBeginnerTasks,
	}

	Cmd.AddCommand(identityCmd, beginnerTasksCmd)
}

func runIdentity(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.WelfareAPI.GetUserIdentity(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rewards/getUserIdentity", ""))
		return nil
	}
	return p.Print(result)
}

func runBeginnerTasks(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.WelfareAPI.GetBeginnerTaskList(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rewards/getBeginnerTaskList", ""))
		return nil
	}
	return p.Print(result)
}

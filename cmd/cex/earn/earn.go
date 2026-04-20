package earn

import "github.com/spf13/cobra"

// Cmd is the root command for the earn module.
var Cmd = &cobra.Command{
	Use:   "earn",
	Short: "Earn & staking commands",
}

func init() {
	Cmd.AddCommand(dualCmd, stakingCmd, fixedCmd, autoInvestCmd, uniCmd)
}

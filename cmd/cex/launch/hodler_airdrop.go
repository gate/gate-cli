package launch

import (
	"encoding/json"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// registerHodlerAirdropCommands attaches the Hodler Airdrop V4 sub-tree under
// `cex launch hodler ...`. These mirror the CandyDrop V4 subtree and close the
// remaining LaunchAPI gap surfaced when cross-referencing gateapi-go v7.2.71
// against the CLI.
func registerHodlerAirdropCommands(parent *cobra.Command) {
	hodlerCmd := &cobra.Command{
		Use:   "hodler",
		Short: "HODLer Airdrop V4 commands",
	}

	projectsCmd := &cobra.Command{
		Use:   "projects",
		Short: "List HODLer Airdrop activities (public; logged-in users get extra participation info)",
		RunE:  runHodlerProjects,
	}
	projectsCmd.Flags().String("status", "", "Filter by activity status")
	projectsCmd.Flags().String("keyword", "", "Filter by currency/project name (fuzzy)")
	projectsCmd.Flags().Int32("join", 0, "Filter by participation status")
	projectsCmd.Flags().Int32("page", 0, "Page number")
	projectsCmd.Flags().Int32("size", 0, "Page size")

	orderCmd := &cobra.Command{
		Use:   "order",
		Short: "Participate in a HODLer Airdrop activity (auth required)",
		RunE:  runHodlerOrder,
	}
	orderCmd.Flags().Int32("hodler-id", 0, "Activity ID (required)")
	orderCmd.MarkFlagRequired("hodler-id")

	orderRecordsCmd := &cobra.Command{
		Use:   "order-records",
		Short: "Query user's HODLer Airdrop participation records (auth required)",
		RunE:  runHodlerOrderRecords,
	}
	orderRecordsCmd.Flags().String("keyword", "", "Filter by currency/project name")
	orderRecordsCmd.Flags().Int32("start-timest", 0, "Start timestamp (seconds)")
	orderRecordsCmd.Flags().Int32("end-timest", 0, "End timestamp (seconds)")
	orderRecordsCmd.Flags().Int32("page", 0, "Page number")
	orderRecordsCmd.Flags().Int32("size", 0, "Page size")

	airdropRecordsCmd := &cobra.Command{
		Use:   "airdrop-records",
		Short: "Query HODLer Airdrop distribution records received by the user (auth required)",
		RunE:  runHodlerAirdropRecords,
	}
	airdropRecordsCmd.Flags().String("keyword", "", "Filter by currency/project name")
	airdropRecordsCmd.Flags().Int32("start-timest", 0, "Start timestamp (seconds)")
	airdropRecordsCmd.Flags().Int32("end-timest", 0, "End timestamp (seconds)")
	airdropRecordsCmd.Flags().Int32("page", 0, "Page number")
	airdropRecordsCmd.Flags().Int32("size", 0, "Page size")

	hodlerCmd.AddCommand(projectsCmd, orderCmd, orderRecordsCmd, airdropRecordsCmd)
	parent.AddCommand(hodlerCmd)
}

func runHodlerProjects(cmd *cobra.Command, args []string) error {
	status, _ := cmd.Flags().GetString("status")
	keyword, _ := cmd.Flags().GetString("keyword")
	join, _ := cmd.Flags().GetInt32("join")
	page, _ := cmd.Flags().GetInt32("page")
	size, _ := cmd.Flags().GetInt32("size")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.GetHodlerAirdropProjectListOpts{}
	if status != "" {
		opts.Status = optional.NewString(status)
	}
	if keyword != "" {
		opts.Keyword = optional.NewString(keyword)
	}
	if join != 0 {
		opts.Join = optional.NewInt32(join)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if size != 0 {
		opts.Size = optional.NewInt32(size)
	}

	result, httpResp, err := c.LaunchAPI.GetHodlerAirdropProjectList(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/hodler_airdrop/v4/projects", ""))
		return nil
	}
	return p.Print(result)
}

func runHodlerOrder(cmd *cobra.Command, args []string) error {
	hodlerID, _ := cmd.Flags().GetInt32("hodler-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.HodlerAirdropV4OrderRequest{HodlerId: hodlerID}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.LaunchAPI.HodlerAirdropOrder(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/hodler_airdrop/v4/order", string(body)))
		return nil
	}
	return p.Print(result)
}

func runHodlerOrderRecords(cmd *cobra.Command, args []string) error {
	keyword, _ := cmd.Flags().GetString("keyword")
	startTimest, _ := cmd.Flags().GetInt32("start-timest")
	endTimest, _ := cmd.Flags().GetInt32("end-timest")
	page, _ := cmd.Flags().GetInt32("page")
	size, _ := cmd.Flags().GetInt32("size")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetHodlerAirdropUserOrderRecordsOpts{}
	if keyword != "" {
		opts.Keyword = optional.NewString(keyword)
	}
	if startTimest != 0 {
		opts.StartTimest = optional.NewInt32(startTimest)
	}
	if endTimest != 0 {
		opts.EndTimest = optional.NewInt32(endTimest)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if size != 0 {
		opts.Size = optional.NewInt32(size)
	}

	result, httpResp, err := c.LaunchAPI.GetHodlerAirdropUserOrderRecords(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/hodler_airdrop/v4/user/order_records", ""))
		return nil
	}
	return p.Print(result)
}

func runHodlerAirdropRecords(cmd *cobra.Command, args []string) error {
	keyword, _ := cmd.Flags().GetString("keyword")
	startTimest, _ := cmd.Flags().GetInt32("start-timest")
	endTimest, _ := cmd.Flags().GetInt32("end-timest")
	page, _ := cmd.Flags().GetInt32("page")
	size, _ := cmd.Flags().GetInt32("size")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.GetHodlerAirdropUserAirdropRecordsOpts{}
	if keyword != "" {
		opts.Keyword = optional.NewString(keyword)
	}
	if startTimest != 0 {
		opts.StartTimest = optional.NewInt32(startTimest)
	}
	if endTimest != 0 {
		opts.EndTimest = optional.NewInt32(endTimest)
	}
	if page != 0 {
		opts.Page = optional.NewInt32(page)
	}
	if size != 0 {
		opts.Size = optional.NewInt32(size)
	}

	result, httpResp, err := c.LaunchAPI.GetHodlerAirdropUserAirdropRecords(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/hodler_airdrop/v4/user/airdrop_records", ""))
		return nil
	}
	return p.Print(result)
}

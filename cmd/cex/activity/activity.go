package activity

import (
	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

// Cmd is the root command for the activity module.
var Cmd = &cobra.Command{
	Use:   "activity",
	Short: "Activity & promotion commands",
}

func init() {
	getEntryCmd := &cobra.Command{
		Use:   "get-entry",
		Short: "Query my activity entry information",
		RunE:  runGetEntry,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List recommended activities (public)",
		RunE:  runList,
	}
	listCmd.Flags().String("recommend-type", "", "Recommendation type: hot, type, scenario")
	listCmd.Flags().String("type-ids", "", "Activity type IDs, comma-separated")
	listCmd.Flags().String("keywords", "", "Activity name keywords")
	listCmd.Flags().Int32("page", 0, "Page number, starting from 1")
	listCmd.Flags().Int32("page-size", 0, "Items per page")
	listCmd.Flags().String("sort-by", "", "Sort order, e.g., hot")

	typesCmd := &cobra.Command{
		Use:   "types",
		Short: "List all activity types (public)",
		RunE:  runTypes,
	}

	Cmd.AddCommand(getEntryCmd, listCmd, typesCmd)
}

func runGetEntry(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.ActivityAPI.GetMyActivityEntry(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rewards/activity/my-activity-entry", ""))
		return nil
	}
	return p.Print(result)
}

func runList(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	opts := &gateapi.ListActivitiesOpts{}
	if v, _ := cmd.Flags().GetString("recommend-type"); v != "" {
		opts.RecommendType = optional.NewString(v)
	}
	if v, _ := cmd.Flags().GetString("type-ids"); v != "" {
		opts.TypeIds = optional.NewString(v)
	}
	if v, _ := cmd.Flags().GetString("keywords"); v != "" {
		opts.Keywords = optional.NewString(v)
	}
	if v, _ := cmd.Flags().GetInt32("page"); v > 0 {
		opts.Page = optional.NewInt32(v)
	}
	if v, _ := cmd.Flags().GetInt32("page-size"); v > 0 {
		opts.PageSize = optional.NewInt32(v)
	}
	if v, _ := cmd.Flags().GetString("sort-by"); v != "" {
		opts.SortBy = optional.NewString(v)
	}

	result, httpResp, err := c.ActivityAPI.ListActivities(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rewards/activity/activity-list", ""))
		return nil
	}
	return p.Print(result)
}

func runTypes(cmd *cobra.Command, args []string) error {
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}

	result, httpResp, err := c.ActivityAPI.ListActivityTypes(c.Context())
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/rewards/activity/activity-type", ""))
		return nil
	}
	return p.Print(result)
}

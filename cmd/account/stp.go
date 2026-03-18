package account

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	gateapi "github.com/gate/gateapi-go/v7"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
)

var stpCmd = &cobra.Command{
	Use:   "stp",
	Short: "Self-Trade Prevention group commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List STP groups",
		RunE:  runSTPList,
	}
	listCmd.Flags().String("name", "", "Filter by group name")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an STP group",
		RunE:  runSTPCreate,
	}
	createCmd.Flags().String("name", "", "Group name (required)")
	createCmd.MarkFlagRequired("name")

	usersCmd := &cobra.Command{
		Use:   "users",
		Short: "List users in an STP group",
		RunE:  runSTPUsers,
	}
	usersCmd.Flags().Int64("id", 0, "STP group ID (required)")
	usersCmd.MarkFlagRequired("id")

	addUsersCmd := &cobra.Command{
		Use:   "add-users",
		Short: "Add users to an STP group",
		RunE:  runSTPAddUsers,
	}
	addUsersCmd.Flags().Int64("id", 0, "STP group ID (required)")
	addUsersCmd.Flags().String("user-ids", "", "Comma-separated user IDs to add (required)")
	addUsersCmd.MarkFlagRequired("id")
	addUsersCmd.MarkFlagRequired("user-ids")

	removeUserCmd := &cobra.Command{
		Use:   "remove-user",
		Short: "Remove a user from an STP group",
		RunE:  runSTPRemoveUser,
	}
	removeUserCmd.Flags().Int64("id", 0, "STP group ID (required)")
	removeUserCmd.Flags().Int64("user-id", 0, "User ID to remove (required)")
	removeUserCmd.MarkFlagRequired("id")
	removeUserCmd.MarkFlagRequired("user-id")

	stpCmd.AddCommand(listCmd, createCmd, usersCmd, addUsersCmd, removeUserCmd)
	Cmd.AddCommand(stpCmd)
}

func runSTPList(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	opts := &gateapi.ListSTPGroupsOpts{}
	if name != "" {
		opts.Name = optional.NewString(name)
	}

	result, httpResp, err := c.AccountAPI.ListSTPGroups(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", "/api/v4/account/stp_groups", ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, g := range result {
		rows[i] = []string{fmt.Sprintf("%d", g.Id), g.Name, fmt.Sprintf("%d", g.CreatorId), fmt.Sprintf("%d", g.CreateTime)}
	}
	return p.Table([]string{"ID", "Name", "Creator ID", "Created At"}, rows)
}

func runSTPCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	req := gateapi.StpGroup{Name: name}
	body, _ := json.Marshal(req)
	result, httpResp, err := c.AccountAPI.CreateSTPGroup(c.Context(), req)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/account/stp_groups", string(body)))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	return p.Table(
		[]string{"ID", "Name", "Creator ID", "Created At"},
		[][]string{{fmt.Sprintf("%d", result.Id), result.Name, fmt.Sprintf("%d", result.CreatorId), fmt.Sprintf("%d", result.CreateTime)}},
	)
}

func runSTPUsers(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt64("id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.AccountAPI.ListSTPGroupsUsers(c.Context(), id)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "GET", fmt.Sprintf("/api/v4/account/stp_groups/%d/users", id), ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, u := range result {
		rows[i] = []string{fmt.Sprintf("%d", u.UserId), fmt.Sprintf("%d", u.StpId), fmt.Sprintf("%d", u.CreateTime)}
	}
	return p.Table([]string{"User ID", "STP ID", "Added At"}, rows)
}

func runSTPAddUsers(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt64("id")
	userIDsStr, _ := cmd.Flags().GetString("user-ids")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	parts := strings.Split(userIDsStr, ",")
	userIDs := make([]int64, 0, len(parts))
	for _, part := range parts {
		uid, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil {
			return fmt.Errorf("invalid user ID %q: %w", part, err)
		}
		userIDs = append(userIDs, uid)
	}

	body, _ := json.Marshal(userIDs)
	result, httpResp, err := c.AccountAPI.AddSTPGroupUsers(c.Context(), id, userIDs)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", fmt.Sprintf("/api/v4/account/stp_groups/%d/users", id), string(body)))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, u := range result {
		rows[i] = []string{fmt.Sprintf("%d", u.UserId), fmt.Sprintf("%d", u.StpId), fmt.Sprintf("%d", u.CreateTime)}
	}
	return p.Table([]string{"User ID", "STP ID", "Added At"}, rows)
}

func runSTPRemoveUser(cmd *cobra.Command, args []string) error {
	id, _ := cmd.Flags().GetInt64("id")
	userID, _ := cmd.Flags().GetInt64("user-id")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	result, httpResp, err := c.AccountAPI.DeleteSTPGroupUsers(c.Context(), id, userID)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "DELETE", fmt.Sprintf("/api/v4/account/stp_groups/%d/users", id), ""))
		return nil
	}
	if p.IsJSON() {
		return p.Print(result)
	}
	rows := make([][]string, len(result))
	for i, u := range result {
		rows[i] = []string{fmt.Sprintf("%d", u.UserId), fmt.Sprintf("%d", u.StpId), fmt.Sprintf("%d", u.CreateTime)}
	}
	return p.Table([]string{"User ID", "STP ID", "Added At"}, rows)
}

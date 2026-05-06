package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/antihax/optional"
	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var adsCmd = &cobra.Command{
	Use:   "ads",
	Short: "P2P advertisement commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List market ads",
		RunE:  runAdsList,
	}
	listCmd.Flags().String("json", "", "JSON body for ads list request (required)")
	listCmd.MarkFlagRequired("json")

	myListCmd := &cobra.Command{
		Use:   "my-list",
		Short: "List my ads",
		RunE:  runAdsMyList,
	}
	myListCmd.Flags().String("json", "", "JSON body for my ads list request (optional)")

	detailCmd := &cobra.Command{
		Use:   "detail",
		Short: "Get ad detail",
		RunE:  runAdsDetail,
	}
	detailCmd.Flags().String("adv-no", "", "Ad number (required)")
	detailCmd.MarkFlagRequired("adv-no")

	updateStatusCmd := &cobra.Command{
		Use:   "update-status",
		Short: "Update ad status",
		RunE:  runAdsUpdateStatus,
	}
	updateStatusCmd.Flags().Int32("adv-no", 0, "Advertisement ID (required)")
	updateStatusCmd.MarkFlagRequired("adv-no")
	updateStatusCmd.Flags().Int32("adv-status", 0, "Ad status: 1=listed, 3=delisted, 4=closed (required)")
	updateStatusCmd.MarkFlagRequired("adv-status")

	adsCmd.AddCommand(listCmd, myListCmd, detailCmd, updateStatusCmd)
	Cmd.AddCommand(adsCmd)
}

func runAdsList(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.AdsListRequest
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantBooksAdsList(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/books/ads_list", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runAdsMyList(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var opts *gateapi.P2pMerchantBooksMyAdsListOpts
	if jsonStr != "" {
		var body gateapi.MyAdsListRequest
		if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
			return fmt.Errorf("invalid --json: %w", err)
		}
		opts = &gateapi.P2pMerchantBooksMyAdsListOpts{
			MyAdsListRequest: optional.NewInterface(body),
		}
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantBooksMyAdsList(c.Context(), opts)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/books/my_ads_list", jsonStr))
		return nil
	}
	return p.Print(result)
}

func runAdsDetail(cmd *cobra.Command, args []string) error {
	advNo, _ := cmd.Flags().GetString("adv-no")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.AdsDetailRequest{
		AdvNo: advNo,
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantBooksAdsDetail(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/books/ads_detail", ""))
		return nil
	}
	return p.Print(result)
}

func runAdsUpdateStatus(cmd *cobra.Command, args []string) error {
	advNo, _ := cmd.Flags().GetInt32("adv-no")
	advStatus, _ := cmd.Flags().GetInt32("adv-status")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.AdsUpdateStatus{
		AdvNo:     advNo,
		AdvStatus: advStatus,
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantBooksAdsUpdateStatus(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/books/ads_update_status", ""))
		return nil
	}
	return p.Print(result)
}

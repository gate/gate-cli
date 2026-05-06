package p2p

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	gateapi "github.com/gate/gateapi-go/v7"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "P2P chat commands",
}

func init() {
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List chat messages for an order",
		Long: `List chat messages for an order.

v7.2.78 contract: --txid is required by the CLI but the value 0 is treated by
the server as 'omit' — pass --txid 0 to fetch the latest order with chat for
the current user. Pass --lastreceived / --firstreceived to paginate forward
or backward from a known timestamp.`,
		RunE: runChatList,
	}
	listCmd.Flags().Int32("txid", 0, "Order ID (required; pass 0 to return the latest order with chat)")
	listCmd.MarkFlagRequired("txid")
	listCmd.Flags().Int32("lastreceived", 0, "Timestamp of last received message; backward incremental fetch")
	listCmd.Flags().Int32("firstreceived", 0, "Timestamp of first received message; forward paging")

	sendCmd := &cobra.Command{
		Use:   "send",
		Short: "Send a chat message",
		RunE:  runChatSend,
	}
	sendCmd.Flags().Int32("txid", 0, "Order ID (required)")
	sendCmd.MarkFlagRequired("txid")
	sendCmd.Flags().String("message", "", "Message content (required)")
	sendCmd.MarkFlagRequired("message")
	sendCmd.Flags().Int32("type", 0, "Message type: 0=Text, 1=File")

	uploadCmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload a chat file",
		RunE:  runChatUpload,
	}
	uploadCmd.Flags().String("json", "", "JSON body for upload request (required)")
	uploadCmd.MarkFlagRequired("json")

	chatCmd.AddCommand(listCmd, sendCmd, uploadCmd)
	Cmd.AddCommand(chatCmd)
}

func runChatList(cmd *cobra.Command, args []string) error {
	txid, _ := cmd.Flags().GetInt32("txid")
	lastreceived, _ := cmd.Flags().GetInt32("lastreceived")
	firstreceived, _ := cmd.Flags().GetInt32("firstreceived")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.GetChatsListRequest{
		Txid: txid,
	}
	if lastreceived != 0 {
		body.Lastreceived = lastreceived
	}
	if firstreceived != 0 {
		body.Firstreceived = firstreceived
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantChatGetChatsList(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/chat/get_chats_list", ""))
		return nil
	}
	return p.Print(result)
}

func runChatSend(cmd *cobra.Command, args []string) error {
	txid, _ := cmd.Flags().GetInt32("txid")
	message, _ := cmd.Flags().GetString("message")
	msgType, _ := cmd.Flags().GetInt32("type")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	body := gateapi.SendChatMessageRequest{
		Txid:    txid,
		Message: message,
		Type:    msgType,
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantChatSendChatMessage(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/chat/send_chat_message", ""))
		return nil
	}
	return p.Print(result)
}

func runChatUpload(cmd *cobra.Command, args []string) error {
	jsonStr, _ := cmd.Flags().GetString("json")
	p := cmdutil.GetPrinter(cmd)
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return err
	}
	if err := c.RequireAuth(); err != nil {
		return err
	}

	var body gateapi.UploadChatFile
	if err := json.Unmarshal([]byte(jsonStr), &body); err != nil {
		return fmt.Errorf("invalid --json: %w", err)
	}

	result, httpResp, err := c.P2pAPI.P2pMerchantChatUploadChatFile(c.Context(), body)
	if err != nil {
		p.PrintError(client.ParseGateError(err, httpResp, "POST", "/api/v4/p2p/merchant/chat/upload_chat_file", jsonStr))
		return nil
	}
	return p.Print(result)
}

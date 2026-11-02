// Package stock provides the public SDK stock commands; identity and authorization stay in the shared client.
package stock

import (
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/antihax/optional"
	"github.com/gate/gate-cli/internal/client"
	"github.com/gate/gate-cli/internal/cmdutil"
	"github.com/gate/gate-cli/internal/output"
	"github.com/spf13/cobra"
)

// Cmd 注册16个股票叶子，所有HTTP均通过既有client/公共SDK集成边界。
var Cmd = &cobra.Command{Use: "stock",
	// 分组自身展示help，未知子命令必须非零，避免下线入口被误判成功。
	Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() }, Short: "Stock market, orders, positions and account commands"}

func sdkClient(cmd *cobra.Command, private bool) (*client.Client, error) {
	c, err := cmdutil.GetClient(cmd)
	if err != nil {
		return nil, err
	}
	if private {
		if err := c.RequireAuth(); err != nil {
			return nil, err
		}
	}
	return c, nil
}
func invalid(message string) error { return output.InvalidArgsError(message) }

var decimalText = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

// positiveDecimal 只校验字符串，不转float、不改精度；128字节是本地输入保护上限，业务精度由后端校验。
func positiveDecimal(value string) error {
	if len(value) > 128 || !decimalText.MatchString(value) || strings.Trim(value, "0.") == "" {
		return invalid("amount/volume/price must be a positive decimal string")
	}
	return nil
}
func text(value, name string, required bool) error {
	if !required && value == "" {
		return nil
	}
	if value == "" || len(value) > 4096 || strings.TrimSpace(value) != value || strings.ContainsAny(value, "\r\n\t") {
		return invalid(name + " is empty, too long or malformed")
	}
	return nil
}
func symbol(value string, required bool) error {
	if err := text(value, "symbol", required); err != nil {
		return err
	}
	if strings.ContainsAny(value, "/\\%?#") {
		return invalid("symbol must be one path segment")
	}
	return nil
}
func flagString(cmd *cobra.Command, key string) string {
	value, _ := cmd.Flags().GetString(key)
	return value
}
func flagInt32(cmd *cobra.Command, key string) int32 {
	value, _ := cmd.Flags().GetInt32(key)
	return value
}
func optionalString(cmd *cobra.Command, key string) optional.String {
	if cmd.Flags().Changed(key) {
		return optional.NewString(flagString(cmd, key))
	}
	return optional.String{}
}
func addPagination(cmd *cobra.Command) {
	cmd.Flags().Int32("page", 1, "Page number")
	cmd.Flags().Int32("page-size", 10, "Page size (1-500)")
}
func pagination(cmd *cobra.Command) error {
	if flagInt32(cmd, "page") < 1 || flagInt32(cmd, "page-size") < 1 || flagInt32(cmd, "page-size") > 500 {
		return invalid("page must be positive; page-size must be 1-500")
	}
	return nil
}
func addPnL(cmd *cobra.Command) {
	cmd.Flags().Int32("pnl-calc-type", 0, "PnL cost: 1=average, 2=diluted (omitted by default)")
	cmd.Flags().Int32("pnl-calc-price", 0, "PnL price: 1=intraday, 2=extended-hours (omitted by default)")
}
func pnl(cmd *cobra.Command) (optional.Int32, optional.Int32, error) {
	var values [2]optional.Int32
	for i, key := range []string{"pnl-calc-type", "pnl-calc-price"} {
		if cmd.Flags().Changed(key) {
			value := flagInt32(cmd, key)
			if value != 1 && value != 2 {
				return values[0], values[1], invalid(key + " must be 1 or 2")
			}
			values[i] = optional.NewInt32(value)
		}
	}
	return values[0], values[1], nil
}
func exchange(cmd *cobra.Command) error {
	value := flagString(cmd, "exchange")
	if value != "" && value != "us" && value != "hk" && value != "kr" && value != "jp" {
		return invalid("exchange must be us, hk, kr or jp")
	}
	return nil
}
func positiveID(cmd *cobra.Command) (int64, error) {
	id, err := cmd.Flags().GetInt64("id")
	if err != nil || id <= 0 {
		return 0, invalid("id must be a positive int64")
	}
	return id, nil
}
func validateOrderIDs(value string) error {
	if value == "" {
		return nil
	}
	ids := strings.Split(value, ",")
	if len(ids) > 20 {
		return invalid("order-ids supports at most 20 IDs")
	}
	for _, id := range ids {
		if v, err := strconv.ParseInt(id, 10, 64); err != nil || v <= 0 {
			return invalid("order-ids must contain positive int64 IDs")
		}
	}
	return nil
}

// finish 保留SDK原始JSON；table使用RawMessage避免int64和十进制经过float64。
func finish(cmd *cobra.Command, data interface{}, response *http.Response, callErr error, method, path string) error {
	if callErr != nil {
		return client.CommandError(callErr, response, method, path, "")
	}
	printer := cmdutil.GetPrinter(cmd)
	if !printer.IsTable() {
		return printer.Print(data)
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(envelope["data"], &fields); err != nil {
		return printer.Print(data)
	}
	if list, exists := fields["list"]; exists {
		var records []map[string]json.RawMessage
		if err := json.Unmarshal(list, &records); err == nil {
			keys := map[string]bool{}
			for _, record := range records {
				for key := range record {
					keys[key] = true
				}
			}
			columns := make([]string, 0, len(keys))
			for key := range keys {
				columns = append(columns, key)
			}
			sort.Strings(columns)
			if len(columns) == 0 {
				columns = []string{"Result"}
			}
			rows := make([][]string, 0, len(records))
			for _, record := range records {
				row := make([]string, len(columns))
				for i, key := range columns {
					row[i] = cell(record[key])
				}
				rows = append(rows, row)
			}
			return printer.Table(columns, rows)
		}
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	rows := make([][]string, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, []string{key, cell(fields[key])})
	}
	return printer.Table([]string{"Field", "Value"}, rows)
}
func cell(raw json.RawMessage) string {
	var value string
	if json.Unmarshal(raw, &value) == nil {
		return value
	}
	return string(raw)
}

// validateTimes 只处理已明确的整数/顺序边界；精确三个月规则沿后端契约，不推断为90天。
func validateTimes(begin, end int64) error {
	if begin < 0 || end < 0 || (begin != 0 && end != 0 && end < begin) {
		return invalid("time must be nonnegative and end must not precede begin")
	}
	return nil
}

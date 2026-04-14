package toolargs

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type MergeOptions struct {
	ReservedFlags map[string]struct{}
}

// MergeFromCommand builds arguments from base JSON sources and changed flags.
func MergeFromCommand(cmd *cobra.Command, opts MergeOptions) (map[string]interface{}, error) {
	base, err := parseBaseJSON(cmd)
	if err != nil {
		return nil, err
	}

	if opts.ReservedFlags == nil {
		opts.ReservedFlags = map[string]struct{}{}
	}

	overlay := map[string]interface{}{}
	cmd.Flags().Visit(func(f *pflag.Flag) {
		if _, ok := opts.ReservedFlags[f.Name]; ok {
			return
		}
		key := strings.ReplaceAll(f.Name, "-", "_")
		overlay[key] = parseFlagValue(f.Value.Type(), f.Value.String())
	})

	for k, v := range overlay {
		base[k] = v
	}
	return base, nil
}

func parseBaseJSON(cmd *cobra.Command) (map[string]interface{}, error) {
	params, _ := cmd.Flags().GetString("params")
	argsJSON, _ := cmd.Flags().GetString("args-json")
	argsFile, _ := cmd.Flags().GetString("args-file")

	count := 0
	if strings.TrimSpace(params) != "" {
		count++
	}
	if strings.TrimSpace(argsJSON) != "" {
		count++
	}
	if strings.TrimSpace(argsFile) != "" {
		count++
	}
	if count > 1 {
		return nil, fmt.Errorf("only one of --params, --args-json, --args-file can be used")
	}

	switch {
	case strings.TrimSpace(params) != "":
		return parseJSONObject(params)
	case strings.TrimSpace(argsJSON) != "":
		return parseJSONObject(argsJSON)
	case strings.TrimSpace(argsFile) != "":
		raw, err := os.ReadFile(argsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read args file: %w", err)
		}
		return parseJSONObject(string(raw))
	default:
		return map[string]interface{}{}, nil
	}
}

func parseJSONObject(raw string) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("invalid json arguments: %w", err)
	}
	if out == nil {
		out = map[string]interface{}{}
	}
	return out, nil
}

func parseFlagValue(flagType, value string) interface{} {
	switch flagType {
	case "bool":
		v, err := strconv.ParseBool(value)
		if err == nil {
			return v
		}
	case "int", "int64":
		v, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return v
		}
	case "stringSlice":
		if value == "" {
			return []string{}
		}
		parts := strings.Split(value, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			out = append(out, strings.TrimSpace(p))
		}
		return out
	}
	return value
}

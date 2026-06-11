// Package intelcmd provides helpers shared only by gate-cli info and news (Intel MCP) commands.
// Do not use it from trading domains (spot, futures, etc.).
package intelcmd

import (
	"errors"
	"strings"

	"github.com/gate/gate-cli/internal/cmdhint"
	"github.com/gate/gate-cli/internal/exitcode"
	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/output"
)

// ErrSilenced is returned after PrintError so callers exit non-zero without cobra duplicating stderr.
var ErrSilenced = errors.New("intel command failed")

// FailAfterPrintError writes gateErr to stderr via the printer, then returns exit code 1.
// The command must use SilenceErrors=true so cobra does not print this error again.
func FailAfterPrintError(p *output.Printer, gateErr *output.GateError) error {
	if gateErr == nil {
		gateErr = &output.GateError{Status: 500, Label: "INTEL_ERROR", Message: "unknown error"}
	}
	p.PrintError(gateErr)
	return exitcode.New(1, ErrSilenced)
}

// FailUnsupportedTable prints the standard unsupported-format error and returns exit code 1.
func FailUnsupportedTable(p *output.Printer) error {
	return FailAfterPrintError(p, output.UnsupportedTableFormatError())
}

// FailLeafUnsupportedTable is for tool invoke/describe paths that are not tabular; backend is "info" or "news".
func FailLeafUnsupportedTable(p *output.Printer, backend string) error {
	base := output.UnsupportedTableFormatError()
	return FailAfterPrintError(p, &output.GateError{
		Status: base.Status,
		Label:  base.Label,
		Message: base.Message + " For a tabular tool index use `gate-cli " + backend +
			" list --format table`. Use `--format pretty` or `--format json` for tool results and `describe` output.",
	})
}

// RenderToolList prints list output in json/table/pretty formats with shared behavior for info/news.
// User-facing output uses gate-cli command paths, not MCP wire tool names (info_*_*).
func RenderToolList(p *output.Printer, backend string, items []intelfacade.ToolSummary) error {
	if p.IsJSON() && cmdhint.AgentModeEnabled() {
		return p.Print(compactToolList(backend, items))
	}
	if p.IsJSON() {
		return p.Print(userFacingListJSON(backend, items))
	}
	if p.IsTable() {
		rows := make([][]string, 0, len(items))
		for _, item := range items {
			params := "no"
			if item.HasInputSchema {
				params = "yes"
			}
			rows = append(rows, []string{
				UserFacingCLICommand(backend, "", item.Name),
				item.Description,
				params,
			})
		}
		return p.Table([]string{"Command", "Description", "Accepts parameters"}, rows)
	}
	return p.WritePretty(listCapabilitiesPrettyText(backend, items))
}

// RenderDescribeTool prints describe output; agent+json uses a compact shape.
func RenderDescribeTool(p *output.Printer, backend string, tool *intelfacade.ToolSummary) error {
	if tool == nil {
		return nil
	}
	if p.IsJSON() && cmdhint.AgentModeEnabled() {
		return p.Print(compactDescribeTool(backend, tool))
	}
	if p.IsJSON() {
		return p.Print(userFacingDescribeJSON(backend, tool))
	}
	return p.WritePretty(describePrettyText(backend, tool))
}

func userFacingListJSON(backend string, items []intelfacade.ToolSummary) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		entry := map[string]interface{}{
			"command":          UserFacingCLICommand(backend, "", item.Name),
			"has_input_schema": item.HasInputSchema,
		}
		if d := strings.TrimSpace(item.Description); d != "" {
			entry["description"] = d
		}
		out = append(out, entry)
	}
	return out
}

func userFacingDescribeJSON(backend string, tool *intelfacade.ToolSummary) map[string]interface{} {
	out := map[string]interface{}{
		"command":          UserFacingCLICommand(backend, "", tool.Name),
		"has_input_schema": tool.HasInputSchema,
	}
	if d := strings.TrimSpace(tool.Description); d != "" {
		out["description"] = d
	}
	if tool.InputSchema != nil {
		out["input_schema"] = tool.InputSchema
	}
	return out
}

func listCapabilitiesPrettyText(backend string, items []intelfacade.ToolSummary) string {
	if len(items) == 0 {
		return "Capabilities\n\n(no entries)\n"
	}
	var b strings.Builder
	b.WriteString("Capabilities\n\n")
	for i, item := range items {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(UserFacingCLICommand(backend, "", item.Name))
		b.WriteByte('\n')
		if d := strings.TrimSpace(item.Description); d != "" {
			b.WriteString(d)
			b.WriteByte('\n')
		}
		if item.HasInputSchema {
			b.WriteString("Accepts parameters: yes\n")
		} else {
			b.WriteString("Accepts parameters: no\n")
		}
	}
	return b.String()
}

func describePrettyText(backend string, tool *intelfacade.ToolSummary) string {
	if tool == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("Overview\n\n")
	b.WriteString(UserFacingCLICommand(backend, "", tool.Name))
	b.WriteByte('\n')
	if d := strings.TrimSpace(tool.Description); d != "" {
		b.WriteString(d)
		b.WriteByte('\n')
	}
	if paramBlock := intelfacade.FormatParameterSummary(tool); paramBlock != "" {
		b.WriteString("\nParameters\n\n")
		b.WriteString(paramBlock)
		b.WriteByte('\n')
	}
	b.WriteString("\nNext steps\n\n")
	b.WriteString("- Use --format json for the full definition suitable for automation.\n")
	b.WriteString("- Use --help on the leaf command that maps to this capability for CLI flags.\n")
	return b.String()
}

func compactDescribeTool(backend string, tool *intelfacade.ToolSummary) map[string]interface{} {
	out := map[string]interface{}{
		"path":        UserFacingCLICommand(backend, "", tool.Name),
		"description": tool.Description,
	}
	if tool.HasInputSchema {
		out["has_input_schema"] = true
	}
	return out
}

func compactToolList(backend string, items []intelfacade.ToolSummary) []map[string]string {
	backend = strings.TrimSpace(backend)
	out := make([]map[string]string, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]string{
			"path": strings.TrimSpace(UserFacingCLICommand(backend, "", item.Name)),
		})
	}
	return out
}

func toolNameToCLIPath(backend, toolName string) string {
	parts := strings.Split(strings.TrimPrefix(toolName, backend+"_"), "_")
	if len(parts) < 2 {
		return strings.ReplaceAll(toolName, "_", " ")
	}
	group := parts[0]
	leaf := strings.Join(parts[1:], "-")
	if backend != "" {
		return group + " " + leaf
	}
	return strings.Join(parts, " ")
}

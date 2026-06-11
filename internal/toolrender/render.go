package toolrender

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gate/gate-cli/internal/cmdhint"
	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
)

// RenderCallResult writes call results via standard printer.
// JSON mode prints the business data value only (PRD §3.7.8). Pretty mode uses fixed sections
// without protocol wrapper fields (PRD §3.7.5).
func RenderCallResult(p *output.Printer, backend, toolName string, result *mcpclient.CallResult, maxBytes int64) error {
	if result == nil {
		return fmt.Errorf("nil tools/call result")
	}
	envelope := BuildCLIEnvelope(toolName, result)
	if !p.IsJSON() && maxBytes <= 0 {
		return writePrettyToolResult(p, toolName, envelope, nil)
	}
	dataJSON, err := json.Marshal(envelope["data"])
	if err != nil {
		return err
	}
	envelope, displayJSON := ApplyOutputLimitWithData(envelope, maxBytes, dataJSON)
	if p.IsJSON() {
		return printJSONToolResult(p, envelope)
	}
	return writePrettyToolResult(p, toolName, envelope, displayJSON)
}

func writePrettyToolResult(p *output.Printer, toolName string, envelope map[string]interface{}, compactJSON []byte) error {
	data, _ := envelope["data"].(map[string]interface{})
	sectioned, ok := prettyOnchainToolResult(toolName, data)
	if !ok {
		sectioned, ok = prettyPlatformmetricsToolResult(toolName, data)
	}
	if ok && strings.TrimSpace(sectioned) != "" {
		var b strings.Builder
		b.WriteString(sectioned)
		b.WriteByte('\n')
		writePrettyNotes(&b, envelope)
		return p.WritePretty(b.String())
	}

	if len(compactJSON) == 0 {
		var err error
		compactJSON, err = json.Marshal(envelope["data"])
		if err != nil {
			return err
		}
	}
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, compactJSON, "", "  "); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("Result\n\n")
	b.Write(prettyJSON.Bytes())
	b.WriteByte('\n')
	writePrettyNotes(&b, envelope)
	return p.WritePretty(b.String())
}

func writePrettyNotes(b *strings.Builder, envelope map[string]interface{}) {
	notes := append(parseWarningsFromEnvelope(envelope), parseNotesFromEnvelope(envelope)...)
	if len(notes) == 0 {
		return
	}
	b.WriteString("\nNotes\n\n")
	for _, w := range notes {
		b.WriteString("- ")
		b.WriteString(w)
		b.WriteByte('\n')
	}
}

func printJSONToolResult(p *output.Printer, envelope map[string]interface{}) error {
	if cmdhint.AgentModeEnabled() {
		meta, _ := envelope["meta"].(map[string]interface{})
		if meta == nil {
			meta = map[string]interface{}{}
		}
		return p.Print(map[string]interface{}{
			"data": envelope["data"],
			"meta": meta,
		})
	}
	return p.Print(envelope["data"])
}

func parseWarningsFromEnvelope(envelope map[string]interface{}) []string {
	return metaStringList(envelope, "parse_warnings")
}

func parseNotesFromEnvelope(envelope map[string]interface{}) []string {
	var notes []string
	for _, key := range []string{"freshness_hints", "agent_reminder"} {
		notes = append(notes, metaStringList(envelope, key)...)
	}
	return notes
}

func metaStringList(envelope map[string]interface{}, key string) []string {
	meta, ok := envelope["meta"].(map[string]interface{})
	if !ok {
		return nil
	}
	raw, ok := meta[key]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case string:
		if strings.TrimSpace(v) != "" {
			return []string{v}
		}
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, x := range v {
			if s, ok := x.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
	return nil
}

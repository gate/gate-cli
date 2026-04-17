package toolrender

import (
	"encoding/json"
	"strings"

	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
)

// RenderCallResult writes call results via standard printer.
// JSON mode prints the business data value only (PRD §3.7.8). Pretty mode uses fixed sections
// without protocol wrapper fields (PRD §3.7.5).
func RenderCallResult(p *output.Printer, toolName string, result *mcpclient.CallResult, maxBytes int64) error {
	envelope := BuildCLIEnvelope(toolName, result)
	envelope = ApplyOutputLimit(envelope, maxBytes)
	if p.IsJSON() {
		return p.Print(envelope["data"])
	}
	return writePrettyToolResult(p, envelope)
}

func writePrettyToolResult(p *output.Printer, envelope map[string]interface{}) error {
	data := envelope["data"]
	dataJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("Result\n\n")
	b.Write(dataJSON)
	b.WriteByte('\n')
	if ws := parseWarningsFromEnvelope(envelope); len(ws) > 0 {
		b.WriteString("\nNotes\n\n")
		for _, w := range ws {
			b.WriteString("- ")
			b.WriteString(w)
			b.WriteByte('\n')
		}
	}
	return p.WritePretty(b.String())
}

func parseWarningsFromEnvelope(envelope map[string]interface{}) []string {
	meta, ok := envelope["meta"].(map[string]interface{})
	if !ok {
		return nil
	}
	raw, ok := meta["parse_warnings"]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case []string:
		return v
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, x := range v {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

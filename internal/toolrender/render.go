package toolrender

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
)

// RenderCallResult writes call results via standard printer.
// JSON mode prints the business data value only (PRD §3.7.8). Pretty mode uses fixed sections
// without protocol wrapper fields (PRD §3.7.5).
func RenderCallResult(p *output.Printer, toolName string, result *mcpclient.CallResult, maxBytes int64) error {
	if result == nil {
		return fmt.Errorf("nil tools/call result")
	}
	envelope := BuildCLIEnvelope(toolName, result)
	if !p.IsJSON() && maxBytes <= 0 {
		return writePrettyToolResult(p, envelope, nil)
	}
	dataJSON, err := json.Marshal(envelope["data"])
	if err != nil {
		return err
	}
	envelope, displayJSON := ApplyOutputLimitWithData(envelope, maxBytes, dataJSON)
	if p.IsJSON() {
		return p.Print(envelope["data"])
	}
	return writePrettyToolResult(p, envelope, displayJSON)
}

func writePrettyToolResult(p *output.Printer, envelope map[string]interface{}, compactJSON []byte) error {
	if len(compactJSON) == 0 {
		var err error
		compactJSON, err = json.Marshal(envelope["data"])
		if err != nil {
			return err
		}
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, compactJSON, "", "  "); err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString("Result\n\n")
	b.Write(pretty.Bytes())
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

package toolrender

import (
	"encoding/json"

	"github.com/gate/gate-cli/internal/output"
)

// RenderIntelPayload prints shortcut or aggregated Intel data with the same limits/meta as tool calls.
func RenderIntelPayload(p *output.Printer, commandPath string, data interface{}, maxBytes int64) error {
	if p == nil {
		return nil
	}
	envelope := map[string]interface{}{
		"status": "success",
		"data":   data,
	}
	AppendResultMeta(commandPath, envelope)
	dataJSON, err := json.Marshal(envelope["data"])
	if err != nil {
		return err
	}
	envelope, displayJSON := ApplyOutputLimitWithData(envelope, maxBytes, dataJSON)
	if !p.IsJSON() && maxBytes <= 0 {
		return writePrettyToolResult(p, commandPath, envelope, nil)
	}
	if p.IsJSON() {
		return printJSONToolResult(p, envelope)
	}
	return writePrettyToolResult(p, commandPath, envelope, displayJSON)
}

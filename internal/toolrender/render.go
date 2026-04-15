package toolrender

import (
	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
)

// RenderCallResult writes call results via standard printer.
func RenderCallResult(p *output.Printer, toolName string, result *mcpclient.CallResult, maxBytes int64) error {
	envelope := BuildCLIEnvelope(toolName, result)
	envelope = ApplyOutputLimit(envelope, maxBytes)
	if err := p.Print(envelope); err != nil {
		return err
	}
	return nil
}

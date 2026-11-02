package spot

import (
	"bytes"
	"encoding/json"
	"fmt"

	gateapi "github.com/gate/gateapi-go/v7"
)

// validateBatchProtection observes, but never rewrites, the already-decoded SDK
// DTO. Only a literal empty input object may request cancellation. Absent/null,
// normal values, ignored extra fields and existing server-side rules stay intact.
func validateBatchProtection(raw string, items []gateapi.BatchAmendItem) error {
	var inputs []struct {
		StopProfit json.RawMessage `json:"stop_profit"`
		StopLoss   json.RawMessage `json:"stop_loss"`
	}
	if err := json.Unmarshal([]byte(raw), &inputs); err != nil {
		return fmt.Errorf("invalid --orders-json protection")
	}
	if len(inputs) != len(items) {
		return fmt.Errorf("invalid --orders-json protection count")
	}
	for i, item := range inputs {
		for _, p := range []struct {
			name    string
			raw     json.RawMessage
			decoded any
		}{{"stop_profit", item.StopProfit, items[i].StopProfit}, {"stop_loss", item.StopLoss, items[i].StopLoss}} {
			value := bytes.TrimSpace(p.raw)
			if len(value) == 0 || bytes.Equal(value, []byte("null")) {
				continue
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(value, &fields); err != nil {
				return fmt.Errorf("invalid --orders-json protection at item %d", i+1)
			}
			if len(fields) == 0 {
				continue
			}
			encoded, err := json.Marshal(p.decoded)
			if err != nil {
				return fmt.Errorf("cannot encode --orders-json protection at item %d", i+1)
			}
			if bytes.Equal(encoded, []byte("{}")) {
				return fmt.Errorf("--orders-json item %d %s must not collapse to an empty cancellation object", i+1, p.name)
			}
		}
	}
	return nil
}

package output_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/revil/gate-cli/internal/output"
)

func TestJSONOutput(t *testing.T) {
	var buf bytes.Buffer
	p := output.New(&buf, output.FormatJSON)

	data := map[string]string{"currency": "BTC", "available": "0.1"}
	err := p.Print(data)
	require.NoError(t, err)

	var result map[string]string
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Equal(t, "BTC", result["currency"])
}

func TestTableOutput(t *testing.T) {
	var buf bytes.Buffer
	p := output.New(&buf, output.FormatTable)

	rows := [][]string{{"BTC", "0.1", "0.0"}, {"USDT", "1000", "0"}}
	err := p.Table([]string{"Currency", "Available", "Locked"}, rows)
	require.NoError(t, err)

	out := buf.String()
	assert.Contains(t, out, "Currency")
	assert.Contains(t, out, "BTC")
	assert.Contains(t, out, "USDT")
}

func TestTableAsJSONWhenFormatJSON(t *testing.T) {
	var buf bytes.Buffer
	p := output.New(&buf, output.FormatJSON)

	rows := [][]string{{"BTC", "0.1"}, {"ETH", "2.0"}}
	err := p.Table([]string{"currency", "available"}, rows)
	require.NoError(t, err)

	var result []map[string]string
	err = json.Unmarshal(buf.Bytes(), &result)
	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "BTC", result[0]["currency"])
}

func TestErrorJSONGateStandard(t *testing.T) {
	var stdout, stderr bytes.Buffer
	p := output.NewWithStderr(&stdout, &stderr, output.FormatJSON)

	gateErr := &output.GateError{
		Status:  400,
		Label:   "INVALID_PARAM_VALUE",
		Message: "Invalid currency pair",
		TraceID: "abc123",
		Request: &output.RequestInfo{
			Method: "POST",
			URL:    "https://api.gateio.ws/api/v4/spot/orders",
			Body:   `{"currency_pair":"INVALID"}`,
		},
	}
	p.PrintError(gateErr)

	var result map[string]interface{}
	err := json.Unmarshal(stderr.Bytes(), &result)
	require.NoError(t, err)
	errObj := result["error"].(map[string]interface{})
	assert.Equal(t, float64(400), errObj["status"])
	assert.Equal(t, "INVALID_PARAM_VALUE", errObj["label"])
	assert.Equal(t, "abc123", errObj["trace_id"])
	assert.NotNil(t, errObj["request"])
}

func TestErrorTableMode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	p := output.NewWithStderr(&stdout, &stderr, output.FormatTable)

	gateErr := &output.GateError{
		Status:  400,
		Label:   "INVALID_PARAM_VALUE",
		Message: "Invalid currency pair",
		TraceID: "abc123",
		Request: &output.RequestInfo{Method: "POST", URL: "https://api.gateio.ws/api/v4/spot/orders"},
	}
	p.PrintError(gateErr)

	out := stderr.String()
	assert.Contains(t, out, "400")
	assert.Contains(t, out, "INVALID_PARAM_VALUE")
	assert.Contains(t, out, "abc123")
	assert.Contains(t, out, "POST")
}

func TestIsJSON(t *testing.T) {
	pJSON := output.New(nil, output.FormatJSON)
	pTable := output.New(nil, output.FormatTable)
	assert.True(t, pJSON.IsJSON())
	assert.False(t, pTable.IsJSON())
}

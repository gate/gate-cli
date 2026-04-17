package output

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

// Format controls output rendering.
type Format string

const (
	FormatPretty Format = "pretty"
	FormatTable  Format = "table"
	FormatJSON   Format = "json"
)

// ParseFormat converts a string flag value to Format.
func ParseFormat(s string) Format {
	switch s {
	case "json":
		return FormatJSON
	case "table":
		return FormatTable
	case "pretty":
		return FormatPretty
	default:
		return FormatPretty
	}
}

// UnsupportedTableFormatError is returned when --format table is used on a command
// whose result is not tabular per product output rules (PRD §3.7).
func UnsupportedTableFormatError() *GateError {
	return &GateError{
		Status:  400,
		Label:   "UNSUPPORTED_FORMAT",
		Message: "this command does not support --format table; use --format pretty or --format json",
	}
}

// RequestInfo captures the HTTP request that triggered an error.
type RequestInfo struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Body   string `json:"body,omitempty"`
}

// GateError is a unified error representation for all Gate API errors.
type GateError struct {
	Status      int          `json:"status"`
	Label       string       `json:"label,omitempty"`
	Message     string       `json:"message"`
	TraceID     string       `json:"trace_id,omitempty"`
	RequestID   string       `json:"request_id,omitempty"`
	ToolName    string       `json:"tool_name,omitempty"`
	JSONRPCCode *int         `json:"jsonrpc_code,omitempty"`
	Request     *RequestInfo `json:"request,omitempty"`
}

// Printer writes structured output to stdout and errors to stderr.
type Printer struct {
	out    io.Writer
	errOut io.Writer
	format Format
}

// New creates a Printer that writes errors to os.Stderr.
func New(out io.Writer, format Format) *Printer {
	return &Printer{out: out, errOut: os.Stderr, format: format}
}

// NewWithStderr creates a Printer with custom stderr writer (useful for testing).
func NewWithStderr(out, errOut io.Writer, format Format) *Printer {
	return &Printer{out: out, errOut: errOut, format: format}
}

// IsJSON returns true if the printer is in JSON mode.
func (p *Printer) IsJSON() bool {
	return p.format == FormatJSON
}

// IsTable returns true if the printer is in table layout mode (--format table).
func (p *Printer) IsTable() bool {
	return p.format == FormatTable
}

// Format returns the active output format.
func (p *Printer) Format() Format {
	return p.format
}

// Print serialises data to stdout as JSON.
// JSON mode uses compact encoding (single line, plus trailing newline) for piping and jq.
// Pretty/table mode uses indented JSON for readability.
func (p *Printer) Print(data interface{}) error {
	var b []byte
	var err error
	if p.format == FormatJSON {
		b, err = json.Marshal(data)
	} else {
		b, err = json.MarshalIndent(data, "", "  ")
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(p.out, string(b))
	return err
}

// WritePretty writes human-oriented text to stdout (for --format pretty or --format table).
// Do not use in JSON mode; machine-readable output must use Print.
func (p *Printer) WritePretty(s string) error {
	if p.format == FormatJSON {
		return fmt.Errorf("output: WritePretty must not be used in JSON mode")
	}
	_, err := io.WriteString(p.out, s)
	if err != nil {
		return err
	}
	if len(s) == 0 || s[len(s)-1] != '\n' {
		_, err = p.out.Write([]byte{'\n'})
	}
	return err
}

// Table renders headers+rows as a table (or JSON array of objects in JSON mode).
func (p *Printer) Table(headers []string, rows [][]string) error {
	if p.format == FormatJSON {
		var result []map[string]string
		for _, row := range rows {
			m := make(map[string]string)
			for i, h := range headers {
				if i < len(row) {
					m[h] = row[i]
				}
			}
			result = append(result, m)
		}
		return p.Print(result)
	}

	t := tablewriter.NewWriter(p.out)
	t.Configure(func(cfg *tablewriter.Config) {
		cfg.Header.Formatting.AutoFormat = tw.Off
	})
	headerCells := make([]interface{}, len(headers))
	for i, h := range headers {
		headerCells[i] = h
	}
	t.Header(headerCells...)
	for _, row := range rows {
		cells := make([]interface{}, len(row))
		for i, c := range row {
			cells[i] = c
		}
		if err := t.Append(cells...); err != nil {
			return err
		}
	}
	return t.Render()
}

// PrintError writes a GateError to stderr in the appropriate format.
// JSON mode: structured JSON with "error" wrapper.
// Table mode: human-readable lines.
func (p *Printer) PrintError(gateErr *GateError) {
	if p.format == FormatJSON {
		out := map[string]interface{}{"error": gateErr}
		b, _ := json.Marshal(out)
		_, _ = fmt.Fprintln(p.errOut, string(b))
		return
	}

	label := gateErr.Label
	if label == "" {
		label = http.StatusText(gateErr.Status)
	}
	_, _ = fmt.Fprintf(p.errOut, "Error [%d %s]: %s\n", gateErr.Status, label, gateErr.Message)
	if gateErr.TraceID != "" {
		_, _ = fmt.Fprintf(p.errOut, "Trace ID: %s\n", gateErr.TraceID)
	}
	if gateErr.RequestID != "" {
		_, _ = fmt.Fprintf(p.errOut, "Request ID: %s\n", gateErr.RequestID)
	}
	if gateErr.ToolName != "" {
		_, _ = fmt.Fprintf(p.errOut, "Tool: %s\n", gateErr.ToolName)
	}
	if gateErr.JSONRPCCode != nil {
		_, _ = fmt.Fprintf(p.errOut, "JSON-RPC Code: %d\n", *gateErr.JSONRPCCode)
	}
	if gateErr.Request != nil {
		_, _ = fmt.Fprintf(p.errOut, "Request: %s %s\n", gateErr.Request.Method, gateErr.Request.URL)
	}
}

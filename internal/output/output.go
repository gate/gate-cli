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
	FormatTable Format = "table"
	FormatJSON  Format = "json"
)

// ParseFormat converts a string flag value to Format.
func ParseFormat(s string) Format {
	if s == "json" {
		return FormatJSON
	}
	return FormatTable
}

// RequestInfo captures the HTTP request that triggered an error.
type RequestInfo struct {
	Method string `json:"method"`
	URL    string `json:"url"`
	Body   string `json:"body,omitempty"`
}

// GateError is a unified error representation for all Gate API errors.
type GateError struct {
	Status  int          `json:"status"`
	Label   string       `json:"label,omitempty"`
	Message string       `json:"message"`
	TraceID string       `json:"trace_id,omitempty"`
	Request *RequestInfo `json:"request,omitempty"`
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

// Print serialises data as indented JSON to stdout.
func (p *Printer) Print(data interface{}) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(p.out, string(b))
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
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintln(p.errOut, string(b))
		return
	}

	label := gateErr.Label
	if label == "" {
		label = http.StatusText(gateErr.Status)
	}
	fmt.Fprintf(p.errOut, "Error [%d %s]: %s\n", gateErr.Status, label, gateErr.Message)
	if gateErr.TraceID != "" {
		fmt.Fprintf(p.errOut, "Trace ID: %s\n", gateErr.TraceID)
	}
	if gateErr.Request != nil {
		fmt.Fprintf(p.errOut, "Request: %s %s\n", gateErr.Request.Method, gateErr.Request.URL)
	}
}

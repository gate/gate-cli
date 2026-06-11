package intelcmd

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gate/gate-cli/internal/mcpclient"
	"github.com/gate/gate-cli/internal/output"
)

// ShortcutToolCaller is the MCP tools/call surface used by info/news shortcuts.
type ShortcutToolCaller interface {
	CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcpclient.CallResult, *http.Response, error)
}

// ShortcutHTTPError wraps a transport error from a shortcut internal tools/call.
type ShortcutHTTPError struct {
	Err      error
	HTTPResp *http.Response
	ToolName string
}

func (e *ShortcutHTTPError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

// ShortcutToolIsError wraps MCP tools/call result.isError=true.
type ShortcutToolIsError struct {
	ToolName string
	HTTPResp *http.Response
	Result   *mcpclient.CallResult
}

func (e *ShortcutToolIsError) Error() string { return "tool returned isError=true" }

// ShortcutArgsError is a shortcut-local argument validation failure.
type ShortcutArgsError string

func (e ShortcutArgsError) Error() string { return string(e) }

// CallShortcutTool prepares args, calls MCP, and parses structured shortcut payload.
func CallShortcutTool(ctx context.Context, caller ShortcutToolCaller, name string, args map[string]interface{}) (map[string]interface{}, error) {
	args, err := PrepareToolArguments(name, args)
	if err != nil {
		return nil, err
	}
	result, resp, err := caller.CallTool(ctx, name, args)
	if err != nil {
		return nil, &ShortcutHTTPError{Err: err, HTTPResp: resp, ToolName: name}
	}
	if result == nil {
		return nil, errors.New("intel tool returned empty response")
	}
	if result.IsError {
		return nil, &ShortcutToolIsError{ToolName: name, HTTPResp: resp, Result: result}
	}
	if len(result.StructuredContent) > 0 {
		return result.StructuredContent, nil
	}
	for _, raw := range result.ContentRaw {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		text, _ := item["text"].(string)
		if strings.TrimSpace(text) == "" {
			continue
		}
		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(text), &parsed); err == nil {
			return parsed, nil
		}
	}
	if len(result.Raw) > 0 {
		return result.Raw, nil
	}
	return map[string]interface{}{}, nil
}

// GateErrorFromShortcutErr maps shortcut runner errors to stderr GateError shape.
func GateErrorFromShortcutErr(err error, path string) *output.GateError {
	if err == nil {
		return output.InvalidArgsError("unknown shortcut error")
	}
	backend := "info"
	if strings.HasPrefix(path, "news/") {
		backend = "news"
	}
	var ge *output.GateError
	var isErr *ShortcutToolIsError
	if errors.As(err, &isErr) {
		ge = GateErrorForIntelToolIsError(isErr.ToolName, isErr.HTTPResp, isErr.Result)
	} else if httpErr, ok := err.(*ShortcutHTTPError); ok {
		ge = mcpclient.ParseError(httpErr.Err, httpErr.HTTPResp, "POST", path, "")
	} else if argErr, ok := err.(ShortcutArgsError); ok {
		return output.InvalidArgsError(argErr.Error())
	} else {
		return output.InvalidArgsError(err.Error())
	}
	SanitizeUserFacingGateError(ge, backend, path, "")
	return ge
}

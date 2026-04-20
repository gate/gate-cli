// Package mcpspec embeds MCP tool input documents shipped with the CLI for offline
// agent/LLM use (parameter enums, defaults, bounds, logic text). Bundled JSON must
// stay byte-identical to specs/mcp/*.json; see TestBundledMatchesSpecs.
package mcpspec

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed bundled/info-mcp-tools-inputs-logic.json
var infoInputsLogicJSON []byte

//go:embed bundled/news-tools-args-and-logic.json
var newsToolsArgsJSON []byte

func init() {
	for _, raw := range [][]byte{infoInputsLogicJSON, newsToolsArgsJSON} {
		var v interface{}
		if err := json.Unmarshal(raw, &v); err != nil {
			panic("mcpspec: invalid embedded JSON: " + err.Error())
		}
	}
}

var (
	infoParsedOnce sync.Once
	newsParsedOnce sync.Once
	infoParsed     interface{}
	newsParsed     interface{}
	infoParseErr   error
	newsParseErr   error
)

// InfoInputsLogic returns the parsed Info MCP inputs/spec document (same shape as specs/mcp/info-mcp-tools-inputs-logic.json).
func InfoInputsLogic() (interface{}, error) {
	infoParsedOnce.Do(func() {
		infoParseErr = json.Unmarshal(infoInputsLogicJSON, &infoParsed)
	})
	return infoParsed, infoParseErr
}

// NewsToolsArgs returns the parsed News tools args/logic document (same shape as specs/mcp/news-tools-args-and-logic.json).
func NewsToolsArgs() (interface{}, error) {
	newsParsedOnce.Do(func() {
		newsParseErr = json.Unmarshal(newsToolsArgsJSON, &newsParsed)
	})
	return newsParsed, newsParseErr
}

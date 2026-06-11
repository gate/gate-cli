package mcpspec

import "encoding/json"

// stripInfoToolDescriptions returns a copy of the info MCP spec with description and
// description_zh removed from tools and meta.tool_entry_keys. Used to compare local
// specs/mcp (QC, not shipped) with the embedded bundled document on logic/fields only.
func stripInfoToolDescriptions(doc interface{}) (interface{}, error) {
	raw, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if meta, ok := m["meta"].(map[string]interface{}); ok {
		if keys, ok := meta["tool_entry_keys"].(map[string]interface{}); ok {
			delete(keys, "description")
			delete(keys, "description_zh")
		}
		// gate-cli release extensions; local specs/mcp QC copy does not carry these.
		delete(meta, "cli_baseline_tools")
		delete(meta, "spec_only_tools")
	}
	rawTools, ok := m["tools"].([]interface{})
	if !ok {
		return m, nil
	}
	for _, item := range rawTools {
		tm, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		delete(tm, "description")
		delete(tm, "description_zh")
	}
	return m, nil
}

// infoSpecEqualExcludingDescriptions reports whether spec and bundled match after
// removing description fields from both (local QC spec must not drive release text).
func infoSpecEqualExcludingDescriptions(specDoc, bundledDoc interface{}) (bool, error) {
	strippedSpec, err := stripInfoToolDescriptions(specDoc)
	if err != nil {
		return false, err
	}
	strippedBundled, err := stripInfoToolDescriptions(bundledDoc)
	if err != nil {
		return false, err
	}
	a, err := json.Marshal(strippedSpec)
	if err != nil {
		return false, err
	}
	b, err := json.Marshal(strippedBundled)
	if err != nil {
		return false, err
	}
	return string(a) == string(b), nil
}

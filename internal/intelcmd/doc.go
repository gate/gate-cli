// Package intelcmd holds shared command wiring for Gate CLI intel backends (info, news):
// grouped and leaf shortcut commands, schema cache loading with baseline merge, MCP JSON
// fallback flags, leaf alias wiring, and helpers for doctor/preflight-style flows.
//
// Schema refresh is controlled by toolschema.ForceRefreshEnabled (GATE_INTEL_REFRESH_SCHEMA),
// not by argv sniffing (CR-826).
package intelcmd

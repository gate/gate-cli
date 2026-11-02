package contracts

import (
	_ "embed"
	"encoding/json"
	"strings"
	"testing"
)

// Scenario keeps a frozen wire oracle separate from the implementation. These
// compatibility inputs were executed against the pre-fix CLI before promotion.
type Scenario struct {
	ID     string `json:"id"`
	Case   Case   `json:"case"`
	Reject bool   `json:"reject,omitempty"`
}

//go:embed compatibility.json
var compatibilityJSON []byte

func CompatibilityScenarios(t testing.TB) []Scenario {
	t.Helper()
	var out []Scenario
	if err := json.Unmarshal(compatibilityJSON, &out); err != nil {
		t.Fatal(err)
	}
	by := map[string]Case{}
	for _, c := range Cases(t) {
		by[c.Name] = c
	}
	clone := func(name string) Case {
		c := by[name]
		c.Args = append([]string{}, c.Args...)
		c.Exchanges = append([]Exchange{}, c.Exchanges...)
		return c
	}
	for _, side := range []string{"long", "short"} {
		for _, size := range []string{"2", "-2"} {
			c := clone("cex futures order add")
			c.Args = []string{"cex", "futures", "order", "add", "--contract", "SYNTH_USDT", "--size", size}
			c.Exchanges[0].Reply = json.RawMessage(`{"in_dual_mode":true,"currency":"USDT"}`)
			c.Exchanges[1].Path = "/api/v4/futures/usdt/dual_comp/positions/SYNTH_USDT"
			if side == "long" {
				c.Exchanges[1].Reply = json.RawMessage(`[{"size":"4"},{"size":"-7"}]`)
			} else {
				c.Exchanges[1].Reply = json.RawMessage(`[{"size":"-7"},{"size":"4"}]`)
			}
			wire := size
			if side == "short" && size == "2" {
				wire = "-2"
			}
			c.Exchanges[2].Body = json.RawMessage(`{"contract":"SYNTH_USDT","size":"` + wire + `","price":"0","tif":"ioc"}`)
			out = append(out, Scenario{ID: "preserve-two-sided-selection-" + side + size, Case: c})
		}
	}
	c := clone("cex futures order close")
	c.Args = append(c.Args, "--side", "shorts")
	out = append(out, Scenario{ID: "single-mode-irrelevant-side-unchanged", Case: c})
	// The original SDK ignores these unknown fields. New local guards cannot
	// impose the single-object 64KiB/32-level policy on a whole legacy batch.
	nested := "1"
	for range 33 {
		nested = `{"next":` + nested + `}`
	}
	for _, extra := range []struct{ id, field string }{{"depth33", `"ignored_nested":` + nested}, {"over64k", `"ignored_note":"` + strings.Repeat("x", 70000) + `"`}, {"unicode-protection", `"ſtop_loſſ":{"trigger_price":"8"}`}} {
		c = clone("cex spot order batch-amend")
		c.Args = []string{"cex", "spot", "order", "batch-amend", "--orders-json", `[{"order_id":"9007199254740993","currency_pair":"SYNTH_USDT","amount":"2",` + extra.field + `}]`}
		protection := "null"
		if extra.id == "unicode-protection" {
			protection = `{"trigger_price":"8"}`
		}
		c.Exchanges[0].Body = json.RawMessage(`[{"order_id":"9007199254740993","currency_pair":"SYNTH_USDT","amount":"2","stop_profit":null,"stop_loss":` + protection + `}]`)
		out = append(out, Scenario{ID: extra.id, Case: c})
	}
	return out
}

// RejectionScenarios extends the known defects without changing their success
// oracle: each invalid batch must stop before its only business POST.
func RejectionScenarios(t testing.TB) []Scenario {
	t.Helper()
	var out []Scenario
	for _, c := range UnsafeIntents(t) {
		out = append(out, Scenario{ID: c.ID, Case: c.Case})
	}
	var spot, futures Case
	for _, c := range Cases(t) {
		if c.Name == "cex spot order batch-amend" {
			spot = c
		}
		if c.Name == "cex futures order batch-create" {
			futures = c
		}
	}
	for _, tc := range []struct{ id, raw, wire string }{
		{"empty-members", `"stop_loss":{"trigger_price":""}`, `"stop_loss":{}`},
		{"unicode-null", `"ſtop_loſſ":{"trigger_price":null}`, `"stop_loss":{}`},
		{"profit-null", `"stop_profit":{"trigger_price":null}`, `"stop_profit":{}`},
		{"outer-duplicate", `"stop_loss":{"trigger_price":"8"},"STOP_LOSS":{}`, `"stop_loss":{}`},
	} {
		c := spot
		c.Args = append([]string{}, spot.Args...)
		c.Exchanges = append([]Exchange{}, spot.Exchanges...)
		c.Args = []string{"cex", "spot", "order", "batch-amend", "--orders-json", `[{"order_id":"9007199254740993","currency_pair":"SYNTH_USDT","amount":"2",` + tc.raw + `}]`}
		body := `{"order_id":"9007199254740993","currency_pair":"SYNTH_USDT","amount":"2","stop_profit":null,"stop_loss":{}}`
		// encoding/json merges a second object into the existing pointer. The
		// duplicate is still ambiguous, but the old wire retains the first value.
		if tc.id == "outer-duplicate" {
			body = `{"order_id":"9007199254740993","currency_pair":"SYNTH_USDT","amount":"2","stop_profit":null,"stop_loss":{"trigger_price":"8"}}`
		}
		if tc.id == "profit-null" {
			body = `{"order_id":"9007199254740993","currency_pair":"SYNTH_USDT","amount":"2","stop_profit":{},"stop_loss":null}`
		}
		c.Exchanges[0].Body = json.RawMessage("[" + body + "]")
		out = append(out, Scenario{ID: tc.id, Case: c})
	}
	for _, key := range []string{"SIZE", "ſize"} {
		c := futures
		c.Args = []string{"cex", "futures", "order", "batch-create", "--orders-json", `[{"contract":"SYNTH_USDT","size":"1","` + key + `":"2","price":"0","tif":"ioc"}]`}
		c.Exchanges = append([]Exchange{}, futures.Exchanges...)
		c.Exchanges[0].Body = json.RawMessage(`[{"contract":"SYNTH_USDT","size":"2","price":"0","tif":"ioc"}]`)
		out = append(out, Scenario{ID: "duplicate-" + key, Case: c})
	}
	c := spot
	c.Args = []string{"cex", "spot", "order", "batch-amend", "--orders-json", `[{"order_id":"9007199254740993","currency_pair":"SYNTH_USDT","amount":"2","stop_loss":{"trigger_price":"8"}},{"order_id":"9007199254740994","currency_pair":"SYNTH_USDT","amount":"2","stop_loss":{"trigger_price":null}}]`}
	c.Exchanges = append([]Exchange{}, spot.Exchanges...)
	c.Exchanges[0].Body = json.RawMessage(`[{"order_id":"9007199254740993","currency_pair":"SYNTH_USDT","amount":"2","stop_profit":null,"stop_loss":{"trigger_price":"8"}},{"order_id":"9007199254740994","currency_pair":"SYNTH_USDT","amount":"2","stop_profit":null,"stop_loss":{}}]`)
	var replies []json.RawMessage
	if err := json.Unmarshal(c.Exchanges[0].Reply, &replies); err != nil {
		t.Fatal(err)
	}
	replies = append(replies, replies[0])
	encoded, err := json.Marshal(replies)
	if err != nil {
		t.Fatal(err)
	}
	c.Exchanges[0].Reply = encoded
	out = append(out, Scenario{ID: "mixed-valid-invalid-whole-batch", Case: c})
	return out
}

// ArgumentScenarios distinguishes boolean values from unowned positional words.
// Reusing the ordinary SDK fixture keeps its response and normal business path.
func ArgumentScenarios(t testing.TB) []Scenario {
	t.Helper()
	var base Case
	for _, c := range Cases(t) {
		if c.Name == "cex futures account dual-mode" {
			base = c
		}
	}
	var out []Scenario
	for _, tc := range []struct {
		id     string
		flags  []string
		query  string
		reject bool
	}{
		{"bare", []string{"--enable"}, "true", false},
		{"equal-true", []string{"--enable=true"}, "true", false},
		{"equal-false", []string{"--enable=false"}, "false", false},
		{"space-true", []string{"--enable", "true"}, "true", false},
		{"space-false", []string{"--enable", "false"}, "false", false},
		{"space-zero", []string{"--enable", "0"}, "false", false},
		{"space-uppercase", []string{"--enable", "TRUE"}, "true", false},
		{"invalid-literal", []string{"--enable", "not-a-bool"}, "true", true},
		{"extra-after-bool", []string{"--enable", "true", "__unexpected_position__"}, "true", true},
		{"terminator", []string{"--", "--enable", "false"}, "false", true},
		{"flag-like-string-value", []string{"--settle", "--enable", "--enable=false"}, "false", false},
	} {
		c := base
		c.Args = append(append([]string{}, base.Args...), tc.flags...)
		c.Exchanges = append([]Exchange{}, base.Exchanges...)
		c.Exchanges[0].Query = "dual_mode=" + tc.query
		if tc.id == "flag-like-string-value" {
			c.Exchanges[0].Path = "/api/v4/futures/--enable/dual_mode"
		}
		out = append(out, Scenario{ID: tc.id, Case: c, Reject: tc.reject})
	}
	return out
}

package contracts

import (
	"encoding/json"
	"testing"
)

// UnsafeIntent records the harmful wire request reached by an invalid intent.
// Its acceptance oracle is NOT that this request should succeed: callers must
// require rejection before the terminal write. The script makes a failure
// diagnostic reproducible without sending anything to a real trading service.
type UnsafeIntent struct {
	ID   string
	Case Case
}

// UnsafeIntents supplies independent boundary counterexamples, while retaining
// the released SDK's baseline response shapes and synthetic credential scope.
func UnsafeIntents(t testing.TB) []UnsafeIntent {
	t.Helper()
	byName := map[string]Case{}
	for _, c := range Cases(t) {
		byName[c.Name] = c
	}
	copyCase := func(name string) Case {
		c, ok := byName[name]
		if !ok {
			t.Fatalf("missing baseline business contract %s", name)
		}
		c.Args = append([]string{}, c.Args...)
		c.Exchanges = append([]Exchange{}, c.Exchanges...)
		return c
	}
	flag := func(c *Case, name, value string) {
		for i, arg := range c.Args {
			if arg == name {
				c.Args[i+1] = value
				return
			}
		}
		c.Args = append(c.Args, name, value)
	}
	write := func(c *Case, body string) { c.Exchanges[len(c.Exchanges)-1].Body = json.RawMessage(body) }
	var out []UnsafeIntent
	c := copyCase("cex futures order long")
	flag(&c, "--size", "-2")
	write(&c, `{"contract":"SYNTH_USDT","size":"-2","price":"0","tif":"ioc"}`)
	out = append(out, UnsafeIntent{"FUT-NEGATIVE-LONG", c})
	c = copyCase("cex futures order add")
	flag(&c, "--size", "-14")
	c.Exchanges[1].Reply = json.RawMessage(`{"contract":"SYNTH_USDT","size":"10","mode":"single"}`)
	write(&c, `{"contract":"SYNTH_USDT","size":"-14","price":"0","tif":"ioc"}`)
	out = append(out, UnsafeIntent{"FUT-NEGATIVE-ADD", c})
	c = copyCase("cex futures order close")
	flag(&c, "--side", "shorts")
	c.Exchanges[0].Reply = json.RawMessage(`{"in_dual_mode":true,"currency":"USDT"}`)
	write(&c, `{"contract":"SYNTH_USDT","size":"0","price":"0","tif":"ioc","auto_size":"close_long"}`)
	out = append(out, UnsafeIntent{"FUT-INVALID-CLOSE-SIDE", c})
	c = copyCase("cex futures order batch-create")
	flag(&c, "--orders-json", `[{"contract":"SYNTH_USDT","size":"1","price":"0","tif":"ioc","reduce_only":true,"reduce_only":false}]`)
	write(&c, `[{"contract":"SYNTH_USDT","size":"1","price":"0","tif":"ioc"}]`)
	out = append(out, UnsafeIntent{"FUT-DUPLICATE-REDUCE-ONLY", c})
	for _, protection := range []struct{ id, raw string }{
		{"SPOT-NULL-PROTECTION-MEMBER", `{"trigger_price":null}`},
		{"SPOT-UNKNOWN-PROTECTION-MEMBER", `{"triger_price":"8"}`},
	} {
		c = copyCase("cex spot order batch-amend")
		flag(&c, "--orders-json", `[{"order_id":"9007199254740993","currency_pair":"SYNTH_USDT","amount":"2","stop_loss":`+protection.raw+`}]`)
		write(&c, `[{"order_id":"9007199254740993","currency_pair":"SYNTH_USDT","amount":"2","stop_profit":null,"stop_loss":{}}]`)
		out = append(out, UnsafeIntent{protection.id, c})
	}
	return out
}

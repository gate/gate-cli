package info

import (
	"testing"

	"github.com/gate/gate-cli/internal/intelcmd"
	"github.com/gate/gate-cli/internal/intelfacade"
	"github.com/gate/gate-cli/internal/toolschema"
)

func TestMergeInfoBaselineIntoFillsGetCoinInfo(t *testing.T) {
	t.Parallel()
	out := map[string]toolschema.ToolSummary{}
	intelcmd.MergeToolBaselineInto(out, intelfacade.InfoToolBaseline, intelfacade.InfoBaselineInputSchema)
	s, ok := out["info_coin_get_coin_info"]
	if !ok || toolschema.IsEmptyInputSchema(s.InputSchema) {
		t.Fatalf("expected baseline schema, got %#v", s)
	}
}

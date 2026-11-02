//go:build agent

package agentcmd

import (
	"fmt"
	"github.com/gate/gate-cli/internal/cmdindex"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"sort"
	"strings"
)

type flagDetail struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Usage    string `json:"usage"`
	Required bool   `json:"required"`
}
type commandDetail struct {
	Path                string       `json:"path"`
	Effect              string       `json:"effect"`
	ConstraintsComplete bool         `json:"constraints_complete"`
	FailureAction       string       `json:"failure_action"`
	Flags               []flagDetail `json:"flags"`
}

// cexDetails reads declarations only, without Cobra Find/Parse/Execute or merging
// flagsets into the target. Values/defaults may be credentials and are excluded.
func cexDetails(root *cobra.Command, selector string) ([]commandDetail, error) {
	entries := cmdindex.FilterByDomain(cmdindex.CollectLeaves(root), "cex")
	out := make([]commandDetail, 0, len(entries))
	for _, entry := range entries {
		if selector != "" && entry.Path != selector {
			continue
		}
		target := root
		for _, part := range strings.Fields(entry.Path) {
			for _, sub := range target.Commands() {
				if sub.Name() == part {
					target = sub
					break
				}
			}
		}
		effect := target.Annotations["gate.effect"]
		if effect != "read" && effect != "write" {
			effect = "unknown"
		}
		d := commandDetail{Path: entry.Path, Effect: effect, FailureAction: "inspect the error and leaf help; reconcile unknown write outcomes before any further submission", Flags: []flagDetail{}}
		flags := map[string]flagDetail{}
		collect := func(f *pflag.Flag) {
			if f.Hidden {
				return
			}
			if _, ok := flags[f.Name]; ok {
				return
			}
			required := false
			for _, v := range f.Annotations[cobra.BashCompOneRequiredFlag] {
				required = required || v == "true"
			}
			flags[f.Name] = flagDetail{f.Name, f.Value.Type(), f.Usage, required}
		}
		target.Flags().VisitAll(collect)
		for node := target; node != nil; node = node.Parent() {
			node.PersistentFlags().VisitAll(collect)
		}
		for _, f := range flags {
			d.Flags = append(d.Flags, f)
		}
		sort.Slice(d.Flags, func(i, j int) bool { return d.Flags[i].Name < d.Flags[j].Name })
		out = append(out, d)
	}
	if selector != "" && len(out) == 0 {
		return nil, fmt.Errorf("--command must name a canonical runnable CEX leaf")
	}
	return out, nil
}

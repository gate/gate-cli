package cmdindex

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestSearchRedeemRecords(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "gate-cli"}
	earn := &cobra.Command{Use: "earn"}
	uni := &cobra.Command{Use: "uni"}
	rec := &cobra.Command{Use: "records", Run: func(*cobra.Command, []string) {}}
	root.AddCommand(earn)
	earn.AddCommand(uni)
	uni.AddCommand(rec)

	hits := Search(CollectLeaves(root), "redeem records", 5)
	if len(hits) == 0 || !strings.Contains(hits[0].Path, "records") {
		t.Fatalf("hits=%v", hits)
	}
}

func TestFilterByDomainCex(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "gate-cli"}
	cex := &cobra.Command{Use: "cex"}
	info := &cobra.Command{Use: "info"}
	leafCex := &cobra.Command{Use: "ticker", Run: func(*cobra.Command, []string) {}}
	leafInfo := &cobra.Command{Use: "get-kline", Run: func(*cobra.Command, []string) {}}
	root.AddCommand(cex, info)
	cex.AddCommand(leafCex)
	info.AddCommand(leafInfo)

	all := CollectLeaves(root)
	filtered := FilterByDomain(all, "cex")
	if len(filtered) != 1 || filtered[0].Path != "cex ticker" {
		t.Fatalf("filtered=%v", filtered)
	}
	if len(FilterByDomain(all, "trading")) != 1 {
		t.Fatalf("trading alias should match cex")
	}
}

func TestSearchSynonymRedeem(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "gate-cli"}
	earn := &cobra.Command{Use: "earn"}
	uni := &cobra.Command{Use: "uni"}
	rec := &cobra.Command{Use: "+redeem-records", Short: "simple earn redeem records", Run: func(*cobra.Command, []string) {}}
	root.AddCommand(earn)
	earn.AddCommand(uni)
	uni.AddCommand(rec)

	hits := Search(CollectLeaves(root), "redeem", 5)
	if len(hits) == 0 || !strings.Contains(hits[0].Path, "redeem") {
		t.Fatalf("hits=%v", hits)
	}
}

func TestCollectSkipsParents(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "gate-cli"}
	parent := &cobra.Command{Use: "cex"}
	leaf := &cobra.Command{Use: "ticker", RunE: func(*cobra.Command, []string) error { return nil }}
	root.AddCommand(parent)
	parent.AddCommand(leaf)
	leaves := CollectLeaves(root)
	if len(leaves) != 1 || leaves[0].Path != "cex ticker" {
		t.Fatalf("leaves=%v", leaves)
	}
}

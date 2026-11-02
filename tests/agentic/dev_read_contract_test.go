//go:build agentic_e2e

package agentic_test

import (
	"strings"
	"testing"
)

// Only this finite reviewed GET command matrix may run against Dev.
var devReadCommands = []string{
	"cex account detail",
	"cex spot account list",
	"cex earn fixed products --page 1 --limit 1",
	"cex earn fixed lends --page 1 --limit 1",
	"cex earn dual plans",
	"cex earn dual orders --page 1 --limit 1",
	"cex earn dual balance",
	"cex earn staking find",
	"cex earn staking assets",
	"cex earn staking orders --page 1",
	"cex earn staking awards --page 1",
	"cex mcl ltv",
	"cex mcl currencies",
	"cex mcl orders --page 1 --limit 1",
	"cex earn uni currencies",
	"cex earn uni lends --page 1 --limit 1",
	"cex earn uni records --page 1 --limit 1",
	"cex earn uni interest-records --page 1 --limit 1",
	"cex earn fixed history --page 1 --limit 1",
}

func TestDevReadMatrixSafety(t *testing.T) {
	if len(devReadCommands) != 19 {
		t.Fatalf("matrix size %d", len(devReadCommands))
	}
	for _, c := range devReadCommands {
		if !strings.HasPrefix(c, "cex ") {
			t.Fatal(c)
		}
		for _, word := range []string{"launch ", "login", "refresh", "pledge ", "redeem ", "participate ", "create ", "cancel ", "claim ", "transfer "} {
			if strings.Contains(c, word) {
				t.Fatalf("mutation: %s", c)
			}
		}
	}
}

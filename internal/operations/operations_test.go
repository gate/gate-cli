package operations

import "testing"

// Diagnostics read the existing catalog by ID without publishing mutable
// pointers or choosing an arbitrary result when the catalog is ambiguous.
func TestOperationByID(t *testing.T) {
	for _, tc := range []struct{ id, method, leaf string }{
		{"createTrailOrder", "POST", "create"}, {"stopTrailOrder", "POST", "stop"}, {"stopAllTrailOrders", "POST", "stop_all"}, {"getTrailOrders", "GET", "list"}, {"getTrailOrderDetail", "GET", "detail"}, {"updateTrailOrder", "POST", "update"}, {"getTrailOrderChangeLog", "GET", "change_log"},
	} {
		op, err := LookupByID(tc.id)
		if err != nil {
			t.Fatal(err)
		}
		path := "/api/v4/futures/{settle}/autoorder/v1/trail/" + tc.leaf
		if op.Method != tc.method || op.Path != path {
			t.Fatalf("wrong catalog entry: %+v", op)
		}
		op.Path = "changed"
		again, err := LookupByID(tc.id)
		if err != nil || again.Path != path {
			t.Fatal("catalog was mutated")
		}
	}
	if _, err := LookupByID("missing"); err == nil {
		t.Fatal("missing ID must not return a usable operation")
	}
	dup := map[string]*Operation{"a": {OperationID: "duplicate"}, "b": {OperationID: "duplicate"}}
	if _, err := lookupOperationID("duplicate", dup, nil); err == nil {
		t.Fatal("ambiguous ID selected arbitrary map entry")
	}
}

func TestLookupExactPublic(t *testing.T) {
	op, err := Lookup("GET", "/api/v4/spot/currencies")
	if err != nil {
		t.Fatalf("Lookup err: %v", err)
	}
	if op == nil {
		t.Fatal("expected /api/v4/spot/currencies to be in catalog")
	}
	if op.AuthRequired {
		t.Errorf("expected /api/v4/spot/currencies to be public, got auth_required=true")
	}
}

func TestLookupTemplatePath(t *testing.T) {
	// /api/v4/spot/currencies/{currency} should match a concrete BTC path.
	op, err := Lookup("GET", "/api/v4/spot/currencies/BTC")
	if err != nil {
		t.Fatalf("Lookup err: %v", err)
	}
	if op == nil {
		t.Fatal("expected template lookup to match GET /api/v4/spot/currencies/BTC")
	}
	if op.AuthRequired {
		t.Errorf("expected /api/v4/spot/currencies/{currency} to be public")
	}
}

func TestIsPublicPrivateEndpoint(t *testing.T) {
	// /api/v4/spot/orders POST must be private (creating an order).
	if IsPublic("POST", "/api/v4/spot/orders") {
		t.Error("POST /api/v4/spot/orders must NOT be public")
	}
}

func TestIsPublicUnknownFailsClosed(t *testing.T) {
	// Unknown endpoints must fail closed (return false → caller will prompt to auth).
	if IsPublic("GET", "/api/v4/this/does/not/exist") {
		t.Error("unknown endpoint must fail closed (return false)")
	}
}

func TestCount(t *testing.T) {
	total, public, err := Count()
	if err != nil {
		t.Fatalf("Count err: %v", err)
	}
	if total == 0 {
		t.Fatal("expected non-zero operation count")
	}
	if public == 0 {
		t.Error("expected at least one public operation in catalog")
	}
	if public >= total {
		t.Errorf("public(%d) should be < total(%d)", public, total)
	}
}

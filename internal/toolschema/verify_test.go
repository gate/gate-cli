package toolschema

import "testing"

func TestValidateToolsWarnings(t *testing.T) {
	tools := []ToolSummary{
		{Name: "", HasInputSchema: true, InputSchema: map[string]interface{}{}},
		{Name: "t1", HasInputSchema: false},
		{Name: "t1", HasInputSchema: true, InputSchema: map[string]interface{}{
			"properties": map[string]interface{}{"a": map[string]interface{}{"type": "string"}},
			"required":   []interface{}{"b"},
		}},
	}
	report := ValidateTools("info", tools, true)
	if report.WarningCount == 0 {
		t.Fatal("expected warnings")
	}
	if report.Status != "warn" {
		t.Fatalf("expected status warn, got %s", report.Status)
	}
}

func TestValidateToolsTypeConsistencyWarnings(t *testing.T) {
	tools := []ToolSummary{
		{
			Name:           "t2",
			HasInputSchema: true,
			InputSchema: map[string]interface{}{
				"properties": map[string]interface{}{
					"limit": map[string]interface{}{
						"type":    "integer",
						"default": "10",
						"enum":    []interface{}{float64(1), "2"},
					},
				},
			},
		},
	}
	report := ValidateTools("info", tools, true)
	if report.WarningCount == 0 {
		t.Fatal("expected type consistency warnings")
	}
}

func TestValidateToolsStatusOK(t *testing.T) {
	tools := []ToolSummary{
		{
			Name:           "ok_tool",
			HasInputSchema: true,
			InputSchema: map[string]interface{}{
				"properties": map[string]interface{}{
					"query": map[string]interface{}{"type": "string", "default": "BTC"},
				},
			},
		},
	}
	report := ValidateTools("news", tools, true)
	if report.WarningCount != 0 {
		t.Fatalf("expected no warnings, got %d", report.WarningCount)
	}
	if report.Status != "ok" {
		t.Fatalf("expected status ok, got %s", report.Status)
	}
}

func TestValidateToolsIntegerDefaultIntLiteralOK(t *testing.T) {
	tools := []ToolSummary{
		{
			Name:           "bounded",
			HasInputSchema: true,
			InputSchema: map[string]interface{}{
				"properties": map[string]interface{}{
					"limit": map[string]interface{}{
						"type":    "integer",
						"default": 10,
						"maximum": 50,
					},
				},
			},
		},
	}
	report := ValidateTools("news", tools, true)
	if report.WarningCount != 0 {
		t.Fatalf("expected no warnings, got %+v", report.Warnings)
	}
	if report.Status != "ok" {
		t.Fatalf("expected status ok, got %s", report.Status)
	}
}

func TestVerifyReportStrictFieldsDefaultFalse(t *testing.T) {
	report := ValidateTools("info", nil, true)
	if report.StrictMode {
		t.Fatal("expected strict_mode default false")
	}
	if report.StrictFailed {
		t.Fatal("expected strict_failed default false")
	}
}

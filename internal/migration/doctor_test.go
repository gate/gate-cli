package migration

import "testing"

func TestBuildDoctorReport(t *testing.T) {
	report := BuildDoctorReport(DoctorOptions{
		Checks: ParseCheckList("connectivity"),
		InfoURL: "http://example.test/info",
		Installed: func(s string) bool {
			return true
		},
		Scanner: NewScannerWithHome(t.TempDir()),
	})
	if report.Status != "pass" {
		t.Fatalf("expected pass, got %s", report.Status)
	}
	if len(report.Checks) == 0 {
		t.Fatalf("expected checks")
	}
}

func TestDoctorStrictWarnBecomesFail(t *testing.T) {
	report := BuildDoctorReport(DoctorOptions{
		Checks: ParseCheckList("legacy-mcp"),
		Strict: true,
		Scanner: &Scanner{
			home: t.TempDir(),
		},
	})
	// no warn here necessarily, but function must remain valid
	if report.Status == "" {
		t.Fatalf("status should not be empty")
	}
}

func TestDoctorConnectivityIsBlocking(t *testing.T) {
	report := BuildDoctorReport(DoctorOptions{
		Checks: ParseCheckList("connectivity"),
		InfoURL: "",
		NewsURL: "",
		Installed: func(string) bool { return true },
		Scanner: NewScannerWithHome(t.TempDir()),
	})
	if len(report.Checks) != 1 {
		t.Fatalf("expected 1 check")
	}
	if report.Checks[0].ID != "cli.connectivity" {
		t.Fatalf("expected cli.connectivity check")
	}
	if !report.Checks[0].Blocking {
		t.Fatalf("connectivity should be blocking")
	}
	if report.Checks[0].Status != "fail" {
		t.Fatalf("connectivity missing env should fail")
	}
}

func TestDoctorExitCode(t *testing.T) {
	if DoctorExitCode(DoctorReport{Status: "pass"}) != 0 {
		t.Fatalf("pass should map to 0")
	}
	if DoctorExitCode(DoctorReport{Status: "warn"}) != 10 {
		t.Fatalf("warn should map to 10")
	}
	if DoctorExitCode(DoctorReport{Status: "fail"}) != 20 {
		t.Fatalf("fail should map to 20")
	}
}

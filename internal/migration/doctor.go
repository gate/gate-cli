package migration

import (
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/gate/gate-cli/internal/config"
	"github.com/gate/gate-cli/internal/version"
)

const MinDoctorVersion = "0.3.0"

type DoctorCheck struct {
	ID       string                 `json:"id"`
	Status   string                 `json:"status"`
	Blocking bool                   `json:"blocking"`
	Message  string                 `json:"message"`
	Detail   map[string]interface{} `json:"detail,omitempty"`
}

type DoctorAction struct {
	ActionCode string `json:"action_code"`
	Command    string `json:"command"`
	Reason     string `json:"reason"`
}

type DoctorSummary struct {
	CLIInstalled           bool     `json:"cli_installed"`
	CLIVersion             string   `json:"cli_version"`
	MinimumRequiredVersion string   `json:"minimum_required_version"`
	LegacyMCPDetected      bool     `json:"legacy_mcp_detected"`
	ProvidersAffected      []string `json:"providers_affected"`
}

type DoctorReport struct {
	Status             string         `json:"status"`
	Summary            DoctorSummary  `json:"summary"`
	Checks             []DoctorCheck  `json:"checks"`
	RecommendedActions []DoctorAction `json:"recommended_actions"`
}

type DoctorOptions struct {
	Profile   string
	Checks    map[string]struct{}
	Strict    bool
	InfoURL   string
	NewsURL   string
	Scanner   *Scanner
	Installed func(string) bool
}

func BuildDoctorReport(opts DoctorOptions) DoctorReport {
	if opts.Scanner == nil {
		opts.Scanner = NewScanner()
	}
	if opts.Installed == nil {
		opts.Installed = isInstalled
	}
	selected := normalizeChecks(opts.Checks)

	out := DoctorReport{
		Summary: DoctorSummary{
			CLIVersion:             version.Version,
			MinimumRequiredVersion: MinDoctorVersion,
			ProvidersAffected:      []string{},
		},
		Checks:             []DoctorCheck{},
		RecommendedActions: []DoctorAction{},
	}

	if has(selected, "cli.binary") {
		ok := opts.Installed("gate-cli")
		out.Summary.CLIInstalled = ok
		out.Checks = append(out.Checks, DoctorCheck{
			ID:       "cli.binary",
			Status:   passFail(ok),
			Blocking: true,
			Message:  ternary(ok, "gate-cli found in PATH", "gate-cli not found in PATH"),
		})
	}

	if has(selected, "cli.version") {
		ok := compareVersion(version.Version, MinDoctorVersion) >= 0
		out.Checks = append(out.Checks, DoctorCheck{
			ID:       "cli.version",
			Status:   passFail(ok),
			Blocking: true,
			Message:  ternary(ok, "cli version meets minimum requirement", "cli version below minimum requirement"),
			Detail: map[string]interface{}{
				"current": version.Version,
				"min":     MinDoctorVersion,
			},
		})
	}

	if has(selected, "cli.config") {
		_, err := config.Load(config.Options{Profile: opts.Profile})
		ok := err == nil
		out.Checks = append(out.Checks, DoctorCheck{
			ID:       "cli.config",
			Status:   passFail(ok),
			Blocking: true,
			Message:  ternary(ok, "config loaded", "config load failed"),
		})
	}

	if has(selected, "cli.connectivity") {
		ok := strings.TrimSpace(opts.InfoURL) != "" || strings.TrimSpace(opts.NewsURL) != ""
		out.Checks = append(out.Checks, DoctorCheck{
			ID:       "cli.connectivity",
			Status:   passFail(ok),
			Blocking: true,
			Message:  ternary(ok, "intel endpoint env configured", "intel endpoint env not configured"),
		})
	}

	if has(selected, "legacy_mcp") {
		for _, s := range opts.Scanner.Scan(nil) {
			warn := s.EntriesFound > 0
			if warn {
				out.Summary.LegacyMCPDetected = true
				out.Summary.ProvidersAffected = append(out.Summary.ProvidersAffected, s.ProviderID)
			}
			out.Checks = append(out.Checks, DoctorCheck{
				ID:       "legacy_mcp." + s.ProviderID,
				Status:   ternary(warn, "warn", "pass"),
				Blocking: false,
				Message:  ternary(warn, "legacy Gate MCP entries detected", "no legacy Gate MCP entries"),
				Detail: map[string]interface{}{
					"entries_found": s.EntriesFound,
					"files_checked": s.FilesChecked,
				},
			})
		}
		sort.Strings(out.Summary.ProvidersAffected)
	}

	blockingFail := false
	hasWarn := false
	for _, c := range out.Checks {
		if c.Blocking && c.Status == "fail" {
			blockingFail = true
		}
		if c.Status == "warn" {
			hasWarn = true
		}
	}
	out.Status = "pass"
	if blockingFail || (opts.Strict && hasWarn) {
		out.Status = "fail"
	} else if hasWarn {
		out.Status = "warn"
	}

	if out.Summary.LegacyMCPDetected {
		out.RecommendedActions = append(out.RecommendedActions, DoctorAction{
			ActionCode: "RUN_MIGRATE",
			Command:    "gate-cli migrate --dry-run",
			Reason:     "legacy Gate MCP entries detected",
		})
	}
	if !out.Summary.CLIInstalled {
		out.RecommendedActions = append(out.RecommendedActions, DoctorAction{
			ActionCode: "INSTALL_CLI",
			Command:    "go install .",
			Reason:     "gate-cli not found in PATH",
		})
	}
	return out
}

func DoctorExitCode(report DoctorReport) int {
	switch report.Status {
	case "pass":
		return 0
	case "warn":
		return 10
	default:
		return 20
	}
}

func ParseCheckList(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	parts := strings.Split(strings.TrimSpace(raw), ",")
	for _, p := range parts {
		v := strings.ToLower(strings.TrimSpace(p))
		if v == "" {
			continue
		}
		out[v] = struct{}{}
	}
	return out
}

func normalizeChecks(in map[string]struct{}) map[string]struct{} {
	if len(in) == 0 {
		return map[string]struct{}{
			"cli.binary":     {},
			"cli.version":    {},
			"cli.config":     {},
			"cli.connectivity": {},
			"legacy_mcp":     {},
		}
	}
	if _, ok := in["all"]; ok {
		return map[string]struct{}{
			"cli.binary":     {},
			"cli.version":    {},
			"cli.config":     {},
			"cli.connectivity": {},
			"legacy_mcp":     {},
		}
	}
	out := map[string]struct{}{}
	for k := range in {
		switch k {
		case "cli":
			out["cli.binary"] = struct{}{}
			out["cli.version"] = struct{}{}
			out["cli.config"] = struct{}{}
			out["cli.connectivity"] = struct{}{}
		case "version":
			out["cli.version"] = struct{}{}
		case "config":
			out["cli.config"] = struct{}{}
		case "connectivity":
			out["cli.connectivity"] = struct{}{}
		case "legacy-mcp":
			out["legacy_mcp"] = struct{}{}
		}
	}
	return out
}

func has(set map[string]struct{}, key string) bool {
	_, ok := set[key]
	return ok
}

func passFail(ok bool) string {
	if ok {
		return "pass"
	}
	return "fail"
}

func isInstalled(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

func compareVersion(v, min string) int {
	parse := func(s string) []int {
		s = strings.TrimPrefix(strings.TrimSpace(s), "v")
		parts := strings.Split(s, ".")
		out := make([]int, 0, 3)
		for _, p := range parts {
			n, _ := strconv.Atoi(p)
			out = append(out, n)
		}
		for len(out) < 3 {
			out = append(out, 0)
		}
		return out[:3]
	}
	a, b := parse(v), parse(min)
	for i := 0; i < 3; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

func ternary[T any](ok bool, yes, no T) T {
	if ok {
		return yes
	}
	return no
}

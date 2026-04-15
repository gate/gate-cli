package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type MigrateProviderResult struct {
	ProviderID      string  `json:"provider_id"`
	FilePath        string  `json:"file_path"`
	DetectedEntries []Entry `json:"detected_entries"`
	Action          string  `json:"action"`
	BackupFile      string  `json:"backup_file"`
	Status          string  `json:"status"`
	ManualPatch     string  `json:"manual_patch"`
}

type MigrateReport struct {
	Mode                string                 `json:"mode"`
	Status              string                 `json:"status"`
	Providers           []MigrateProviderResult `json:"providers"`
	RecommendedNextStep string                 `json:"recommended_next_step"`
}

type MigrateOptions struct {
	Apply      bool
	ProviderIDs []string
	BackupDir  string
	Scanner    *Scanner
}

func RunMigrate(opts MigrateOptions) (MigrateReport, error) {
	if opts.Scanner == nil {
		opts.Scanner = NewScanner()
	}
	if strings.TrimSpace(opts.BackupDir) == "" {
		opts.BackupDir = filepath.Join(os.TempDir(), "gate-cli-migrate-backups")
	}
	mode := "dry_run"
	if opts.Apply {
		mode = "apply"
	}

	if err := os.MkdirAll(opts.BackupDir, 0o755); err != nil {
		return MigrateReport{}, err
	}

	scans := opts.Scanner.Scan(opts.ProviderIDs)
	results := make([]MigrateProviderResult, 0, len(scans))
	anyFail := false
	anyWarn := false

	for _, s := range scans {
		result := MigrateProviderResult{
			ProviderID:      s.ProviderID,
			DetectedEntries: s.Entries,
			Status:          "pass",
			Action:          "none",
		}
		if len(s.FilesChecked) > 0 {
			result.FilePath = s.FilesChecked[0]
		}
		if s.EntriesFound == 0 {
			results = append(results, result)
			continue
		}

		anyWarn = true
		result.Status = "warn"
		result.Action = "manual_patch"
		result.ManualPatch = "remove Gate MCP entries from provider config"

		target := pickExisting(s.FilesChecked)
		if target == "" {
			results = append(results, result)
			continue
		}
		result.FilePath = target

		if opts.Apply {
			bak, err := backupFile(target, opts.BackupDir)
			if err != nil {
				result.Status = "fail"
				anyFail = true
				result.ManualPatch = "failed to backup file: " + err.Error()
				results = append(results, result)
				continue
			}
			result.BackupFile = bak
			updated, changed, err := removeGateMarkers(target)
			if err != nil {
				result.Status = "fail"
				anyFail = true
				result.ManualPatch = "failed to modify file: " + err.Error()
				results = append(results, result)
				continue
			}
			if changed {
				if err := os.WriteFile(target, []byte(updated), 0o644); err != nil {
					result.Status = "fail"
					anyFail = true
					result.ManualPatch = "failed to write modified file: " + err.Error()
				} else {
					result.Status = "pass"
					result.Action = "comment"
					result.ManualPatch = ""
				}
			} else {
				// Keep JSON and other non-comment-safe formats in manual patch mode.
				result.Action = "manual_patch"
				result.Status = "warn"
				result.ManualPatch = "auto-change not safe for this file format; apply manual patch"
			}
		}
		results = append(results, result)
	}

	report := MigrateReport{
		Mode:                mode,
		Status:              "pass",
		Providers:           results,
		RecommendedNextStep: "run_skill_again",
	}
	if anyFail {
		report.Status = "fail"
		report.RecommendedNextStep = "manual_edit_required"
	} else if anyWarn {
		report.Status = "warn"
		report.RecommendedNextStep = "rerun_doctor"
	}
	sort.Slice(report.Providers, func(i, j int) bool {
		return report.Providers[i].ProviderID < report.Providers[j].ProviderID
	})
	return report, nil
}

func pickExisting(paths []string) string {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func backupFile(src, backupDir string) (string, error) {
	data, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	name := filepath.Base(src) + "." + time.Now().Format("20060102-150405") + ".bak"
	dst := filepath.Join(backupDir, name)
	return dst, os.WriteFile(dst, data, 0o644)
}

func removeGateMarkers(path string) (string, bool, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".json" {
		return "", false, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}
	lines := strings.Split(string(data), "\n")
	changed := false
	for i, line := range lines {
		lc := strings.ToLower(line)
		for _, marker := range gateMarkers {
			if strings.Contains(lc, marker) && !strings.HasPrefix(strings.TrimSpace(line), "#") && !strings.HasPrefix(strings.TrimSpace(line), "//") {
				lines[i] = "# " + line
				changed = true
				break
			}
		}
	}
	return strings.Join(lines, "\n"), changed, nil
}

func MigrateExitCode(report MigrateReport) int {
	switch report.Status {
	case "pass":
		return 0
	case "warn":
		return 10
	default:
		return 20
	}
}

func ParseProviders(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.ToLower(strings.TrimSpace(p))
		if v == "" {
			continue
		}
		out = append(out, v)
	}
	return out
}

func ValidateMode(apply bool, dryRun bool) error {
	if apply && dryRun {
		return fmt.Errorf("cannot set both --apply and --dry-run")
	}
	return nil
}

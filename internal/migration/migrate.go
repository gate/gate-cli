package migration

import (
	"fmt"
	"io"
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
	Mode                string                  `json:"mode"`
	Status              string                  `json:"status"`
	Providers           []MigrateProviderResult `json:"providers"`
	RecommendedNextStep string                  `json:"recommended_next_step"`
}

type MigrateOptions struct {
	Apply       bool
	ProviderIDs []string
	BackupDir   string
	Scanner     *Scanner
}

func RunMigrate(opts MigrateOptions) (MigrateReport, error) {
	if opts.Scanner == nil {
		opts.Scanner = NewScanner()
	}
	if strings.TrimSpace(opts.BackupDir) == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return MigrateReport{}, err
		}
		opts.BackupDir = filepath.Join(home, ".gate-cli", "migrate-backups")
	}
	mode := "dry_run"
	if opts.Apply {
		mode = "apply"
	}

	if err := os.MkdirAll(opts.BackupDir, 0o700); err != nil {
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
				if err := atomicWritePreservePerm(target, []byte(updated)); err != nil {
					result.Status = "fail"
					anyFail = true
					result.ManualPatch = "failed to write modified file: " + err.Error()
				} else {
					result.Status = "pass"
					result.Action = "comment"
					result.ManualPatch = ""
				}
			} else {
				// No safe structural change was produced for this file; keep manual patch mode.
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
	f, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("backup source is not a regular file: %s", src)
	}
	const maxBackup = 1 << 20
	data, err := io.ReadAll(io.LimitReader(f, maxBackup+1))
	if err != nil {
		return "", err
	}
	if len(data) > maxBackup {
		return "", fmt.Errorf("source file exceeds %d bytes: %s", maxBackup, src)
	}
	name := filepath.Base(src) + "." + time.Now().Format("20060102-150405") + ".bak"
	dst := filepath.Join(backupDir, name)
	return dst, os.WriteFile(dst, data, info.Mode().Perm())
}

func removeGateMarkers(path string) (string, bool, error) {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return removeGateMarkersJSON(path)
	case ".toml":
		return removeGateMarkersTOML(path)
	}
	return "", false, nil
}

func removeGateMarkersJSON(path string) (string, bool, error) {
	// Keep JSON in manual patch mode to avoid rewriting user formatting
	// (indentation/key-order/newline style).
	_, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}
	return "", false, nil
}

func removeGateMarkersTOML(path string) (string, bool, error) {
	// Keep TOML in manual patch mode: structural unmarshal/marshal drops comments
	// and key order, which is too destructive for user-managed config.
	_, err := os.ReadFile(path)
	if err != nil {
		return "", false, err
	}
	return "", false, nil
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

func atomicWritePreservePerm(path string, data []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmp)
		}
	}()

	if err := f.Chmod(info.Mode().Perm()); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	cleanup = false

	// Best-effort directory sync to reduce rename durability risk.
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
	}
	return nil
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

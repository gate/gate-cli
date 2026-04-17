package migration

const (
	PreflightMsgMigrateHint = "检测到本地仍加载 Gate MCP，建议运行 gate-cli migrate"
	PreflightMsgInstallHint = "当前环境未安装 gate-cli。请先安装: go install ."
	PreflightMsgDoctorHint  = "环境检查未通过，请运行 gate-cli doctor"
)

type PreflightProvider struct {
	ProviderID   string   `json:"provider_id"`
	FilesChecked []string `json:"files_checked"`
	EntriesFound int      `json:"entries_found"`
}

type PreflightLegacyEntry struct {
	ProviderID string `json:"provider_id"`
	FilePath   string `json:"file_path"`
	ServerName string `json:"server_name"`
}

type PreflightResult struct {
	Status            string                 `json:"status"`
	Route             string                 `json:"route"`
	ActionCode        string                 `json:"action_code"`
	CLIInstalled      bool                   `json:"cli_installed"`
	CLIVersion        string                 `json:"cli_version"`
	VersionOK         bool                   `json:"version_ok"`
	LegacyMCPDetected bool                   `json:"legacy_mcp_detected"`
	LegacyMCPEntries  []PreflightLegacyEntry `json:"legacy_mcp_entries"`
	ProvidersScanned  []PreflightProvider    `json:"providers_scanned"`
	FallbackEnabled   bool                   `json:"fallback_enabled"`
	BlockingReason    string                 `json:"blocking_reason"`
	UserMessage       string                 `json:"user_message"`
}

type PreflightOptions struct {
	FallbackEnabled bool
	Scanner         *Scanner
	Installed       func(string) bool
	Version         string
}

func BuildPreflight(opts PreflightOptions) PreflightResult {
	if opts.Scanner == nil {
		opts.Scanner = NewScanner()
	}
	if opts.Installed == nil {
		opts.Installed = isInstalled
	}
	if opts.Version == "" {
		opts.Version = "0.0.0"
	}

	scans := opts.Scanner.Scan(nil)
	res := PreflightResult{
		CLIInstalled:      opts.Installed("gate-cli"),
		CLIVersion:        opts.Version,
		VersionOK:         compareVersion(opts.Version, MinDoctorVersion) >= 0,
		LegacyMCPDetected: false,
		LegacyMCPEntries:  []PreflightLegacyEntry{},
		ProvidersScanned:  []PreflightProvider{},
		FallbackEnabled:   opts.FallbackEnabled,
	}

	for _, s := range scans {
		res.ProvidersScanned = append(res.ProvidersScanned, PreflightProvider{
			ProviderID:   s.ProviderID,
			FilesChecked: s.FilesChecked,
			EntriesFound: s.EntriesFound,
		})
		for _, e := range s.Entries {
			res.LegacyMCPDetected = true
			res.LegacyMCPEntries = append(res.LegacyMCPEntries, PreflightLegacyEntry{
				ProviderID: s.ProviderID,
				FilePath:   e.FilePath,
				ServerName: e.ServerName,
			})
		}
	}

	switch {
	case res.CLIInstalled && !res.VersionOK:
		res.Status = "run_doctor_required"
		res.Route = "BLOCK"
		res.ActionCode = "SHOW_DOCTOR_HINT"
		res.BlockingReason = "cli_version_not_supported"
		res.UserMessage = preflightMessageForAction(res.ActionCode)
	case res.CLIInstalled && res.VersionOK && !res.LegacyMCPDetected:
		res.Status = "ready"
		res.Route = "CLI"
		res.ActionCode = "NONE"
		res.UserMessage = preflightMessageForAction(res.ActionCode)
	case res.CLIInstalled && res.VersionOK && res.LegacyMCPDetected:
		res.Status = "ready_with_migration_warning"
		res.Route = "CLI"
		res.ActionCode = "SHOW_MIGRATE_HINT"
		res.UserMessage = preflightMessageForAction(res.ActionCode)
	case !res.CLIInstalled && res.LegacyMCPDetected && res.FallbackEnabled:
		res.Status = "fallback_to_mcp"
		res.Route = "MCP_FALLBACK"
		res.ActionCode = "SHOW_INSTALL_HINT"
		res.UserMessage = preflightMessageForAction(res.ActionCode)
	default:
		res.Status = "install_cli_required"
		res.Route = "BLOCK"
		res.ActionCode = "SHOW_INSTALL_HINT"
		res.BlockingReason = "cli_not_installed"
		res.UserMessage = preflightMessageForAction(res.ActionCode)
	}
	return res
}

func preflightMessageForAction(actionCode string) string {
	switch actionCode {
	case "SHOW_MIGRATE_HINT":
		return PreflightMsgMigrateHint
	case "SHOW_INSTALL_HINT":
		return PreflightMsgInstallHint
	case "SHOW_DOCTOR_HINT":
		return PreflightMsgDoctorHint
	default:
		return ""
	}
}

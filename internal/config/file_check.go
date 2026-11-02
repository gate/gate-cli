package config

import (
	"os"
	"runtime"
)

// FileCheck reports a local metadata snapshot, not YAML correctness, ACL safety
// or API authorization. It never reads credential contents or repairs files.
type FileCheck struct {
	SchemaVersion int      `json:"schema_version"`
	Scope         string   `json:"scope"`
	Status        string   `json:"status"`
	Checks        []string `json:"checks"`
	Remediation   string   `json:"remediation"`
}

// CheckFile belongs to the existing config dependency boundary. Special files
// and symlinks are not opened. The configured directory is a trusted local path;
// this check is not protection against adversarial replacement during the check.
func CheckFile(path string) FileCheck {
	d := FileCheck{SchemaVersion: 1, Scope: "file", Status: "pass", Checks: []string{}}
	fail := func(message string) FileCheck {
		d.Status = "fail"
		d.Checks = append(d.Checks, message)
		d.Remediation = "check the config file path, type and read permissions; no changes were made"
		return d
	}
	info, err := os.Lstat(path)
	if err != nil {
		return fail("config file is missing or its metadata is unreadable")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		d.Status = "warn"
		d.Checks = append(d.Checks, "symlink target was not evaluated")
		d.Remediation = "review the symlink target and its permissions explicitly"
		return d
	}
	if !info.Mode().IsRegular() {
		return fail("config path is not a regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return fail("config file is not readable")
	}
	if err = f.Close(); err != nil {
		return fail("config file could not be closed")
	}
	d.Checks = append(d.Checks, "regular file is readable; contents were not inspected")
	if runtime.GOOS == "windows" {
		d.Status = "warn"
		d.Checks = append(d.Checks, "Windows ACL was not evaluated")
		d.Remediation = "review file access permissions with Windows security tools"
	} else if info.Mode().Perm()&0077 != 0 {
		d.Status = "warn"
		d.Checks = append(d.Checks, "group or other permission bits are set")
		d.Remediation = "review ownership and restrict group/other permissions if appropriate"
	}
	return d
}

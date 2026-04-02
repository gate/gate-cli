# gate-cli Install & Distribution Design

**Date:** 2026-04-02
**Status:** Approved

## Overview

Add one-line install support for gate-cli across two distribution channels: shell scripts (Unix + Windows) and Homebrew. Binaries continue to be built and published to GitHub Releases by the existing goreleaser + GitHub Actions pipeline.

## Goals

- Users can install gate-cli with a single command on any supported platform
- No new build infrastructure required beyond what goreleaser already produces
- Version pinning is possible in all channels for CI/CD reproducibility
- Checksum verification on all script-based installs

## Distribution Channels

### 1. Shell Scripts

Two scripts at the repo root, served via raw GitHub URLs.

#### `install.sh` (macOS + Linux)

- Detects OS (`uname -s`) and arch (`uname -m`), maps to goreleaser archive naming convention (e.g., `linux_amd64`, `darwin_arm64`)
- Fetches latest release tag from GitHub API by default
- `--version vX.Y.Z` flag overrides to a specific release
- Downloads `gate-cli_<version>_<os>_<arch>.tar.gz` and `checksums.txt` from GitHub Releases
  - The `v` prefix is stripped from the tag before constructing the archive filename (e.g. tag `v0.3.1` → filename `gate-cli_0.3.1_linux_amd64.tar.gz`)
- Verifies SHA256 checksum before extracting
- Install directory auto-detection:
  1. Try `~/.local/bin` — attempt to write there; if write fails (permission or missing dir), fall back
  2. Fall back to `/usr/local/bin` with sudo
- Prints a `$PATH` hint if installed to `~/.local/bin` and that directory is not yet in `$PATH`

**One-liner:**
```sh
# Latest
curl -fsSL https://raw.githubusercontent.com/gate/gate-cli/main/install.sh | sh

# Specific version
curl -fsSL https://raw.githubusercontent.com/gate/gate-cli/main/install.sh | sh -s -- --version v0.3.1
```

#### `install.ps1` (Windows PowerShell)

- Detects architecture via `$env:PROCESSOR_ARCHITECTURE`
- Fetches latest release tag from GitHub API by default
- `$env:GATE_CLI_VERSION` env var overrides to a specific release
- Downloads `gate-cli_<version>_windows_amd64.zip` from GitHub Releases (only amd64 is built for Windows)
  - The `v` prefix is stripped from the tag before constructing the archive filename (e.g. tag `v0.3.1` → `gate-cli_0.3.1_windows_amd64.zip`)
- Verifies SHA256 hash with `Get-FileHash` before extracting
- Installs binary to `$env:LOCALAPPDATA\gate-cli`
- Adds install directory to user `PATH` via registry if not already present

**One-liner:**
```powershell
# Latest
irm https://raw.githubusercontent.com/gate/gate-cli/main/install.ps1 | iex

# Specific version
$env:GATE_CLI_VERSION="v0.3.1"; irm https://raw.githubusercontent.com/gate/gate-cli/main/install.ps1 | iex
```

---

### 2. Homebrew Tap (goreleaser)

goreleaser generates the Homebrew formula automatically from the release archives and pushes it to a dedicated tap repository: **`gate/homebrew-tap`**.

**`.goreleaser.yaml` addition:**
```yaml
brews:
  - name: gate-cli
    repository:
      owner: gate
      name: homebrew-tap
      branch: main
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    homepage: "https://www.gate.com"
    description: "Gate CLI - command-line interface for Gate"
    license: "MIT"
    install: |
      bin.install "gate-cli"
    test: |
      system bin/"gate-cli", "--version"
```

goreleaser computes the SHA256 of the darwin and linux tarballs, renders `Formula/gate-cli.rb` with the correct URLs and hashes, and pushes a commit to `gate/homebrew-tap` on every release.

**Prerequisites:**
1. Create the public `gate/homebrew-tap` repository and initialize it with at least one commit (e.g., a README) so the `main` branch exists before the first release runs.
2. Add a `HOMEBREW_TAP_TOKEN` GitHub secret (fine-grained PAT with `contents: write` on `gate/homebrew-tap`) to the `gate/gate-cli` repo.

**User experience:**
```sh
brew install gate/tap/gate-cli

# or explicit tap first:
brew tap gate/tap
brew install gate-cli
```

---

## Release Pipeline Changes

### `.github/workflows/release.yaml`

No changes required. goreleaser reads `HOMEBREW_TAP_TOKEN` from the environment automatically when the `brews` section is present in `.goreleaser.yaml`.

**New secret required:**
| Secret | Used by | Purpose |
|---|---|---|
| `HOMEBREW_TAP_TOKEN` | goreleaser | Push formula to `gate/homebrew-tap` |

---

## Files Added / Modified

| File | Change |
|---|---|
| `install.sh` | New — Unix install script |
| `install.ps1` | New — Windows PowerShell install script |
| `.goreleaser.yaml` | Add `brews` section |
| `docs/quickstart.md` | Update Installation section to list all install methods: shell script, Homebrew, and build from source |

---

## Out of Scope

- npm distribution — `gate-cli` package name is taken on the public registry; deferred
- Scoop (Windows package manager) — can be added later
- apt/yum packages — goreleaser supports `.deb`/`.rpm` but not in this iteration
- Signed binaries / notarization — not in scope

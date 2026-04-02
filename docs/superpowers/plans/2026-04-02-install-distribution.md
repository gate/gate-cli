# Install Distribution Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add one-line install support for gate-cli via a Unix shell script, a Windows PowerShell script, and a Homebrew tap driven by goreleaser.

**Architecture:** Two standalone install scripts (`install.sh`, `install.ps1`) at the repo root fetch the correct release archive from GitHub Releases, verify its checksum, and place the binary on the user's PATH. Homebrew distribution is handled entirely by goreleaser's `brews` config, which auto-generates and pushes `Formula/gate-cli.rb` to `gate/homebrew-tap` on every release tag.

**Tech Stack:** bash, PowerShell, goreleaser v2 (`brews`), GitHub Releases API

---

## File Map

| File | Change |
|---|---|
| `install.sh` | New — Unix install script (macOS + Linux) |
| `install.ps1` | New — Windows PowerShell install script |
| `.goreleaser.yaml` | Modify — add `brews` section |
| `docs/quickstart.md` | Modify — update Installation section |

---

## Task 1: Write `install.sh`

**Files:**
- Create: `install.sh`

Background: goreleaser archive names follow `gate-cli_<version>_<os>_<arch>.tar.gz` where `<version>` has no `v` prefix (e.g. `0.3.2`), `<os>` is `linux` or `darwin`, and `<arch>` is `amd64` or `arm64`. The GitHub release tag includes the `v` prefix (e.g. `v0.3.2`), so it must be stripped when constructing the filename.

- [ ] **Step 1: Create `install.sh`**

```bash
#!/bin/sh
set -e

REPO="gate/gate-cli"
BINARY="gate-cli"

# --- Parse flags ---
VERSION=""
while [ $# -gt 0 ]; do
  case "$1" in
    --version)
      VERSION="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

# --- Detect OS and arch ---
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

case "$OS" in
  linux|darwin) ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# --- Resolve version ---
if [ -z "$VERSION" ]; then
  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
fi

# Strip leading 'v' for the archive filename
BARE_VERSION="${VERSION#v}"
ARCHIVE="${BINARY}_${BARE_VERSION}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

# --- Download ---
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

echo "Downloading ${ARCHIVE}..."
curl -fsSL "${BASE_URL}/${ARCHIVE}" -o "${TMP}/${ARCHIVE}"
curl -fsSL "${BASE_URL}/checksums.txt" -o "${TMP}/checksums.txt"

# --- Verify checksum ---
echo "Verifying checksum..."
cd "$TMP"
if command -v sha256sum > /dev/null 2>&1; then
  grep "${ARCHIVE}" checksums.txt | sha256sum --check --status
elif command -v shasum > /dev/null 2>&1; then
  grep "${ARCHIVE}" checksums.txt | shasum -a 256 --check --status
else
  echo "Warning: no sha256sum or shasum found, skipping checksum verification" >&2
fi
cd - > /dev/null

# --- Extract ---
tar -xzf "${TMP}/${ARCHIVE}" -C "$TMP" "${BINARY}"

# --- Install ---
install_bin() {
  local dir="$1"
  local use_sudo="$2"
  if [ "$use_sudo" = "true" ]; then
    sudo install -m 755 "${TMP}/${BINARY}" "${dir}/${BINARY}"
  else
    install -m 755 "${TMP}/${BINARY}" "${dir}/${BINARY}"
  fi
}

LOCAL_BIN="${HOME}/.local/bin"
mkdir -p "$LOCAL_BIN" 2>/dev/null || true

if install_bin "$LOCAL_BIN" "false" 2>/dev/null; then
  INSTALL_DIR="$LOCAL_BIN"
  # Check if it's on PATH
  case ":$PATH:" in
    *":${LOCAL_BIN}:"*) ;;
    *)
      echo ""
      echo "Installed to ${LOCAL_BIN}/${BINARY}"
      echo "Add the following to your shell profile to use it:"
      echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
      ;;
  esac
else
  SYSTEM_BIN="/usr/local/bin"
  install_bin "$SYSTEM_BIN" "true"
  INSTALL_DIR="$SYSTEM_BIN"
fi

echo ""
echo "gate-cli ${VERSION} installed to ${INSTALL_DIR}/${BINARY}"
echo "Run: gate-cli --version"
```

- [ ] **Step 2: Make executable**

```bash
chmod +x install.sh
```

- [ ] **Step 3: Smoke test against latest release**

```bash
sh install.sh
```

Expected: downloads, verifies checksum, installs binary, prints success message.

```bash
gate-cli --version
```

Expected: `gate-cli version v0.3.2` (or whatever the latest release is)

- [ ] **Step 4: Smoke test version pinning**

```bash
sh install.sh --version v0.3.2
```

Expected: downloads `gate-cli_0.3.2_<os>_<arch>.tar.gz`, installs successfully.

- [ ] **Step 5: Commit**

```bash
git add install.sh
git commit -m "feat: add install.sh for Unix one-line install"
```

---

## Task 2: Write `install.ps1`

**Files:**
- Create: `install.ps1`

Background: Windows only has amd64 builds. Archive format is `.zip`. Version is taken from `$env:GATE_CLI_VERSION` if set, otherwise latest from GitHub API. The `v` prefix must be stripped for the archive filename.

- [ ] **Step 1: Create `install.ps1`**

```powershell
$ErrorActionPreference = 'Stop'

$Repo = "gate/gate-cli"
$Binary = "gate-cli.exe"
$InstallDir = "$env:LOCALAPPDATA\gate-cli"

# --- Resolve version ---
$Version = $env:GATE_CLI_VERSION
if (-not $Version) {
    $release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $release.tag_name
}

# Strip leading 'v' for the archive filename
$BareVersion = $Version.TrimStart('v')
$Archive = "gate-cli_${BareVersion}_windows_amd64.zip"
$BaseUrl = "https://github.com/$Repo/releases/download/$Version"

# --- Download ---
$Tmp = New-TemporaryFile | ForEach-Object { $_.DirectoryName + "\" + [System.IO.Path]::GetRandomFileName() }
New-Item -ItemType Directory -Path $Tmp | Out-Null

try {
    Write-Host "Downloading $Archive..."
    Invoke-WebRequest "$BaseUrl/$Archive" -OutFile "$Tmp\$Archive"
    Invoke-WebRequest "$BaseUrl/checksums.txt" -OutFile "$Tmp\checksums.txt"

    # --- Verify checksum ---
    Write-Host "Verifying checksum..."
    $Expected = (Get-Content "$Tmp\checksums.txt" | Where-Object { $_ -match $Archive }) -split '\s+' | Select-Object -First 1
    $Actual = (Get-FileHash "$Tmp\$Archive" -Algorithm SHA256).Hash.ToLower()
    if ($Actual -ne $Expected) {
        Write-Error "Checksum mismatch: expected $Expected, got $Actual"
        exit 1
    }

    # --- Extract ---
    Expand-Archive "$Tmp\$Archive" -DestinationPath $Tmp -Force

    # --- Install ---
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item "$Tmp\gate-cli.exe" "$InstallDir\gate-cli.exe" -Force

    # --- Add to PATH ---
    $UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($UserPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("PATH", "$UserPath;$InstallDir", "User")
        Write-Host ""
        Write-Host "Added $InstallDir to your PATH."
        Write-Host "Restart your terminal for the change to take effect."
    }

    Write-Host ""
    Write-Host "gate-cli $Version installed to $InstallDir\gate-cli.exe"
    Write-Host "Run: gate-cli --version"
}
finally {
    Remove-Item -Recurse -Force $Tmp -ErrorAction SilentlyContinue
}
```

- [ ] **Step 2: Smoke test on Windows**

Since `install.ps1` is not yet on `main`, test locally by running the script directly in PowerShell:
```powershell
.\install.ps1
```

Expected: downloads, verifies checksum, installs to `$env:LOCALAPPDATA\gate-cli`, adds to PATH.

```powershell
gate-cli --version
```

Expected: `gate-cli version v0.3.2` (after restarting terminal or refreshing PATH)

- [ ] **Step 3: Smoke test version pinning on Windows**

```powershell
$env:GATE_CLI_VERSION="v0.3.2"; irm https://raw.githubusercontent.com/gate/gate-cli/main/install.ps1 | iex
```

Expected: installs the specified version.

- [ ] **Step 4: Commit**

```bash
git add install.ps1
git commit -m "feat: add install.ps1 for Windows one-line install"
```

---

## Task 3: Add Homebrew tap to goreleaser

**Files:**
- Modify: `.goreleaser.yaml`

Prerequisites (must be done manually before this task runs):
1. `gate/homebrew-tap` repo exists on GitHub, initialized with a README and `Formula/.gitkeep`
2. `HOMEBREW_TAP_TOKEN` secret is set on `gate/gate-cli` (fine-grained PAT, `contents: write` on `gate/homebrew-tap`)

- [ ] **Step 1: Add `brews` section to `.goreleaser.yaml`**

Append after the `checksum` block:

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

- [ ] **Step 2: Validate goreleaser config**

```bash
goreleaser check
```

Expected: `• config is valid`

If `goreleaser` is not installed locally:
```bash
go run github.com/goreleaser/goreleaser/v2@latest check
```

- [ ] **Step 3: Commit**

```bash
git add .goreleaser.yaml
git commit -m "feat: add homebrew tap to goreleaser config"
```

---

## Task 4: Update `docs/quickstart.md`

**Files:**
- Modify: `docs/quickstart.md`

- [ ] **Step 1: Replace the Installation section**

Find the current Installation section (from `## Installation` down to and including the `gate-cli --help` block) and replace with the block below. Remove the "Verify the install" line — it is now covered by the version pinning examples.

```markdown
## Installation

### macOS / Linux — shell script

```sh
curl -fsSL https://raw.githubusercontent.com/gate/gate-cli/main/install.sh | sh
```

### macOS — Homebrew

```sh
brew install gate/tap/gate-cli
```

### Windows — PowerShell

```powershell
irm https://raw.githubusercontent.com/gate/gate-cli/main/install.ps1 | iex
```

### Pin to a specific version

```sh
# Unix
curl -fsSL https://raw.githubusercontent.com/gate/gate-cli/main/install.sh | sh -s -- --version v0.3.2

# Windows
$env:GATE_CLI_VERSION="v0.3.2"; irm https://raw.githubusercontent.com/gate/gate-cli/main/install.ps1 | iex
```

### Build from source (requires Go 1.21+)

```bash
git clone https://github.com/gate/gate-cli.git
cd gate-cli
go build -o gate-cli .
sudo mv gate-cli /usr/local/bin/
```
```

- [ ] **Step 2: Verify the file renders correctly**

```bash
cat docs/quickstart.md | head -60
```

Expected: clean markdown with all four install methods shown.

- [ ] **Step 3: Commit**

```bash
git add docs/quickstart.md
git commit -m "docs: update installation instructions with all install methods"
```

---

## Task 5: End-to-end release test

- [ ] **Step 1: Tag and push to trigger the release workflow**

```bash
git tag v0.3.3
git push origin main --tags
```

- [ ] **Step 2: Watch the release workflow**

Go to `https://github.com/gate/gate-cli/actions` and confirm the release job completes successfully, including the goreleaser step that pushes to `gate/homebrew-tap`.

- [ ] **Step 3: Verify Homebrew formula was pushed**

Check `https://github.com/gate/homebrew-tap/blob/main/Formula/gate-cli.rb` — it should contain the v0.3.3 URLs and SHA256 hashes.

- [ ] **Step 4: Test Homebrew install**

```bash
brew tap gate/tap
brew install gate-cli
gate-cli --version
```

Expected: `gate-cli version v0.3.3`

- [ ] **Step 5: Test shell script install**

```bash
curl -fsSL https://raw.githubusercontent.com/gate/gate-cli/main/install.sh | sh
gate-cli --version
```

Expected: `gate-cli version v0.3.3`

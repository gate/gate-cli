$ErrorActionPreference = 'Stop'
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12 -bor [Net.SecurityProtocolType]::Tls13

$Repo = "gate/gate-cli"
$Binary = "gate-cli.exe"
$InstallDir = "$env:LOCALAPPDATA\gate-cli"

# --- Resolve version ---
$Version = $env:GATE_CLI_VERSION
if (-not $Version) {
    $release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $release.tag_name
}

if ($Version -notmatch '^v?\d+\.\d+\.\d+') {
    throw "Invalid version format: $Version"
}

# Strip leading 'v' for the archive filename
$BareVersion = $Version.TrimStart('v')
$Archive = "gate-cli_${BareVersion}_windows_amd64.zip"
$BaseUrl = "https://github.com/$Repo/releases/download/$Version"

# --- Download ---
$Tmp = New-TemporaryFile | ForEach-Object { $_.DirectoryName + "\" + [System.IO.Path]::GetRandomFileName() }

try {
    New-Item -ItemType Directory -Path $Tmp | Out-Null

    Write-Host "Downloading $Archive..."
    Invoke-WebRequest "$BaseUrl/$Archive" -OutFile "$Tmp\$Archive"
    Invoke-WebRequest "$BaseUrl/checksums.txt" -OutFile "$Tmp\checksums.txt"

    # --- Verify checksum ---
    Write-Host "Verifying checksum..."
    $Expected = (Get-Content "$Tmp\checksums.txt" | Where-Object { $_ -match [regex]::Escape("  $Archive") }) -split '\s+' | Select-Object -First 1
    if (-not $Expected) {
        throw "Archive '$Archive' not found in checksums.txt"
    }
    $Actual = (Get-FileHash "$Tmp\$Archive" -Algorithm SHA256).Hash.ToLower()
    if ($Actual -ne $Expected) {
        throw "Checksum mismatch: expected $Expected, got $Actual"
    }

    # --- Extract ---
    Expand-Archive "$Tmp\$Archive" -DestinationPath $Tmp -Force

    # --- Install ---
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item "$Tmp\gate-cli.exe" "$InstallDir\gate-cli.exe" -Force

    # --- Add to PATH ---
    $UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    $PathEntries = $UserPath -split ';'
    if ($InstallDir -notin $PathEntries) {
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

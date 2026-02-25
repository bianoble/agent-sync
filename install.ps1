# install.ps1 — install agent-sync on Windows
#
# Usage:
#   irm https://raw.githubusercontent.com/bianoble/agent-sync/main/install.ps1 | iex
#
# Environment variables:
#   VERSION      — version to install (default: latest)
#   INSTALL_DIR  — directory to install to (default: $env:LOCALAPPDATA\agent-sync\bin)

$ErrorActionPreference = "Stop"

$Repo = "bianoble/agent-sync"
$Binary = "agent-sync"

function Main {
    $arch = Get-Arch

    if ($env:VERSION) {
        $version = $env:VERSION
    } else {
        $version = Get-LatestVersion
    }

    $versionDisplay = $version -replace "^v", ""
    Write-Host "Installing $Binary v$versionDisplay (windows/$arch)"

    if ($env:INSTALL_DIR) {
        $installDir = $env:INSTALL_DIR
    } else {
        $installDir = Join-Path $env:LOCALAPPDATA "agent-sync\bin"
    }

    if (-not (Test-Path $installDir)) {
        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    }

    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) "agent-sync-install-$(Get-Random)"
    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

    try {
        $archiveName = "${Binary}_windows_${arch}.zip"
        $checksumsName = "checksums.txt"
        $baseUrl = "https://github.com/$Repo/releases/download/$version"

        Write-Host "Downloading $archiveName..."
        Download-File "$baseUrl/$archiveName" (Join-Path $tmpDir $archiveName)
        Download-File "$baseUrl/$checksumsName" (Join-Path $tmpDir $checksumsName)

        Write-Host "Verifying checksum..."
        Verify-Checksum $tmpDir $archiveName

        Write-Host "Extracting..."
        Expand-Archive -Path (Join-Path $tmpDir $archiveName) -DestinationPath $tmpDir -Force

        $src = Join-Path $tmpDir "$Binary.exe"
        $dest = Join-Path $installDir "$Binary.exe"
        Copy-Item -Path $src -Destination $dest -Force

        Write-Host "Installed $Binary to $dest"

        # Add to PATH if not already present.
        $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
        if ($userPath -notlike "*$installDir*") {
            [Environment]::SetEnvironmentVariable("PATH", "$userPath;$installDir", "User")
            $env:PATH = "$env:PATH;$installDir"
            Write-Host "Added $installDir to user PATH"
        }

        Write-Host ""
        & $dest version
    } finally {
        Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
    }
}

function Get-Arch {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "x86"   { throw "32-bit Windows is not supported. A 64-bit system is required." }
        default { throw "Unsupported architecture: $arch" }
    }
}

function Get-LatestVersion {
    $url = "https://api.github.com/repos/$Repo/releases/latest"
    $response = Invoke-RestMethod -Uri $url -UseBasicParsing
    if (-not $response.tag_name) {
        throw "Could not determine latest version from GitHub API"
    }
    return $response.tag_name
}

function Download-File($url, $dest) {
    try {
        Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing
    } catch {
        throw "Failed to download: $url"
    }
}

function Verify-Checksum($dir, $file) {
    $checksumsFile = Join-Path $dir "checksums.txt"
    $checksums = Get-Content $checksumsFile
    $entry = $checksums | Where-Object { $_.Contains($file) }

    if (-not $entry) {
        throw "Checksum not found for $file in checksums.txt"
    }

    $expected = ($entry -split "\s+")[0]
    $actual = (Get-FileHash -Path (Join-Path $dir $file) -Algorithm SHA256).Hash.ToLower()

    if ($actual -ne $expected) {
        throw "Checksum mismatch for $file (expected $expected, got $actual)"
    }
}

Main

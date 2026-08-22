<#
.SYNOPSIS
    Installs agpass globally and configures PATH and AI Agent skills.
.DESCRIPTION
    Works both locally from the repository and as a remote one-liner:
    irm https://raw.githubusercontent.com/agpass/agpass/main/install.ps1 | iex
#>

$ErrorActionPreference = "Stop"

Write-Host "═══════════════════════════════════════════" -ForegroundColor Magenta
Write-Host "         Installing agpass CLI             " -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════`n" -ForegroundColor Magenta

# Refresh PATH in current session
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")

$InstallDir = Join-Path $HOME ".agpass\bin"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$TargetExe = Join-Path $InstallDir "agpass.exe"

# If running inside repo directory
if (Test-Path "main.go") {
    Write-Host "Building from local repository..." -ForegroundColor Yellow
    go build -ldflags="-s -w" -o $TargetExe .
} else {
    # Check if Go is installed
    if (Get-Command go -ErrorAction SilentlyContinue) {
        Write-Host "Installing via 'go install'..." -ForegroundColor Yellow
        go install agpass@latest
        $GopathBin = Join-Path (go env GOPATH) "bin\agpass.exe"
        if (Test-Path $GopathBin) {
            Copy-Item $GopathBin $TargetExe -Force
        }
    } else {
        Write-Host "Go is not detected. Please install Go (winget install GoLang.Go) or download pre-built binary." -ForegroundColor Red
        exit 1
    }
}

# 1. Register to User PATH
$UserPath = [System.Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -split ";" -notcontains $InstallDir) {
    [System.Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "✓ Added $InstallDir to user PATH." -ForegroundColor Green
} else {
    Write-Host "✓ $InstallDir is already in user PATH." -ForegroundColor DarkGray
}

# 2. Install AI Agent Skill
& $TargetExe install-skill

Write-Host "`n✓ Installation complete! Run 'agpass' to get started." -ForegroundColor Green
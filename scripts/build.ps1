<#
.SYNOPSIS
    Builds the agpass binary into the bin/ directory.
#>
param(
    [string]$OutputDir = "bin"
)

$ErrorActionPreference = "Stop"
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
$target = Join-Path $OutputDir "agpass.exe"

Write-Host "Building agpass -> $target..." -ForegroundColor Cyan
go build -ldflags="-s -w" -o $target .

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful! Output: $target" -ForegroundColor Green
} else {
    Write-Error "Build failed."
    exit 1
}
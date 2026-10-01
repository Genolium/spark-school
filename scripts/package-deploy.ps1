# проект «так называемый SPARK» — PowerShell Deployment Packager
$ErrorActionPreference = "Stop"

Write-Host "================================================" -ForegroundColor Cyan
Write-Host "📦  проект «так называемый SPARK» — Packaging for Deploy" -ForegroundColor Cyan
Write-Host "================================================" -ForegroundColor Cyan

$ScriptPath = Join-Path $PSScriptRoot "package-deploy.js"
if (Get-Command node -ErrorAction SilentlyContinue) {
    node $ScriptPath
} else {
    Write-Error "Node.js is not found in PATH. Please install Node.js."
}

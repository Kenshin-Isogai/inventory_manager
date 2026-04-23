param(
    [switch]$SkipInstall,
    [switch]$SkipDatabase
)

$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
$backend = Join-Path $root "backend"
$frontend = Join-Path $root "frontend"

Write-Host "[inventory_manager] workspace: $root"

if (-not $SkipDatabase) {
    Write-Host "[inventory_manager] starting PostgreSQL via docker compose"
    & docker compose up -d db
}

Write-Host "[inventory_manager] running backend migrations"
Push-Location $backend
try {
    & go run .\cmd\migrate up
}
finally {
    Pop-Location
}

if (-not $SkipInstall) {
    if (-not (Test-Path (Join-Path $frontend "node_modules"))) {
        Write-Host "[inventory_manager] installing frontend dependencies"
        Push-Location $frontend
        try {
            & npm install
        }
        finally {
            Pop-Location
        }
    }
}

$backendCommand = "Set-Location '$backend'; go run .\cmd\server"
$frontendCommand = "Set-Location '$frontend'; npm run dev"

Write-Host "[inventory_manager] opening backend window"
Start-Process powershell -ArgumentList "-NoExit", "-Command", $backendCommand | Out-Null

Write-Host "[inventory_manager] opening frontend window"
Start-Process powershell -ArgumentList "-NoExit", "-Command", $frontendCommand | Out-Null

Write-Host "[inventory_manager] backend: http://localhost:8080"
Write-Host "[inventory_manager] frontend: http://localhost:5173"

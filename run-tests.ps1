<#
.SYNOPSIS
    Runs the complete test suite for inventory_manager.
.DESCRIPTION
    1. Starts a test PostgreSQL container (port 5433)
    2. Runs Go integration tests against it
    3. Optionally runs Playwright E2E tests
    4. Tears down the test container
.EXAMPLE
    ./run-tests.ps1              # Run backend integration tests only
    ./run-tests.ps1 -E2E         # Also run Playwright E2E tests
    ./run-tests.ps1 -Verbose     # Show full test output
#>
param(
    [switch]$E2E,
    [switch]$KeepDB
)

$ErrorActionPreference = "Stop"

Write-Host "=== inventory_manager test runner ===" -ForegroundColor Cyan

# 1. Start test database
Write-Host "`n[1/4] Starting test database..." -ForegroundColor Yellow
docker compose -f docker-compose.test.yml up -d test-db
Write-Host "  Waiting for PostgreSQL to be ready..."
$retries = 0
while ($retries -lt 30) {
    $result = docker compose -f docker-compose.test.yml exec test-db pg_isready -U postgres -d inventory_manager_test 2>&1
    if ($LASTEXITCODE -eq 0) { break }
    Start-Sleep -Seconds 1
    $retries++
}
if ($retries -eq 30) {
    Write-Host "  ERROR: Database did not become ready" -ForegroundColor Red
    exit 1
}
Write-Host "  Database is ready." -ForegroundColor Green

# 2. Run Go integration tests
Write-Host "`n[2/4] Running backend integration tests..." -ForegroundColor Yellow
$env:TEST_DATABASE_URL = "postgres://postgres:postgres@localhost:5433/inventory_manager_test?sslmode=disable"
$env:DATABASE_URL = $env:TEST_DATABASE_URL

Push-Location backend
try {
    go test -v -tags=integration -timeout=120s -count=1 ./internal/integration/...
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  Backend tests FAILED" -ForegroundColor Red
    } else {
        Write-Host "  Backend tests PASSED" -ForegroundColor Green
    }
} finally {
    Pop-Location
}

# 3. Run Playwright E2E tests (optional)
if ($E2E) {
    Write-Host "`n[3/4] Running E2E Playwright tests..." -ForegroundColor Yellow
    Push-Location e2e
    try {
        npm install
        npx playwright install --with-deps chromium
        npx playwright test
        if ($LASTEXITCODE -ne 0) {
            Write-Host "  E2E tests FAILED" -ForegroundColor Red
        } else {
            Write-Host "  E2E tests PASSED" -ForegroundColor Green
        }
    } finally {
        Pop-Location
    }
} else {
    Write-Host "`n[3/4] Skipping E2E tests (use -E2E flag to enable)" -ForegroundColor DarkGray
}

# 4. Cleanup
if (-not $KeepDB) {
    Write-Host "`n[4/4] Cleaning up test database..." -ForegroundColor Yellow
    docker compose -f docker-compose.test.yml down -v
    Write-Host "  Cleanup complete." -ForegroundColor Green
} else {
    Write-Host "`n[4/4] Keeping test database (use 'docker compose -f docker-compose.test.yml down -v' to clean up)" -ForegroundColor DarkGray
}

Write-Host "`n=== Test run complete ===" -ForegroundColor Cyan

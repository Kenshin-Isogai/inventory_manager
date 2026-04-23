param(
    [string]$ProjectId,
    [string]$Region = "asia-northeast1",
    [string]$Repository = "inventory-manager",
    [string]$ImageTag = "latest",
    [string]$JobName = "inventory-manager-migrate",
    [string]$RuntimeServiceAccount = "inventory-manager-migrate-runtime",
    [string]$DatabaseUrlSecretName = "INVENTORY_MANAGER_DATABASE_URL",
    [string]$CloudSqlInstance = ""
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($ProjectId)) {
    throw "ProjectId is required. Example: .\run-cloud-migrations.ps1 -ProjectId my-gcp-project -ImageTag <sha>"
}

$image = "$Region-docker.pkg.dev/$ProjectId/$Repository/inventory-manager-backend:$ImageTag"
$serviceAccount = "$RuntimeServiceAccount@$ProjectId.iam.gserviceaccount.com"
$manifestPath = Join-Path $PSScriptRoot "backend\cloudrun.migrate.job.yaml"
$manifest = (Get-Content $manifestPath -Raw).
    Replace("IMAGE_PLACEHOLDER", $image).
    Replace("DATABASE_URL_SECRET_PLACEHOLDER", $DatabaseUrlSecretName)
$tempFile = Join-Path $env:TEMP "inventory-manager-migrate-job.yaml"

Set-Content -Path $tempFile -Value $manifest -NoNewline

Write-Host "[migrate] project=$ProjectId region=$Region image=$image"
& gcloud config set project $ProjectId | Out-Null
& gcloud run jobs replace $tempFile --region $Region | Out-Null
& gcloud run jobs update $JobName --region $Region --service-account $serviceAccount | Out-Null
if (-not [string]::IsNullOrWhiteSpace($CloudSqlInstance)) {
    & gcloud run jobs update $JobName --region $Region --set-cloudsql-instances $CloudSqlInstance | Out-Null
}
& gcloud run jobs execute $JobName --region $Region --wait

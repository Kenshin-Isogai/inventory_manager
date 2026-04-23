param(
    [string]$ProjectId,
    [string]$Region = "asia-northeast1",
    [string]$Repository = "inventory-manager",
    [string]$FrontendService = "inventory-manager-frontend",
    [string]$BackendService = "inventory-manager-backend",
    [string]$MigrationJob = "inventory-manager-migrate",
    [string]$DeployServiceAccount = "inventory-manager-deployer",
    [string]$BackendRuntimeServiceAccount = "inventory-manager-backend-runtime",
    [string]$FrontendRuntimeServiceAccount = "inventory-manager-frontend-runtime",
    [string]$MigrationRuntimeServiceAccount = "inventory-manager-migrate-runtime",
    [string]$DatabaseUrlSecretName = "INVENTORY_MANAGER_DATABASE_URL",
    [string]$ProcurementWebhookSecretName = "PROCUREMENT_WEBHOOK_SECRET",
    [string]$WorkloadIdentityPool = "github-actions-pool",
    [string]$WorkloadIdentityProvider = "github-actions-provider",
    [switch]$SkipApiEnablement,
    [switch]$SkipArtifactRegistry,
    [switch]$SkipServiceAccounts,
    [switch]$SkipSecrets,
    [switch]$SkipWorkloadIdentity
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($ProjectId)) {
    throw "ProjectId is required. Example: .\bootstrap-cloud-run.ps1 -ProjectId my-gcp-project"
}

function Invoke-GCloud {
    param([string[]]$Args)
    Write-Host ("gcloud " + ($Args -join " "))
    & gcloud @Args
}

Write-Host "[bootstrap] project=$ProjectId region=$Region"
Invoke-GCloud @("config", "set", "project", $ProjectId)

if (-not $SkipApiEnablement) {
    Invoke-GCloud @(
        "services", "enable",
        "run.googleapis.com",
        "artifactregistry.googleapis.com",
        "secretmanager.googleapis.com",
        "iam.googleapis.com",
        "iamcredentials.googleapis.com",
        "sts.googleapis.com"
    )
}

if (-not $SkipArtifactRegistry) {
    $repoCheck = & gcloud artifacts repositories describe $Repository --location $Region 2>$null
    if ($LASTEXITCODE -ne 0) {
        Invoke-GCloud @("artifacts", "repositories", "create", $Repository, "--repository-format=docker", "--location", $Region)
    } else {
        Write-Host "[bootstrap] artifact registry repository already exists"
    }
}

$deploySaEmail = "$DeployServiceAccount@$ProjectId.iam.gserviceaccount.com"
$backendRuntimeSaEmail = "$BackendRuntimeServiceAccount@$ProjectId.iam.gserviceaccount.com"
$frontendRuntimeSaEmail = "$FrontendRuntimeServiceAccount@$ProjectId.iam.gserviceaccount.com"
$migrationRuntimeSaEmail = "$MigrationRuntimeServiceAccount@$ProjectId.iam.gserviceaccount.com"

if (-not $SkipServiceAccounts) {
    foreach ($pair in @(
        @{ Name = $DeployServiceAccount; Display = "Inventory Manager Deploy" },
        @{ Name = $BackendRuntimeServiceAccount; Display = "Inventory Manager Backend Runtime" },
        @{ Name = $FrontendRuntimeServiceAccount; Display = "Inventory Manager Frontend Runtime" },
        @{ Name = $MigrationRuntimeServiceAccount; Display = "Inventory Manager Migration Runtime" }
    )) {
        & gcloud iam service-accounts describe "$($pair.Name)@$ProjectId.iam.gserviceaccount.com" 2>$null | Out-Null
        if ($LASTEXITCODE -ne 0) {
            Invoke-GCloud @("iam", "service-accounts", "create", $pair.Name, "--display-name", $pair.Display)
        } else {
            Write-Host "[bootstrap] service account $($pair.Name) already exists"
        }
    }

    Invoke-GCloud @("projects", "add-iam-policy-binding", $ProjectId, "--member", "serviceAccount:$deploySaEmail", "--role", "roles/run.admin")
    Invoke-GCloud @("projects", "add-iam-policy-binding", $ProjectId, "--member", "serviceAccount:$deploySaEmail", "--role", "roles/artifactregistry.writer")
    foreach ($serviceAccount in @($backendRuntimeSaEmail, $frontendRuntimeSaEmail, $migrationRuntimeSaEmail)) {
        Invoke-GCloud @("iam", "service-accounts", "add-iam-policy-binding", $serviceAccount, "--member", "serviceAccount:$deploySaEmail", "--role", "roles/iam.serviceAccountUser")
    }
    Invoke-GCloud @("projects", "add-iam-policy-binding", $ProjectId, "--member", "serviceAccount:$backendRuntimeSaEmail", "--role", "roles/secretmanager.secretAccessor")
    Invoke-GCloud @("projects", "add-iam-policy-binding", $ProjectId, "--member", "serviceAccount:$migrationRuntimeSaEmail", "--role", "roles/secretmanager.secretAccessor")
}

if (-not $SkipSecrets) {
    foreach ($secretName in @($DatabaseUrlSecretName, $ProcurementWebhookSecretName)) {
        & gcloud secrets describe $secretName 2>$null | Out-Null
        if ($LASTEXITCODE -ne 0) {
            Invoke-GCloud @("secrets", "create", $secretName, "--replication-policy=automatic")
            Write-Host "[bootstrap] created secret $secretName. Add first version manually:"
            Write-Host "  echo VALUE | gcloud secrets versions add $secretName --data-file=-"
        } else {
            Write-Host "[bootstrap] secret $secretName already exists"
        }
    }
}

if (-not $SkipWorkloadIdentity) {
    $projectNumber = (& gcloud projects describe $ProjectId --format "value(projectNumber)").Trim()
    & gcloud iam workload-identity-pools describe $WorkloadIdentityPool --location global 2>$null | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Invoke-GCloud @("iam", "workload-identity-pools", "create", $WorkloadIdentityPool, "--location=global", "--display-name=GitHub Actions Pool")
    }
    & gcloud iam workload-identity-pools providers describe $WorkloadIdentityProvider --workload-identity-pool $WorkloadIdentityPool --location global 2>$null | Out-Null
    if ($LASTEXITCODE -ne 0) {
        Invoke-GCloud @(
            "iam", "workload-identity-pools", "providers", "create-oidc", $WorkloadIdentityProvider,
            "--location=global",
            "--workload-identity-pool=$WorkloadIdentityPool",
            "--display-name=GitHub Provider",
            "--issuer-uri=https://token.actions.githubusercontent.com",
            "--attribute-mapping=google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.ref=assertion.ref"
        )
    }
    Invoke-GCloud @(
        "iam", "service-accounts", "add-iam-policy-binding", $deploySaEmail,
        "--role", "roles/iam.workloadIdentityUser",
        "--member", "principalSet://iam.googleapis.com/projects/$projectNumber/locations/global/workloadIdentityPools/$WorkloadIdentityPool/attribute.repository/Kenshin-Isogai/inventory_manager"
    )
}

Write-Host ""
Write-Host "[bootstrap] GitHub repository variables"
Write-Host "  GCP_PROJECT_ID=$ProjectId"
Write-Host "  GCP_REGION=$Region"
Write-Host "  GAR_LOCATION=$Region"
Write-Host "  GAR_REPOSITORY=$Repository"
Write-Host "  CLOUD_RUN_FRONTEND_SERVICE=$FrontendService"
Write-Host "  CLOUD_RUN_BACKEND_SERVICE=$BackendService"
Write-Host "  CLOUD_RUN_MIGRATION_JOB=$MigrationJob"
Write-Host "  DATABASE_URL_SECRET_NAME=$DatabaseUrlSecretName"
Write-Host "  AUTH_MODE=dry_run"
Write-Host "  RBAC_MODE=dry_run"
Write-Host "  JWT_VERIFIER=jwks"
Write-Host "  STORAGE_MODE=local"
Write-Host "  JWT_SIGNING_ALGORITHMS=RS256"
Write-Host "  OIDC_REQUIRE_EMAIL_VERIFIED=true"
Write-Host "  FIREBASE_API_KEY=<identity-platform-web-api-key>"
Write-Host "  FIREBASE_AUTH_DOMAIN=$ProjectId.firebaseapp.com"
Write-Host "  FIREBASE_APP_ID=<firebase-web-app-id>"
Write-Host "  CLOUD_SQL_INSTANCE=<project:region:instance>  # optional"
Write-Host "  CLOUD_STORAGE_BUCKET=<bucket-name>  # required when STORAGE_MODE=cloud"
Write-Host "  GOOGLE_CLOUD_PROJECT=$ProjectId  # optional"
Write-Host "  VERTEX_AI_LOCATION=asia-northeast1  # optional"
Write-Host "  GEMINI_MODEL=gemini-3-flash-preview  # optional"
Write-Host "  FRONTEND_PUBLIC_URL=  # optional override"
Write-Host ""
Write-Host "[bootstrap] GitHub repository secrets"
Write-Host "  WORKLOAD_IDENTITY_PROVIDER=projects/<PROJECT_NUMBER>/locations/global/workloadIdentityPools/$WorkloadIdentityPool/providers/$WorkloadIdentityProvider"
Write-Host "  WORKLOAD_IDENTITY_SERVICE_ACCOUNT=$deploySaEmail"
Write-Host "  BACKEND_RUNTIME_SERVICE_ACCOUNT=$backendRuntimeSaEmail"
Write-Host "  FRONTEND_RUNTIME_SERVICE_ACCOUNT=$frontendRuntimeSaEmail"
Write-Host "  MIGRATE_RUNTIME_SERVICE_ACCOUNT=$migrationRuntimeSaEmail"
Write-Host ""
Write-Host "[bootstrap] next steps"
Write-Host "  1. Add secret versions for $DatabaseUrlSecretName and $ProcurementWebhookSecretName"
Write-Host "  2. Grant Cloud SQL / Storage / Vertex AI roles to runtime service accounts as needed"
Write-Host "  3. Push main or run the deploy workflow manually"
Write-Host "  4. If you keep migrations manual, run .\run-cloud-migrations.ps1 before first backend traffic"

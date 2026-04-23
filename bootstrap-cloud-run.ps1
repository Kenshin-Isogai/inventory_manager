param(
    [string]$ProjectId,
    [string]$Region = "asia-northeast1",
    [string]$Repository = "inventory-manager",
    [string]$FrontendService = "inventory-manager-frontend",
    [string]$BackendService = "inventory-manager-backend",
    [string]$MigrationJob = "inventory-manager-migrate",
    [string]$DeployServiceAccount = "inventory-manager-deployer",
    [string]$RuntimeServiceAccount = "inventory-manager-runtime",
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
$runtimeSaEmail = "$RuntimeServiceAccount@$ProjectId.iam.gserviceaccount.com"

if (-not $SkipServiceAccounts) {
    foreach ($pair in @(
        @{ Name = $DeployServiceAccount; Display = "Inventory Manager Deploy" },
        @{ Name = $RuntimeServiceAccount; Display = "Inventory Manager Runtime" }
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
    Invoke-GCloud @("projects", "add-iam-policy-binding", $ProjectId, "--member", "serviceAccount:$deploySaEmail", "--role", "roles/iam.serviceAccountUser")
    Invoke-GCloud @("projects", "add-iam-policy-binding", $ProjectId, "--member", "serviceAccount:$runtimeSaEmail", "--role", "roles/secretmanager.secretAccessor")
}

if (-not $SkipSecrets) {
    foreach ($secretName in @("DATABASE_URL", "PROCUREMENT_WEBHOOK_SECRET")) {
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
Write-Host "  AUTH_PROVIDER=local"
Write-Host ""
Write-Host "[bootstrap] GitHub repository secrets"
Write-Host "  WORKLOAD_IDENTITY_PROVIDER=projects/<PROJECT_NUMBER>/locations/global/workloadIdentityPools/$WorkloadIdentityPool/providers/$WorkloadIdentityProvider"
Write-Host "  WORKLOAD_IDENTITY_SERVICE_ACCOUNT=$deploySaEmail"
Write-Host ""
Write-Host "[bootstrap] next steps"
Write-Host "  1. Add secret versions for DATABASE_URL and PROCUREMENT_WEBHOOK_SECRET"
Write-Host "  2. Push main or run the deploy workflow manually"
Write-Host "  3. Deploy the migration job with .\run-cloud-migrations.ps1 before first backend traffic"

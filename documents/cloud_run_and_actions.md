# Cloud Run / GitHub Actions Notes

## Deployment Shape

- `frontend`: static SPA container on Cloud Run
- `backend`: Go API container on Cloud Run
- `migration job`: Cloud Run Job using the backend image with `/app/migrate up`
- deploy trigger: manual `workflow_dispatch` only
- manual deploy targets: `backend`, `frontend`, `full`
- deploy guardrail: run backend/frontend verification before image push and deploy

## Required Repository Variables

- `GCP_PROJECT_ID`
- `GCP_REGION`
- `GAR_LOCATION`
- `GAR_REPOSITORY`
- `CLOUD_RUN_FRONTEND_SERVICE`
- `CLOUD_RUN_BACKEND_SERVICE`
- `CLOUD_RUN_MIGRATION_JOB`
- `DATABASE_URL_SECRET_NAME`
- `AUTH_MODE`
- `JWT_VERIFIER`
- `RBAC_MODE`
- `STORAGE_MODE`
- `JWT_SIGNING_ALGORITHMS`
- `OIDC_REQUIRE_EMAIL_VERIFIED`
- `FIREBASE_API_KEY`
- `FIREBASE_AUTH_DOMAIN`
- `FIREBASE_APP_ID`

Optional repository variables:

- `CLOUD_SQL_INSTANCE`
- `CLOUD_STORAGE_BUCKET`
- `GOOGLE_CLOUD_PROJECT`
- `VERTEX_AI_LOCATION`
- `GEMINI_MODEL`
- `FRONTEND_PUBLIC_URL`

## Required Repository Secrets

- `WORKLOAD_IDENTITY_PROVIDER`
- `WORKLOAD_IDENTITY_SERVICE_ACCOUNT`
- `BACKEND_RUNTIME_SERVICE_ACCOUNT`
- `FRONTEND_RUNTIME_SERVICE_ACCOUNT`
- `MIGRATE_RUNTIME_SERVICE_ACCOUNT`

## Required Secret Manager Entries

- the secret referenced by `DATABASE_URL_SECRET_NAME`
- `PROCUREMENT_WEBHOOK_SECRET`

Recommended secret names:

- `INVENTORY_MANAGER_DATABASE_URL`
- `PROCUREMENT_WEBHOOK_SECRET`

## Workflow Behavior

The deploy workflow now does the following:

1. verify backend for `backend|full`
2. verify frontend for `frontend|full`
3. build and push backend image for `backend|full`
4. deploy and execute the migration job for `backend|full`
5. deploy backend with explicit runtime service account
6. resolve backend URL
7. build and push frontend image for `frontend|full`
8. deploy frontend with explicit runtime service account
9. resolve frontend URL
10. update backend `APP_BASE_URL` and `ALLOWED_ORIGINS` after frontend URL is known
11. verify backend `/health`, `/ready`, and frontend root URL

## Service Accounts

Recommended split:

- deploy SA:
  `Cloud Run Admin`, `Artifact Registry Writer`, and `Service Account User` on backend/frontend/migrate runtime SAs
- backend runtime SA:
  `Secret Manager Secret Accessor`, plus `Cloud SQL Client`, `Storage Object Admin`, `Vertex AI User` only when those integrations are used
- frontend runtime SA:
  no extra project roles by default
- migrate runtime SA:
  `Secret Manager Secret Accessor` and `Cloud SQL Client`

The current workflow passes service accounts explicitly with `--service-account`, so IAM on these accounts is effective only when the matching GitHub secret points at the intended SA.

## Initial Bootstrap Order

1. run [bootstrap-cloud-run.ps1](/C:/Users/IsogaiKenshin/Documents/Yaqumo/applications/inventory_manager/bootstrap-cloud-run.ps1) with the target GCP project
2. add secret versions for `DATABASE_URL_SECRET_NAME` and `PROCUREMENT_WEBHOOK_SECRET`
3. set the printed GitHub repository variables and secrets
4. grant runtime SA roles for Cloud SQL / Cloud Storage / Vertex AI as needed
5. run the deploy workflow once with `deploy_target=backend` or `deploy_target=full`
6. confirm backend `/health` and `/ready`
7. confirm frontend root URL

The bootstrap split is intentional:

- infrastructure creation is one-time or infrequent
- service deploy is repeatable CI/CD
- schema migration remains explicit enough to audit, even when the workflow runs it automatically

## Staging Rollout Checklist

- create one staging Cloud Run pair for `frontend` and `backend`
- create the `inventory-manager-migrate` Cloud Run Job from [backend/cloudrun.migrate.job.yaml](/C:/Users/IsogaiKenshin/Documents/Yaqumo/applications/inventory_manager/backend/cloudrun.migrate.job.yaml)
- wire GitHub OIDC to GCP Workload Identity Federation
- create Artifact Registry repository
- create Secret Manager values for the secret named by `DATABASE_URL_SECRET_NAME` and for `PROCUREMENT_WEBHOOK_SECRET`
- bind deploy/runtime service accounts to the intended roles
- confirm `/health` and `/ready` return success after each deploy
- keep at least one prior Cloud Run revision available for rollback

## Operational Notes

- backend is still deployed with `--allow-unauthenticated`; application auth is enforced by bearer-token verification
- if `STORAGE_MODE=cloud`, also set `CLOUD_STORAGE_BUCKET` and grant the backend runtime SA bucket object permissions
- if `CLOUD_SQL_INSTANCE` is set, the workflow attaches it to both the migration job and backend service
- frontend no longer needs build-time API URL injection because runtime config is written at container startup

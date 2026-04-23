# Cloud Run / GitHub Actions Notes

## Deployment Shape

- `frontend`: static SPA container on Cloud Run
- `backend`: Go API container on Cloud Run
- `migration job`: Cloud Run Job using the backend image with `/app/migrate up`
- deploy trigger: `main` push or manual `workflow_dispatch`
- deploy guardrail: run backend/frontend verification before image push and deploy

## Required Repository Variables

- `GCP_PROJECT_ID`
- `GCP_REGION`
- `GAR_LOCATION`
- `GAR_REPOSITORY`
- `CLOUD_RUN_FRONTEND_SERVICE`
- `CLOUD_RUN_BACKEND_SERVICE`
- `AUTH_MODE`
- `JWT_VERIFIER`
- `RBAC_MODE`
- `STORAGE_MODE`
- `FIREBASE_API_KEY`
- `FIREBASE_AUTH_DOMAIN`
- `FIREBASE_APP_ID`

## Required Repository Secrets

- `WORKLOAD_IDENTITY_PROVIDER`
- `WORKLOAD_IDENTITY_SERVICE_ACCOUNT`

## Required Secret Manager Entries

- `DATABASE_URL`
- `PROCUREMENT_WEBHOOK_SECRET`

Recommended next additions when moving from local/mock to cloud adapters:

- `CLOUD_STORAGE_BUCKET`
- `GOOGLE_CLOUD_PROJECT`
- `VERTEX_AI_LOCATION`
- `GEMINI_MODEL`
- `JWKS_URL`
- `JWT_ISSUER`
- `JWT_AUDIENCE`

## Workflow Behavior

The deploy workflow now does the following in one path:

1. backend `go mod tidy`, `go test ./...`, `go build ./cmd/server`
2. frontend `npm ci`, `npm run lint`, `npm run test`, `npm run build`
3. build and push frontend/backend images to Artifact Registry
4. deploy both services to Cloud Run
5. inject backend runtime env and Secret Manager values
6. verify deployed `/health`, `/ready`, and frontend root URL

## Initial Bootstrap Order

1. run [bootstrap-cloud-run.ps1](/C:/Users/IsogaiKenshin/Documents/Yaqumo/applications/inventory_manager/bootstrap-cloud-run.ps1) with the target GCP project
2. add secret versions for `DATABASE_URL` and `PROCUREMENT_WEBHOOK_SECRET`
3. set the printed GitHub repository variables and secrets
4. run the deploy workflow once so the backend image exists in Artifact Registry
5. run [run-cloud-migrations.ps1](/C:/Users/IsogaiKenshin/Documents/Yaqumo/applications/inventory_manager/run-cloud-migrations.ps1) with the backend image tag to create and execute the migration job
6. confirm backend `/health` and `/ready`
7. confirm frontend root URL

The bootstrap split is intentional:

- infrastructure creation is one-time or infrequent
- service deploy is repeatable CI/CD
- schema migration remains explicit and idempotent

## Staging Rollout Checklist

- create one staging Cloud Run pair for `frontend` and `backend`
- create the `inventory-manager-migrate` Cloud Run Job from [backend/cloudrun.migrate.job.yaml](/C:/Users/IsogaiKenshin/Documents/Yaqumo/applications/inventory_manager/backend/cloudrun.migrate.job.yaml)
- wire GitHub OIDC to GCP Workload Identity Federation
- create Artifact Registry repository
- create Secret Manager values for `DATABASE_URL` and `PROCUREMENT_WEBHOOK_SECRET`
- grant runtime service account access to Secret Manager and backing services
- confirm `/health` and `/ready` return success after each deploy
- keep at least one prior Cloud Run revision available for rollback

## Operational Notes

- `--allow-unauthenticated` is still present for the current skeleton; tighten ingress/auth when cloud auth is enforced
- deploy health verification is intentionally fail-fast so missing env/secrets stop the rollout immediately
- DB migration execution is intentionally separate from service startup; use the dedicated Cloud Run Job for first bootstrap and later controlled schema changes

# inventory_manager

Implementation scaffold for the inventory manager described in [documents/pre_implementation_plan.md](/C:/Users/IsogaiKenshin/Documents/Yaqumo/applications/inventory_manager/documents/pre_implementation_plan.md).

## Layout

- `backend`: Go API service with config, health endpoints, storage abstraction, and SQL-file migration runner
- `frontend`: React/Vite SPA shell with route guards, app sections, and mock read-model contracts
- `.github/workflows`: CI and deploy skeleton for separate frontend/backend services

## Local Development

### Backend

1. Start PostgreSQL with `docker compose up -d db`
2. Set `DATABASE_URL` from `.env.example`
3. Run `go run ./cmd/migrate up`
4. Run `go run ./cmd/server`

Local auth profiles are available when `AUTH_MODE=dry_run` or `AUTH_MODE=enforced` with `JWT_VERIFIER=local`.
Default dev bearer tokens:

- `local-admin-token`
- `local-operator-token`
- `local-inventory-token`
- `local-procurement-token`

You can also use `local:<email>|<displayName>` to simulate a new unregistered identity and walk through the registration and approval flow.

For Vertex AI local development, prefer `gcloud auth application-default login`.
Use `GOOGLE_APPLICATION_CREDENTIALS` only when you intentionally want to point the backend at a specific local service-account JSON key.
On Cloud Run, rely on the attached runtime service account instead of a JSON key.

### Frontend

1. Run `npm install`
2. Run `npm run dev`

Frontend runtime config is injected at container start via `runtime-config.js`, so Cloud Run does not need a rebuild just to change API or Firebase endpoints.

### One-command local start

- `.\start-app.ps1`
- `start-app.bat`

The script starts PostgreSQL, runs backend migrations, and opens backend/frontend dev servers in separate PowerShell windows.

## Cloud Bootstrap

Initial Cloud Run bootstrap is separated from normal deploys:

1. `.\bootstrap-cloud-run.ps1 -ProjectId <gcp-project-id>`
2. add first versions for the database secret named by `DATABASE_URL_SECRET_NAME` and for `PROCUREMENT_WEBHOOK_SECRET`
3. configure the GitHub repository variables/secrets printed by the script
4. grant runtime service-account roles for Cloud SQL, Cloud Storage, and Vertex AI as needed
5. run the deploy workflow to publish frontend/backend images and services
6. if needed, run `.\run-cloud-migrations.ps1 -ProjectId <gcp-project-id> -ImageTag <backend-image-tag>`

The backend image now contains both `/app/server` and `/app/migrate`, so schema bootstrap can run as a dedicated Cloud Run Job instead of at service startup.

For Identity Platform production auth, configure:

- backend env: `JWT_VERIFIER=jwks`, `OIDC_EXPECTED_ISSUER`, `OIDC_EXPECTED_AUDIENCE`, `OIDC_JWKS_URL`, `JWT_SIGNING_ALGORITHMS=RS256`
- frontend runtime env: `FIREBASE_API_KEY`, `FIREBASE_AUTH_DOMAIN`, `FIREBASE_PROJECT_ID`, `FIREBASE_APP_ID`

## Deployment Direction

- Frontend and backend are deployed as separate services
- Cloud Run is the target runtime
- GitHub Actions is the CI/CD entrypoint
- backend, frontend, and migration job each use an explicit Cloud Run service account
- deploy workflow is manual-only and supports `backend`, `frontend`, and `full` targets via `workflow_dispatch`

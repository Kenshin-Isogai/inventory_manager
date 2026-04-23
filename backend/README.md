# Backend

Go `net/http` based API server for the inventory manager.

## Commands

- `go run ./cmd/server`
- `go run ./cmd/migrate up`
- `go run ./cmd/migrate status`
- container image also includes `/app/migrate` for Cloud Run Job based schema bootstrap

## Environment Variables

See `.env.example` for the full list.

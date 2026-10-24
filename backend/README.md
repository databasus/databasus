# Before run

> Copy .env.example to .env in the repo root (single source for backend, frontend, and docker compose)

# Run

Commands run from the repository root, with the tools from `mise.toml` installed.

To run the backend together with the frontend, the dev database, the test fixtures and the WAL backup source:

> mise run dev

The backend runs under [air](.air.toml), which rebuilds and restarts it when a Go file is saved; a build that fails leaves the previous backend serving. Once the backend accepts connections, the [seed command](cmd/seed_dev/doc.go) creates the dev account and the seeded databases if they are missing. To run the seed again against the running app, from `backend/`:

> go run ./cmd/seed_dev

To run tests:

> mise run test

Before commit:

> mise run lint

# Migrations

The backend applies pending migrations on startup. To create a migration:

> goose -dir backend/migrations create MIGRATION_NAME sql

To apply or roll back migrations on the dev database, with the variables from `.env` exported (goose reads `GOOSE_DRIVER` and `GOOSE_DBSTRING` from the environment):

> goose -dir backend/migrations up

> goose -dir backend/migrations down

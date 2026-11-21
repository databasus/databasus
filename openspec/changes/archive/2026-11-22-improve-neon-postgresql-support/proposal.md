## Why

The connection string Neon shows by default points at its connection pooler, a PgBouncer in transaction mode. Such a pooler may move a client to another server process between transactions, and a manual check against a real Neon project saw session settings reset that way under concurrent load. `pg_dump` and `pg_restore` set their session up once and rely on it, so a backup or restore through the pooler can break on a large or busy database. The Supabase transaction pooler on port 6543 has the same flaw.

A physical backup of a Neon database cannot work at all, because Neon refuses physical replication connections. Today the connection test explains that refusal as a missing `pg_hba.conf` entry and tells the user to edit a file they cannot reach.

Nothing checks Databasus against a real managed PostgreSQL service. The manual Neon check from this conversation cannot be repeated, so a regression in backups or restores of Neon databases would go unnoticed until a user reports it.

## What Changes

- Importing a connection string replaces a Neon pooler host (`ep-<name>-pooler.<region>.neon.tech`) with the direct host of the same endpoint, without a message in the form.
- Importing a connection string for the Supabase transaction pooler (`*.pooler.supabase.com` on port 6543) switches the port to 5432, the Supabase session pooler on the same host, without a message in the form.
- A physical PostgreSQL database whose host belongs to Neon is refused before any connection opens: the connection test fails with a new reason, `PHYSICAL_REPLICATION_UNSUPPORTED`, and a create or update answers `database_connection_failed` saying that Neon offers only logical backups. The form explains the reason. The API enum `DatabaseConnectionFailure` gains this value.
- A new end-to-end test backs up a Neon database through the HTTP API and restores it, both into a new database and over the source, with the owner role and with a Databasus-created limited user. The same cycle also runs against a PostgreSQL container, so the test's own logic is checked without Neon. Both belong to the logical end-to-end packages, so `mise run test-full` and the CI `logical` group run them.
- Locally, the test skips with a stated reason when the Neon connection string is missing from `.env`. CI makes the secret mandatory: without it the `logical` check fails.
- The website gains a Neon FAQ page next to the Supabase one: use the direct host, only logical backups are possible, restore into a Neon database or branch.

## Out of scope

- A UI or browser scenario for Neon; the API covers the backend path, and the frontend parser has unit tests.
- Supabase, or any other managed PostgreSQL, in the new end-to-end test. A free Supabase project pauses after a week without activity, which would fail CI for reasons outside the code.
- Rewriting pooler hosts on the backend, or refusing them. A small database backs up through the pooler, so a server-side refusal would block working setups.
- A hint or warning in the form for a pooler host typed by hand.
- Neon branch management through the Neon API, and any secret besides one connection string.
- Physical backup support for Neon in any form.

## Capabilities

### New Capabilities

- `managed-postgresql-backup-tests`: the end-to-end check of backups and restores against a real managed PostgreSQL service, its isolation between concurrent runs, its cleanup, and when it skips or fails.

### Modified Capabilities

- `frontend-database-connection`: a PostgreSQL connection string for a transaction-mode pooler of Neon or Supabase fills the form with the address of a direct or session-mode connection.
- `physical-backups/backup-method`: a Neon source is refused for physical backups with its own reason, `PHYSICAL_REPLICATION_UNSUPPORTED`.
- `ci-pipeline`: the `logical` group receives the Neon secret and fails when it is missing.

## Impact

- Frontend: `PostgresqlConnectionStringParser.ts` and its tests; the explanation of connection failures in the PostgreSQL physical form; every interface dictionary for the new reason.
- Backend: a Neon host check in `engines/postgresql/physical`, called at the start of `OpenTunnel` before any dial, `pgshared` connection error codes, `backend/api/openapi.yaml` and the generated code; a new package under `backend/internal/modules/backups/e2e/postgresql/`.
- CI: `.github/workflows/ci-release.yml` passes a repository secret to the `logical` group on both architectures. The secret points at a throwaway Neon project that holds no user data.
- Website: a new FAQ page in English and the translated languages, linked from the FAQ index and the sitemap.
- Documents: `docs/testing.md`, `docs/glossary.md` (transaction-mode pooler, managed PostgreSQL service), `.env.example`.
- The change answers to the root `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md` and `website/AGENTS.md`.
- **BREAKING** for API clients that treat `DatabaseConnectionFailure` as a closed set: the enum gains one value. Nothing is removed.

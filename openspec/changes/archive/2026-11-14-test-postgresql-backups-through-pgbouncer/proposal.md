## Why

The backup end-to-end suites exercise direct PostgreSQL connections, SSH tunnels and TLS, but do not prove that backups remain restorable when Databasus connects through PgBouncer. Two standalone tests will cover logical backup and restore, and physical full/incremental backup with streamed WAL and point-in-time recovery, without changing existing scenarios.

## What Changes

- Add a real, session-pooling PgBouncer fixture with a private PostgreSQL source and a published PgBouncer endpoint. Pin the test image for both amd64 and arm64.
- Add one standalone PostgreSQL 18 logical end-to-end test. Back up through PgBouncer, restore through it into a different database with four restore jobs, and compare restored rows and schema with the source.
- Add one standalone PostgreSQL 18 physical end-to-end test. Take a full backup and two incrementals, archive WAL containing writes before and after a selected recovery time, then reconstruct and boot an independent cluster and verify the recovered rows.
- Keep existing tests and their direct-connection coverage. Reuse test helpers rather than copying their backup and recovery machinery.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

None. This change adds test coverage and fixtures without changing product requirements. It sets `skip_specs: true` and creates no delta specs.

## Impact

- Governing instructions: root `AGENTS.md` and `backend/AGENTS.md`.
- Shared container support under `backend/internal/lib/testcontainers/` and the immutable image catalog at `backend/internal/lib/testimages/images.json`.
- A new logical test file under `backend/internal/modules/backups/e2e/postgresql/`.
- A new physical test file under `backend/internal/modules/backups/e2e/postgresqlphysical/pg18/` and a focused PgBouncer scenario helper under `backend/internal/app/apptest/physicaltest/`.
- Test-only dependency: PgBouncer 1.26.1, using the published `edoburu/pgbouncer:v1.26.1-p0` image pinned by digest during implementation.
- No production dependencies, HTTP contracts, database migrations or runtime configuration change. Nothing is breaking.
- No shared documentation or agent-instruction changes are needed: existing test ownership, isolation and execution procedures still apply. Fixture-specific setup belongs with the fixture.

## Out of Scope

- Replacing, repurposing or extending existing test scenarios to run through PgBouncer.
- Testing PgBouncer against every supported PostgreSQL version. Both new tests use PostgreSQL 18; this does not claim that PgBouncer requires PostgreSQL 18.
- Transaction or statement pooling, PgBouncer failover or restart behavior, and combinations with SSH or TLS.
- Physical restore over a database connection: physical recovery consumes downloaded files and archived WAL, then starts a new PostgreSQL server.
- New product settings, compatibility fallbacks, broad production refactoring, CI workflow changes or a new test runner.

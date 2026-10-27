## Why

Only the 14 end-to-end scenarios in `backend/internal/features/tests/physical` check physical PostgreSQL backups as a whole, from the first FULL to a cluster restored by the recovery script. They are built on the old code they are meant to protect. They assemble their own Gin router around the old controllers and seed the database with SQL helpers behind the `testhelpers` build tag. They start the old scheduler and WAL stream supervisor through tagged factories, and they read the catalog through the old repositories, `chain_view` and `util/walmath`. Milestone 10 of the backend rewrite plan moves exactly that code: `engines-physical` moves the source and process code into `internal/engines`, and `physical-backups-module` moves the state and the loops into the `backups` module. If the scenarios still depended on it, each of those changes would have to rewrite the safety net in the same diff as the code under it. This change lands first and moves the scenarios onto the real test application, so that the next two changes can run them unchanged.

## What Changes

- The 14 scenario bodies move from `features/tests/physical/postgresql/shared/bodies.go` into a shared scenario package next to `enginetest`, and two thin per-version packages under `internal/modules/backups/e2e` run them: 14 tests on PostgreSQL 17 and 12 on PostgreSQL 18, which skips the two SSH bastion scenarios as today. None of the new code needs the `testhelpers` build tag. The scenarios check the same outcomes as before, scenario by scenario.
- The scenarios prepare their data through the API, as a user does:
  - the users, the workspace and the local storage come from the `apptest` helpers;
  - the physical database is created with `createDatabase`, with its backup type and, for the bastion scenarios, its SSH tunnel;
  - the replication-only user comes from `createLimitedUser` followed by `updateDatabase`;
  - backups are enabled with `updateBackupConfig`.
- The operations the contract does not describe yet go through the old routes: starting an incremental, the backup list, the restore token, the WAL stream statistics, the restore stream and the recovery script. `apptest` gains a call of an old public route without a session, and a download of an old public route into a file.
- A test can ask `apptest` to run the old physical backup scheduler and WAL stream supervisor live for the lifetime of its application. A test also runs one pass of the old physical cleaner explicitly. This takes a small production change that keeps the behavior:
  - the old loop package gains exported constructors for the scheduler, the cleaner and the supervisor, and its singletons are built by them;
  - the cleaner gains an exported single pass, which is the body of its tick;
  - `internal/app` gains a method that starts a fresh scheduler and supervisor on a given context, and a method that runs one cleaner pass. `Serve` keeps using the singletons.
- The scenarios read the catalog through the old routes. A direct query on the internal database remains only for data the API does not return, and the design lists each query, because `physical-backups-module` renames those tables. The scenarios import nothing that the next two changes move: no `util/walmath`, no `engines/postgresql/physical`, no old DTO packages and no old test helpers.
- The CI test groups follow the move: the `physical` group lists the new per-version packages, and the `logical` group leaves them out.
- The Go suite under `backend/internal/features/tests/physical` is deleted once the new scenarios pass. Its `testdata/mtls` assets stay at their existing paths for the old mTLS usecase tests and their commit-specific gitleaks fingerprints. The tagged helpers that the old package tests still use stay until `physical-backups-module`.

- Repair the physical backup list so concurrent WAL uploads cannot merge ranges across an LSN gap. Deterministic API regression tests cover out-of-order receive times and filtering. This enforces the existing `physical-backup-wal-ranges` requirements.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `api-contract`: the requirement "Backend API tests run against the real application" allows one exception to "no background loops running on their own". A test may ask for the old physical backup scheduler and WAL stream supervisor to run live for the lifetime of its application, until they become background jobs of the backups module. They stop before the application closes its database, and so do the physical backups they started.

Backup creation, streaming, retention and restore keep their behavior. The list correctly exposes existing WAL gaps even when uploads arrive out of LSN order. The specs `physical-backups`, `physical-backup-wal-ranges` and `postgresql-physical-backup-failover` do not change, and the `ci-pipeline` spec already says that the `physical` group runs the end-to-end physical backup tests.

## Out of scope

- Any change to how physical backups, WAL streaming, retention, the restore stream or the recovery script behave.
- New end-to-end scenarios or stronger end-to-end assertions beyond what the old scenarios check. The focused WAL list regression tests are included. Where an API-based check replaces a catalog read, it checks the same outcome.
- Moving `pg_basebackup` and `pg_receivewal` onto `lib/process` and moving `util/walmath` into the engine. That is `engines-physical`.
- Moving the state, the loops, the tables and the old package tests into the `backups` module, and deleting the tagged physical test helpers. That is `physical-backups-module`, which also decides what the live-loops option becomes once the loops are `background.Runner` jobs.
- Describing the physical operations in the contract and moving the scenarios to the generated client. That is `physical-backups-api`.
- Apart from the focused WAL list regression tests, the old package tests of the physical code (`controller_test`, `wal_supervisor_test`, `cleaner_test`, `scheduler_test` and the tests of `usecases/physical/postgresql`) and the mTLS tests, which run in the `wal-usecases` group.

## Impact

- **Module rules:** the change answers to the root `AGENTS.md` and `backend/AGENTS.md`, and its tests follow the testing skill.
- **Backend, production code:**
  - `backend/internal/features/backups/backups/backuping/physical` (exported constructors and the cleaner's single pass);
  - `backend/internal/app` (starting the old physical loops on a context and running a cleaner pass).
  - `backend/internal/features/backups/backups/core/physical/service` (WAL ranges grouped by LSN continuity within their display boundaries).
- **Backend, test code, new or changed:**
  - the old physical controller package (deterministic WAL list regression tests);
  - `backend/internal/app/apptest` (the live-loops option, the cleaner pass, calls of old public routes);
  - `backend/internal/app/apptest/enginetest` (physical database settings, options for the team's application);
  - the new shared scenario package under `backend/internal/app/apptest`;
  - the new per-version packages under `backend/internal/modules/backups/e2e`.
- **Deleted:** the Go packages under `backend/internal/features/tests/physical`; their mTLS testdata remains for the old usecase tests.
- **CI:** `mise-tasks/backend/test-group-packages` and the test durations in `backend/test-shard-weights.txt`.
- **Not affected:** the schema, the contract, the frontend and backup execution; WAL list rows now preserve real gaps under concurrent uploads.
- **Documents:**
  - `docs/testing.md`, "The test application": the exception for the live physical loops;
  - `temp/refactor/plan.md`, which lives only in the main checkout: milestone 10a without "production code does not change" and without mTLS, and the relaxation row about the live loops.

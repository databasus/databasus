## 1. Image and standalone logical recovery

Run the commands below from the repository root with the pinned mise tools active. Before commands that execute PostgreSQL client tools on Fedora, prepare the existing shim in that shell:

```sh
source scripts/fedora.sh
apply_libedit_shim
```

- [x] 1.1 Resolve the index and amd64/arm64 manifests for `edoburu/pgbouncer:v1.26.1-p0` and register the immutable references in `backend/internal/lib/testimages/images.json`, without changing unrelated pins. Verify the catalog path with `go -C backend test -count=1 ./internal/lib/testimages`; the real image is exercised by task 1.2.
- [x] 1.2 Add the isolated, password-authenticated session-pooling fixture in `backend/internal/lib/testcontainers/pgbouncer.go` and the standalone `TestBackupAndRestorePostgresqlThroughPgBouncer` in a new logical `pgbouncer_test.go`. Use PostgreSQL 18 with no published source port, authenticated readiness, database-name passthrough, dynamic pooler port and shared cleanup. Back up several populated tables through the API, restore into a different database through PgBouncer with four jobs, and check table definitions, primary keys, every seeded row and the downloaded backup size. Reuse existing cycle/assertion helpers; leave existing tests unchanged. Verify with:

  ```sh
  TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout=20m -run '^TestBackupAndRestorePostgresqlThroughPgBouncer$' ./internal/modules/backups/e2e/postgresql
  ```

## 2. Standalone physical chain and point-in-time recovery

- [x] 2.1 Extend the new fixture to compose the existing replication-ready PostgreSQL request and to add/reload the API-created limited role's credentials before Databasus switches to them. Add physical orchestration in `backend/internal/app/apptest/physicaltest/pgbouncer.go` and the standalone `TestRestorePhysicalBackupThroughPgBouncer` in a new PG18 `pgbouncer_test.go`. Exercise `FULL_INCREMENTAL_WAL_STREAM` through PgBouncer with the limited replication role, one completed full backup, two completed incrementals and at least two distinct completed WAL segments after the second incremental. Check parent links and the actual restore-set selection of both incrementals and WAL. Reuse the existing chain builder, download and recovery script; boot the recovered cluster and assert that all four pre-target markers exist and `after-target` does not. Preserve existing physical test entry points and scenarios. Verify with:

  ```sh
  TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout=20m -run '^TestRestorePhysicalBackupThroughPgBouncer$' ./internal/modules/backups/e2e/postgresqlphysical/pg18
  ```

## 3. Regression checks and handoff

- [x] 3.1 Check the existing direct PG18 logical single-job and parallel restore scenarios, plus the existing direct PG18 physical chain and PITR scenarios, to verify that reuse of the shared helpers preserves their behavior. Run the commands below. If fixing a failure changes either new scenario, rerun its task's command too.

  ```sh
  TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout=20m -run '^TestBackupAndRestorePostgresql$/^PostgreSQL_18$/restores the rows with' ./internal/modules/backups/e2e/postgresql
  TESTCONTAINERS_RYUK_DISABLED=true go -C backend test -count=1 -timeout=20m -run '^(TestRestorePhysicalBackupChain|TestRestorePhysicalBackupToTargetTime)$/^recovers rows' ./internal/modules/backups/e2e/postgresqlphysical/pg18
  ```

- [x] 3.2 Run `mise run backend:format` and `mise run backend:lint`, inspect the diff for changes outside the new fixtures/tests/image pin, and fix any findings in the touched code. Record the commands and actual test outcomes in this change's artifacts, then finish the turn so the mandatory review hooks can review the implementation. Resolve blocking findings and leave the change ready to commit. No shared-documentation or agent-instruction edits are planned because the existing architecture and test procedures remain valid.

Verification is targeted: this change adds end-to-end scenarios and their test-support code, without adding or changing ordinary backend or frontend tests. It does not run the full test command.

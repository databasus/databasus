## Context

See `proposal.md` for motivation and scope. This design is needed because the change introduces a real external test dependency and must preserve authentication, replication semantics and fixture isolation.

- The logical suite already exposes a complete API-driven backup/restore cycle and row checks (`backend/internal/modules/backups/e2e/postgresql/engine_postgresql_test.go:1203` and `:1264`). Its version matrix remains direct (`:67`).
- The physical helpers already build one full backup and two incrementals, commit markers before and after a recovery target, and wait for archived WAL (`backend/internal/app/apptest/physicaltest/pitr_chain.go:10`). The existing recovery path downloads the restore set, reconstructs the cluster and boots it (`backend/internal/app/apptest/physicaltest/restore_target_time.go:11`).
- Physical setup creates a limited replication role through the API, changes the saved source credentials and checks that the role is not a superuser (`backend/internal/app/apptest/physicaltest/team.go:95`). PgBouncer must authenticate that newly created role too.
- Container fixtures can put PostgreSQL on an isolated network without publishing its port (`backend/internal/lib/testcontainers/postgres_physical.go:216`). Image selection already resolves immutable artifacts for the host architecture (`backend/internal/lib/testcontainers/image.go:11`).
- PgBouncer added replication support in [1.23.0](https://www.pgbouncer.org/changelog.html). Its upstream tests exercise both [pg_basebackup and pg_receivewal](https://github.com/pgbouncer/pgbouncer/blob/pgbouncer_1_26_1/test/test_replication.py). The [image publisher](https://hub.docker.com/r/edoburu/pgbouncer/tags) lists `v1.26.1-p0` for amd64 and arm64.

The change follows root `AGENTS.md`, `backend/AGENTS.md`, and the testing skill. Test composition stays in `internal/app/apptest`, engine-independent container support stays in `internal/lib/testcontainers`, and end-to-end entry points stay in the backup suites. This follows the engine and shared-code boundaries in `docs/architecture.md`; no production architecture changes are needed.

## Goals / Non-Goals

**Goals:**

- Make each PgBouncer scenario independently selectable by its own top-level Go test name and give it its own source, pooler and application.
- Ensure every Databasus source connection, and every logical restore connection, reaches PostgreSQL through the real PgBouncer endpoint.
- Establish success from restored data in a running database and from the physical artifacts actually selected for recovery.

**Non-Goals:**

- Changing the behavior or topology of existing tests. Their helpers may be reused without changing existing call sites.
- Building a configurable pooler framework or introducing pooler awareness into production code.
- Expanding the product support matrix or changing the physical recovery mechanism.

## Decisions

### 1. Add two standalone tests on PostgreSQL 18

Add `pgbouncer_test.go` in each existing suite directory:

- `backend/internal/modules/backups/e2e/postgresql/`: `TestBackupAndRestorePostgresqlThroughPgBouncer`.
- `backend/internal/modules/backups/e2e/postgresqlphysical/pg18/`: `TestRestorePhysicalBackupThroughPgBouncer`.

Each test has one descriptive `t.Run` scenario and uses `t.Parallel()` where independent. Existing runner discovery includes these packages, so no runner or CI changes are needed. Reuse the logical cycle helpers and physical chain/recovery helpers; place the new physical orchestration in `backend/internal/app/apptest/physicaltest/pgbouncer.go`.

PostgreSQL 18 is the newest major already represented in both suites. Physical incrementals are available from PostgreSQL 17; using 18 bounds the additional coverage and does not establish a new minimum PostgreSQL version.

**Alternatives:** Replacing existing direct scenarios would violate the requested separation and remove existing coverage. Running the new pair for every supported major would multiply the expensive recovery runs without isolating another PgBouncer behavior.

### 2. Use an isolated source and a real session-pooling PgBouncer

Add a focused fixture in `backend/internal/lib/testcontainers/pgbouncer.go`. It composes the existing PostgreSQL requests and network/container lifecycle helpers. The logical source uses the ordinary request; the physical source uses the replication-ready request with WAL summarization. Do not duplicate PostgreSQL startup configuration.

```text
Databasus / test SQL --> published PgBouncer port --> private PostgreSQL:5432

Logical restore     --> same PgBouncer endpoint --> new database

Physical restore set --> recovery script --> independent PostgreSQL target
```

The source has no published host port. PgBouncer joins its network and publishes a dynamically allocated loopback port. Configure `pool_mode=session`, database-name passthrough for source and dynamically created restore databases, and the route used by physical replication clients. Allow the lib/pq startup parameter `extra_float_digits` through the documented PgBouncer `ignore_startup_parameters` setting; do not ignore `replication`. Size the pool for four logical restore workers or the physical SQL and replication connections without a deliberately constrained pool.

Keep password authentication on both legs and forward the client user without a forced backend superuser. The fixture starts with the test administrator in its authentication file. In the physical test, create the limited role through the existing Databasus API, add its returned credentials to that file and reload PgBouncer before saving the limited credentials. Check the resulting role attributes as the existing physical setup does. Do not use `trust`, `auth_type=any`, or an ignored replication startup parameter to make connections succeed.

Use the shared lifecycle helpers for unique networks and cleanup, including failed starts. Wait for an authenticated SQL query through PgBouncer, rather than a fixed delay or an open TCP port. Preserve the shared log redaction when reporting fixture failures.

**Alternatives:** A TCP forwarder would not exercise PgBouncer protocol handling or authentication. Publishing a direct PostgreSQL port would make accidental bypass possible. [Transaction pooling](https://www.pgbouncer.org/features.html) has session-state limitations and would require a separate compatibility decision; these tests establish session-pooling behavior only.

### 3. Pin the published image for both architectures

Resolve `edoburu/pgbouncer:v1.26.1-p0` from its registry during implementation and add its index and platform digests to `backend/internal/lib/testimages/images.json`. Start it through the existing pinned-image path. Supply explicit fixture configuration rather than relying on image-specific generated authentication defaults.

**Alternatives:** A floating tag makes results depend on pull time. A custom PgBouncer build adds build tooling and maintenance when a small published multi-architecture image already exists. Architecture-specific skips would remove coverage from a supported test platform.

### 4. Verify logical recovery through the pooler with parallel restore jobs

The logical test creates several populated tables through PgBouncer, creates the Databasus database and backup configuration through the API, and waits for a completed backup. It restores that backup into a different database on the same source server, also through PgBouncer, with four jobs. Reuse the existing cycle helper so the API contract checks and backup pipeline stay real.

Verify the restored tables, their column definitions and primary keys, and compare all seeded rows with the source. Check that the backup is downloadable and its recorded size matches the download, using the existing assertion. Database-name passthrough must preserve the requested source and restore database names.

**Alternatives:** Testing only a connection or a successful backup status cannot establish restorability. A single restore job would miss the multi-connection path that a pooler must also handle. A new application or engine stub would skip the integration this test exists to exercise.

### 5. Verify a complete physical chain and WAL replay across the target

The physical test registers the PgBouncer endpoint with `FULL_INCREMENTAL_WAL_STREAM` and uses the limited replication role for Databasus. Test SQL uses the administrator through the same endpoint. Reuse `seedChainAndStreamPastTarget` for the following sequence:

1. Insert `before-full`, then complete the full backup.
2. Insert `after-full`, then complete the first incremental.
3. Insert `after-incr1`, then complete the second incremental.
4. Insert `before-target`, produce WAL beyond the second incremental and record the target time.
5. Insert `after-target`, produce more WAL, and wait until the required completed segments are published to storage.

Assert that the catalog contains a completed full backup and two completed incrementals with the correct parent links. Check that at least two distinct completed WAL segments after the second incremental have been published; a running receiver alone is insufficient. Inspect the downloaded archive entries and backup manifests to establish that the downloaded restore set uses that full backup, both incrementals and archived WAL reaching the target, rather than proving only full-plus-WAL recovery.

Download the restore set through the API and run the existing recovery script in the isolated restore target. Boot PostgreSQL and wait for recovery to finish. The recovered marker set must be exactly `before-full`, `after-full`, `after-incr1`, and `before-target`; `after-target` must be absent. Physical recovery reads downloaded artifacts rather than reconnecting through PgBouncer.

**Alternatives:** Full-only or incremental-only recovery misses the continuous WAL path. Checking only WAL catalog entries misses invalid or unusable archives. Repeating the existing recovery script or chain builder would create a second implementation of the same test machinery.

## Risks / Trade-offs

- PgBouncer database routing can differ between SQL, physical replication and a newly created restore database. Mitigation: use the actual Databasus connection paths and client tools against an unpublished source, with no direct-connect fallback.
- PgBouncer needs credentials for the API-created replication role. Mitigation: explicitly update and reload its authentication file before switching saved credentials; continue to assert the limited privileges.
- PgBouncer retains session-pooled backend connections after clients disconnect, which can prevent dropping a logical restore database. Mitigation: retire the affected pools before database cleanup, or let the isolated source container own their disposal; never reset an unrelated source.
- Extra recovery work increases test duration. Mitigation: add only the two requested scenarios on PG18, reuse the existing condition-based waits and let the runner govern concurrency.
- A passing pair establishes only the pinned PgBouncer release, session pooling and the selected PG18 paths. Mitigation: keep these limits explicit and avoid broader support claims.

## Migration Plan

No runtime deployment or data migration is needed. The change adds test fixtures, image pins and two tests. Removing those additions rolls it back without affecting user data or existing scenarios. Test-only behavior does not require delta specs; `skip_specs: true` records that decision.
